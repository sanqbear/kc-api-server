package users

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
	CreateFunc  func(ctx context.Context, req *CreateUserRequest) (*UserListResponse, error)
	GetByIDFunc func(ctx context.Context, publicID string) (*UserDetailResponse, error)
	ListFunc    func(ctx context.Context, page, limit int) (*UserListResponseWrapper, error)
	UpdateFunc  func(ctx context.Context, publicID string, req *UpdateUserRequest) (*UserListResponse, error)
	DeleteFunc  func(ctx context.Context, publicID string) error
	SearchFunc  func(ctx context.Context, criteria *SearchUserRequest, page, limit int) (*UserListResponseWrapper, error)
}

func (m *MockService) Create(ctx context.Context, req *CreateUserRequest) (*UserListResponse, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) GetByID(ctx context.Context, publicID string) (*UserDetailResponse, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, publicID)
	}
	return nil, nil
}

func (m *MockService) List(ctx context.Context, page, limit int) (*UserListResponseWrapper, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, page, limit)
	}
	return nil, nil
}

func (m *MockService) Update(ctx context.Context, publicID string, req *UpdateUserRequest) (*UserListResponse, error) {
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

func (m *MockService) Search(ctx context.Context, criteria *SearchUserRequest, page, limit int) (*UserListResponseWrapper, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, criteria, page, limit)
	}
	return nil, nil
}

func TestHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockReturn     *UserListResponseWrapper
		mockError      error
		expectedStatus int
	}{
		{
			name:        "successful list with default pagination",
			queryParams: "",
			mockReturn: &UserListResponseWrapper{
				Data: []UserListResponse{
					{
						ID:      "01912345-6789-7abc-def0-123456789abc",
						LoginID: "john.doe",
						Name:    json.RawMessage(`{"en-US": "John Doe"}`),
						Email:   "john.doe@example.com",
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
		{
			name:        "successful list with custom pagination",
			queryParams: "?page=2&limit=5",
			mockReturn: &UserListResponseWrapper{
				Data:       []UserListResponse{},
				Page:       2,
				Limit:      5,
				TotalCount: 5,
				TotalPages: 1,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				ListFunc: func(ctx context.Context, page, limit int) (*UserListResponseWrapper, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			req := httptest.NewRequest(http.MethodGet, "/users"+tt.queryParams, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockReturn     *UserListResponse
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful creation",
			requestBody: CreateUserRequest{
				Email: "john.doe@example.com",
				Name:  json.RawMessage(`{"en-US": "John Doe"}`),
			},
			mockReturn: &UserListResponse{
				ID:      "01912345-6789-7abc-def0-123456789abc",
				LoginID: "john.doe@example.com",
				Name:    json.RawMessage(`{"en-US": "John Doe"}`),
				Email:   "john.doe@example.com",
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid email",
			requestBody: CreateUserRequest{
				Email: "invalid-email",
				Name:  json.RawMessage(`{"en-US": "John Doe"}`),
			},
			mockReturn:     nil,
			mockError:      ErrInvalidEmail,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "email already exists",
			requestBody: CreateUserRequest{
				Email: "existing@example.com",
				Name:  json.RawMessage(`{"en-US": "John Doe"}`),
			},
			mockReturn:     nil,
			mockError:      ErrEmailExists,
			expectedStatus: http.StatusConflict,
		},
		{
			name: "invalid name - empty",
			requestBody: CreateUserRequest{
				Email: "john.doe@example.com",
				Name:  json.RawMessage(`{}`),
			},
			mockReturn:     nil,
			mockError:      ErrInvalidName,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid json",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				CreateFunc: func(ctx context.Context, req *CreateUserRequest) (*UserListResponse, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_GetByID(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		mockReturn     *UserDetailResponse
		mockError      error
		expectedStatus int
	}{
		{
			name:   "successful get",
			userID: "01912345-6789-7abc-def0-123456789abc",
			mockReturn: &UserDetailResponse{
				ID:            "01912345-6789-7abc-def0-123456789abc",
				LoginID:       "john.doe",
				Name:          json.RawMessage(`{"en-US": "John Doe"}`),
				Email:         "john.doe@example.com",
				DeptName:      json.RawMessage(`{"en-US": "Engineering"}`),
				RankName:      json.RawMessage(`{}`),
				DutyName:      json.RawMessage(`{}`),
				TitleName:     json.RawMessage(`{}`),
				PositionName:  json.RawMessage(`{}`),
				LocationName:  json.RawMessage(`{}`),
				ContactMobile: "***-****-5678",
				ContactOffice: "***-****-1234",
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user not found",
			userID:         "non-existent-id",
			mockReturn:     nil,
			mockError:      ErrUserNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				GetByIDFunc: func(ctx context.Context, publicID string) (*UserDetailResponse, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.userID, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_Update(t *testing.T) {
	loginID := "john.doe.updated"
	tests := []struct {
		name           string
		userID         string
		requestBody    interface{}
		mockReturn     *UserListResponse
		mockError      error
		expectedStatus int
	}{
		{
			name:   "successful update",
			userID: "01912345-6789-7abc-def0-123456789abc",
			requestBody: UpdateUserRequest{
				LoginID: &loginID,
			},
			mockReturn: &UserListResponse{
				ID:      "01912345-6789-7abc-def0-123456789abc",
				LoginID: "john.doe.updated",
				Name:    json.RawMessage(`{"en-US": "John Doe"}`),
				Email:   "john.doe@example.com",
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:   "user not found",
			userID: "non-existent-id",
			requestBody: UpdateUserRequest{
				LoginID: &loginID,
			},
			mockReturn:     nil,
			mockError:      ErrUserNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid request body",
			userID:         "01912345-6789-7abc-def0-123456789abc",
			requestBody:    "invalid json",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				UpdateFunc: func(ctx context.Context, publicID string, req *UpdateUserRequest) (*UserListResponse, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPut, "/users/"+tt.userID, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_Delete(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		mockError      error
		expectedStatus int
	}{
		{
			name:           "successful delete",
			userID:         "01912345-6789-7abc-def0-123456789abc",
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user not found",
			userID:         "non-existent-id",
			mockError:      ErrUserNotFound,
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
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			req := httptest.NewRequest(http.MethodDelete, "/users/"+tt.userID, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_Search(t *testing.T) {
	name := "John"
	tests := []struct {
		name           string
		queryParams    string
		requestBody    interface{}
		mockReturn     *UserListResponseWrapper
		mockError      error
		expectedStatus int
	}{
		{
			name:        "successful search by name",
			queryParams: "",
			requestBody: SearchUserRequest{
				Name: &name,
			},
			mockReturn: &UserListResponseWrapper{
				Data: []UserListResponse{
					{
						ID:      "01912345-6789-7abc-def0-123456789abc",
						LoginID: "john.doe",
						Name:    json.RawMessage(`{"en-US": "John Doe"}`),
						Email:   "john.doe@example.com",
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
		{
			name:           "invalid request body",
			queryParams:    "",
			requestBody:    "invalid json",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				SearchFunc: func(ctx context.Context, criteria *SearchUserRequest, page, limit int) (*UserListResponseWrapper, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/users/search"+tt.queryParams, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}
