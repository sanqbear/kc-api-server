package roles

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Repository defines the interface for role data access operations
type Repository interface {
	Create(ctx context.Context, role *Role) error
	GetByID(ctx context.Context, id int) (*Role, error)
	GetByName(ctx context.Context, name string) (*Role, error)
	List(ctx context.Context, page, limit int) ([]Role, int, error)
	Update(ctx context.Context, id int, role *Role) error
	Delete(ctx context.Context, id int) error
	Search(ctx context.Context, criteria *SearchRoleRequest, page, limit int) ([]Role, int, error)
	BatchCreate(ctx context.Context, roles []Role) (int, error)
	BatchUpdate(ctx context.Context, updates []Role) (int, []int, error)
	BatchDelete(ctx context.Context, ids []int) (int, []int, error)
	ExistsByName(ctx context.Context, name string) (bool, error)
	ExistsByNameExcludingID(ctx context.Context, name string, id int) (bool, error)

	// User role operations
	GetUserRoles(ctx context.Context, userID int) ([]UserRoleResponse, error)
	AssignUserRoles(ctx context.Context, userID int, roleIDs []int) error
	RemoveUserRole(ctx context.Context, userID int, roleID int) error
	RemoveAllUserRoles(ctx context.Context, userID int) error
	GetUsersWithRole(ctx context.Context, roleID int) ([]int, error)
}

type repository struct {
	db *sql.DB
}

// NewRepository creates a new role repository
func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// Create inserts a new role into the database
func (r *repository) Create(ctx context.Context, role *Role) error {
	query := `
		INSERT INTO organizations.roles (name, description)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		role.Name,
		role.Description,
	).Scan(&role.ID, &role.CreatedAt, &role.UpdatedAt)
}

// GetByID retrieves a role by ID
func (r *repository) GetByID(ctx context.Context, id int) (*Role, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM organizations.roles
		WHERE id = $1`

	role := &Role{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&role.ID,
		&role.Name,
		&role.Description,
		&role.CreatedAt,
		&role.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return role, nil
}

// GetByName retrieves a role by name
func (r *repository) GetByName(ctx context.Context, name string) (*Role, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM organizations.roles
		WHERE name = $1`

	role := &Role{}
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&role.ID,
		&role.Name,
		&role.Description,
		&role.CreatedAt,
		&role.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return role, nil
}

// List retrieves a paginated list of roles
func (r *repository) List(ctx context.Context, page, limit int) ([]Role, int, error) {
	offset := (page - 1) * limit

	// Get total count
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM organizations.roles`
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// Get roles
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM organizations.roles
		ORDER BY name
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(
			&role.ID,
			&role.Name,
			&role.Description,
			&role.CreatedAt,
			&role.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return roles, totalCount, nil
}

// Update updates an existing role
func (r *repository) Update(ctx context.Context, id int, role *Role) error {
	query := `
		UPDATE organizations.roles SET
			name = $1,
			description = $2,
			updated_at = NOW()
		WHERE id = $3`

	result, err := r.db.ExecContext(ctx, query,
		role.Name,
		role.Description,
		id,
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

// Delete deletes a role
func (r *repository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM organizations.roles WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
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

// Search searches for roles based on criteria
func (r *repository) Search(ctx context.Context, criteria *SearchRoleRequest, page, limit int) ([]Role, int, error) {
	offset := (page - 1) * limit

	var conditions []string
	var args []interface{}
	argIndex := 1

	if criteria.Name != nil && *criteria.Name != "" {
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argIndex))
		args = append(args, "%"+*criteria.Name+"%")
		argIndex++
	}

	if criteria.Description != nil && *criteria.Description != "" {
		conditions = append(conditions, fmt.Sprintf("description::text ILIKE $%d", argIndex))
		args = append(args, "%"+*criteria.Description+"%")
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM organizations.roles %s", whereClause)
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// Data query
	dataQuery := fmt.Sprintf(`
		SELECT id, name, description, created_at, updated_at
		FROM organizations.roles
		%s
		ORDER BY name
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(
			&role.ID,
			&role.Name,
			&role.Description,
			&role.CreatedAt,
			&role.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return roles, totalCount, nil
}

// BatchCreate creates multiple roles
func (r *repository) BatchCreate(ctx context.Context, roles []Role) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO organizations.roles (name, description)
		VALUES ($1, $2)`

	successCount := 0
	for _, role := range roles {
		_, err := tx.ExecContext(ctx, query,
			role.Name,
			role.Description,
		)
		if err != nil {
			continue
		}
		successCount++
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return successCount, nil
}

// BatchUpdate updates multiple roles
func (r *repository) BatchUpdate(ctx context.Context, updates []Role) (int, []int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback()

	query := `
		UPDATE organizations.roles SET
			name = $1,
			description = $2,
			updated_at = NOW()
		WHERE id = $3`

	successCount := 0
	var failedIDs []int

	for _, role := range updates {
		result, err := tx.ExecContext(ctx, query,
			role.Name,
			role.Description,
			role.ID,
		)
		if err != nil {
			failedIDs = append(failedIDs, role.ID)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			failedIDs = append(failedIDs, role.ID)
			continue
		}

		successCount++
	}

	if err := tx.Commit(); err != nil {
		return 0, nil, err
	}

	return successCount, failedIDs, nil
}

// BatchDelete deletes multiple roles
func (r *repository) BatchDelete(ctx context.Context, ids []int) (int, []int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback()

	query := `DELETE FROM organizations.roles WHERE id = $1`

	successCount := 0
	var failedIDs []int

	for _, id := range ids {
		result, err := tx.ExecContext(ctx, query, id)
		if err != nil {
			failedIDs = append(failedIDs, id)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			failedIDs = append(failedIDs, id)
			continue
		}

		successCount++
	}

	if err := tx.Commit(); err != nil {
		return 0, nil, err
	}

	return successCount, failedIDs, nil
}

// ExistsByName checks if a role exists with the given name
func (r *repository) ExistsByName(ctx context.Context, name string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM organizations.roles WHERE name = $1)`
	err := r.db.QueryRowContext(ctx, query, name).Scan(&exists)
	return exists, err
}

// ExistsByNameExcludingID checks if a role exists excluding a specific ID
func (r *repository) ExistsByNameExcludingID(ctx context.Context, name string, id int) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM organizations.roles WHERE name = $1 AND id != $2)`
	err := r.db.QueryRowContext(ctx, query, name, id).Scan(&exists)
	return exists, err
}

// GetUserRoles retrieves all roles for a user
func (r *repository) GetUserRoles(ctx context.Context, userID int) ([]UserRoleResponse, error) {
	query := `
		SELECT ur.user_id, ur.role_id, r.name
		FROM organizations.user_roles ur
		JOIN organizations.roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.name`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userRoles []UserRoleResponse
	for rows.Next() {
		var ur UserRoleResponse
		if err := rows.Scan(&ur.UserID, &ur.RoleID, &ur.RoleName); err != nil {
			return nil, err
		}
		userRoles = append(userRoles, ur)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return userRoles, nil
}

// AssignUserRoles assigns roles to a user (replaces existing)
func (r *repository) AssignUserRoles(ctx context.Context, userID int, roleIDs []int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing roles
	_, err = tx.ExecContext(ctx, `DELETE FROM organizations.user_roles WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	// Insert new roles
	query := `INSERT INTO organizations.user_roles (user_id, role_id) VALUES ($1, $2)`
	for _, roleID := range roleIDs {
		_, err := tx.ExecContext(ctx, query, userID, roleID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// RemoveUserRole removes a specific role from a user
func (r *repository) RemoveUserRole(ctx context.Context, userID int, roleID int) error {
	query := `DELETE FROM organizations.user_roles WHERE user_id = $1 AND role_id = $2`
	result, err := r.db.ExecContext(ctx, query, userID, roleID)
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

// RemoveAllUserRoles removes all roles from a user
func (r *repository) RemoveAllUserRoles(ctx context.Context, userID int) error {
	query := `DELETE FROM organizations.user_roles WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// GetUsersWithRole retrieves all user IDs that have a specific role
func (r *repository) GetUsersWithRole(ctx context.Context, roleID int) ([]int, error) {
	query := `SELECT user_id FROM organizations.user_roles WHERE role_id = $1`

	rows, err := r.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []int
	for rows.Next() {
		var userID int
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return userIDs, nil
}
