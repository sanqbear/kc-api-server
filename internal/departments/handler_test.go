package departments

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
	CreateFunc      func(ctx context.Context, req *CreateDepartmentRequest) (*DepartmentResponse, error)
	GetByPublicIDFunc func(ctx context.Context, publicID string) (*DepartmentResponse, error)
	ListFunc        func(ctx context.Context, page, limit int) (*DepartmentListResponseWrapper, error)
	UpdateFunc      func(ctx context.Context, publicID string, req *UpdateDepartmentRequest) (*DepartmentResponse, error)
	DeleteFunc      func(ctx context.Context, publicID string) error
	SearchFunc      func(ctx context.Context, criteria *SearchDepartmentRequest, page, limit int) (*DepartmentListResponseWrapper, error)
	BatchCreateFunc func(ctx context.Context, req *BatchCreateDepartmentRequest) (*BatchOperationResponse, error)
	BatchUpdateFunc func(ctx context.Context, req *BatchUpdateDepartmentRequest) (*BatchOperationResponse, error)
	BatchDeleteFunc func(ctx context.Context, req *BatchDeleteDepartmentRequest) (*BatchOperationResponse, error)
	GetTreeFunc     func(ctx context.Context) ([]DepartmentTreeResponse, error)
	GetSubtreeFunc  func(ctx context.Context, publicID string) (*DepartmentTreeResponse, error)
	MoveFunc        func(ctx context.Context, publicID string, req *MoveDepartmentRequest) error
	GetChildrenFunc func(ctx context.Context, publicID string) ([]DepartmentResponse, error)
}

func (m *MockService) Create(ctx context.Context, req *CreateDepartmentRequest) (*DepartmentResponse, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) GetByPublicID(ctx context.Context, publicID string) (*DepartmentResponse, error) {
	if m.GetByPublicIDFunc != nil {
		return m.GetByPublicIDFunc(ctx, publicID)
	}
	return nil, nil
}

func (m *MockService) List(ctx context.Context, page, limit int) (*DepartmentListResponseWrapper, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, page, limit)
	}
	return nil, nil
}

func (m *MockService) Update(ctx context.Context, publicID string, req *UpdateDepartmentRequest) (*DepartmentResponse, error) {
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

func (m *MockService) Search(ctx context.Context, criteria *SearchDepartmentRequest, page, limit int) (*DepartmentListResponseWrapper, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, criteria, page, limit)
	}
	return nil, nil
}

func (m *MockService) BatchCreate(ctx context.Context, req *BatchCreateDepartmentRequest) (*BatchOperationResponse, error) {
	if m.BatchCreateFunc != nil {
		return m.BatchCreateFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) BatchUpdate(ctx context.Context, req *BatchUpdateDepartmentRequest) (*BatchOperationResponse, error) {
	if m.BatchUpdateFunc != nil {
		return m.BatchUpdateFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) BatchDelete(ctx context.Context, req *BatchDeleteDepartmentRequest) (*BatchOperationResponse, error) {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) GetTree(ctx context.Context) ([]DepartmentTreeResponse, error) {
	if m.GetTreeFunc != nil {
		return m.GetTreeFunc(ctx)
	}
	return nil, nil
}

func (m *MockService) GetSubtree(ctx context.Context, publicID string) (*DepartmentTreeResponse, error) {
	if m.GetSubtreeFunc != nil {
		return m.GetSubtreeFunc(ctx, publicID)
	}
	return nil, nil
}

func (m *MockService) Move(ctx context.Context, publicID string, req *MoveDepartmentRequest) error {
	if m.MoveFunc != nil {
		return m.MoveFunc(ctx, publicID, req)
	}
	return nil
}

func (m *MockService) GetChildren(ctx context.Context, publicID string) ([]DepartmentResponse, error) {
	if m.GetChildrenFunc != nil {
		return m.GetChildrenFunc(ctx, publicID)
	}
	return nil, nil
}

func TestHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockReturn     *DepartmentListResponseWrapper
		mockError      error
		expectedStatus int
	}{
		{
			name:        "successful list with default pagination",
			queryParams: "",
			mockReturn: &DepartmentListResponseWrapper{
				Data: []DepartmentResponse{
					{
						ID:          1,
						PublicID:    "dept-abc123",
						Name:        json.RawMessage(`{"en-US": "Engineering"}`),
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
				ListFunc: func(ctx context.Context, page, limit int) (*DepartmentListResponseWrapper, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			req := httptest.NewRequest(http.MethodGet, "/departments"+tt.queryParams, nil)
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
		mockReturn     *DepartmentResponse
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful create",
			requestBody: CreateDepartmentRequest{
				Name:        json.RawMessage(`{"en-US": "Engineering"}`),
				Description: nil,
			},
			mockReturn: &DepartmentResponse{
				ID:          1,
				PublicID:    "dept-abc123",
				Name:        json.RawMessage(`{"en-US": "Engineering"}`),
				Description: json.RawMessage(`{}`),
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid name",
			requestBody: CreateDepartmentRequest{
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
				CreateFunc: func(ctx context.Context, req *CreateDepartmentRequest) (*DepartmentResponse, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/departments", bytes.NewReader(body))
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
		mockReturn     *DepartmentResponse
		mockError      error
		expectedStatus int
	}{
		{
			name:     "successful get",
			publicID: "dept-abc123",
			mockReturn: &DepartmentResponse{
				ID:          1,
				PublicID:    "dept-abc123",
				Name:        json.RawMessage(`{"en-US": "Engineering"}`),
				Description: json.RawMessage(`{}`),
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found",
			publicID:       "dept-notfound",
			mockReturn:     nil,
			mockError:      ErrDepartmentNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				GetByPublicIDFunc: func(ctx context.Context, publicID string) (*DepartmentResponse, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			req := httptest.NewRequest(http.MethodGet, "/departments/"+tt.publicID, nil)
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
			publicID:       "dept-abc123",
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found",
			publicID:       "dept-notfound",
			mockError:      ErrDepartmentNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "has children",
			publicID:       "dept-parent",
			mockError:      ErrHasChildren,
			expectedStatus: http.StatusBadRequest,
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
			req := httptest.NewRequest(http.MethodDelete, "/departments/"+tt.publicID, nil)
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

func TestHandler_GetTree(t *testing.T) {
	tests := []struct {
		name           string
		mockReturn     []DepartmentTreeResponse
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful get tree",
			mockReturn: []DepartmentTreeResponse{
				{
					ID:          1,
					PublicID:    "dept-root",
					Name:        json.RawMessage(`{"en-US": "Root"}`),
					Description: json.RawMessage(`{}`),
					Children: []DepartmentTreeResponse{
						{
							ID:          2,
							PublicID:    "dept-child",
							Name:        json.RawMessage(`{"en-US": "Child"}`),
							Description: json.RawMessage(`{}`),
						},
					},
				},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				GetTreeFunc: func(ctx context.Context) ([]DepartmentTreeResponse, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			req := httptest.NewRequest(http.MethodGet, "/departments/tree", nil)
			w := httptest.NewRecorder()

			handler.GetTree(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
