package users

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailExists       = errors.New("email already exists")
	ErrLoginIDExists     = errors.New("login_id already exists")
	ErrInvalidEmail      = errors.New("invalid email format")
	ErrInvalidName       = errors.New("name must have at least one locale value")
	ErrEncryptionFailed  = errors.New("encryption failed")
	ErrDecryptionFailed  = errors.New("decryption failed")
)

// Argon2 parameters
const (
	argon2Time    = 1
	argon2Memory  = 64 * 1024
	argon2Threads = 4
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

// Service defines the interface for user business logic
type Service interface {
	Create(ctx context.Context, req *CreateUserRequest) (*UserListResponse, error)
	GetByID(ctx context.Context, publicID string) (*UserDetailResponse, error)
	List(ctx context.Context, page, limit int) (*UserListResponseWrapper, error)
	Update(ctx context.Context, publicID string, req *UpdateUserRequest) (*UserListResponse, error)
	Delete(ctx context.Context, publicID string) error
	Search(ctx context.Context, criteria *SearchUserRequest, page, limit int) (*UserListResponseWrapper, error)
}

type service struct {
	repo          Repository
	encryptionKey []byte
}

// NewService creates a new user service with the given repository and encryption key
func NewService(repo Repository, encryptionKey string) Service {
	key := sha256.Sum256([]byte(encryptionKey))
	return &service{
		repo:          repo,
		encryptionKey: key[:],
	}
}

// Create creates a new user
func (s *service) Create(ctx context.Context, req *CreateUserRequest) (*UserListResponse, error) {
	// Validate email
	if !isValidEmail(req.Email) {
		return nil, ErrInvalidEmail
	}

	// Validate name has at least one locale
	if !hasAtLeastOneLocale(req.Name) {
		return nil, ErrInvalidName
	}

	// Check if email already exists
	exists, err := s.repo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		return nil, ErrEmailExists
	}

	// Set login_id to email if not provided
	loginID := req.Email
	if req.LoginID != nil && *req.LoginID != "" {
		loginID = *req.LoginID
	}

	// Check if login_id already exists
	exists, err = s.repo.ExistsByLoginID(ctx, loginID)
	if err != nil {
		return nil, fmt.Errorf("failed to check login_id existence: %w", err)
	}
	if exists {
		return nil, ErrLoginIDExists
	}

	user := &User{
		LoginID:   loginID,
		Name:      req.Name,
		Email:     req.Email,
		IsVisible: true,
		IsDeleted: false,
	}

	// Set optional ID fields
	if req.DeptID != nil {
		user.DeptID = sql.NullInt64{Int64: *req.DeptID, Valid: true}
	}
	if req.RankID != nil {
		user.RankID = sql.NullInt64{Int64: *req.RankID, Valid: true}
	}
	if req.DutyID != nil {
		user.DutyID = sql.NullInt64{Int64: *req.DutyID, Valid: true}
	}
	if req.TitleID != nil {
		user.TitleID = sql.NullInt64{Int64: *req.TitleID, Valid: true}
	}
	if req.PositionID != nil {
		user.PositionID = sql.NullInt64{Int64: *req.PositionID, Valid: true}
	}
	if req.LocationID != nil {
		user.LocationID = sql.NullInt64{Int64: *req.LocationID, Valid: true}
	}

	// Handle contact_mobile encryption
	if req.ContactMobile != nil && *req.ContactMobile != "" {
		encrypted, err := s.encryptAESGCM(*req.ContactMobile)
		if err != nil {
			return nil, ErrEncryptionFailed
		}
		user.ContactMobile = sql.NullString{String: encrypted, Valid: true}
		user.ContactMobileHash = sql.NullString{String: s.hashSHA256(*req.ContactMobile), Valid: true}
		user.ContactMobileID = sql.NullString{String: getLast4Digits(*req.ContactMobile), Valid: true}
	}

	// Handle contact_office encryption
	if req.ContactOffice != nil && *req.ContactOffice != "" {
		encrypted, err := s.encryptAESGCM(*req.ContactOffice)
		if err != nil {
			return nil, ErrEncryptionFailed
		}
		user.ContactOffice = sql.NullString{String: encrypted, Valid: true}
		user.ContactOfficeHash = sql.NullString{String: s.hashSHA256(*req.ContactOffice), Valid: true}
		user.ContactOfficeID = sql.NullString{String: getLast4Digits(*req.ContactOffice), Valid: true}
	}

	// Handle password hashing
	if req.Password != nil && *req.Password != "" {
		hashedPassword, err := s.hashPassword(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		user.PasswordHash = sql.NullString{String: hashedPassword, Valid: true}
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	response := user.ToListResponse()
	return &response, nil
}

// GetByID retrieves a user by their public ID with full details
func (s *service) GetByID(ctx context.Context, publicID string) (*UserDetailResponse, error) {
	detail, err := s.repo.GetDetailByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return detail, nil
}

// List retrieves a paginated list of users
func (s *service) List(ctx context.Context, page, limit int) (*UserListResponseWrapper, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	users, totalCount, err := s.repo.List(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	var responses []UserListResponse
	for _, user := range users {
		responses = append(responses, user.ToListResponse())
	}

	totalPages := (totalCount + limit - 1) / limit

	return &UserListResponseWrapper{
		Data:       responses,
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

// Update updates an existing user
func (s *service) Update(ctx context.Context, publicID string, req *UpdateUserRequest) (*UserListResponse, error) {
	// Get existing user
	existingUser, err := s.repo.GetByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Update fields if provided
	if req.LoginID != nil && *req.LoginID != "" && *req.LoginID != existingUser.LoginID {
		exists, err := s.repo.ExistsByLoginID(ctx, *req.LoginID)
		if err != nil {
			return nil, fmt.Errorf("failed to check login_id existence: %w", err)
		}
		if exists {
			return nil, ErrLoginIDExists
		}
		existingUser.LoginID = *req.LoginID
	}

	if req.Name != nil {
		if !hasAtLeastOneLocale(*req.Name) {
			return nil, ErrInvalidName
		}
		existingUser.Name = *req.Name
	}

	if req.Email != nil && *req.Email != "" && *req.Email != existingUser.Email {
		if !isValidEmail(*req.Email) {
			return nil, ErrInvalidEmail
		}
		exists, err := s.repo.ExistsByEmail(ctx, *req.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to check email existence: %w", err)
		}
		if exists {
			return nil, ErrEmailExists
		}
		existingUser.Email = *req.Email
	}

	// Update ID fields
	if req.DeptID != nil {
		existingUser.DeptID = sql.NullInt64{Int64: *req.DeptID, Valid: true}
	}
	if req.RankID != nil {
		existingUser.RankID = sql.NullInt64{Int64: *req.RankID, Valid: true}
	}
	if req.DutyID != nil {
		existingUser.DutyID = sql.NullInt64{Int64: *req.DutyID, Valid: true}
	}
	if req.TitleID != nil {
		existingUser.TitleID = sql.NullInt64{Int64: *req.TitleID, Valid: true}
	}
	if req.PositionID != nil {
		existingUser.PositionID = sql.NullInt64{Int64: *req.PositionID, Valid: true}
	}
	if req.LocationID != nil {
		existingUser.LocationID = sql.NullInt64{Int64: *req.LocationID, Valid: true}
	}

	// Handle contact_mobile update
	if req.ContactMobile != nil {
		if *req.ContactMobile != "" {
			encrypted, err := s.encryptAESGCM(*req.ContactMobile)
			if err != nil {
				return nil, ErrEncryptionFailed
			}
			existingUser.ContactMobile = sql.NullString{String: encrypted, Valid: true}
			existingUser.ContactMobileHash = sql.NullString{String: s.hashSHA256(*req.ContactMobile), Valid: true}
			existingUser.ContactMobileID = sql.NullString{String: getLast4Digits(*req.ContactMobile), Valid: true}
		} else {
			existingUser.ContactMobile = sql.NullString{}
			existingUser.ContactMobileHash = sql.NullString{}
			existingUser.ContactMobileID = sql.NullString{}
		}
	}

	// Handle contact_office update
	if req.ContactOffice != nil {
		if *req.ContactOffice != "" {
			encrypted, err := s.encryptAESGCM(*req.ContactOffice)
			if err != nil {
				return nil, ErrEncryptionFailed
			}
			existingUser.ContactOffice = sql.NullString{String: encrypted, Valid: true}
			existingUser.ContactOfficeHash = sql.NullString{String: s.hashSHA256(*req.ContactOffice), Valid: true}
			existingUser.ContactOfficeID = sql.NullString{String: getLast4Digits(*req.ContactOffice), Valid: true}
		} else {
			existingUser.ContactOffice = sql.NullString{}
			existingUser.ContactOfficeHash = sql.NullString{}
			existingUser.ContactOfficeID = sql.NullString{}
		}
	}

	// Handle password update
	if req.Password != nil && *req.Password != "" {
		hashedPassword, err := s.hashPassword(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		existingUser.PasswordHash = sql.NullString{String: hashedPassword, Valid: true}
	}

	// Handle visibility update
	if req.IsVisible != nil {
		existingUser.IsVisible = *req.IsVisible
	}

	if err := s.repo.Update(ctx, publicID, existingUser); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	response := existingUser.ToListResponse()
	return &response, nil
}

// Delete performs a soft delete on a user
func (s *service) Delete(ctx context.Context, publicID string) error {
	if err := s.repo.Delete(ctx, publicID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// Search searches for users based on criteria
func (s *service) Search(ctx context.Context, criteria *SearchUserRequest, page, limit int) (*UserListResponseWrapper, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Hash mobile/office numbers for full search
	searchCriteria := &SearchUserRequest{
		Name:        criteria.Name,
		Email:       criteria.Email,
		MobileLast4: criteria.MobileLast4,
		OfficeLast4: criteria.OfficeLast4,
	}

	if criteria.MobileFull != nil && *criteria.MobileFull != "" {
		hash := s.hashSHA256(*criteria.MobileFull)
		searchCriteria.MobileFull = &hash
	}

	if criteria.OfficeFull != nil && *criteria.OfficeFull != "" {
		hash := s.hashSHA256(*criteria.OfficeFull)
		searchCriteria.OfficeFull = &hash
	}

	users, totalCount, err := s.repo.Search(ctx, searchCriteria, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	var responses []UserListResponse
	for _, user := range users {
		responses = append(responses, user.ToListResponse())
	}

	totalPages := (totalCount + limit - 1) / limit

	return &UserListResponseWrapper{
		Data:       responses,
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

// encryptAESGCM encrypts plaintext using AES-256-GCM
func (s *service) encryptAESGCM(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptAESGCM decrypts ciphertext using AES-256-GCM
func (s *service) decryptAESGCM(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrDecryptionFailed
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// hashSHA256 creates a SHA-256 hash of the input
func (s *service) hashSHA256(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// hashPassword hashes a password using Argon2id
func (s *service) hashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Format: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argon2Memory, argon2Time, argon2Threads, b64Salt, b64Hash), nil
}

// getLast4Digits extracts the last 4 digits from a phone number
func getLast4Digits(phone string) string {
	// Remove all non-digit characters
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)

	if len(digits) < 4 {
		return digits
	}
	return digits[len(digits)-4:]
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	atIndex := strings.LastIndex(email, "@")
	if atIndex < 1 || atIndex >= len(email)-1 {
		return false
	}
	return strings.Contains(email[atIndex:], ".")
}

// hasAtLeastOneLocale checks if the JSON has at least one key-value pair
func hasAtLeastOneLocale(jsonData json.RawMessage) bool {
	if len(jsonData) == 0 {
		return false
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return false
	}

	return len(data) > 0
}
