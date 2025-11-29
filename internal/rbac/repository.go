package rbac

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
)

// APIPermission represents a permission rule from the database
type APIPermission struct {
	ID            int64
	Method        string
	PathPattern   string
	RequiredRoles []string
}

// Repository defines the interface for RBAC data access
type Repository interface {
	GetAllPermissions(ctx context.Context) ([]APIPermission, error)
}

// repository implements the Repository interface
type repository struct {
	db *sql.DB
}

// NewRepository creates a new RBAC repository
func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// GetAllPermissions fetches all API permissions from the database
func (r *repository) GetAllPermissions(ctx context.Context) ([]APIPermission, error) {
	query := `
		SELECT id, method, path_pattern, required_roles
		FROM managements.api_permissions
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []APIPermission
	for rows.Next() {
		var perm APIPermission
		err := rows.Scan(&perm.ID, &perm.Method, &perm.PathPattern, pq.Array(&perm.RequiredRoles))
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, perm)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}
