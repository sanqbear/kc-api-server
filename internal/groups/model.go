package groups

import (
	"encoding/json"
	"time"
)

// Group represents a group in the system
type Group struct {
	ID          int             `json:"-"`
	PublicID    string          `json:"public_id"`
	Name        json.RawMessage `json:"name"`
	Description json.RawMessage `json:"description"`
	CreatedAt   time.Time       `json:"-"`
	UpdatedAt   time.Time       `json:"-"`
}

// GroupResponse represents the API response for a group
type GroupResponse struct {
	ID          int             `json:"id" example:"1"`
	PublicID    string          `json:"public_id" example:"grp-abc123"`
	Name        json.RawMessage `json:"name" swaggertype:"object"`
	Description json.RawMessage `json:"description" swaggertype:"object"`
}

// CreateGroupRequest represents the request to create a group
type CreateGroupRequest struct {
	Name        json.RawMessage  `json:"name" swaggertype:"object"`
	Description *json.RawMessage `json:"description,omitempty" swaggertype:"object"`
}

// UpdateGroupRequest represents the request to update a group
type UpdateGroupRequest struct {
	Name        *json.RawMessage `json:"name,omitempty" swaggertype:"object"`
	Description *json.RawMessage `json:"description,omitempty" swaggertype:"object"`
}

// SearchGroupRequest represents the search criteria for groups
type SearchGroupRequest struct {
	Name        *string `json:"name,omitempty" example:"Admins"`
	Description *string `json:"description,omitempty" example:"administrator"`
}

// BatchCreateGroupRequest represents batch create request
type BatchCreateGroupRequest struct {
	Groups []CreateGroupRequest `json:"groups"`
}

// BatchUpdateGroupRequest represents batch update request
type BatchUpdateGroupRequest struct {
	Updates []struct {
		PublicID    string           `json:"public_id" example:"grp-abc123"`
		Name        *json.RawMessage `json:"name,omitempty" swaggertype:"object"`
		Description *json.RawMessage `json:"description,omitempty" swaggertype:"object"`
	} `json:"updates"`
}

// BatchDeleteGroupRequest represents batch delete request
type BatchDeleteGroupRequest struct {
	PublicIDs []string `json:"public_ids" example:"grp-abc123,grp-xyz789"`
}

// GroupUser represents the group-user mapping
type GroupUser struct {
	GroupID   int       `json:"group_id"`
	UserID    int       `json:"user_id"`
	CreatedAt time.Time `json:"-"`
}

// GroupUserResponse represents the API response for group-user mapping
type GroupUserResponse struct {
	GroupPublicID string `json:"group_public_id" example:"grp-abc123"`
	UserID        int    `json:"user_id" example:"1"`
}

// GroupRole represents the group-role mapping
type GroupRole struct {
	GroupID   int       `json:"group_id"`
	RoleID    int       `json:"role_id"`
	CreatedAt time.Time `json:"-"`
}

// GroupRoleResponse represents the API response for group-role mapping
type GroupRoleResponse struct {
	GroupPublicID string `json:"group_public_id" example:"grp-abc123"`
	RoleID        int    `json:"role_id" example:"1"`
	RoleName      string `json:"role_name,omitempty" example:"admin"`
}

// AssignUsersRequest represents the request to assign users to a group
type AssignUsersRequest struct {
	UserIDs []int `json:"user_ids" example:"1,2,3"`
}

// AssignRolesRequest represents the request to assign roles to a group
type AssignRolesRequest struct {
	RoleIDs []int `json:"role_ids" example:"1,2,3"`
}

// GroupListResponseWrapper wraps the list response with pagination
type GroupListResponseWrapper struct {
	Data       []GroupResponse `json:"data"`
	Page       int             `json:"page" example:"1"`
	Limit      int             `json:"limit" example:"10"`
	TotalCount int             `json:"total_count" example:"100"`
	TotalPages int             `json:"total_pages" example:"10"`
}

// BatchOperationResponse represents batch operation result
type BatchOperationResponse struct {
	SuccessCount int      `json:"success_count" example:"5"`
	FailedCount  int      `json:"failed_count" example:"0"`
	FailedIDs    []string `json:"failed_ids,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error" example:"Bad Request"`
	Message string `json:"message" example:"Invalid group name"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string `json:"message" example:"Group deleted successfully"`
}

// ToResponse converts a Group to GroupResponse
func (g *Group) ToResponse() GroupResponse {
	return GroupResponse{
		ID:          g.ID,
		PublicID:    g.PublicID,
		Name:        g.Name,
		Description: g.Description,
	}
}
