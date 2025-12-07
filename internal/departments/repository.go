package departments

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Repository defines the interface for department data access operations
type Repository interface {
	Create(ctx context.Context, dept *Department) error
	GetByID(ctx context.Context, id int) (*Department, error)
	GetByPublicID(ctx context.Context, publicID string) (*Department, error)
	List(ctx context.Context, page, limit int) ([]Department, int, error)
	Update(ctx context.Context, id int, dept *Department) error
	Delete(ctx context.Context, id int) error
	Search(ctx context.Context, criteria *SearchDepartmentRequest, page, limit int) ([]Department, int, error)
	BatchCreate(ctx context.Context, depts []Department) (int, error)
	BatchUpdate(ctx context.Context, updates []Department) (int, []string, error)
	BatchDelete(ctx context.Context, ids []int) (int, []string, error)
	GetChildren(ctx context.Context, parentID int) ([]Department, error)
	GetAllDescendants(ctx context.Context, parentID int) ([]Department, error)
	GetRootDepartments(ctx context.Context) ([]Department, error)
	GetParentPublicID(ctx context.Context, parentID *int) (*string, error)
	GetIDByPublicID(ctx context.Context, publicID string) (int, error)
	HasChildren(ctx context.Context, id int) (bool, error)
}

type repository struct {
	db *sql.DB
}

// NewRepository creates a new department repository
func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// Create inserts a new department into the database
func (r *repository) Create(ctx context.Context, dept *Department) error {
	if dept.PublicID == "" {
		dept.PublicID = uuid.New().String()
	}

	query := `
		INSERT INTO organizations.departments (public_id, name, description, parent_department_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		dept.PublicID,
		dept.Name,
		dept.Description,
		dept.ParentDepartmentID,
	).Scan(&dept.ID, &dept.CreatedAt, &dept.UpdatedAt)
}

// GetByID retrieves a department by ID
func (r *repository) GetByID(ctx context.Context, id int) (*Department, error) {
	query := `
		SELECT id, public_id, name, description, parent_department_id, is_deleted, created_at, updated_at
		FROM organizations.departments
		WHERE id = $1 AND is_deleted = false`

	dept := &Department{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&dept.ID,
		&dept.PublicID,
		&dept.Name,
		&dept.Description,
		&dept.ParentDepartmentID,
		&dept.IsDeleted,
		&dept.CreatedAt,
		&dept.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return dept, nil
}

// GetByPublicID retrieves a department by public ID
func (r *repository) GetByPublicID(ctx context.Context, publicID string) (*Department, error) {
	query := `
		SELECT id, public_id, name, description, parent_department_id, is_deleted, created_at, updated_at
		FROM organizations.departments
		WHERE public_id = $1 AND is_deleted = false`

	dept := &Department{}
	err := r.db.QueryRowContext(ctx, query, publicID).Scan(
		&dept.ID,
		&dept.PublicID,
		&dept.Name,
		&dept.Description,
		&dept.ParentDepartmentID,
		&dept.IsDeleted,
		&dept.CreatedAt,
		&dept.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return dept, nil
}

// List retrieves a paginated list of departments
func (r *repository) List(ctx context.Context, page, limit int) ([]Department, int, error) {
	offset := (page - 1) * limit

	// Get total count
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM organizations.departments WHERE is_deleted = false`
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// Get departments
	query := `
		SELECT id, public_id, name, description, parent_department_id, is_deleted, created_at, updated_at
		FROM organizations.departments
		WHERE is_deleted = false
		ORDER BY name
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var depts []Department
	for rows.Next() {
		var dept Department
		if err := rows.Scan(
			&dept.ID,
			&dept.PublicID,
			&dept.Name,
			&dept.Description,
			&dept.ParentDepartmentID,
			&dept.IsDeleted,
			&dept.CreatedAt,
			&dept.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		depts = append(depts, dept)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return depts, totalCount, nil
}

// Update updates an existing department
func (r *repository) Update(ctx context.Context, id int, dept *Department) error {
	query := `
		UPDATE organizations.departments SET
			name = $1,
			description = $2,
			parent_department_id = $3,
			updated_at = NOW()
		WHERE id = $4 AND is_deleted = false`

	result, err := r.db.ExecContext(ctx, query,
		dept.Name,
		dept.Description,
		dept.ParentDepartmentID,
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

// Delete soft-deletes a department
func (r *repository) Delete(ctx context.Context, id int) error {
	query := `UPDATE organizations.departments SET is_deleted = true, updated_at = NOW() WHERE id = $1 AND is_deleted = false`

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

// Search searches for departments based on criteria
func (r *repository) Search(ctx context.Context, criteria *SearchDepartmentRequest, page, limit int) ([]Department, int, error) {
	offset := (page - 1) * limit

	var conditions []string
	var args []interface{}
	argIndex := 1

	conditions = append(conditions, "is_deleted = false")

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

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM organizations.departments %s", whereClause)
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// Data query
	dataQuery := fmt.Sprintf(`
		SELECT id, public_id, name, description, parent_department_id, is_deleted, created_at, updated_at
		FROM organizations.departments
		%s
		ORDER BY name
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var depts []Department
	for rows.Next() {
		var dept Department
		if err := rows.Scan(
			&dept.ID,
			&dept.PublicID,
			&dept.Name,
			&dept.Description,
			&dept.ParentDepartmentID,
			&dept.IsDeleted,
			&dept.CreatedAt,
			&dept.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		depts = append(depts, dept)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return depts, totalCount, nil
}

// BatchCreate creates multiple departments
func (r *repository) BatchCreate(ctx context.Context, depts []Department) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO organizations.departments (public_id, name, description, parent_department_id)
		VALUES ($1, $2, $3, $4)`

	successCount := 0
	for _, dept := range depts {
		publicID := dept.PublicID
		if publicID == "" {
			publicID = uuid.New().String()
		}

		_, err := tx.ExecContext(ctx, query,
			publicID,
			dept.Name,
			dept.Description,
			dept.ParentDepartmentID,
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

// BatchUpdate updates multiple departments
func (r *repository) BatchUpdate(ctx context.Context, updates []Department) (int, []string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback()

	query := `
		UPDATE organizations.departments SET
			name = $1,
			description = $2,
			parent_department_id = $3,
			updated_at = NOW()
		WHERE id = $4 AND is_deleted = false`

	successCount := 0
	var failedIDs []string

	for _, dept := range updates {
		result, err := tx.ExecContext(ctx, query,
			dept.Name,
			dept.Description,
			dept.ParentDepartmentID,
			dept.ID,
		)
		if err != nil {
			failedIDs = append(failedIDs, dept.PublicID)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			failedIDs = append(failedIDs, dept.PublicID)
			continue
		}

		successCount++
	}

	if err := tx.Commit(); err != nil {
		return 0, nil, err
	}

	return successCount, failedIDs, nil
}

// BatchDelete soft-deletes multiple departments
func (r *repository) BatchDelete(ctx context.Context, ids []int) (int, []string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback()

	query := `UPDATE organizations.departments SET is_deleted = true, updated_at = NOW() WHERE id = $1 AND is_deleted = false`

	successCount := 0
	var failedIDs []string

	for _, id := range ids {
		// Get public_id for error reporting
		var publicID string
		r.db.QueryRowContext(ctx, "SELECT public_id FROM organizations.departments WHERE id = $1", id).Scan(&publicID)

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

// GetChildren retrieves all direct children of a department
func (r *repository) GetChildren(ctx context.Context, parentID int) ([]Department, error) {
	query := `
		SELECT id, public_id, name, description, parent_department_id, is_deleted, created_at, updated_at
		FROM organizations.departments
		WHERE parent_department_id = $1 AND is_deleted = false
		ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var depts []Department
	for rows.Next() {
		var dept Department
		if err := rows.Scan(
			&dept.ID,
			&dept.PublicID,
			&dept.Name,
			&dept.Description,
			&dept.ParentDepartmentID,
			&dept.IsDeleted,
			&dept.CreatedAt,
			&dept.UpdatedAt,
		); err != nil {
			return nil, err
		}
		depts = append(depts, dept)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return depts, nil
}

// GetAllDescendants retrieves all descendants of a department using recursive CTE
func (r *repository) GetAllDescendants(ctx context.Context, parentID int) ([]Department, error) {
	query := `
		WITH RECURSIVE dept_tree AS (
			SELECT id, public_id, name, description, parent_department_id, is_deleted, created_at, updated_at
			FROM organizations.departments
			WHERE parent_department_id = $1 AND is_deleted = false

			UNION ALL

			SELECT d.id, d.public_id, d.name, d.description, d.parent_department_id, d.is_deleted, d.created_at, d.updated_at
			FROM organizations.departments d
			INNER JOIN dept_tree dt ON d.parent_department_id = dt.id
			WHERE d.is_deleted = false
		)
		SELECT id, public_id, name, description, parent_department_id, is_deleted, created_at, updated_at
		FROM dept_tree
		ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var depts []Department
	for rows.Next() {
		var dept Department
		if err := rows.Scan(
			&dept.ID,
			&dept.PublicID,
			&dept.Name,
			&dept.Description,
			&dept.ParentDepartmentID,
			&dept.IsDeleted,
			&dept.CreatedAt,
			&dept.UpdatedAt,
		); err != nil {
			return nil, err
		}
		depts = append(depts, dept)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return depts, nil
}

// GetRootDepartments retrieves all departments without a parent
func (r *repository) GetRootDepartments(ctx context.Context) ([]Department, error) {
	query := `
		SELECT id, public_id, name, description, parent_department_id, is_deleted, created_at, updated_at
		FROM organizations.departments
		WHERE parent_department_id IS NULL AND is_deleted = false
		ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var depts []Department
	for rows.Next() {
		var dept Department
		if err := rows.Scan(
			&dept.ID,
			&dept.PublicID,
			&dept.Name,
			&dept.Description,
			&dept.ParentDepartmentID,
			&dept.IsDeleted,
			&dept.CreatedAt,
			&dept.UpdatedAt,
		); err != nil {
			return nil, err
		}
		depts = append(depts, dept)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return depts, nil
}

// GetParentPublicID retrieves the public_id of a parent department
func (r *repository) GetParentPublicID(ctx context.Context, parentID *int) (*string, error) {
	if parentID == nil {
		return nil, nil
	}

	var publicID string
	query := `SELECT public_id FROM organizations.departments WHERE id = $1 AND is_deleted = false`
	err := r.db.QueryRowContext(ctx, query, *parentID).Scan(&publicID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &publicID, nil
}

// GetIDByPublicID retrieves the internal ID by public ID
func (r *repository) GetIDByPublicID(ctx context.Context, publicID string) (int, error) {
	var id int
	query := `SELECT id FROM organizations.departments WHERE public_id = $1 AND is_deleted = false`
	err := r.db.QueryRowContext(ctx, query, publicID).Scan(&id)
	return id, err
}

// HasChildren checks if a department has children
func (r *repository) HasChildren(ctx context.Context, id int) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM organizations.departments WHERE parent_department_id = $1 AND is_deleted = false)`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	return exists, err
}
