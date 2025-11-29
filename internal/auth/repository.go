package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// Repository defines the interface for auth data access operations
type Repository interface {
	// User operations
	GetUserByLoginID(ctx context.Context, loginID string) (*AuthUser, error)
	GetUserByEmail(ctx context.Context, email string) (*AuthUser, error)
	CreateUser(ctx context.Context, user *AuthUser) error
	GetUserInternalID(ctx context.Context, publicID string) (int, error)

	// Token operations
	CreateToken(ctx context.Context, token *UserToken) error
	GetTokenByHash(ctx context.Context, tokenHash string) (*UserToken, error)
	RevokeToken(ctx context.Context, tokenID int64) error
	RevokeAllUserTokens(ctx context.Context, userID int) error
	UpdateTokenReplacement(ctx context.Context, oldTokenID, newTokenID int64) error

	// Group operations
	GetGroupByPublicID(ctx context.Context, publicID string) (*Group, error)
	AddUserToGroup(ctx context.Context, userID, groupID int, assignedBy *int) error
	GetUserGroups(ctx context.Context, userID int) ([]Group, error)

	// Role operations
	GetUserRoles(ctx context.Context, userID int) ([]string, error)
	GetGroupRoles(ctx context.Context, groupID int) ([]string, error)
	GetAllUserRoles(ctx context.Context, userID int) ([]string, error)
}

type repository struct {
	db *sql.DB
}

// NewRepository creates a new auth repository
func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// GetUserByLoginID retrieves a user by their login ID
func (r *repository) GetUserByLoginID(ctx context.Context, loginID string) (*AuthUser, error) {
	query := `
		SELECT id, public_id, login_id, email, name, password_hash, is_deleted
		FROM users
		WHERE login_id = $1 AND is_deleted = false`

	user := &AuthUser{}
	var passwordHash sql.NullString
	err := r.db.QueryRowContext(ctx, query, loginID).Scan(
		&user.ID,
		&user.PublicID,
		&user.LoginID,
		&user.Email,
		&user.Name,
		&passwordHash,
		&user.IsDeleted,
	)
	if err != nil {
		return nil, err
	}
	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}
	return user, nil
}

// GetUserByEmail retrieves a user by their email
func (r *repository) GetUserByEmail(ctx context.Context, email string) (*AuthUser, error) {
	query := `
		SELECT id, public_id, login_id, email, name, password_hash, is_deleted
		FROM users
		WHERE email = $1 AND is_deleted = false`

	user := &AuthUser{}
	var passwordHash sql.NullString
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.PublicID,
		&user.LoginID,
		&user.Email,
		&user.Name,
		&passwordHash,
		&user.IsDeleted,
	)
	if err != nil {
		return nil, err
	}
	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}
	return user, nil
}

// CreateUser creates a new user and returns the created user with ID
func (r *repository) CreateUser(ctx context.Context, user *AuthUser) error {
	query := `
		INSERT INTO users (login_id, email, name, password_hash, is_visible, is_deleted)
		VALUES ($1, $2, $3, $4, true, false)
		RETURNING id, public_id`

	return r.db.QueryRowContext(ctx, query,
		user.LoginID,
		user.Email,
		user.Name,
		user.PasswordHash,
	).Scan(&user.ID, &user.PublicID)
}

// GetUserInternalID retrieves the internal ID from a public UUID
func (r *repository) GetUserInternalID(ctx context.Context, publicID string) (int, error) {
	var id int
	query := `SELECT id FROM users WHERE public_id = $1 AND is_deleted = false`
	err := r.db.QueryRowContext(ctx, query, publicID).Scan(&id)
	return id, err
}

// CreateToken stores a new refresh token
func (r *repository) CreateToken(ctx context.Context, token *UserToken) error {
	query := `
		INSERT INTO user_tokens (user_id, token_hash, expires_at, is_revoked, parent_token_id, client_ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.IsRevoked,
		token.ParentTokenID,
		token.ClientIP,
		token.UserAgent,
	).Scan(&token.ID, &token.CreatedAt, &token.UpdatedAt)
}

// GetTokenByHash retrieves a token by its hash
func (r *repository) GetTokenByHash(ctx context.Context, tokenHash string) (*UserToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, is_revoked, replaced_by_token_id, parent_token_id, client_ip, user_agent, created_at, updated_at
		FROM user_tokens
		WHERE token_hash = $1`

	token := &UserToken{}
	var clientIP, userAgent sql.NullString
	var replacedBy, parentID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.IsRevoked,
		&replacedBy,
		&parentID,
		&clientIP,
		&userAgent,
		&token.CreatedAt,
		&token.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if clientIP.Valid {
		token.ClientIP = &clientIP.String
	}
	if userAgent.Valid {
		token.UserAgent = &userAgent.String
	}
	if replacedBy.Valid {
		token.ReplacedByTokenID = &replacedBy.Int64
	}
	if parentID.Valid {
		token.ParentTokenID = &parentID.Int64
	}

	return token, nil
}

// RevokeToken marks a token as revoked
func (r *repository) RevokeToken(ctx context.Context, tokenID int64) error {
	query := `UPDATE user_tokens SET is_revoked = true, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, tokenID)
	return err
}

// RevokeAllUserTokens revokes all tokens for a user
func (r *repository) RevokeAllUserTokens(ctx context.Context, userID int) error {
	query := `UPDATE user_tokens SET is_revoked = true, updated_at = NOW() WHERE user_id = $1 AND is_revoked = false`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// UpdateTokenReplacement marks a token as replaced by a new token
func (r *repository) UpdateTokenReplacement(ctx context.Context, oldTokenID, newTokenID int64) error {
	query := `UPDATE user_tokens SET replaced_by_token_id = $1, is_revoked = true, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, newTokenID, oldTokenID)
	return err
}

// GetGroupByPublicID retrieves a group by its public ID
func (r *repository) GetGroupByPublicID(ctx context.Context, publicID string) (*Group, error) {
	query := `
		SELECT id, public_id, name, description, created_at, updated_at
		FROM groups
		WHERE public_id = $1`

	group := &Group{}
	err := r.db.QueryRowContext(ctx, query, publicID).Scan(
		&group.ID,
		&group.PublicID,
		&group.Name,
		&group.Description,
		&group.CreatedAt,
		&group.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return group, nil
}

// AddUserToGroup adds a user to a group
func (r *repository) AddUserToGroup(ctx context.Context, userID, groupID int, assignedBy *int) error {
	query := `
		INSERT INTO group_users (group_id, user_id, assigned_by, assigned_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (group_id, user_id) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query, groupID, userID, assignedBy)
	return err
}

// GetUserGroups retrieves all groups a user belongs to
func (r *repository) GetUserGroups(ctx context.Context, userID int) ([]Group, error) {
	query := `
		SELECT g.id, g.public_id, g.name, g.description, g.created_at, g.updated_at
		FROM groups g
		INNER JOIN group_users gu ON g.id = gu.group_id
		WHERE gu.user_id = $1`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var group Group
		if err := rows.Scan(
			&group.ID,
			&group.PublicID,
			&group.Name,
			&group.Description,
			&group.CreatedAt,
			&group.UpdatedAt,
		); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}

	return groups, rows.Err()
}

// GetUserRoles retrieves direct roles assigned to a user
func (r *repository) GetUserRoles(ctx context.Context, userID int) ([]string, error) {
	query := `
		SELECT r.name
		FROM roles r
		INNER JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	return roles, rows.Err()
}

// GetGroupRoles retrieves roles assigned to a group
func (r *repository) GetGroupRoles(ctx context.Context, groupID int) ([]string, error) {
	query := `
		SELECT r.name
		FROM roles r
		INNER JOIN group_roles gr ON r.id = gr.role_id
		WHERE gr.group_id = $1`

	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	return roles, rows.Err()
}

// GetAllUserRoles retrieves all roles for a user including inherited group roles (deduplicated)
func (r *repository) GetAllUserRoles(ctx context.Context, userID int) ([]string, error) {
	query := `
		SELECT DISTINCT r.name
		FROM roles r
		WHERE r.id IN (
			-- Direct user roles
			SELECT role_id FROM user_roles WHERE user_id = $1
			UNION
			-- Group roles (inherited through group membership)
			SELECT gr.role_id
			FROM group_roles gr
			INNER JOIN group_users gu ON gr.group_id = gu.group_id
			WHERE gu.user_id = $1
		)
		ORDER BY r.name`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	return roles, rows.Err()
}

// GetUserByID retrieves a user by their internal ID
func (r *repository) GetUserByID(ctx context.Context, userID int) (*AuthUser, error) {
	query := `
		SELECT id, public_id, login_id, email, name, password_hash, is_deleted
		FROM users
		WHERE id = $1 AND is_deleted = false`

	user := &AuthUser{}
	var passwordHash sql.NullString
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.PublicID,
		&user.LoginID,
		&user.Email,
		&user.Name,
		&passwordHash,
		&user.IsDeleted,
	)
	if err != nil {
		return nil, err
	}
	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}
	return user, nil
}

// ToUserInfo converts AuthUser to UserInfo
func (u *AuthUser) ToUserInfo() UserInfo {
	return UserInfo{
		ID:      u.PublicID,
		LoginID: u.LoginID,
		Name:    u.Name,
		Email:   u.Email,
	}
}

// Helper function to check if name has at least one locale
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

// Helper function to get current time
func now() time.Time {
	return time.Now()
}
