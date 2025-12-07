package roles

import (
	"encoding/json"
	"time"
)

// Role represents a role in the system
type Role struct {
	ID          int             `json:"-"`
	Name        string          `json:"name"`
	Description json.RawMessage `json:"description"`
	CreatedAt   time.Time       `json:"-"`
	UpdatedAt   time.Time       `json:"-"`
}

// RoleResponse represents the API response for a role
type RoleResponse struct {
	ID          int             `json:"id" example:"1"`
	Name        string          `json:"name" example:"admin"`
	Description json.RawMessage `json:"description" swaggertype:"object"`
}

// CreateRoleRequest represents the request to create a role
type CreateRoleRequest struct {
	Name        string           `json:"name" example:"admin"`
	Description *json.RawMessage `json:"description,omitempty" swaggertype:"object"`
}

// UpdateRoleRequest represents the request to update a role
type UpdateRoleRequest struct {
	Name        *string          `json:"name,omitempty" example:"admin"`
	Description *json.RawMessage `json:"description,omitempty" swaggertype:"object"`
}

// SearchRoleRequest represents the search criteria for roles
type SearchRoleRequest struct {
	Name        *string `json:"name,omitempty" example:"admin"`
	Description *string `json:"description,omitempty" example:"administrator"`
}

// BatchCreateRoleRequest represents batch create request
type BatchCreateRoleRequest struct {
	Roles []CreateRoleRequest `json:"roles"`
}

// BatchUpdateRoleRequest represents batch update request
type BatchUpdateRoleRequest struct {
	Updates []struct {
		ID          int              `json:"id" example:"1"`
		Name        *string          `json:"name,omitempty" example:"admin"`
		Description *json.RawMessage `json:"description,omitempty" swaggertype:"object"`
	} `json:"updates"`
}

// BatchDeleteRoleRequest represents batch delete request
type BatchDeleteRoleRequest struct {
	IDs []int `json:"ids" example:"1,2,3"`
}

// UserRole represents the user-role mapping
type UserRole struct {
	UserID    int       `json:"user_id"`
	RoleID    int       `json:"role_id"`
	CreatedAt time.Time `json:"-"`
}

// UserRoleResponse represents the API response for user-role mapping
type UserRoleResponse struct {
	UserID   int    `json:"user_id" example:"1"`
	RoleID   int    `json:"role_id" example:"1"`
	RoleName string `json:"role_name,omitempty" example:"admin"`
}

// AssignUserRolesRequest represents the request to assign roles to a user
type AssignUserRolesRequest struct {
	RoleIDs []int `json:"role_ids" example:"1,2,3"`
}

// RoleListResponseWrapper wraps the list response with pagination
type RoleListResponseWrapper struct {
	Data       []RoleResponse `json:"data"`
	Page       int            `json:"page" example:"1"`
	Limit      int            `json:"limit" example:"10"`
	TotalCount int            `json:"total_count" example:"100"`
	TotalPages int            `json:"total_pages" example:"10"`
}

// BatchOperationResponse represents batch operation result
type BatchOperationResponse struct {
	SuccessCount int   `json:"success_count" example:"5"`
	FailedCount  int   `json:"failed_count" example:"0"`
	FailedIDs    []int `json:"failed_ids,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error" example:"Bad Request"`
	Message string `json:"message" example:"Invalid role name"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string `json:"message" example:"Role deleted successfully"`
}

// ToResponse converts a Role to RoleResponse
func (r *Role) ToResponse() RoleResponse {
	return RoleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
	}
}
