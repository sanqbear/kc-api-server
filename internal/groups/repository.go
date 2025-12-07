package groups

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Repository defines the interface for group data access operations
type Repository interface {
	Create(ctx context.Context, group *Group) error
	GetByID(ctx context.Context, id int) (*Group, error)
	GetByPublicID(ctx context.Context, publicID string) (*Group, error)
	List(ctx context.Context, page, limit int) ([]Group, int, error)
	Update(ctx context.Context, id int, group *Group) error
	Delete(ctx context.Context, id int) error
	Search(ctx context.Context, criteria *SearchGroupRequest, page, limit int) ([]Group, int, error)
	BatchCreate(ctx context.Context, groups []Group) (int, error)
	BatchUpdate(ctx context.Context, updates []Group) (int, []string, error)
	BatchDelete(ctx context.Context, ids []int) (int, []string, error)
	GetIDByPublicID(ctx context.Context, publicID string) (int, error)

	// Group user operations
	GetGroupUsers(ctx context.Context, groupID int) ([]int, error)
	AddUsersToGroup(ctx context.Context, groupID int, userIDs []int) error
	RemoveUserFromGroup(ctx context.Context, groupID int, userID int) error
	RemoveAllUsersFromGroup(ctx context.Context, groupID int) error
	GetUserGroups(ctx context.Context, userID int) ([]Group, error)

	// Group role operations
	GetGroupRoles(ctx context.Context, groupID int) ([]GroupRoleResponse, error)
	AssignRolesToGroup(ctx context.Context, groupID int, roleIDs []int) error
	RemoveRoleFromGroup(ctx context.Context, groupID int, roleID int) error
	RemoveAllRolesFromGroup(ctx context.Context, groupID int) error
}

type repository struct {
	db *sql.DB
}

// NewRepository creates a new group repository
func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// Create inserts a new group into the database
func (r *repository) Create(ctx context.Context, group *Group) error {
	if group.PublicID == "" {
		group.PublicID = uuid.New().String()
	}

	query := `
		INSERT INTO organizations.groups (public_id, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		group.PublicID,
		group.Name,
		group.Description,
	).Scan(&group.ID, &group.CreatedAt, &group.UpdatedAt)
}

// GetByID retrieves a group by ID
func (r *repository) GetByID(ctx context.Context, id int) (*Group, error) {
	query := `
		SELECT id, public_id, name, description, created_at, updated_at
		FROM organizations.groups
		WHERE id = $1`

	group := &Group{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
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

// GetByPublicID retrieves a group by public ID
func (r *repository) GetByPublicID(ctx context.Context, publicID string) (*Group, error) {
	query := `
		SELECT id, public_id, name, description, created_at, updated_at
		FROM organizations.groups
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

// List retrieves a paginated list of groups
func (r *repository) List(ctx context.Context, page, limit int) ([]Group, int, error) {
	offset := (page - 1) * limit

	// Get total count
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM organizations.groups`
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// Get groups
	query := `
		SELECT id, public_id, name, description, created_at, updated_at
		FROM organizations.groups
		ORDER BY name
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
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
			return nil, 0, err
		}
		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return groups, totalCount, nil
}

// Update updates an existing group
func (r *repository) Update(ctx context.Context, id int, group *Group) error {
	query := `
		UPDATE organizations.groups SET
			name = $1,
			description = $2,
			updated_at = NOW()
		WHERE id = $3`

	result, err := r.db.ExecContext(ctx, query,
		group.Name,
		group.Description,
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

// Delete deletes a group
func (r *repository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM organizations.groups WHERE id = $1`

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

// Search searches for groups based on criteria
func (r *repository) Search(ctx context.Context, criteria *SearchGroupRequest, page, limit int) ([]Group, int, error) {
	offset := (page - 1) * limit

	var conditions []string
	var args []interface{}
	argIndex := 1

	if criteria.Name != nil && *criteria.Name != "" {
		conditions = append(conditions, fmt.Sprintf("name::text ILIKE $%d", argIndex))
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
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM organizations.groups %s", whereClause)
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// Data query
	dataQuery := fmt.Sprintf(`
		SELECT id, public_id, name, description, created_at, updated_at
		FROM organizations.groups
		%s
		ORDER BY name
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
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
			return nil, 0, err
		}
		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return groups, totalCount, nil
}

// BatchCreate creates multiple groups
func (r *repository) BatchCreate(ctx context.Context, groups []Group) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO organizations.groups (public_id, name, description)
		VALUES ($1, $2, $3)`

	successCount := 0
	for _, group := range groups {
		publicID := group.PublicID
		if publicID == "" {
			publicID = uuid.New().String()
		}

		_, err := tx.ExecContext(ctx, query,
			publicID,
			group.Name,
			group.Description,
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

// BatchUpdate updates multiple groups
func (r *repository) BatchUpdate(ctx context.Context, updates []Group) (int, []string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback()

	query := `
		UPDATE organizations.groups SET
			name = $1,
			description = $2,
			updated_at = NOW()
		WHERE id = $3`

	successCount := 0
	var failedIDs []string

	for _, group := range updates {
		result, err := tx.ExecContext(ctx, query,
			group.Name,
			group.Description,
			group.ID,
		)
		if err != nil {
			failedIDs = append(failedIDs, group.PublicID)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			failedIDs = append(failedIDs, group.PublicID)
			continue
		}

		successCount++
	}

	if err := tx.Commit(); err != nil {
		return 0, nil, err
	}

	return successCount, failedIDs, nil
}

// BatchDelete deletes multiple groups
func (r *repository) BatchDelete(ctx context.Context, ids []int) (int, []string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback()

	query := `DELETE FROM organizations.groups WHERE id = $1`

	successCount := 0
	var failedIDs []string

	for _, id := range ids {
		// Get public_id for error reporting
		var publicID string
		r.db.QueryRowContext(ctx, "SELECT public_id FROM organizations.groups WHERE id = $1", id).Scan(&publicID)

		result, err := tx.ExecContext(ctx, query, id)
		if err != nil {
			failedIDs = append(failedIDs, publicID)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			failedIDs = append(failedIDs, publicID)
			continue
		}

		successCount++
	}

	if err := tx.Commit(); err != nil {
		return 0, nil, err
	}

	return successCount, failedIDs, nil
}

// GetIDByPublicID retrieves the internal ID by public ID
func (r *repository) GetIDByPublicID(ctx context.Context, publicID string) (int, error) {
	var id int
	query := `SELECT id FROM organizations.groups WHERE public_id = $1`
	err := r.db.QueryRowContext(ctx, query, publicID).Scan(&id)
	return id, err
}

// GetGroupUsers retrieves all user IDs in a group
func (r *repository) GetGroupUsers(ctx context.Context, groupID int) ([]int, error) {
	query := `SELECT user_id FROM organizations.group_users WHERE group_id = $1`

	rows, err := r.db.QueryContext(ctx, query, groupID)
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

// AddUsersToGroup adds users to a group (replaces existing)
func (r *repository) AddUsersToGroup(ctx context.Context, groupID int, userIDs []int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing users
	_, err = tx.ExecContext(ctx, `DELETE FROM organizations.group_users WHERE group_id = $1`, groupID)
	if err != nil {
		return err
	}

	// Insert new users
	query := `INSERT INTO organizations.group_users (group_id, user_id) VALUES ($1, $2)`
	for _, userID := range userIDs {
		_, err := tx.ExecContext(ctx, query, groupID, userID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// RemoveUserFromGroup removes a user from a group
func (r *repository) RemoveUserFromGroup(ctx context.Context, groupID int, userID int) error {
	query := `DELETE FROM organizations.group_users WHERE group_id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, groupID, userID)
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

// RemoveAllUsersFromGroup removes all users from a group
func (r *repository) RemoveAllUsersFromGroup(ctx context.Context, groupID int) error {
	query := `DELETE FROM organizations.group_users WHERE group_id = $1`
	_, err := r.db.ExecContext(ctx, query, groupID)
	return err
}

// GetUserGroups retrieves all groups a user belongs to
func (r *repository) GetUserGroups(ctx context.Context, userID int) ([]Group, error) {
	query := `
		SELECT g.id, g.public_id, g.name, g.description, g.created_at, g.updated_at
		FROM organizations.groups g
		JOIN organizations.group_users gu ON g.id = gu.group_id
		WHERE gu.user_id = $1
		ORDER BY g.name`

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

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return groups, nil
}

// GetGroupRoles retrieves all roles assigned to a group
func (r *repository) GetGroupRoles(ctx context.Context, groupID int) ([]GroupRoleResponse, error) {
	query := `
		SELECT g.public_id, gr.role_id, r.name
		FROM organizations.group_roles gr
		JOIN organizations.groups g ON gr.group_id = g.id
		JOIN organizations.roles r ON gr.role_id = r.id
		WHERE gr.group_id = $1
		ORDER BY r.name`

	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []GroupRoleResponse
	for rows.Next() {
		var role GroupRoleResponse
		if err := rows.Scan(&role.GroupPublicID, &role.RoleID, &role.RoleName); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return roles, nil
}

// AssignRolesToGroup assigns roles to a group (replaces existing)
func (r *repository) AssignRolesToGroup(ctx context.Context, groupID int, roleIDs []int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing roles
	_, err = tx.ExecContext(ctx, `DELETE FROM organizations.group_roles WHERE group_id = $1`, groupID)
	if err != nil {
		return err
	}

	// Insert new roles
	query := `INSERT INTO organizations.group_roles (group_id, role_id) VALUES ($1, $2)`
	for _, roleID := range roleIDs {
		_, err := tx.ExecContext(ctx, query, groupID, roleID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// RemoveRoleFromGroup removes a role from a group
func (r *repository) RemoveRoleFromGroup(ctx context.Context, groupID int, roleID int) error {
	query := `DELETE FROM organizations.group_roles WHERE group_id = $1 AND role_id = $2`
	result, err := r.db.ExecContext(ctx, query, groupID, roleID)
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

// RemoveAllRolesFromGroup removes all roles from a group
func (r *repository) RemoveAllRolesFromGroup(ctx context.Context, groupID int) error {
	query := `DELETE FROM organizations.group_roles WHERE group_id = $1`
	_, err := r.db.ExecContext(ctx, query, groupID)
	return err
}
