package roles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrRoleNotFound      = errors.New("role not found")
	ErrRoleNameExists    = errors.New("role name already exists")
	ErrInvalidRoleName   = errors.New("role name is required")
	ErrEmptyBatchRequest = errors.New("batch request cannot be empty")
	ErrUserRoleNotFound  = errors.New("user role not found")
)

// Service defines the interface for role business logic
type Service interface {
	Create(ctx context.Context, req *CreateRoleRequest) (*RoleResponse, error)
	GetByID(ctx context.Context, id int) (*RoleResponse, error)
	List(ctx context.Context, page, limit int) (*RoleListResponseWrapper, error)
	Update(ctx context.Context, id int, req *UpdateRoleRequest) (*RoleResponse, error)
	Delete(ctx context.Context, id int) error
	Search(ctx context.Context, criteria *SearchRoleRequest, page, limit int) (*RoleListResponseWrapper, error)
	BatchCreate(ctx context.Context, req *BatchCreateRoleRequest) (*BatchOperationResponse, error)
	BatchUpdate(ctx context.Context, req *BatchUpdateRoleRequest) (*BatchOperationResponse, error)
	BatchDelete(ctx context.Context, req *BatchDeleteRoleRequest) (*BatchOperationResponse, error)

	// User role operations
	GetUserRoles(ctx context.Context, userID int) ([]UserRoleResponse, error)
	AssignUserRoles(ctx context.Context, userID int, req *AssignUserRolesRequest) error
	RemoveUserRole(ctx context.Context, userID int, roleID int) error
	GetUsersWithRole(ctx context.Context, roleID int) ([]int, error)
}

type service struct {
	repo Repository
}

// NewService creates a new role service
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// Create creates a new role
func (s *service) Create(ctx context.Context, req *CreateRoleRequest) (*RoleResponse, error) {
	if req.Name == "" {
		return nil, ErrInvalidRoleName
	}

	// Check if name exists
	exists, err := s.repo.ExistsByName(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check existence: %w", err)
	}
	if exists {
		return nil, ErrRoleNameExists
	}

	role := &Role{
		Name:        req.Name,
		Description: getOrDefault(req.Description, json.RawMessage("{}")),
	}

	if err := s.repo.Create(ctx, role); err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	response := role.ToResponse()
	return &response, nil
}

// GetByID retrieves a role by ID
func (s *service) GetByID(ctx context.Context, id int) (*RoleResponse, error) {
	role, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	response := role.ToResponse()
	return &response, nil
}

// List retrieves a paginated list of roles
func (s *service) List(ctx context.Context, page, limit int) (*RoleListResponseWrapper, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	roles, totalCount, err := s.repo.List(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	var responses []RoleResponse
	for _, role := range roles {
		responses = append(responses, role.ToResponse())
	}

	totalPages := (totalCount + limit - 1) / limit

	return &RoleListResponseWrapper{
		Data:       responses,
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

// Update updates an existing role
func (s *service) Update(ctx context.Context, id int, req *UpdateRoleRequest) (*RoleResponse, error) {
	existingRole, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	if req.Name != nil && *req.Name != "" {
		existingRole.Name = *req.Name
	}
	if req.Description != nil {
		existingRole.Description = *req.Description
	}

	// Check if name exists (excluding current ID)
	if req.Name != nil {
		exists, err := s.repo.ExistsByNameExcludingID(ctx, existingRole.Name, id)
		if err != nil {
			return nil, fmt.Errorf("failed to check existence: %w", err)
		}
		if exists {
			return nil, ErrRoleNameExists
		}
	}

	if err := s.repo.Update(ctx, id, existingRole); err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	response := existingRole.ToResponse()
	return &response, nil
}

// Delete deletes a role
func (s *service) Delete(ctx context.Context, id int) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRoleNotFound
		}
		return fmt.Errorf("failed to delete role: %w", err)
	}
	return nil
}

// Search searches for roles based on criteria
func (s *service) Search(ctx context.Context, criteria *SearchRoleRequest, page, limit int) (*RoleListResponseWrapper, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	roles, totalCount, err := s.repo.Search(ctx, criteria, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search roles: %w", err)
	}

	var responses []RoleResponse
	for _, role := range roles {
		responses = append(responses, role.ToResponse())
	}

	totalPages := (totalCount + limit - 1) / limit

	return &RoleListResponseWrapper{
		Data:       responses,
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

// BatchCreate creates multiple roles
func (s *service) BatchCreate(ctx context.Context, req *BatchCreateRoleRequest) (*BatchOperationResponse, error) {
	if len(req.Roles) == 0 {
		return nil, ErrEmptyBatchRequest
	}

	var roles []Role
	for _, r := range req.Roles {
		if r.Name == "" {
			continue
		}

		roles = append(roles, Role{
			Name:        r.Name,
			Description: getOrDefault(r.Description, json.RawMessage("{}")),
		})
	}

	successCount, err := s.repo.BatchCreate(ctx, roles)
	if err != nil {
		return nil, fmt.Errorf("failed to batch create: %w", err)
	}

	failedCount := len(req.Roles) - successCount

	return &BatchOperationResponse{
		SuccessCount: successCount,
		FailedCount:  failedCount,
	}, nil
}

// BatchUpdate updates multiple roles
func (s *service) BatchUpdate(ctx context.Context, req *BatchUpdateRoleRequest) (*BatchOperationResponse, error) {
	if len(req.Updates) == 0 {
		return nil, ErrEmptyBatchRequest
	}

	var updates []Role
	for _, u := range req.Updates {
		existingRole, err := s.repo.GetByID(ctx, u.ID)
		if err != nil {
			continue
		}

		if u.Name != nil && *u.Name != "" {
			existingRole.Name = *u.Name
		}
		if u.Description != nil {
			existingRole.Description = *u.Description
		}

		updates = append(updates, *existingRole)
	}

	successCount, failedIDs, err := s.repo.BatchUpdate(ctx, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to batch update: %w", err)
	}

	return &BatchOperationResponse{
		SuccessCount: successCount,
		FailedCount:  len(failedIDs),
		FailedIDs:    failedIDs,
	}, nil
}

// BatchDelete deletes multiple roles
func (s *service) BatchDelete(ctx context.Context, req *BatchDeleteRoleRequest) (*BatchOperationResponse, error) {
	if len(req.IDs) == 0 {
		return nil, ErrEmptyBatchRequest
	}

	successCount, failedIDs, err := s.repo.BatchDelete(ctx, req.IDs)
	if err != nil {
		return nil, fmt.Errorf("failed to batch delete: %w", err)
	}

	return &BatchOperationResponse{
		SuccessCount: successCount,
		FailedCount:  len(failedIDs),
		FailedIDs:    failedIDs,
	}, nil
}

// GetUserRoles retrieves all roles for a user
func (s *service) GetUserRoles(ctx context.Context, userID int) ([]UserRoleResponse, error) {
	roles, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	return roles, nil
}

// AssignUserRoles assigns roles to a user
func (s *service) AssignUserRoles(ctx context.Context, userID int, req *AssignUserRolesRequest) error {
	if err := s.repo.AssignUserRoles(ctx, userID, req.RoleIDs); err != nil {
		return fmt.Errorf("failed to assign user roles: %w", err)
	}
	return nil
}

// RemoveUserRole removes a specific role from a user
func (s *service) RemoveUserRole(ctx context.Context, userID int, roleID int) error {
	if err := s.repo.RemoveUserRole(ctx, userID, roleID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserRoleNotFound
		}
		return fmt.Errorf("failed to remove user role: %w", err)
	}
	return nil
}

// GetUsersWithRole retrieves all users that have a specific role
func (s *service) GetUsersWithRole(ctx context.Context, roleID int) ([]int, error) {
	userIDs, err := s.repo.GetUsersWithRole(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users with role: %w", err)
	}
	return userIDs, nil
}

// Helper function
func getOrDefault(value *json.RawMessage, defaultValue json.RawMessage) json.RawMessage {
	if value != nil {
		return *value
	}
	return defaultValue
}
