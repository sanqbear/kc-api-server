package users

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// Repository defines the interface for user data access operations
type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByPublicID(ctx context.Context, publicID string) (*User, error)
	GetDetailByPublicID(ctx context.Context, publicID string) (*UserDetailResponse, error)
	List(ctx context.Context, page, limit int) ([]User, int, error)
	Update(ctx context.Context, publicID string, user *User) error
	Delete(ctx context.Context, publicID string) error
	Search(ctx context.Context, criteria *SearchUserRequest, page, limit int) ([]User, int, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByLoginID(ctx context.Context, loginID string) (bool, error)
}

type repository struct {
	db *sql.DB
}

// NewRepository creates a new user repository
func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// Create inserts a new user into the database
func (r *repository) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO organizations.users (
			login_id, name, email,
			dept_id, rank_id, duty_id, title_id, position_id, location_id,
			contact_mobile, contact_mobile_hash, contact_mobile_id,
			contact_office, contact_office_hash, contact_office_id,
			password_hash, is_visible, is_deleted
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		) RETURNING id, public_id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		user.LoginID,
		user.Name,
		user.Email,
		user.DeptID,
		user.RankID,
		user.DutyID,
		user.TitleID,
		user.PositionID,
		user.LocationID,
		user.ContactMobile,
		user.ContactMobileHash,
		user.ContactMobileID,
		user.ContactOffice,
		user.ContactOfficeHash,
		user.ContactOfficeID,
		user.PasswordHash,
		user.IsVisible,
		user.IsDeleted,
	).Scan(&user.ID, &user.PublicID, &user.CreatedAt, &user.UpdatedAt)
}

// GetByPublicID retrieves a user by their public UUID
func (r *repository) GetByPublicID(ctx context.Context, publicID string) (*User, error) {
	query := `
		SELECT id, public_id, login_id, name, email,
			dept_id, rank_id, duty_id, title_id, position_id, location_id,
			contact_mobile, contact_mobile_hash, contact_mobile_id,
			contact_office, contact_office_hash, contact_office_id,
			created_at, updated_at, password_hash, is_visible, is_deleted
		FROM organizations.users
		WHERE public_id = $1 AND is_deleted = false`

	user := &User{}
	err := r.db.QueryRowContext(ctx, query, publicID).Scan(
		&user.ID,
		&user.PublicID,
		&user.LoginID,
		&user.Name,
		&user.Email,
		&user.DeptID,
		&user.RankID,
		&user.DutyID,
		&user.TitleID,
		&user.PositionID,
		&user.LocationID,
		&user.ContactMobile,
		&user.ContactMobileHash,
		&user.ContactMobileID,
		&user.ContactOffice,
		&user.ContactOfficeHash,
		&user.ContactOfficeID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.PasswordHash,
		&user.IsVisible,
		&user.IsDeleted,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetDetailByPublicID retrieves detailed user information including related names
func (r *repository) GetDetailByPublicID(ctx context.Context, publicID string) (*UserDetailResponse, error) {
	query := `
		SELECT
			u.public_id,
			u.login_id,
			u.name,
			u.email,
			COALESCE(d.name, '{}')::jsonb as dept_name,
			COALESCE(r.name, '{}')::jsonb as rank_name,
			COALESCE(du.name, '{}')::jsonb as duty_name,
			COALESCE(t.name, '{}')::jsonb as title_name,
			COALESCE(p.name, '{}')::jsonb as position_name,
			COALESCE(l.name, '{}')::jsonb as location_name,
			u.contact_mobile_id,
			u.contact_office_id
		FROM organizations.users u
		LEFT JOIN organizations.departments d ON u.dept_id = d.id
		LEFT JOIN organizations.common_codes r ON u.rank_id = r.id
		LEFT JOIN organizations.common_codes du ON u.duty_id = du.id
		LEFT JOIN organizations.common_codes t ON u.title_id = t.id
		LEFT JOIN organizations.common_codes p ON u.position_id = p.id
		LEFT JOIN organizations.common_codes l ON u.location_id = l.id
		WHERE u.public_id = $1 AND u.is_deleted = false`

	var mobileID, officeID sql.NullString
	detail := &UserDetailResponse{}
	err := r.db.QueryRowContext(ctx, query, publicID).Scan(
		&detail.ID,
		&detail.LoginID,
		&detail.Name,
		&detail.Email,
		&detail.DeptName,
		&detail.RankName,
		&detail.DutyName,
		&detail.TitleName,
		&detail.PositionName,
		&detail.LocationName,
		&mobileID,
		&officeID,
	)
	if err != nil {
		return nil, err
	}

	// Format contact information as masked
	if mobileID.Valid && mobileID.String != "" {
		detail.ContactMobile = fmt.Sprintf("***-****-%s", mobileID.String)
	}
	if officeID.Valid && officeID.String != "" {
		detail.ContactOffice = fmt.Sprintf("***-****-%s", officeID.String)
	}

	return detail, nil
}

// List retrieves a paginated list of users
func (r *repository) List(ctx context.Context, page, limit int) ([]User, int, error) {
	offset := (page - 1) * limit

	// Get total count
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM organizations.users WHERE is_deleted = false`
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// Get users
	query := `
		SELECT id, public_id, login_id, name, email,
			dept_id, rank_id, duty_id, title_id, position_id, location_id,
			contact_mobile, contact_mobile_hash, contact_mobile_id,
			contact_office, contact_office_hash, contact_office_id,
			created_at, updated_at, password_hash, is_visible, is_deleted
		FROM organizations.users
		WHERE is_deleted = false
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(
			&user.ID,
			&user.PublicID,
			&user.LoginID,
			&user.Name,
			&user.Email,
			&user.DeptID,
			&user.RankID,
			&user.DutyID,
			&user.TitleID,
			&user.PositionID,
			&user.LocationID,
			&user.ContactMobile,
			&user.ContactMobileHash,
			&user.ContactMobileID,
			&user.ContactOffice,
			&user.ContactOfficeHash,
			&user.ContactOfficeID,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.PasswordHash,
			&user.IsVisible,
			&user.IsDeleted,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, totalCount, nil
}

// Update updates an existing user
func (r *repository) Update(ctx context.Context, publicID string, user *User) error {
	query := `
		UPDATE organizations.users SET
			login_id = $1,
			name = $2,
			email = $3,
			dept_id = $4,
			rank_id = $5,
			duty_id = $6,
			title_id = $7,
			position_id = $8,
			location_id = $9,
			contact_mobile = $10,
			contact_mobile_hash = $11,
			contact_mobile_id = $12,
			contact_office = $13,
			contact_office_hash = $14,
			contact_office_id = $15,
			password_hash = $16,
			is_visible = $17,
			updated_at = NOW()
		WHERE public_id = $18 AND is_deleted = false`

	result, err := r.db.ExecContext(ctx, query,
		user.LoginID,
		user.Name,
		user.Email,
		user.DeptID,
		user.RankID,
		user.DutyID,
		user.TitleID,
		user.PositionID,
		user.LocationID,
		user.ContactMobile,
		user.ContactMobileHash,
		user.ContactMobileID,
		user.ContactOffice,
		user.ContactOfficeHash,
		user.ContactOfficeID,
		user.PasswordHash,
		user.IsVisible,
		publicID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Delete performs a soft delete on a user
func (r *repository) Delete(ctx context.Context, publicID string) error {
	query := `UPDATE organizations.users SET is_deleted = true, updated_at = NOW() WHERE public_id = $1 AND is_deleted = false`

	result, err := r.db.ExecContext(ctx, query, publicID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Search searches for users based on various criteria
func (r *repository) Search(ctx context.Context, criteria *SearchUserRequest, page, limit int) ([]User, int, error) {
	offset := (page - 1) * limit

	var conditions []string
	var args []interface{}
	argIndex := 1

	conditions = append(conditions, "is_deleted = false")

	if criteria.Name != nil && *criteria.Name != "" {
		// Search in JSONB name field for any locale
		conditions = append(conditions, fmt.Sprintf("name::text ILIKE $%d", argIndex))
		args = append(args, "%"+*criteria.Name+"%")
		argIndex++
	}

	if criteria.Email != nil && *criteria.Email != "" {
		conditions = append(conditions, fmt.Sprintf("email ILIKE $%d", argIndex))
		args = append(args, "%"+*criteria.Email+"%")
		argIndex++
	}

	if criteria.MobileFull != nil && *criteria.MobileFull != "" {
		conditions = append(conditions, fmt.Sprintf("contact_mobile_hash = $%d", argIndex))
		args = append(args, *criteria.MobileFull) // Note: This should be hashed by service layer
		argIndex++
	}

	if criteria.OfficeFull != nil && *criteria.OfficeFull != "" {
		conditions = append(conditions, fmt.Sprintf("contact_office_hash = $%d", argIndex))
		args = append(args, *criteria.OfficeFull) // Note: This should be hashed by service layer
		argIndex++
	}

	if criteria.MobileLast4 != nil && *criteria.MobileLast4 != "" {
		conditions = append(conditions, fmt.Sprintf("contact_mobile_id = $%d", argIndex))
		args = append(args, *criteria.MobileLast4)
		argIndex++
	}

	if criteria.OfficeLast4 != nil && *criteria.OfficeLast4 != "" {
		conditions = append(conditions, fmt.Sprintf("contact_office_id = $%d", argIndex))
		args = append(args, *criteria.OfficeLast4)
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM organizations.users WHERE %s", whereClause)
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// Data query
	dataQuery := fmt.Sprintf(`
		SELECT id, public_id, login_id, name, email,
			dept_id, rank_id, duty_id, title_id, position_id, location_id,
			contact_mobile, contact_mobile_hash, contact_mobile_id,
			contact_office, contact_office_hash, contact_office_id,
			created_at, updated_at, password_hash, is_visible, is_deleted
		FROM organizations.users
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(
			&user.ID,
			&user.PublicID,
			&user.LoginID,
			&user.Name,
			&user.Email,
			&user.DeptID,
			&user.RankID,
			&user.DutyID,
			&user.TitleID,
			&user.PositionID,
			&user.LocationID,
			&user.ContactMobile,
			&user.ContactMobileHash,
			&user.ContactMobileID,
			&user.ContactOffice,
			&user.ContactOfficeHash,
			&user.ContactOfficeID,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.PasswordHash,
			&user.IsVisible,
			&user.IsDeleted,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, totalCount, nil
}

// ExistsByEmail checks if a user with the given email exists
func (r *repository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM organizations.users WHERE email = $1 AND is_deleted = false)`
	err := r.db.QueryRowContext(ctx, query, email).Scan(&exists)
	return exists, err
}

// ExistsByLoginID checks if a user with the given login ID exists
func (r *repository) ExistsByLoginID(ctx context.Context, loginID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM organizations.users WHERE login_id = $1 AND is_deleted = false)`
	err := r.db.QueryRowContext(ctx, query, loginID).Scan(&exists)
	return exists, err
}

// ToListResponse converts a User to UserListResponse
func (u *User) ToListResponse() UserListResponse {
	return UserListResponse{
		ID:      u.PublicID,
		LoginID: u.LoginID,
		Name:    u.Name,
		Email:   u.Email,
	}
}

// ToDetailResponse converts a User to UserDetailResponse with default values
func (u *User) ToDetailResponse() UserDetailResponse {
	detail := UserDetailResponse{
		ID:           u.PublicID,
		LoginID:      u.LoginID,
		Name:         u.Name,
		Email:        u.Email,
		DeptName:     json.RawMessage("{}"),
		RankName:     json.RawMessage("{}"),
		DutyName:     json.RawMessage("{}"),
		TitleName:    json.RawMessage("{}"),
		PositionName: json.RawMessage("{}"),
		LocationName: json.RawMessage("{}"),
	}

	if u.ContactMobileID.Valid && u.ContactMobileID.String != "" {
		detail.ContactMobile = fmt.Sprintf("***-****-%s", u.ContactMobileID.String)
	}
	if u.ContactOfficeID.Valid && u.ContactOfficeID.String != "" {
		detail.ContactOffice = fmt.Sprintf("***-****-%s", u.ContactOfficeID.String)
	}

	return detail
}
