package roles

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
	CreateFunc          func(ctx context.Context, req *CreateRoleRequest) (*RoleResponse, error)
	GetByIDFunc         func(ctx context.Context, id int) (*RoleResponse, error)
	ListFunc            func(ctx context.Context, page, limit int) (*RoleListResponseWrapper, error)
	UpdateFunc          func(ctx context.Context, id int, req *UpdateRoleRequest) (*RoleResponse, error)
	DeleteFunc          func(ctx context.Context, id int) error
	SearchFunc          func(ctx context.Context, criteria *SearchRoleRequest, page, limit int) (*RoleListResponseWrapper, error)
	BatchCreateFunc     func(ctx context.Context, req *BatchCreateRoleRequest) (*BatchOperationResponse, error)
	BatchUpdateFunc     func(ctx context.Context, req *BatchUpdateRoleRequest) (*BatchOperationResponse, error)
	BatchDeleteFunc     func(ctx context.Context, req *BatchDeleteRoleRequest) (*BatchOperationResponse, error)
	GetUserRolesFunc    func(ctx context.Context, userID int) ([]UserRoleResponse, error)
	AssignUserRolesFunc func(ctx context.Context, userID int, req *AssignUserRolesRequest) error
	RemoveUserRoleFunc  func(ctx context.Context, userID int, roleID int) error
	GetUsersWithRoleFunc func(ctx context.Context, roleID int) ([]int, error)
}

func (m *MockService) Create(ctx context.Context, req *CreateRoleRequest) (*RoleResponse, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) GetByID(ctx context.Context, id int) (*RoleResponse, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockService) List(ctx context.Context, page, limit int) (*RoleListResponseWrapper, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, page, limit)
	}
	return nil, nil
}

func (m *MockService) Update(ctx context.Context, id int, req *UpdateRoleRequest) (*RoleResponse, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, req)
	}
	return nil, nil
}

func (m *MockService) Delete(ctx context.Context, id int) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockService) Search(ctx context.Context, criteria *SearchRoleRequest, page, limit int) (*RoleListResponseWrapper, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, criteria, page, limit)
	}
	return nil, nil
}

func (m *MockService) BatchCreate(ctx context.Context, req *BatchCreateRoleRequest) (*BatchOperationResponse, error) {
	if m.BatchCreateFunc != nil {
		return m.BatchCreateFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) BatchUpdate(ctx context.Context, req *BatchUpdateRoleRequest) (*BatchOperationResponse, error) {
	if m.BatchUpdateFunc != nil {
		return m.BatchUpdateFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) BatchDelete(ctx context.Context, req *BatchDeleteRoleRequest) (*BatchOperationResponse, error) {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) GetUserRoles(ctx context.Context, userID int) ([]UserRoleResponse, error) {
	if m.GetUserRolesFunc != nil {
		return m.GetUserRolesFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockService) AssignUserRoles(ctx context.Context, userID int, req *AssignUserRolesRequest) error {
	if m.AssignUserRolesFunc != nil {
		return m.AssignUserRolesFunc(ctx, userID, req)
	}
	return nil
}

func (m *MockService) RemoveUserRole(ctx context.Context, userID int, roleID int) error {
	if m.RemoveUserRoleFunc != nil {
		return m.RemoveUserRoleFunc(ctx, userID, roleID)
	}
	return nil
}

func (m *MockService) GetUsersWithRole(ctx context.Context, roleID int) ([]int, error) {
	if m.GetUsersWithRoleFunc != nil {
		return m.GetUsersWithRoleFunc(ctx, roleID)
	}
	return nil, nil
}

func TestHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockReturn     *RoleListResponseWrapper
		mockError      error
		expectedStatus int
	}{
		{
			name:        "successful list with default pagination",
			queryParams: "",
			mockReturn: &RoleListResponseWrapper{
				Data: []RoleResponse{
					{
						ID:          1,
						Name:        "admin",
						Description: json.RawMessage(`{"en-US": "Administrator"}`),
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
				ListFunc: func(ctx context.Context, page, limit int) (*RoleListResponseWrapper, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			req := httptest.NewRequest(http.MethodGet, "/roles"+tt.queryParams, nil)
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
		mockReturn     *RoleResponse
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful create",
			requestBody: CreateRoleRequest{
				Name:        "admin",
				Description: nil,
			},
			mockReturn: &RoleResponse{
				ID:          1,
				Name:        "admin",
				Description: json.RawMessage(`{}`),
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "role name exists",
			requestBody: CreateRoleRequest{
				Name: "admin",
			},
			mockReturn:     nil,
			mockError:      ErrRoleNameExists,
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				CreateFunc: func(ctx context.Context, req *CreateRoleRequest) (*RoleResponse, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/roles", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.Create(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_GetByID(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		mockReturn     *RoleResponse
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful get",
			id:   "1",
			mockReturn: &RoleResponse{
				ID:          1,
				Name:        "admin",
				Description: json.RawMessage(`{}`),
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found",
			id:             "999",
			mockReturn:     nil,
			mockError:      ErrRoleNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				GetByIDFunc: func(ctx context.Context, id int) (*RoleResponse, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			req := httptest.NewRequest(http.MethodGet, "/roles/"+tt.id, nil)
			w := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.GetByID(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_Delete(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		mockError      error
		expectedStatus int
	}{
		{
			name:           "successful delete",
			id:             "1",
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found",
			id:             "999",
			mockError:      ErrRoleNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				DeleteFunc: func(ctx context.Context, id int) error {
					return tt.mockError
				},
			}

			handler := NewHandler(mockService)
			req := httptest.NewRequest(http.MethodDelete, "/roles/"+tt.id, nil)
			w := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.Delete(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_GetUserRoles(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		mockReturn     []UserRoleResponse
		mockError      error
		expectedStatus int
	}{
		{
			name:   "successful get user roles",
			userID: "1",
			mockReturn: []UserRoleResponse{
				{
					UserID:   1,
					RoleID:   1,
					RoleName: "admin",
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				GetUserRolesFunc: func(ctx context.Context, userID int) ([]UserRoleResponse, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.userID+"/roles", nil)
			w := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("userId", tt.userID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.GetUserRoles(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
