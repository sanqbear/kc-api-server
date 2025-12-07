package departments

import (
	"encoding/json"
	"time"
)

// Department represents a department in the system
type Department struct {
	ID                 int             `json:"-"`
	PublicID           string          `json:"public_id"`
	Name               json.RawMessage `json:"name"`
	Description        json.RawMessage `json:"description"`
	ParentDepartmentID *int            `json:"-"`
	IsDeleted          bool            `json:"-"`
	CreatedAt          time.Time       `json:"-"`
	UpdatedAt          time.Time       `json:"-"`
}

// DepartmentResponse represents the API response for a department
type DepartmentResponse struct {
	ID                       int             `json:"id" example:"1"`
	PublicID                 string          `json:"public_id" example:"dept-abc123"`
	Name                     json.RawMessage `json:"name" swaggertype:"object"`
	Description              json.RawMessage `json:"description" swaggertype:"object"`
	ParentDepartmentPublicID *string         `json:"parent_department_public_id,omitempty" example:"dept-parent123"`
}

// DepartmentTreeResponse represents a department with its children for tree structure
type DepartmentTreeResponse struct {
	ID                       int                      `json:"id" example:"1"`
	PublicID                 string                   `json:"public_id" example:"dept-abc123"`
	Name                     json.RawMessage          `json:"name" swaggertype:"object"`
	Description              json.RawMessage          `json:"description" swaggertype:"object"`
	ParentDepartmentPublicID *string                  `json:"parent_department_public_id,omitempty" example:"dept-parent123"`
	Children                 []DepartmentTreeResponse `json:"children,omitempty"`
}

// CreateDepartmentRequest represents the request to create a department
type CreateDepartmentRequest struct {
	Name                     json.RawMessage  `json:"name" swaggertype:"object"`
	Description              *json.RawMessage `json:"description,omitempty" swaggertype:"object"`
	ParentDepartmentPublicID *string          `json:"parent_department_public_id,omitempty" example:"dept-parent123"`
}

// UpdateDepartmentRequest represents the request to update a department
type UpdateDepartmentRequest struct {
	Name                     *json.RawMessage `json:"name,omitempty" swaggertype:"object"`
	Description              *json.RawMessage `json:"description,omitempty" swaggertype:"object"`
	ParentDepartmentPublicID *string          `json:"parent_department_public_id,omitempty" example:"dept-parent123"`
}

// SearchDepartmentRequest represents the search criteria for departments
type SearchDepartmentRequest struct {
	Name        *string `json:"name,omitempty" example:"Engineering"`
	Description *string `json:"description,omitempty" example:"tech"`
}

// BatchCreateDepartmentRequest represents batch create request
type BatchCreateDepartmentRequest struct {
	Departments []CreateDepartmentRequest `json:"departments"`
}

// BatchUpdateDepartmentRequest represents batch update request
type BatchUpdateDepartmentRequest struct {
	Updates []struct {
		PublicID                 string           `json:"public_id" example:"dept-abc123"`
		Name                     *json.RawMessage `json:"name,omitempty" swaggertype:"object"`
		Description              *json.RawMessage `json:"description,omitempty" swaggertype:"object"`
		ParentDepartmentPublicID *string          `json:"parent_department_public_id,omitempty" example:"dept-parent123"`
	} `json:"updates"`
}

// BatchDeleteDepartmentRequest represents batch delete request
type BatchDeleteDepartmentRequest struct {
	PublicIDs []string `json:"public_ids" example:"dept-abc123,dept-xyz789"`
}

// MoveDepartmentRequest represents request to move a department to a new parent
type MoveDepartmentRequest struct {
	NewParentDepartmentPublicID *string `json:"new_parent_department_public_id,omitempty" example:"dept-parent123"`
}

// DepartmentListResponseWrapper wraps the list response with pagination
type DepartmentListResponseWrapper struct {
	Data       []DepartmentResponse `json:"data"`
	Page       int                  `json:"page" example:"1"`
	Limit      int                  `json:"limit" example:"10"`
	TotalCount int                  `json:"total_count" example:"100"`
	TotalPages int                  `json:"total_pages" example:"10"`
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
	Message string `json:"message" example:"Invalid department name"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string `json:"message" example:"Department deleted successfully"`
}

// ToResponse converts a Department to DepartmentResponse
func (d *Department) ToResponse(parentPublicID *string) DepartmentResponse {
	return DepartmentResponse{
		ID:                       d.ID,
		PublicID:                 d.PublicID,
		Name:                     d.Name,
		Description:              d.Description,
		ParentDepartmentPublicID: parentPublicID,
	}
}

// ToTreeResponse converts a Department to DepartmentTreeResponse
func (d *Department) ToTreeResponse(parentPublicID *string, children []DepartmentTreeResponse) DepartmentTreeResponse {
	return DepartmentTreeResponse{
		ID:                       d.ID,
		PublicID:                 d.PublicID,
		Name:                     d.Name,
		Description:              d.Description,
		ParentDepartmentPublicID: parentPublicID,
		Children:                 children,
	}
}
