package commoncodes

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Repository defines the interface for common code data access operations
type Repository interface {
	Create(ctx context.Context, code *CommonCode) error
	GetByID(ctx context.Context, id int) (*CommonCode, error)
	List(ctx context.Context, page, limit int) ([]CommonCode, int, error)
	Update(ctx context.Context, id int, code *CommonCode) error
	Delete(ctx context.Context, id int) error
	Search(ctx context.Context, criteria *SearchCommonCodeRequest, page, limit int) ([]CommonCode, int, error)
	BatchCreate(ctx context.Context, codes []CommonCode) (int, error)
	BatchUpdate(ctx context.Context, updates []CommonCode) (int, []int, error)
	BatchDelete(ctx context.Context, ids []int) (int, []int, error)
	ListCategories(ctx context.Context) ([]string, error)
	GetByCategory(ctx context.Context, category string) ([]CommonCode, error)
	Reorder(ctx context.Context, category string, orders map[int]int) error
	ExistsByCategoryAndCode(ctx context.Context, category, code string) (bool, error)
	ExistsByCategoryAndCodeExcludingID(ctx context.Context, category, code string, id int) (bool, error)
}

type repository struct {
	db *sql.DB
}

// NewRepository creates a new common code repository
func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// Create inserts a new common code into the database
func (r *repository) Create(ctx context.Context, code *CommonCode) error {
	query := `
		INSERT INTO organizations.common_codes (
			category, code, name, description, extra_payload, sort_order
		) VALUES (
			$1, $2, $3, $4, $5, $6
		) RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		code.Category,
		code.Code,
		code.Name,
		code.Description,
		code.ExtraPayload,
		code.SortOrder,
	).Scan(&code.ID, &code.CreatedAt, &code.UpdatedAt)
}

// GetByID retrieves a common code by ID
func (r *repository) GetByID(ctx context.Context, id int) (*CommonCode, error) {
	query := `
		SELECT id, category, code, name, description, extra_payload, sort_order, created_at, updated_at
		FROM organizations.common_codes
		WHERE id = $1`

	code := &CommonCode{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&code.ID,
		&code.Category,
		&code.Code,
		&code.Name,
		&code.Description,
		&code.ExtraPayload,
		&code.SortOrder,
		&code.CreatedAt,
		&code.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return code, nil
}

// List retrieves a paginated list of common codes
func (r *repository) List(ctx context.Context, page, limit int) ([]CommonCode, int, error) {
	offset := (page - 1) * limit

	// Get total count
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM organizations.common_codes`
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// Get codes
	query := `
		SELECT id, category, code, name, description, extra_payload, sort_order, created_at, updated_at
		FROM organizations.common_codes
		ORDER BY category, sort_order, code
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var codes []CommonCode
	for rows.Next() {
		var code CommonCode
		if err := rows.Scan(
			&code.ID,
			&code.Category,
			&code.Code,
			&code.Name,
			&code.Description,
			&code.ExtraPayload,
			&code.SortOrder,
			&code.CreatedAt,
			&code.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		codes = append(codes, code)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return codes, totalCount, nil
}

// Update updates an existing common code
func (r *repository) Update(ctx context.Context, id int, code *CommonCode) error {
	query := `
		UPDATE organizations.common_codes SET
			category = $1,
			code = $2,
			name = $3,
			description = $4,
			extra_payload = $5,
			sort_order = $6,
			updated_at = NOW()
		WHERE id = $7`

	result, err := r.db.ExecContext(ctx, query,
		code.Category,
		code.Code,
		code.Name,
		code.Description,
		code.ExtraPayload,
		code.SortOrder,
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

// Delete deletes a common code
func (r *repository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM organizations.common_codes WHERE id = $1`

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

// Search searches for common codes based on criteria
func (r *repository) Search(ctx context.Context, criteria *SearchCommonCodeRequest, page, limit int) ([]CommonCode, int, error) {
	offset := (page - 1) * limit

	var conditions []string
	var args []interface{}
	argIndex := 1

	if criteria.Category != nil && *criteria.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, *criteria.Category)
		argIndex++
	}

	if criteria.Code != nil && *criteria.Code != "" {
		conditions = append(conditions, fmt.Sprintf("code ILIKE $%d", argIndex))
		args = append(args, "%"+*criteria.Code+"%")
		argIndex++
	}

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
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM organizations.common_codes %s", whereClause)
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// Data query
	dataQuery := fmt.Sprintf(`
		SELECT id, category, code, name, description, extra_payload, sort_order, created_at, updated_at
		FROM organizations.common_codes
		%s
		ORDER BY category, sort_order, code
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var codes []CommonCode
	for rows.Next() {
		var code CommonCode
		if err := rows.Scan(
			&code.ID,
			&code.Category,
			&code.Code,
			&code.Name,
			&code.Description,
			&code.ExtraPayload,
			&code.SortOrder,
			&code.CreatedAt,
			&code.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		codes = append(codes, code)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return codes, totalCount, nil
}

// BatchCreate creates multiple common codes
func (r *repository) BatchCreate(ctx context.Context, codes []CommonCode) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO organizations.common_codes (
			category, code, name, description, extra_payload, sort_order
		) VALUES ($1, $2, $3, $4, $5, $6)`

	successCount := 0
	for _, code := range codes {
		_, err := tx.ExecContext(ctx, query,
			code.Category,
			code.Code,
			code.Name,
			code.Description,
			code.ExtraPayload,
			code.SortOrder,
		)
		if err != nil {
			continue // Skip failed entries
		}
		successCount++
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return successCount, nil
}

// BatchUpdate updates multiple common codes
func (r *repository) BatchUpdate(ctx context.Context, updates []CommonCode) (int, []int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback()

	query := `
		UPDATE organizations.common_codes SET
			category = $1,
			code = $2,
			name = $3,
			description = $4,
			extra_payload = $5,
			sort_order = $6,
			updated_at = NOW()
		WHERE id = $7`

	successCount := 0
	var failedIDs []int

	for _, code := range updates {
		result, err := tx.ExecContext(ctx, query,
			code.Category,
			code.Code,
			code.Name,
			code.Description,
			code.ExtraPayload,
			code.SortOrder,
			code.ID,
		)
		if err != nil {
			failedIDs = append(failedIDs, code.ID)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			failedIDs = append(failedIDs, code.ID)
			continue
		}

		successCount++
	}

	if err := tx.Commit(); err != nil {
		return 0, nil, err
	}

	return successCount, failedIDs, nil
}

// BatchDelete deletes multiple common codes
func (r *repository) BatchDelete(ctx context.Context, ids []int) (int, []int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback()

	query := `DELETE FROM organizations.common_codes WHERE id = $1`

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

// ListCategories retrieves all unique categories
func (r *repository) ListCategories(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT category FROM organizations.common_codes ORDER BY category`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

// GetByCategory retrieves all codes in a category
func (r *repository) GetByCategory(ctx context.Context, category string) ([]CommonCode, error) {
	query := `
		SELECT id, category, code, name, description, extra_payload, sort_order, created_at, updated_at
		FROM organizations.common_codes
		WHERE category = $1
		ORDER BY sort_order, code`

	rows, err := r.db.QueryContext(ctx, query, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []CommonCode
	for rows.Next() {
		var code CommonCode
		if err := rows.Scan(
			&code.ID,
			&code.Category,
			&code.Code,
			&code.Name,
			&code.Description,
			&code.ExtraPayload,
			&code.SortOrder,
			&code.CreatedAt,
			&code.UpdatedAt,
		); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return codes, nil
}

// Reorder updates sort orders for codes in a category
func (r *repository) Reorder(ctx context.Context, category string, orders map[int]int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		UPDATE organizations.common_codes
		SET sort_order = $1, updated_at = NOW()
		WHERE id = $2 AND category = $3`

	for id, sortOrder := range orders {
		_, err := tx.ExecContext(ctx, query, sortOrder, id, category)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ExistsByCategoryAndCode checks if a code exists with the given category and code
func (r *repository) ExistsByCategoryAndCode(ctx context.Context, category, code string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM organizations.common_codes WHERE category = $1 AND code = $2)`
	err := r.db.QueryRowContext(ctx, query, category, code).Scan(&exists)
	return exists, err
}

// ExistsByCategoryAndCodeExcludingID checks if a code exists excluding a specific ID
func (r *repository) ExistsByCategoryAndCodeExcludingID(ctx context.Context, category, code string, id int) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM organizations.common_codes WHERE category = $1 AND code = $2 AND id != $3)`
	err := r.db.QueryRowContext(ctx, query, category, code, id).Scan(&exists)
	return exists, err
}
