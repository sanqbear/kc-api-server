package commoncodes

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
	CreateFunc         func(ctx context.Context, req *CreateCommonCodeRequest) (*CommonCodeResponse, error)
	GetByIDFunc        func(ctx context.Context, id int) (*CommonCodeResponse, error)
	ListFunc           func(ctx context.Context, page, limit int) (*CommonCodeListResponseWrapper, error)
	UpdateFunc         func(ctx context.Context, id int, req *UpdateCommonCodeRequest) (*CommonCodeResponse, error)
	DeleteFunc         func(ctx context.Context, id int) error
	SearchFunc         func(ctx context.Context, criteria *SearchCommonCodeRequest, page, limit int) (*CommonCodeListResponseWrapper, error)
	BatchCreateFunc    func(ctx context.Context, req *BatchCreateRequest) (*BatchOperationResponse, error)
	BatchUpdateFunc    func(ctx context.Context, req *BatchUpdateRequest) (*BatchOperationResponse, error)
	BatchDeleteFunc    func(ctx context.Context, req *BatchDeleteRequest) (*BatchOperationResponse, error)
	ListCategoriesFunc func(ctx context.Context) ([]CategoryResponse, error)
	GetByCategoryFunc  func(ctx context.Context, category string) ([]CommonCodeResponse, error)
	ReorderFunc        func(ctx context.Context, category string, req *ReorderRequest) error
}

func (m *MockService) Create(ctx context.Context, req *CreateCommonCodeRequest) (*CommonCodeResponse, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) GetByID(ctx context.Context, id int) (*CommonCodeResponse, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockService) List(ctx context.Context, page, limit int) (*CommonCodeListResponseWrapper, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, page, limit)
	}
	return nil, nil
}

func (m *MockService) Update(ctx context.Context, id int, req *UpdateCommonCodeRequest) (*CommonCodeResponse, error) {
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

func (m *MockService) Search(ctx context.Context, criteria *SearchCommonCodeRequest, page, limit int) (*CommonCodeListResponseWrapper, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, criteria, page, limit)
	}
	return nil, nil
}

func (m *MockService) BatchCreate(ctx context.Context, req *BatchCreateRequest) (*BatchOperationResponse, error) {
	if m.BatchCreateFunc != nil {
		return m.BatchCreateFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) BatchUpdate(ctx context.Context, req *BatchUpdateRequest) (*BatchOperationResponse, error) {
	if m.BatchUpdateFunc != nil {
		return m.BatchUpdateFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) BatchDelete(ctx context.Context, req *BatchDeleteRequest) (*BatchOperationResponse, error) {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) ListCategories(ctx context.Context) ([]CategoryResponse, error) {
	if m.ListCategoriesFunc != nil {
		return m.ListCategoriesFunc(ctx)
	}
	return nil, nil
}

func (m *MockService) GetByCategory(ctx context.Context, category string) ([]CommonCodeResponse, error) {
	if m.GetByCategoryFunc != nil {
		return m.GetByCategoryFunc(ctx, category)
	}
	return nil, nil
}

func (m *MockService) Reorder(ctx context.Context, category string, req *ReorderRequest) error {
	if m.ReorderFunc != nil {
		return m.ReorderFunc(ctx, category, req)
	}
	return nil
}

func TestHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockReturn     *CommonCodeListResponseWrapper
		mockError      error
		expectedStatus int
	}{
		{
			name:        "successful list with default pagination",
			queryParams: "",
			mockReturn: &CommonCodeListResponseWrapper{
				Data: []CommonCodeResponse{
					{
						ID:           1,
						Category:     "rank",
						Code:         "MANAGER",
						Name:         json.RawMessage(`{"en-US": "Manager", "ko-KR": "매니저"}`),
						Description:  json.RawMessage(`{}`),
						ExtraPayload: json.RawMessage(`{}`),
						SortOrder:    10,
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
				ListFunc: func(ctx context.Context, page, limit int) (*CommonCodeListResponseWrapper, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			req := httptest.NewRequest(http.MethodGet, "/common-codes"+tt.queryParams, nil)
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
		mockReturn     *CommonCodeResponse
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful create",
			requestBody: CreateCommonCodeRequest{
				Category:     "rank",
				Code:         "MANAGER",
				Name:         json.RawMessage(`{"en-US": "Manager"}`),
				Description:  nil,
				ExtraPayload: nil,
				SortOrder:    nil,
			},
			mockReturn: &CommonCodeResponse{
				ID:           1,
				Category:     "rank",
				Code:         "MANAGER",
				Name:         json.RawMessage(`{"en-US": "Manager"}`),
				Description:  json.RawMessage(`{}`),
				ExtraPayload: json.RawMessage(`{}`),
				SortOrder:    0,
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "category code exists",
			requestBody: CreateCommonCodeRequest{
				Category: "rank",
				Code:     "MANAGER",
				Name:     json.RawMessage(`{"en-US": "Manager"}`),
			},
			mockReturn:     nil,
			mockError:      ErrCategoryCodeExists,
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				CreateFunc: func(ctx context.Context, req *CreateCommonCodeRequest) (*CommonCodeResponse, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/common-codes", bytes.NewReader(body))
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
		mockReturn     *CommonCodeResponse
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful get",
			id:   "1",
			mockReturn: &CommonCodeResponse{
				ID:           1,
				Category:     "rank",
				Code:         "MANAGER",
				Name:         json.RawMessage(`{"en-US": "Manager"}`),
				Description:  json.RawMessage(`{}`),
				ExtraPayload: json.RawMessage(`{}`),
				SortOrder:    0,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "not found",
			id:             "999",
			mockReturn:     nil,
			mockError:      ErrCommonCodeNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				GetByIDFunc: func(ctx context.Context, id int) (*CommonCodeResponse, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			req := httptest.NewRequest(http.MethodGet, "/common-codes/"+tt.id, nil)
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
			mockError:      ErrCommonCodeNotFound,
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
			req := httptest.NewRequest(http.MethodDelete, "/common-codes/"+tt.id, nil)
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
