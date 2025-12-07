package commoncodes

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrCommonCodeNotFound    = errors.New("common code not found")
	ErrCategoryCodeExists    = errors.New("category and code combination already exists")
	ErrInvalidCategory       = errors.New("category is required")
	ErrInvalidCode           = errors.New("code is required")
	ErrInvalidName           = errors.New("name must have at least one locale value")
	ErrEmptyBatchRequest     = errors.New("batch request cannot be empty")
	ErrInvalidReorderRequest = errors.New("invalid reorder request")
)

// Service defines the interface for common code business logic
type Service interface {
	Create(ctx context.Context, req *CreateCommonCodeRequest) (*CommonCodeResponse, error)
	GetByID(ctx context.Context, id int) (*CommonCodeResponse, error)
	List(ctx context.Context, page, limit int) (*CommonCodeListResponseWrapper, error)
	Update(ctx context.Context, id int, req *UpdateCommonCodeRequest) (*CommonCodeResponse, error)
	Delete(ctx context.Context, id int) error
	Search(ctx context.Context, criteria *SearchCommonCodeRequest, page, limit int) (*CommonCodeListResponseWrapper, error)
	BatchCreate(ctx context.Context, req *BatchCreateRequest) (*BatchOperationResponse, error)
	BatchUpdate(ctx context.Context, req *BatchUpdateRequest) (*BatchOperationResponse, error)
	BatchDelete(ctx context.Context, req *BatchDeleteRequest) (*BatchOperationResponse, error)
	ListCategories(ctx context.Context) ([]CategoryResponse, error)
	GetByCategory(ctx context.Context, category string) ([]CommonCodeResponse, error)
	Reorder(ctx context.Context, category string, req *ReorderRequest) error
}

type service struct {
	repo Repository
}

// NewService creates a new common code service
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// Create creates a new common code
func (s *service) Create(ctx context.Context, req *CreateCommonCodeRequest) (*CommonCodeResponse, error) {
	// Validate required fields
	if req.Category == "" {
		return nil, ErrInvalidCategory
	}
	if req.Code == "" {
		return nil, ErrInvalidCode
	}
	if !hasAtLeastOneLocale(req.Name) {
		return nil, ErrInvalidName
	}

	// Check if category + code combination exists
	exists, err := s.repo.ExistsByCategoryAndCode(ctx, req.Category, req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to check existence: %w", err)
	}
	if exists {
		return nil, ErrCategoryCodeExists
	}

	code := &CommonCode{
		Category:     req.Category,
		Code:         req.Code,
		Name:         req.Name,
		Description:  getOrDefault(req.Description, json.RawMessage("{}")),
		ExtraPayload: getOrDefault(req.ExtraPayload, json.RawMessage("{}")),
		SortOrder:    getIntOrDefault(req.SortOrder, 0),
	}

	if err := s.repo.Create(ctx, code); err != nil {
		return nil, fmt.Errorf("failed to create common code: %w", err)
	}

	response := code.ToResponse()
	return &response, nil
}

// GetByID retrieves a common code by ID
func (s *service) GetByID(ctx context.Context, id int) (*CommonCodeResponse, error) {
	code, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCommonCodeNotFound
		}
		return nil, fmt.Errorf("failed to get common code: %w", err)
	}

	response := code.ToResponse()
	return &response, nil
}

// List retrieves a paginated list of common codes
func (s *service) List(ctx context.Context, page, limit int) (*CommonCodeListResponseWrapper, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	codes, totalCount, err := s.repo.List(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list common codes: %w", err)
	}

	var responses []CommonCodeResponse
	for _, code := range codes {
		responses = append(responses, code.ToResponse())
	}

	totalPages := (totalCount + limit - 1) / limit

	return &CommonCodeListResponseWrapper{
		Data:       responses,
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

// Update updates an existing common code
func (s *service) Update(ctx context.Context, id int, req *UpdateCommonCodeRequest) (*CommonCodeResponse, error) {
	// Get existing code
	existingCode, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCommonCodeNotFound
		}
		return nil, fmt.Errorf("failed to get common code: %w", err)
	}

	// Update fields if provided
	if req.Category != nil && *req.Category != "" {
		existingCode.Category = *req.Category
	}
	if req.Code != nil && *req.Code != "" {
		existingCode.Code = *req.Code
	}
	if req.Name != nil {
		if !hasAtLeastOneLocale(*req.Name) {
			return nil, ErrInvalidName
		}
		existingCode.Name = *req.Name
	}
	if req.Description != nil {
		existingCode.Description = *req.Description
	}
	if req.ExtraPayload != nil {
		existingCode.ExtraPayload = *req.ExtraPayload
	}
	if req.SortOrder != nil {
		existingCode.SortOrder = *req.SortOrder
	}

	// Check if category + code combination exists (excluding current ID)
	if req.Category != nil || req.Code != nil {
		exists, err := s.repo.ExistsByCategoryAndCodeExcludingID(ctx, existingCode.Category, existingCode.Code, id)
		if err != nil {
			return nil, fmt.Errorf("failed to check existence: %w", err)
		}
		if exists {
			return nil, ErrCategoryCodeExists
		}
	}

	if err := s.repo.Update(ctx, id, existingCode); err != nil {
		return nil, fmt.Errorf("failed to update common code: %w", err)
	}

	response := existingCode.ToResponse()
	return &response, nil
}

// Delete deletes a common code
func (s *service) Delete(ctx context.Context, id int) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCommonCodeNotFound
		}
		return fmt.Errorf("failed to delete common code: %w", err)
	}
	return nil
}

// Search searches for common codes based on criteria
func (s *service) Search(ctx context.Context, criteria *SearchCommonCodeRequest, page, limit int) (*CommonCodeListResponseWrapper, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	codes, totalCount, err := s.repo.Search(ctx, criteria, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search common codes: %w", err)
	}

	var responses []CommonCodeResponse
	for _, code := range codes {
		responses = append(responses, code.ToResponse())
	}

	totalPages := (totalCount + limit - 1) / limit

	return &CommonCodeListResponseWrapper{
		Data:       responses,
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

// BatchCreate creates multiple common codes
func (s *service) BatchCreate(ctx context.Context, req *BatchCreateRequest) (*BatchOperationResponse, error) {
	if len(req.Codes) == 0 {
		return nil, ErrEmptyBatchRequest
	}

	var codes []CommonCode
	for _, r := range req.Codes {
		// Validate each code
		if r.Category == "" || r.Code == "" || !hasAtLeastOneLocale(r.Name) {
			continue // Skip invalid entries
		}

		codes = append(codes, CommonCode{
			Category:     r.Category,
			Code:         r.Code,
			Name:         r.Name,
			Description:  getOrDefault(r.Description, json.RawMessage("{}")),
			ExtraPayload: getOrDefault(r.ExtraPayload, json.RawMessage("{}")),
			SortOrder:    getIntOrDefault(r.SortOrder, 0),
		})
	}

	successCount, err := s.repo.BatchCreate(ctx, codes)
	if err != nil {
		return nil, fmt.Errorf("failed to batch create: %w", err)
	}

	failedCount := len(req.Codes) - successCount

	return &BatchOperationResponse{
		SuccessCount: successCount,
		FailedCount:  failedCount,
	}, nil
}

// BatchUpdate updates multiple common codes
func (s *service) BatchUpdate(ctx context.Context, req *BatchUpdateRequest) (*BatchOperationResponse, error) {
	if len(req.Updates) == 0 {
		return nil, ErrEmptyBatchRequest
	}

	var updates []CommonCode
	for _, u := range req.Updates {
		// Get existing code
		existingCode, err := s.repo.GetByID(ctx, u.ID)
		if err != nil {
			continue // Skip if not found
		}

		// Apply updates
		if u.Category != nil && *u.Category != "" {
			existingCode.Category = *u.Category
		}
		if u.Code != nil && *u.Code != "" {
			existingCode.Code = *u.Code
		}
		if u.Name != nil {
			if !hasAtLeastOneLocale(*u.Name) {
				continue // Skip invalid
			}
			existingCode.Name = *u.Name
		}
		if u.Description != nil {
			existingCode.Description = *u.Description
		}
		if u.ExtraPayload != nil {
			existingCode.ExtraPayload = *u.ExtraPayload
		}
		if u.SortOrder != nil {
			existingCode.SortOrder = *u.SortOrder
		}

		updates = append(updates, *existingCode)
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

// BatchDelete deletes multiple common codes
func (s *service) BatchDelete(ctx context.Context, req *BatchDeleteRequest) (*BatchOperationResponse, error) {
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

// ListCategories retrieves all unique categories
func (s *service) ListCategories(ctx context.Context) ([]CategoryResponse, error) {
	categories, err := s.repo.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}

	var responses []CategoryResponse
	for _, cat := range categories {
		responses = append(responses, CategoryResponse{Category: cat})
	}

	return responses, nil
}

// GetByCategory retrieves all codes in a category
func (s *service) GetByCategory(ctx context.Context, category string) ([]CommonCodeResponse, error) {
	codes, err := s.repo.GetByCategory(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("failed to get codes by category: %w", err)
	}

	var responses []CommonCodeResponse
	for _, code := range codes {
		responses = append(responses, code.ToResponse())
	}

	return responses, nil
}

// Reorder updates sort orders for codes in a category
func (s *service) Reorder(ctx context.Context, category string, req *ReorderRequest) error {
	if len(req.Orders) == 0 {
		return ErrInvalidReorderRequest
	}

	orders := make(map[int]int)
	for _, o := range req.Orders {
		orders[o.ID] = o.SortOrder
	}

	if err := s.repo.Reorder(ctx, category, orders); err != nil {
		return fmt.Errorf("failed to reorder: %w", err)
	}

	return nil
}

// Helper functions

func hasAtLeastOneLocale(jsonData json.RawMessage) bool {
	if len(jsonData) == 0 {
		return false
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return false
	}

	return len(data) > 0
}

func getOrDefault(value *json.RawMessage, defaultValue json.RawMessage) json.RawMessage {
	if value != nil {
		return *value
	}
	return defaultValue
}

func getIntOrDefault(value *int, defaultValue int) int {
	if value != nil {
		return *value
	}
	return defaultValue
}
