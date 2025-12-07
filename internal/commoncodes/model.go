package commoncodes

import (
	"encoding/json"
	"time"
)

// CommonCode represents the internal common code entity in the database
type CommonCode struct {
	ID           int             `json:"-"`
	Category     string          `json:"category"`
	Code         string          `json:"code"`
	Name         json.RawMessage `json:"name"`
	Description  json.RawMessage `json:"description"`
	ExtraPayload json.RawMessage `json:"extra_payload"`
	SortOrder    int             `json:"sort_order"`
	CreatedAt    time.Time       `json:"-"`
	UpdatedAt    time.Time       `json:"-"`
}

// CommonCodeResponse represents a common code response for API
type CommonCodeResponse struct {
	ID           int             `json:"id" example:"1"`
	Category     string          `json:"category" example:"rank"`
	Code         string          `json:"code" example:"MANAGER"`
	Name         json.RawMessage `json:"name" swaggertype:"object"`
	Description  json.RawMessage `json:"description" swaggertype:"object"`
	ExtraPayload json.RawMessage `json:"extra_payload" swaggertype:"object"`
	SortOrder    int             `json:"sort_order" example:"10"`
}

// CreateCommonCodeRequest represents the request body for creating a common code
type CreateCommonCodeRequest struct {
	Category     string           `json:"category" example:"rank"`
	Code         string           `json:"code" example:"MANAGER"`
	Name         json.RawMessage  `json:"name" swaggertype:"object"`
	Description  *json.RawMessage `json:"description,omitempty" swaggertype:"object"`
	ExtraPayload *json.RawMessage `json:"extra_payload,omitempty" swaggertype:"object"`
	SortOrder    *int             `json:"sort_order,omitempty" example:"10"`
}

// UpdateCommonCodeRequest represents the request body for updating a common code
type UpdateCommonCodeRequest struct {
	Category     *string          `json:"category,omitempty" example:"rank"`
	Code         *string          `json:"code,omitempty" example:"MANAGER"`
	Name         *json.RawMessage `json:"name,omitempty" swaggertype:"object"`
	Description  *json.RawMessage `json:"description,omitempty" swaggertype:"object"`
	ExtraPayload *json.RawMessage `json:"extra_payload,omitempty" swaggertype:"object"`
	SortOrder    *int             `json:"sort_order,omitempty" example:"10"`
}

// SearchCommonCodeRequest represents the search criteria for common codes
type SearchCommonCodeRequest struct {
	Category    *string `json:"category,omitempty" example:"rank"`
	Code        *string `json:"code,omitempty" example:"MANAGER"`
	Name        *string `json:"name,omitempty" example:"Manager"`
	Description *string `json:"description,omitempty" example:"description text"`
}

// BatchCreateRequest represents batch create request
type BatchCreateRequest struct {
	Codes []CreateCommonCodeRequest `json:"codes"`
}

// BatchUpdateRequest represents batch update request with ID-based updates
type BatchUpdateRequest struct {
	Updates []struct {
		ID           int              `json:"id" example:"1"`
		Category     *string          `json:"category,omitempty" example:"rank"`
		Code         *string          `json:"code,omitempty" example:"MANAGER"`
		Name         *json.RawMessage `json:"name,omitempty" swaggertype:"object"`
		Description  *json.RawMessage `json:"description,omitempty" swaggertype:"object"`
		ExtraPayload *json.RawMessage `json:"extra_payload,omitempty" swaggertype:"object"`
		SortOrder    *int             `json:"sort_order,omitempty" example:"10"`
	} `json:"updates"`
}

// BatchDeleteRequest represents batch delete request
type BatchDeleteRequest struct {
	IDs []int `json:"ids" example:"1,2,3"`
}

// ReorderRequest represents reorder request for codes in a category
type ReorderRequest struct {
	Orders []struct {
		ID        int `json:"id" example:"1"`
		SortOrder int `json:"sort_order" example:"10"`
	} `json:"orders"`
}

// CategoryResponse represents a category name
type CategoryResponse struct {
	Category string `json:"category" example:"rank"`
}

// CommonCodeListResponseWrapper wraps the list response with pagination info
type CommonCodeListResponseWrapper struct {
	Data       []CommonCodeResponse `json:"data"`
	Page       int                  `json:"page" example:"1"`
	Limit      int                  `json:"limit" example:"10"`
	TotalCount int                  `json:"total_count" example:"100"`
	TotalPages int                  `json:"total_pages" example:"10"`
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
	Message string `json:"message" example:"Invalid category or code"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string `json:"message" example:"Common code deleted successfully"`
}

// ToResponse converts a CommonCode to CommonCodeResponse
func (c *CommonCode) ToResponse() CommonCodeResponse {
	return CommonCodeResponse{
		ID:           c.ID,
		Category:     c.Category,
		Code:         c.Code,
		Name:         c.Name,
		Description:  c.Description,
		ExtraPayload: c.ExtraPayload,
		SortOrder:    c.SortOrder,
	}
}
