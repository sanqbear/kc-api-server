package groups

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// MockService is a mock implementation of the Service interface for testing
type MockService struct {
	CreateFunc            func(ctx context.Context, req *CreateGroupRequest) (*GroupResponse, error)
	GetByPublicIDFunc     func(ctx context.Context, publicID string) (*GroupResponse, error)
	ListFunc              func(ctx context.Context, page, limit int) (*GroupListResponseWrapper, error)
	UpdateFunc            func(ctx context.Context, publicID string, req *UpdateGroupRequest) (*GroupResponse, error)
	DeleteFunc            func(ctx context.Context, publicID string) error
	SearchFunc            func(ctx context.Context, criteria *SearchGroupRequest, page, limit int) (*GroupListResponseWrapper, error)
	BatchCreateFunc       func(ctx context.Context, req *BatchCreateGroupRequest) (*BatchOperationResponse, error)
	BatchUpdateFunc       func(ctx context.Context, req *BatchUpdateGroupRequest) (*BatchOperationResponse, error)
	BatchDeleteFunc       func(ctx context.Context, req *BatchDeleteGroupRequest) (*BatchOperationResponse, error)
	GetGroupUsersFunc     func(ctx context.Context, publicID string) ([]int, error)
	AssignUsersToGroupFunc func(ctx context.Context, publicID string, req *AssignUsersRequest) error
	RemoveUserFromGroupFunc func(ctx context.Context, publicID string, userID int) error
	GetUserGroupsFunc     func(ctx context.Context, userID int) ([]GroupResponse, error)
	GetGroupRolesFunc     func(ctx context.Context, publicID string) ([]GroupRoleResponse, error)
	AssignRolesToGroupFunc func(ctx context.Context, publicID string, req *AssignRolesRequest) error
	RemoveRoleFromGroupFunc func(ctx context.Context, publicID string, roleID int) error
}

func (m *MockService) Create(ctx context.Context, req *CreateGroupRequest) (*GroupResponse, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) GetByPublicID(ctx context.Context, publicID string) (*GroupResponse, error) {
	if m.GetByPublicIDFunc != nil {
		return m.GetByPublicIDFunc(ctx, publicID)
	}
	return nil, nil
}

func (m *MockService) List(ctx context.Context, page, limit int) (*GroupListResponseWrapper, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, page, limit)
	}
	return nil, nil
}

func (m *MockService) Update(ctx context.Context, publicID string, req *UpdateGroupRequest) (*GroupResponse, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, publicID, req)
	}
	return nil, nil
}

func (m *MockService) Delete(ctx context.Context, publicID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, publicID)
	}
	return nil
}

func (m *MockService) Search(ctx context.Context, criteria *SearchGroupRequest, page, limit int) (*GroupListResponseWrapper, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, criteria, page, limit)
	}
	return nil, nil
}

func (m *MockService) BatchCreate(ctx context.Context, req *BatchCreateGroupRequest) (*BatchOperationResponse, error) {
	if m.BatchCreateFunc != nil {
		return m.BatchCreateFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) BatchUpdate(ctx context.Context, req *BatchUpdateGroupRequest) (*BatchOperationResponse, error) {
	if m.BatchUpdateFunc != nil {
		return m.BatchUpdateFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) BatchDelete(ctx context.Context, req *BatchDeleteGroupRequest) (*BatchOperationResponse, error) {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) GetGroupUsers(ctx context.Context, publicID string) ([]int, error) {
	if m.GetGroupUsersFunc != nil {
		return m.GetGroupUsersFunc(ctx, publicID)
	}
	return nil, nil
}

func (m *MockService) AssignUsersToGroup(ctx context.Context, publicID string, req *AssignUsersRequest) error {
	if m.AssignUsersToGroupFunc != nil {
		return m.AssignUsersToGroupFunc(ctx, publicID, req)
	}
	return nil
}

func (m *MockService) RemoveUserFromGroup(ctx context.Context, publicID string, userID int) error {
	if m.RemoveUserFromGroupFunc != nil {
		return m.RemoveUserFromGroupFunc(ctx, publicID, userID)
	}
	return nil
}

func (m *MockService) GetUserGroups(ctx context.Context, userID int) ([]GroupResponse, error) {
	if m.GetUserGroupsFunc != nil {
		return m.GetUserGroupsFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockService) GetGroupRoles(ctx context.Context, publicID string) ([]GroupRoleResponse, error) {
	if m.GetGroupRolesFunc != nil {
		return m.GetGroupRolesFunc(ctx, publicID)
	}
	return nil, nil
}

func (m *MockService) AssignRolesToGroup(ctx context.Context, publicID string, req *AssignRolesRequest) error {
	if m.AssignRolesToGroupFunc != nil {
		return m.AssignRolesToGroupFunc(ctx, publicID, req)
	}
	return nil
}

func (m *MockService) RemoveRoleFromGroup(ctx context.Context, publicID string, roleID int) error {
	if m.RemoveRoleFromGroupFunc != nil {
		return m.RemoveRoleFromGroupFunc(ctx, publicID, roleID)
	}
	return nil
}

func TestHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockReturn     *GroupListResponseWrapper
		mockError      error
		expectedStatus int
	}{
		{
			name:        "successful list with default pagination",
			queryParams: "",
			mockReturn: &GroupListResponseWrapper{
				Data: []GroupResponse{
					{
						ID:          1,
						PublicID:    "grp-abc123",
						Name:        json.RawMessage(`{"en-US": "Admins"}`),
						Description: json.RawMessage(`{}`),
					},
				},
				Page:       1,
				Limit:      10,
				TotalCount: 1,
				TotalPages: 1,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				ListFunc: func(ctx context.Context, page, limit int) (*GroupListResponseWrapper, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			req := httptest.NewRequest(http.MethodGet, "/groups"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			handler.List(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockReturn     *GroupResponse
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful create",
			requestBody: CreateGroupRequest{
				Name:        json.RawMessage(`{"en-US": "Admins"}`),
				Description: nil,
			},
			mockReturn: &GroupResponse{
				ID:          1,
				PublicID:    "grp-abc123",
				Name:        json.RawMessage(`{"en-US": "Admins"}`),
				Description: json.RawMessage(`{}`),
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid name",
			requestBody: CreateGroupRequest{
				Name: json.RawMessage(`{}`),
			},
			mockReturn:     nil,
			mockError:      ErrInvalidName,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				CreateFunc: func(ctx context.Context, req *CreateGroupRequest) (*GroupResponse, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/groups", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.Create(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_GetByPublicID(t *testing.T) {
	tests := []struct {
		name           string
		publicID       string
		mockReturn     *GroupResponse
		mockError      error
		expectedStatus int
	}{
		{
			name:     "successful get",
			publicID: "grp-abc123",
			mockReturn: &GroupResponse{
				ID:          1,
				PublicID:    "grp-abc123",
				Name:        json.RawMessage(`{"en-US": "Admins"}`),
				Description: json.RawMessage(`{}`),
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found",
			publicID:       "grp-notfound",
			mockReturn:     nil,
			mockError:      ErrGroupNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				GetByPublicIDFunc: func(ctx context.Context, publicID string) (*GroupResponse, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			req := httptest.NewRequest(http.MethodGet, "/groups/"+tt.publicID, nil)
			w := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("publicId", tt.publicID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.GetByPublicID(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_Delete(t *testing.T) {
	tests := []struct {
		name           string
		publicID       string
		mockError      error
		expectedStatus int
	}{
		{
			name:           "successful delete",
			publicID:       "grp-abc123",
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found",
			publicID:       "grp-notfound",
			mockError:      ErrGroupNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				DeleteFunc: func(ctx context.Context, publicID string) error {
					return tt.mockError
				},
			}

			handler := NewHandler(mockService)
			req := httptest.NewRequest(http.MethodDelete, "/groups/"+tt.publicID, nil)
			w := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("publicId", tt.publicID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.Delete(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_GetGroupUsers(t *testing.T) {
	tests := []struct {
		name           string
		publicID       string
		mockReturn     []int
		mockError      error
		expectedStatus int
	}{
		{
			name:           "successful get group users",
			publicID:       "grp-abc123",
			mockReturn:     []int{1, 2, 3},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "group not found",
			publicID:       "grp-notfound",
			mockReturn:     nil,
			mockError:      ErrGroupNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				GetGroupUsersFunc: func(ctx context.Context, publicID string) ([]int, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			req := httptest.NewRequest(http.MethodGet, "/groups/"+tt.publicID+"/users", nil)
			w := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("publicId", tt.publicID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.GetGroupUsers(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_GetGroupRoles(t *testing.T) {
	tests := []struct {
		name           string
		publicID       string
		mockReturn     []GroupRoleResponse
		mockError      error
		expectedStatus int
	}{
		{
			name:     "successful get group roles",
			publicID: "grp-abc123",
			mockReturn: []GroupRoleResponse{
				{
					GroupPublicID: "grp-abc123",
					RoleID:        1,
					RoleName:      "admin",
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "group not found",
			publicID:       "grp-notfound",
			mockReturn:     nil,
			mockError:      ErrGroupNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				GetGroupRolesFunc: func(ctx context.Context, publicID string) ([]GroupRoleResponse, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			req := httptest.NewRequest(http.MethodGet, "/groups/"+tt.publicID+"/roles", nil)
			w := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("publicId", tt.publicID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.GetGroupRoles(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
