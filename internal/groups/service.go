package groups

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrGroupNotFound     = errors.New("group not found")
	ErrInvalidName       = errors.New("name must have at least one locale value")
	ErrEmptyBatchRequest = errors.New("batch request cannot be empty")
	ErrGroupUserNotFound = errors.New("group user not found")
	ErrGroupRoleNotFound = errors.New("group role not found")
)

// Service defines the interface for group business logic
type Service interface {
	Create(ctx context.Context, req *CreateGroupRequest) (*GroupResponse, error)
	GetByPublicID(ctx context.Context, publicID string) (*GroupResponse, error)
	List(ctx context.Context, page, limit int) (*GroupListResponseWrapper, error)
	Update(ctx context.Context, publicID string, req *UpdateGroupRequest) (*GroupResponse, error)
	Delete(ctx context.Context, publicID string) error
	Search(ctx context.Context, criteria *SearchGroupRequest, page, limit int) (*GroupListResponseWrapper, error)
	BatchCreate(ctx context.Context, req *BatchCreateGroupRequest) (*BatchOperationResponse, error)
	BatchUpdate(ctx context.Context, req *BatchUpdateGroupRequest) (*BatchOperationResponse, error)
	BatchDelete(ctx context.Context, req *BatchDeleteGroupRequest) (*BatchOperationResponse, error)

	// Group user operations
	GetGroupUsers(ctx context.Context, publicID string) ([]int, error)
	AssignUsersToGroup(ctx context.Context, publicID string, req *AssignUsersRequest) error
	RemoveUserFromGroup(ctx context.Context, publicID string, userID int) error
	GetUserGroups(ctx context.Context, userID int) ([]GroupResponse, error)

	// Group role operations
	GetGroupRoles(ctx context.Context, publicID string) ([]GroupRoleResponse, error)
	AssignRolesToGroup(ctx context.Context, publicID string, req *AssignRolesRequest) error
	RemoveRoleFromGroup(ctx context.Context, publicID string, roleID int) error
}

type service struct {
	repo Repository
}

// NewService creates a new group service
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// Create creates a new group
func (s *service) Create(ctx context.Context, req *CreateGroupRequest) (*GroupResponse, error) {
	if !hasAtLeastOneLocale(req.Name) {
		return nil, ErrInvalidName
	}

	group := &Group{
		Name:        req.Name,
		Description: getOrDefault(req.Description, json.RawMessage("{}")),
	}

	if err := s.repo.Create(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	response := group.ToResponse()
	return &response, nil
}

// GetByPublicID retrieves a group by public ID
func (s *service) GetByPublicID(ctx context.Context, publicID string) (*GroupResponse, error) {
	group, err := s.repo.GetByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	response := group.ToResponse()
	return &response, nil
}

// List retrieves a paginated list of groups
func (s *service) List(ctx context.Context, page, limit int) (*GroupListResponseWrapper, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	groups, totalCount, err := s.repo.List(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	var responses []GroupResponse
	for _, group := range groups {
		responses = append(responses, group.ToResponse())
	}

	totalPages := (totalCount + limit - 1) / limit

	return &GroupListResponseWrapper{
		Data:       responses,
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

// Update updates an existing group
func (s *service) Update(ctx context.Context, publicID string, req *UpdateGroupRequest) (*GroupResponse, error) {
	existingGroup, err := s.repo.GetByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	if req.Name != nil {
		if !hasAtLeastOneLocale(*req.Name) {
			return nil, ErrInvalidName
		}
		existingGroup.Name = *req.Name
	}
	if req.Description != nil {
		existingGroup.Description = *req.Description
	}

	if err := s.repo.Update(ctx, existingGroup.ID, existingGroup); err != nil {
		return nil, fmt.Errorf("failed to update group: %w", err)
	}

	response := existingGroup.ToResponse()
	return &response, nil
}

// Delete deletes a group
func (s *service) Delete(ctx context.Context, publicID string) error {
	group, err := s.repo.GetByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrGroupNotFound
		}
		return fmt.Errorf("failed to get group: %w", err)
	}

	if err := s.repo.Delete(ctx, group.ID); err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}
	return nil
}

// Search searches for groups based on criteria
func (s *service) Search(ctx context.Context, criteria *SearchGroupRequest, page, limit int) (*GroupListResponseWrapper, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	groups, totalCount, err := s.repo.Search(ctx, criteria, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search groups: %w", err)
	}

	var responses []GroupResponse
	for _, group := range groups {
		responses = append(responses, group.ToResponse())
	}

	totalPages := (totalCount + limit - 1) / limit

	return &GroupListResponseWrapper{
		Data:       responses,
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

// BatchCreate creates multiple groups
func (s *service) BatchCreate(ctx context.Context, req *BatchCreateGroupRequest) (*BatchOperationResponse, error) {
	if len(req.Groups) == 0 {
		return nil, ErrEmptyBatchRequest
	}

	var groups []Group
	for _, r := range req.Groups {
		if !hasAtLeastOneLocale(r.Name) {
			continue
		}

		groups = append(groups, Group{
			Name:        r.Name,
			Description: getOrDefault(r.Description, json.RawMessage("{}")),
		})
	}

	successCount, err := s.repo.BatchCreate(ctx, groups)
	if err != nil {
		return nil, fmt.Errorf("failed to batch create: %w", err)
	}

	failedCount := len(req.Groups) - successCount

	return &BatchOperationResponse{
		SuccessCount: successCount,
		FailedCount:  failedCount,
	}, nil
}

// BatchUpdate updates multiple groups
func (s *service) BatchUpdate(ctx context.Context, req *BatchUpdateGroupRequest) (*BatchOperationResponse, error) {
	if len(req.Updates) == 0 {
		return nil, ErrEmptyBatchRequest
	}

	var updates []Group
	for _, u := range req.Updates {
		existingGroup, err := s.repo.GetByPublicID(ctx, u.PublicID)
		if err != nil {
			continue
		}

		if u.Name != nil {
			if !hasAtLeastOneLocale(*u.Name) {
				continue
			}
			existingGroup.Name = *u.Name
		}
		if u.Description != nil {
			existingGroup.Description = *u.Description
		}

		updates = append(updates, *existingGroup)
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

// BatchDelete deletes multiple groups
func (s *service) BatchDelete(ctx context.Context, req *BatchDeleteGroupRequest) (*BatchOperationResponse, error) {
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

// GetGroupUsers retrieves all users in a group
func (s *service) GetGroupUsers(ctx context.Context, publicID string) ([]int, error) {
	group, err := s.repo.GetByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	userIDs, err := s.repo.GetGroupUsers(ctx, group.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group users: %w", err)
	}
	return userIDs, nil
}

// AssignUsersToGroup assigns users to a group
func (s *service) AssignUsersToGroup(ctx context.Context, publicID string, req *AssignUsersRequest) error {
	group, err := s.repo.GetByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrGroupNotFound
		}
		return fmt.Errorf("failed to get group: %w", err)
	}

	if err := s.repo.AddUsersToGroup(ctx, group.ID, req.UserIDs); err != nil {
		return fmt.Errorf("failed to assign users to group: %w", err)
	}
	return nil
}

// RemoveUserFromGroup removes a user from a group
func (s *service) RemoveUserFromGroup(ctx context.Context, publicID string, userID int) error {
	group, err := s.repo.GetByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrGroupNotFound
		}
		return fmt.Errorf("failed to get group: %w", err)
	}

	if err := s.repo.RemoveUserFromGroup(ctx, group.ID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrGroupUserNotFound
		}
		return fmt.Errorf("failed to remove user from group: %w", err)
	}
	return nil
}

// GetUserGroups retrieves all groups a user belongs to
func (s *service) GetUserGroups(ctx context.Context, userID int) ([]GroupResponse, error) {
	groups, err := s.repo.GetUserGroups(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user groups: %w", err)
	}

	var responses []GroupResponse
	for _, group := range groups {
		responses = append(responses, group.ToResponse())
	}
	return responses, nil
}

// GetGroupRoles retrieves all roles assigned to a group
func (s *service) GetGroupRoles(ctx context.Context, publicID string) ([]GroupRoleResponse, error) {
	group, err := s.repo.GetByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	roles, err := s.repo.GetGroupRoles(ctx, group.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group roles: %w", err)
	}
	return roles, nil
}

// AssignRolesToGroup assigns roles to a group
func (s *service) AssignRolesToGroup(ctx context.Context, publicID string, req *AssignRolesRequest) error {
	group, err := s.repo.GetByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrGroupNotFound
		}
		return fmt.Errorf("failed to get group: %w", err)
	}

	if err := s.repo.AssignRolesToGroup(ctx, group.ID, req.RoleIDs); err != nil {
		return fmt.Errorf("failed to assign roles to group: %w", err)
	}
	return nil
}

// RemoveRoleFromGroup removes a role from a group
func (s *service) RemoveRoleFromGroup(ctx context.Context, publicID string, roleID int) error {
	group, err := s.repo.GetByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrGroupNotFound
		}
		return fmt.Errorf("failed to get group: %w", err)
	}

	if err := s.repo.RemoveRoleFromGroup(ctx, group.ID, roleID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrGroupRoleNotFound
		}
		return fmt.Errorf("failed to remove role from group: %w", err)
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
