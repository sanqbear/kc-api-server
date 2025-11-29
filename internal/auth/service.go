package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUserNotFound        = errors.New("user not found")
	ErrEmailExists         = errors.New("email already exists")
	ErrLoginIDExists       = errors.New("login_id already exists")
	ErrInvalidEmail        = errors.New("invalid email format")
	ErrInvalidName         = errors.New("name must have at least one locale value")
	ErrInvalidPassword     = errors.New("password must be at least 8 characters")
	ErrInvalidToken        = errors.New("invalid or expired token")
	ErrTokenRevoked        = errors.New("token has been revoked")
	ErrTokenExpired        = errors.New("token has expired")
	ErrPublicGroupNotFound = errors.New("public group not found")
)

// Argon2 parameters (must match users service)
const (
	argon2Time    = 1
	argon2Memory  = 64 * 1024
	argon2Threads = 4
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

// Service defines the interface for authentication business logic
type Service interface {
	Register(ctx context.Context, req *RegisterRequest, clientIP, userAgent string) (*RegisterResponse, string, error)
	Login(ctx context.Context, req *LoginRequest, clientIP, userAgent string) (*LoginResponse, string, error)
	Refresh(ctx context.Context, refreshToken, clientIP, userAgent string) (*TokenResponse, string, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID string) error
	GetMe(ctx context.Context, userID string) (*MeResponse, error)
	ValidateAccessToken(tokenString string) (*TokenClaims, error)
}

type service struct {
	repo      Repository
	jwtSecret []byte
}

// NewService creates a new auth service
func NewService(repo Repository, jwtSecret string) Service {
	return &service{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
	}
}

// Register creates a new user account and returns tokens
func (s *service) Register(ctx context.Context, req *RegisterRequest, clientIP, userAgent string) (*RegisterResponse, string, error) {
	// Validate email
	if !isValidEmail(req.Email) {
		return nil, "", ErrInvalidEmail
	}

	// Validate name
	if !hasAtLeastOneLocale(req.Name) {
		return nil, "", ErrInvalidName
	}

	// Validate password
	if len(req.Password) < 8 {
		return nil, "", ErrInvalidPassword
	}

	// Check if email already exists
	_, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err == nil {
		return nil, "", ErrEmailExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, "", fmt.Errorf("failed to check email: %w", err)
	}

	// Set login_id to email if not provided
	loginID := req.Email
	if req.LoginID != nil && *req.LoginID != "" {
		loginID = *req.LoginID
	}

	// Check if login_id already exists
	_, err = s.repo.GetUserByLoginID(ctx, loginID)
	if err == nil {
		return nil, "", ErrLoginIDExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, "", fmt.Errorf("failed to check login_id: %w", err)
	}

	// Hash password
	passwordHash, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &AuthUser{
		LoginID:      loginID,
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: passwordHash,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, "", fmt.Errorf("failed to create user: %w", err)
	}

	// Add user to 'public' group
	publicGroup, err := s.repo.GetGroupByPublicID(ctx, PublicGroupID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", ErrPublicGroupNotFound
		}
		return nil, "", fmt.Errorf("failed to get public group: %w", err)
	}

	if err := s.repo.AddUserToGroup(ctx, user.ID, publicGroup.ID, nil); err != nil {
		return nil, "", fmt.Errorf("failed to add user to public group: %w", err)
	}

	// Get user roles for token
	roles, err := s.repo.GetAllUserRoles(ctx, user.ID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user roles: %w", err)
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(user, roles)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token
	if err := s.storeRefreshToken(ctx, user.ID, refreshToken, nil, clientIP, userAgent); err != nil {
		return nil, "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &RegisterResponse{
		User: user.ToUserInfo(),
		Tokens: TokenResponse{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresIn:   int64(AccessTokenDuration.Seconds()),
		},
		Message: "User registered successfully",
	}, refreshToken, nil
}

// Login authenticates a user and returns tokens
func (s *service) Login(ctx context.Context, req *LoginRequest, clientIP, userAgent string) (*LoginResponse, string, error) {
	// Get user by login_id (can be email or login_id)
	user, err := s.repo.GetUserByLoginID(ctx, req.LoginID)
	if errors.Is(err, sql.ErrNoRows) {
		// Try by email
		user, err = s.repo.GetUserByEmail(ctx, req.LoginID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", ErrInvalidCredentials
		}
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	if user.PasswordHash == "" {
		return nil, "", ErrInvalidCredentials
	}

	if !s.verifyPassword(req.Password, user.PasswordHash) {
		return nil, "", ErrInvalidCredentials
	}

	// Get user roles for token
	roles, err := s.repo.GetAllUserRoles(ctx, user.ID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user roles: %w", err)
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(user, roles)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token
	if err := s.storeRefreshToken(ctx, user.ID, refreshToken, nil, clientIP, userAgent); err != nil {
		return nil, "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &LoginResponse{
		User: user.ToUserInfo(),
		Tokens: TokenResponse{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresIn:   int64(AccessTokenDuration.Seconds()),
		},
	}, refreshToken, nil
}

// Refresh generates new tokens using a valid refresh token
func (s *service) Refresh(ctx context.Context, refreshToken, clientIP, userAgent string) (*TokenResponse, string, error) {
	// Hash the refresh token
	tokenHash := s.hashToken(refreshToken)

	// Get stored token
	storedToken, err := s.repo.GetTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", ErrInvalidToken
		}
		return nil, "", fmt.Errorf("failed to get token: %w", err)
	}

	// Check if token is revoked
	if storedToken.IsRevoked {
		// Token reuse detected - revoke all user tokens
		_ = s.repo.RevokeAllUserTokens(ctx, storedToken.UserID)
		return nil, "", ErrTokenRevoked
	}

	// Check if token is expired
	if time.Now().After(storedToken.ExpiresAt) {
		return nil, "", ErrTokenExpired
	}

	// Get user by internal ID
	user, err := s.getUserByInternalID(ctx, storedToken.UserID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user: %w", err)
	}

	// Get user roles for token
	roles, err := s.repo.GetAllUserRoles(ctx, user.ID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user roles: %w", err)
	}

	// Generate new tokens
	newAccessToken, err := s.generateAccessToken(user, roles)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := s.generateRefreshToken()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store new refresh token with parent reference
	parentID := storedToken.ID
	if err := s.storeRefreshToken(ctx, user.ID, newRefreshToken, &parentID, clientIP, userAgent); err != nil {
		return nil, "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Get new token ID
	newTokenHash := s.hashToken(newRefreshToken)
	newStoredToken, err := s.repo.GetTokenByHash(ctx, newTokenHash)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get new token: %w", err)
	}

	// Mark old token as replaced
	if err := s.repo.UpdateTokenReplacement(ctx, storedToken.ID, newStoredToken.ID); err != nil {
		return nil, "", fmt.Errorf("failed to update token replacement: %w", err)
	}

	return &TokenResponse{
		AccessToken: newAccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(AccessTokenDuration.Seconds()),
	}, newRefreshToken, nil
}

// Logout revokes the current refresh token
func (s *service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := s.hashToken(refreshToken)

	storedToken, err := s.repo.GetTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil // Token not found, consider it already logged out
		}
		return fmt.Errorf("failed to get token: %w", err)
	}

	return s.repo.RevokeToken(ctx, storedToken.ID)
}

// LogoutAll revokes all refresh tokens for a user
func (s *service) LogoutAll(ctx context.Context, userPublicID string) error {
	userID, err := s.repo.GetUserInternalID(ctx, userPublicID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	return s.repo.RevokeAllUserTokens(ctx, userID)
}

// GetMe returns the current user information and roles
func (s *service) GetMe(ctx context.Context, userPublicID string) (*MeResponse, error) {
	userID, err := s.repo.GetUserInternalID(ctx, userPublicID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user, err := s.getUserByInternalID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user details: %w", err)
	}

	roles, err := s.repo.GetAllUserRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	return &MeResponse{
		User:  user.ToUserInfo(),
		Roles: roles,
	}, nil
}

// ValidateAccessToken validates and parses an access token
func (s *service) ValidateAccessToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	// Extract claims
	tokenClaims := &TokenClaims{
		Issuer: TokenIssuer,
	}

	if userID, ok := claims["user_id"].(string); ok {
		tokenClaims.UserID = userID
	}
	if loginID, ok := claims["login_id"].(string); ok {
		tokenClaims.LoginID = loginID
	}
	if email, ok := claims["email"].(string); ok {
		tokenClaims.Email = email
	}
	if jti, ok := claims["jti"].(string); ok {
		tokenClaims.TokenID = jti
	}
	if iat, ok := claims["iat"].(float64); ok {
		tokenClaims.IssuedAt = int64(iat)
	}
	if exp, ok := claims["exp"].(float64); ok {
		tokenClaims.ExpireAt = int64(exp)
	}
	if rolesInterface, ok := claims["roles"].([]interface{}); ok {
		for _, r := range rolesInterface {
			if role, ok := r.(string); ok {
				tokenClaims.Roles = append(tokenClaims.Roles, role)
			}
		}
	}

	return tokenClaims, nil
}

// generateAccessToken creates a new JWT access token
func (s *service) generateAccessToken(user *AuthUser, roles []string) (string, error) {
	now := time.Now()
	tokenID, err := s.generateTokenID()
	if err != nil {
		return "", err
	}

	claims := jwt.MapClaims{
		"user_id":  user.PublicID,
		"login_id": user.LoginID,
		"email":    user.Email,
		"roles":    roles,
		"jti":      tokenID,
		"iat":      now.Unix(),
		"exp":      now.Add(AccessTokenDuration).Unix(),
		"iss":      TokenIssuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// generateRefreshToken creates a secure random refresh token
func (s *service) generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// generateTokenID creates a unique token ID
func (s *service) generateTokenID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashToken creates a SHA-256 hash of the token for storage
func (s *service) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// storeRefreshToken stores a refresh token in the database
func (s *service) storeRefreshToken(ctx context.Context, userID int, token string, parentTokenID *int64, clientIP, userAgent string) error {
	tokenHash := s.hashToken(token)

	userToken := &UserToken{
		UserID:        userID,
		TokenHash:     tokenHash,
		ExpiresAt:     time.Now().Add(RefreshTokenDuration),
		IsRevoked:     false,
		ParentTokenID: parentTokenID,
	}

	if clientIP != "" {
		userToken.ClientIP = &clientIP
	}
	if userAgent != "" {
		userToken.UserAgent = &userAgent
	}

	return s.repo.CreateToken(ctx, userToken)
}

// hashPassword hashes a password using Argon2id
func (s *service) hashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argon2Memory, argon2Time, argon2Threads, b64Salt, b64Hash), nil
}

// verifyPassword verifies a password against an Argon2id hash
func (s *service) verifyPassword(password, encodedHash string) bool {
	// Parse the encoded hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false
	}

	var version int
	var memory, time uint32
	var threads uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false
	}
	_, err = fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	// Compute hash with the same parameters
	computedHash := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(decodedHash)))

	// Constant-time comparison
	return subtle.ConstantTimeCompare(decodedHash, computedHash) == 1
}

// getUserByInternalID retrieves a user by internal ID using repository interface
func (s *service) getUserByInternalID(ctx context.Context, userID int) (*AuthUser, error) {
	// Use the repository's GetUserByID method
	repo, ok := s.repo.(*repository)
	if !ok {
		return nil, errors.New("invalid repository type")
	}
	return repo.GetUserByID(ctx, userID)
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
