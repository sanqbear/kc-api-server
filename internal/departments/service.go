package departments

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrDepartmentNotFound     = errors.New("department not found")
	ErrInvalidName            = errors.New("name must have at least one locale value")
	ErrEmptyBatchRequest      = errors.New("batch request cannot be empty")
	ErrCircularReference      = errors.New("circular reference detected")
	ErrHasChildren            = errors.New("department has children and cannot be deleted")
	ErrParentDepartmentNotFound = errors.New("parent department not found")
)

// Service defines the interface for department business logic
type Service interface {
	Create(ctx context.Context, req *CreateDepartmentRequest) (*DepartmentResponse, error)
	GetByPublicID(ctx context.Context, publicID string) (*DepartmentResponse, error)
	List(ctx context.Context, page, limit int) (*DepartmentListResponseWrapper, error)
	Update(ctx context.Context, publicID string, req *UpdateDepartmentRequest) (*DepartmentResponse, error)
	Delete(ctx context.Context, publicID string) error
	Search(ctx context.Context, criteria *SearchDepartmentRequest, page, limit int) (*DepartmentListResponseWrapper, error)
	BatchCreate(ctx context.Context, req *BatchCreateDepartmentRequest) (*BatchOperationResponse, error)
	BatchUpdate(ctx context.Context, req *BatchUpdateDepartmentRequest) (*BatchOperationResponse, error)
	BatchDelete(ctx context.Context, req *BatchDeleteDepartmentRequest) (*BatchOperationResponse, error)
	GetTree(ctx context.Context) ([]DepartmentTreeResponse, error)
	GetSubtree(ctx context.Context, publicID string) (*DepartmentTreeResponse, error)
	Move(ctx context.Context, publicID string, req *MoveDepartmentRequest) error
	GetChildren(ctx context.Context, publicID string) ([]DepartmentResponse, error)
}

type service struct {
	repo Repository
}

// NewService creates a new department service
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// Create creates a new department
func (s *service) Create(ctx context.Context, req *CreateDepartmentRequest) (*DepartmentResponse, error) {
	if !hasAtLeastOneLocale(req.Name) {
		return nil, ErrInvalidName
	}

	var parentID *int
	if req.ParentDepartmentPublicID != nil && *req.ParentDepartmentPublicID != "" {
		id, err := s.repo.GetIDByPublicID(ctx, *req.ParentDepartmentPublicID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrParentDepartmentNotFound
			}
			return nil, fmt.Errorf("failed to get parent department: %w", err)
		}
		parentID = &id
	}

	dept := &Department{
		Name:               req.Name,
		Description:        getOrDefault(req.Description, json.RawMessage("{}")),
		ParentDepartmentID: parentID,
	}

	if err := s.repo.Create(ctx, dept); err != nil {
		return nil, fmt.Errorf("failed to create department: %w", err)
	}

	parentPublicID, _ := s.repo.GetParentPublicID(ctx, dept.ParentDepartmentID)
	response := dept.ToResponse(parentPublicID)
	return &response, nil
}

// GetByPublicID retrieves a department by public ID
func (s *service) GetByPublicID(ctx context.Context, publicID string) (*DepartmentResponse, error) {
	dept, err := s.repo.GetByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDepartmentNotFound
		}
		return nil, fmt.Errorf("failed to get department: %w", err)
	}

	parentPublicID, _ := s.repo.GetParentPublicID(ctx, dept.ParentDepartmentID)
	response := dept.ToResponse(parentPublicID)
	return &response, nil
}

// List retrieves a paginated list of departments
func (s *service) List(ctx context.Context, page, limit int) (*DepartmentListResponseWrapper, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	depts, totalCount, err := s.repo.List(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list departments: %w", err)
	}

	var responses []DepartmentResponse
	for _, dept := range depts {
		parentPublicID, _ := s.repo.GetParentPublicID(ctx, dept.ParentDepartmentID)
		responses = append(responses, dept.ToResponse(parentPublicID))
	}

	totalPages := (totalCount + limit - 1) / limit

	return &DepartmentListResponseWrapper{
		Data:       responses,
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

// Update updates an existing department
func (s *service) Update(ctx context.Context, publicID string, req *UpdateDepartmentRequest) (*DepartmentResponse, error) {
	existingDept, err := s.repo.GetByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDepartmentNotFound
		}
		return nil, fmt.Errorf("failed to get department: %w", err)
	}

	if req.Name != nil {
		if !hasAtLeastOneLocale(*req.Name) {
			return nil, ErrInvalidName
		}
		existingDept.Name = *req.Name
	}
	if req.Description != nil {
		existingDept.Description = *req.Description
	}
	if req.ParentDepartmentPublicID != nil {
		if *req.ParentDepartmentPublicID == "" {
			existingDept.ParentDepartmentID = nil
		} else {
			// Check for circular reference
			if err := s.checkCircularReference(ctx, existingDept.ID, *req.ParentDepartmentPublicID); err != nil {
				return nil, err
			}

			parentID, err := s.repo.GetIDByPublicID(ctx, *req.ParentDepartmentPublicID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, ErrParentDepartmentNotFound
				}
				return nil, fmt.Errorf("failed to get parent department: %w", err)
			}
			existingDept.ParentDepartmentID = &parentID
		}
	}

	if err := s.repo.Update(ctx, existingDept.ID, existingDept); err != nil {
		return nil, fmt.Errorf("failed to update department: %w", err)
	}

	parentPublicID, _ := s.repo.GetParentPublicID(ctx, existingDept.ParentDepartmentID)
	response := existingDept.ToResponse(parentPublicID)
	return &response, nil
}

// Delete soft-deletes a department
func (s *service) Delete(ctx context.Context, publicID string) error {
	dept, err := s.repo.GetByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrDepartmentNotFound
		}
		return fmt.Errorf("failed to get department: %w", err)
	}

	// Check if department has children
	hasChildren, err := s.repo.HasChildren(ctx, dept.ID)
	if err != nil {
		return fmt.Errorf("failed to check children: %w", err)
	}
	if hasChildren {
		return ErrHasChildren
	}

	if err := s.repo.Delete(ctx, dept.ID); err != nil {
		return fmt.Errorf("failed to delete department: %w", err)
	}
	return nil
}

// Search searches for departments based on criteria
func (s *service) Search(ctx context.Context, criteria *SearchDepartmentRequest, page, limit int) (*DepartmentListResponseWrapper, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	depts, totalCount, err := s.repo.Search(ctx, criteria, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search departments: %w", err)
	}

	var responses []DepartmentResponse
	for _, dept := range depts {
		parentPublicID, _ := s.repo.GetParentPublicID(ctx, dept.ParentDepartmentID)
		responses = append(responses, dept.ToResponse(parentPublicID))
	}

	totalPages := (totalCount + limit - 1) / limit

	return &DepartmentListResponseWrapper{
		Data:       responses,
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

// BatchCreate creates multiple departments
func (s *service) BatchCreate(ctx context.Context, req *BatchCreateDepartmentRequest) (*BatchOperationResponse, error) {
	if len(req.Departments) == 0 {
		return nil, ErrEmptyBatchRequest
	}

	var depts []Department
	for _, r := range req.Departments {
		if !hasAtLeastOneLocale(r.Name) {
			continue
		}

		var parentID *int
		if r.ParentDepartmentPublicID != nil && *r.ParentDepartmentPublicID != "" {
			id, err := s.repo.GetIDByPublicID(ctx, *r.ParentDepartmentPublicID)
			if err != nil {
				continue
			}
			parentID = &id
		}

		depts = append(depts, Department{
			Name:               r.Name,
			Description:        getOrDefault(r.Description, json.RawMessage("{}")),
			ParentDepartmentID: parentID,
		})
	}

	successCount, err := s.repo.BatchCreate(ctx, depts)
	if err != nil {
		return nil, fmt.Errorf("failed to batch create: %w", err)
	}

	failedCount := len(req.Departments) - successCount

	return &BatchOperationResponse{
		SuccessCount: successCount,
		FailedCount:  failedCount,
	}, nil
}

// BatchUpdate updates multiple departments
func (s *service) BatchUpdate(ctx context.Context, req *BatchUpdateDepartmentRequest) (*BatchOperationResponse, error) {
	if len(req.Updates) == 0 {
		return nil, ErrEmptyBatchRequest
	}

	var updates []Department
	for _, u := range req.Updates {
		existingDept, err := s.repo.GetByPublicID(ctx, u.PublicID)
		if err != nil {
			continue
		}

		if u.Name != nil {
			if !hasAtLeastOneLocale(*u.Name) {
				continue
			}
			existingDept.Name = *u.Name
		}
		if u.Description != nil {
			existingDept.Description = *u.Description
		}
		if u.ParentDepartmentPublicID != nil {
			if *u.ParentDepartmentPublicID == "" {
				existingDept.ParentDepartmentID = nil
			} else {
				parentID, err := s.repo.GetIDByPublicID(ctx, *u.ParentDepartmentPublicID)
				if err != nil {
					continue
				}
				existingDept.ParentDepartmentID = &parentID
			}
		}

		updates = append(updates, *existingDept)
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

// BatchDelete soft-deletes multiple departments
func (s *service) BatchDelete(ctx context.Context, req *BatchDeleteDepartmentRequest) (*BatchOperationResponse, error) {
	if len(req.PublicIDs) == 0 {
		return nil, ErrEmptyBatchRequest
	}

	var ids []int
	for _, publicID := range req.PublicIDs {
		id, err := s.repo.GetIDByPublicID(ctx, publicID)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}

	successCount, failedIDs, err := s.repo.BatchDelete(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to batch delete: %w", err)
	}

	return &BatchOperationResponse{
		SuccessCount: successCount,
		FailedCount:  len(failedIDs),
		FailedIDs:    failedIDs,
	}, nil
}

// GetTree retrieves the full department tree
func (s *service) GetTree(ctx context.Context) ([]DepartmentTreeResponse, error) {
	rootDepts, err := s.repo.GetRootDepartments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get root departments: %w", err)
	}

	var tree []DepartmentTreeResponse
	for _, dept := range rootDepts {
		treeNode, err := s.buildTreeNode(ctx, &dept)
		if err != nil {
			return nil, err
		}
		tree = append(tree, *treeNode)
	}

	return tree, nil
}

// GetSubtree retrieves a subtree starting from a specific department
func (s *service) GetSubtree(ctx context.Context, publicID string) (*DepartmentTreeResponse, error) {
	dept, err := s.repo.GetByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDepartmentNotFound
		}
		return nil, fmt.Errorf("failed to get department: %w", err)
	}

	return s.buildTreeNode(ctx, dept)
}

// Move moves a department to a new parent
func (s *service) Move(ctx context.Context, publicID string, req *MoveDepartmentRequest) error {
	dept, err := s.repo.GetByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrDepartmentNotFound
		}
		return fmt.Errorf("failed to get department: %w", err)
	}

	if req.NewParentDepartmentPublicID == nil || *req.NewParentDepartmentPublicID == "" {
		dept.ParentDepartmentID = nil
	} else {
		// Check for circular reference
		if err := s.checkCircularReference(ctx, dept.ID, *req.NewParentDepartmentPublicID); err != nil {
			return err
		}

		parentID, err := s.repo.GetIDByPublicID(ctx, *req.NewParentDepartmentPublicID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrParentDepartmentNotFound
			}
			return fmt.Errorf("failed to get parent department: %w", err)
		}
		dept.ParentDepartmentID = &parentID
	}

	if err := s.repo.Update(ctx, dept.ID, dept); err != nil {
		return fmt.Errorf("failed to move department: %w", err)
	}

	return nil
}

// GetChildren retrieves direct children of a department
func (s *service) GetChildren(ctx context.Context, publicID string) ([]DepartmentResponse, error) {
	dept, err := s.repo.GetByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDepartmentNotFound
		}
		return nil, fmt.Errorf("failed to get department: %w", err)
	}

	children, err := s.repo.GetChildren(ctx, dept.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get children: %w", err)
	}

	var responses []DepartmentResponse
	for _, child := range children {
		responses = append(responses, child.ToResponse(&publicID))
	}

	return responses, nil
}

// buildTreeNode recursively builds a tree node
func (s *service) buildTreeNode(ctx context.Context, dept *Department) (*DepartmentTreeResponse, error) {
	children, err := s.repo.GetChildren(ctx, dept.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get children: %w", err)
	}

	var childNodes []DepartmentTreeResponse
	for _, child := range children {
		childNode, err := s.buildTreeNode(ctx, &child)
		if err != nil {
			return nil, err
		}
		childNodes = append(childNodes, *childNode)
	}

	parentPublicID, _ := s.repo.GetParentPublicID(ctx, dept.ParentDepartmentID)
	treeResponse := dept.ToTreeResponse(parentPublicID, childNodes)
	return &treeResponse, nil
}

// checkCircularReference checks if setting a parent would create a circular reference
func (s *service) checkCircularReference(ctx context.Context, deptID int, newParentPublicID string) error {
	newParentID, err := s.repo.GetIDByPublicID(ctx, newParentPublicID)
	if err != nil {
		return err
	}

	// Can't set self as parent
	if deptID == newParentID {
		return ErrCircularReference
	}

	// Check if new parent is a descendant of the department
	descendants, err := s.repo.GetAllDescendants(ctx, deptID)
	if err != nil {
		return fmt.Errorf("failed to get descendants: %w", err)
	}

	for _, desc := range descendants {
		if desc.ID == newParentID {
			return ErrCircularReference
		}
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
