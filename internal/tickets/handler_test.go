package tickets

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// MockService is a mock implementation of the Service interface for testing
type MockService struct {
	CreateTicketFunc        func(ctx context.Context, req *CreateTicketRequest, authorUserID *int64) (*TicketDetailResponse, error)
	GetTicketByIDFunc       func(ctx context.Context, publicID string) (*TicketDetailResponse, error)
	ListTicketsFunc         func(ctx context.Context, page, limit int) (*TicketListResponseWrapper, error)
	UpdateTicketFunc        func(ctx context.Context, publicID string, req *UpdateTicketRequest) (*TicketListResponse, error)
	DeleteTicketFunc        func(ctx context.Context, publicID string) error
	SearchTicketsFunc       func(ctx context.Context, criteria *SearchTicketRequest, page, limit int) (*TicketListResponseWrapper, error)
	CreateEntryFunc         func(ctx context.Context, ticketPublicID string, req *CreateEntryRequest, authorUserID *int64) (*EntryDetailResponse, error)
	GetEntryByIDFunc        func(ctx context.Context, entryID int64) (*EntryDetailResponse, error)
	UpdateEntryFunc         func(ctx context.Context, entryID int64, req *UpdateEntryRequest) (*EntryListResponse, error)
	DeleteEntryFunc         func(ctx context.Context, entryID int64) error
	CreateTagFunc           func(ctx context.Context, req *CreateTagRequest) (*TagResponse, error)
	GetTagByIDFunc          func(ctx context.Context, tagID int64) (*TagResponse, error)
	ListTagsFunc            func(ctx context.Context, page, limit int) (*TagListResponseWrapper, error)
	UpdateTagFunc           func(ctx context.Context, tagID int64, req *UpdateTagRequest) (*TagResponse, error)
	DeleteTagFunc           func(ctx context.Context, tagID int64) error
	AddTagsToTicketFunc     func(ctx context.Context, ticketPublicID string, req *AddTagRequest) error
	RemoveTagFromTicketFunc func(ctx context.Context, ticketPublicID string, tagID int64) error
	AddTagsToEntryFunc      func(ctx context.Context, entryID int64, req *AddTagRequest) error
	RemoveTagFromEntryFunc  func(ctx context.Context, entryID int64, tagID int64) error
}

func (m *MockService) CreateTicket(ctx context.Context, req *CreateTicketRequest, authorUserID *int64) (*TicketDetailResponse, error) {
	if m.CreateTicketFunc != nil {
		return m.CreateTicketFunc(ctx, req, authorUserID)
	}
	return nil, nil
}

func (m *MockService) GetTicketByID(ctx context.Context, publicID string) (*TicketDetailResponse, error) {
	if m.GetTicketByIDFunc != nil {
		return m.GetTicketByIDFunc(ctx, publicID)
	}
	return nil, nil
}

func (m *MockService) ListTickets(ctx context.Context, page, limit int) (*TicketListResponseWrapper, error) {
	if m.ListTicketsFunc != nil {
		return m.ListTicketsFunc(ctx, page, limit)
	}
	return nil, nil
}

func (m *MockService) UpdateTicket(ctx context.Context, publicID string, req *UpdateTicketRequest) (*TicketListResponse, error) {
	if m.UpdateTicketFunc != nil {
		return m.UpdateTicketFunc(ctx, publicID, req)
	}
	return nil, nil
}

func (m *MockService) DeleteTicket(ctx context.Context, publicID string) error {
	if m.DeleteTicketFunc != nil {
		return m.DeleteTicketFunc(ctx, publicID)
	}
	return nil
}

func (m *MockService) SearchTickets(ctx context.Context, criteria *SearchTicketRequest, page, limit int) (*TicketListResponseWrapper, error) {
	if m.SearchTicketsFunc != nil {
		return m.SearchTicketsFunc(ctx, criteria, page, limit)
	}
	return nil, nil
}

func (m *MockService) CreateEntry(ctx context.Context, ticketPublicID string, req *CreateEntryRequest, authorUserID *int64) (*EntryDetailResponse, error) {
	if m.CreateEntryFunc != nil {
		return m.CreateEntryFunc(ctx, ticketPublicID, req, authorUserID)
	}
	return nil, nil
}

func (m *MockService) GetEntryByID(ctx context.Context, entryID int64) (*EntryDetailResponse, error) {
	if m.GetEntryByIDFunc != nil {
		return m.GetEntryByIDFunc(ctx, entryID)
	}
	return nil, nil
}

func (m *MockService) UpdateEntry(ctx context.Context, entryID int64, req *UpdateEntryRequest) (*EntryListResponse, error) {
	if m.UpdateEntryFunc != nil {
		return m.UpdateEntryFunc(ctx, entryID, req)
	}
	return nil, nil
}

func (m *MockService) DeleteEntry(ctx context.Context, entryID int64) error {
	if m.DeleteEntryFunc != nil {
		return m.DeleteEntryFunc(ctx, entryID)
	}
	return nil
}

func (m *MockService) CreateTag(ctx context.Context, req *CreateTagRequest) (*TagResponse, error) {
	if m.CreateTagFunc != nil {
		return m.CreateTagFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockService) GetTagByID(ctx context.Context, tagID int64) (*TagResponse, error) {
	if m.GetTagByIDFunc != nil {
		return m.GetTagByIDFunc(ctx, tagID)
	}
	return nil, nil
}

func (m *MockService) ListTags(ctx context.Context, page, limit int) (*TagListResponseWrapper, error) {
	if m.ListTagsFunc != nil {
		return m.ListTagsFunc(ctx, page, limit)
	}
	return nil, nil
}

func (m *MockService) UpdateTag(ctx context.Context, tagID int64, req *UpdateTagRequest) (*TagResponse, error) {
	if m.UpdateTagFunc != nil {
		return m.UpdateTagFunc(ctx, tagID, req)
	}
	return nil, nil
}

func (m *MockService) DeleteTag(ctx context.Context, tagID int64) error {
	if m.DeleteTagFunc != nil {
		return m.DeleteTagFunc(ctx, tagID)
	}
	return nil
}

func (m *MockService) AddTagsToTicket(ctx context.Context, ticketPublicID string, req *AddTagRequest) error {
	if m.AddTagsToTicketFunc != nil {
		return m.AddTagsToTicketFunc(ctx, ticketPublicID, req)
	}
	return nil
}

func (m *MockService) RemoveTagFromTicket(ctx context.Context, ticketPublicID string, tagID int64) error {
	if m.RemoveTagFromTicketFunc != nil {
		return m.RemoveTagFromTicketFunc(ctx, ticketPublicID, tagID)
	}
	return nil
}

func (m *MockService) AddTagsToEntry(ctx context.Context, entryID int64, req *AddTagRequest) error {
	if m.AddTagsToEntryFunc != nil {
		return m.AddTagsToEntryFunc(ctx, entryID, req)
	}
	return nil
}

func (m *MockService) RemoveTagFromEntry(ctx context.Context, entryID int64, tagID int64) error {
	if m.RemoveTagFromEntryFunc != nil {
		return m.RemoveTagFromEntryFunc(ctx, entryID, tagID)
	}
	return nil
}

func TestHandler_ListTickets(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name           string
		queryParams    string
		mockReturn     *TicketListResponseWrapper
		mockError      error
		expectedStatus int
	}{
		{
			name:        "successful list with default pagination",
			queryParams: "",
			mockReturn: &TicketListResponseWrapper{
				Data: []TicketListResponse{
					{
						ID:          "01912345-6789-7abc-def0-123456789abc",
						Title:       "Test Ticket",
						Status:      TicketStatusOpen,
						Priority:    TicketPriorityMedium,
						RequestType: TicketRequestTypeBug,
						CreatedAt:   now,
						UpdatedAt:   now,
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
			mockReturn: &TicketListResponseWrapper{
				Data:       []TicketListResponse{},
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
				ListTicketsFunc: func(ctx context.Context, page, limit int) (*TicketListResponseWrapper, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			req := httptest.NewRequest(http.MethodGet, "/tickets"+tt.queryParams, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_CreateTicket(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name           string
		requestBody    interface{}
		mockReturn     *TicketDetailResponse
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful creation",
			requestBody: CreateTicketRequest{
				Title: "Bug in login page",
				InitialEntry: CreateEntryRequest{
					EntryType: EntryTypeComment,
					Body:      ptrString("This is the initial comment"),
				},
			},
			mockReturn: &TicketDetailResponse{
				ID:          "01912345-6789-7abc-def0-123456789abc",
				Title:       "Bug in login page",
				Status:      TicketStatusOpen,
				Priority:    TicketPriorityMedium,
				RequestType: TicketRequestTypeGeneralInquiry,
				Tags:        []TagResponse{},
				Entries:     []EntryListResponse{},
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid title - empty",
			requestBody: CreateTicketRequest{
				Title: "",
				InitialEntry: CreateEntryRequest{
					EntryType: EntryTypeComment,
				},
			},
			mockReturn:     nil,
			mockError:      ErrInvalidTitle,
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
				CreateTicketFunc: func(ctx context.Context, req *CreateTicketRequest, authorUserID *int64) (*TicketDetailResponse, error) {
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

			req := httptest.NewRequest(http.MethodPost, "/tickets", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_GetTicketByID(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name           string
		ticketID       string
		mockReturn     *TicketDetailResponse
		mockError      error
		expectedStatus int
	}{
		{
			name:     "successful get",
			ticketID: "01912345-6789-7abc-def0-123456789abc",
			mockReturn: &TicketDetailResponse{
				ID:          "01912345-6789-7abc-def0-123456789abc",
				Title:       "Test Ticket",
				Status:      TicketStatusOpen,
				Priority:    TicketPriorityMedium,
				RequestType: TicketRequestTypeBug,
				Tags:        []TagResponse{},
				Entries:     []EntryListResponse{},
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "ticket not found",
			ticketID:       "non-existent-id",
			mockReturn:     nil,
			mockError:      ErrTicketNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				GetTicketByIDFunc: func(ctx context.Context, publicID string) (*TicketDetailResponse, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			req := httptest.NewRequest(http.MethodGet, "/tickets/"+tt.ticketID, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_UpdateTicket(t *testing.T) {
	now := time.Now()
	newTitle := "Updated Title"
	tests := []struct {
		name           string
		ticketID       string
		requestBody    interface{}
		mockReturn     *TicketListResponse
		mockError      error
		expectedStatus int
	}{
		{
			name:     "successful update",
			ticketID: "01912345-6789-7abc-def0-123456789abc",
			requestBody: UpdateTicketRequest{
				Title: &newTitle,
			},
			mockReturn: &TicketListResponse{
				ID:          "01912345-6789-7abc-def0-123456789abc",
				Title:       "Updated Title",
				Status:      TicketStatusOpen,
				Priority:    TicketPriorityMedium,
				RequestType: TicketRequestTypeBug,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:     "ticket not found",
			ticketID: "non-existent-id",
			requestBody: UpdateTicketRequest{
				Title: &newTitle,
			},
			mockReturn:     nil,
			mockError:      ErrTicketNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid request body",
			ticketID:       "01912345-6789-7abc-def0-123456789abc",
			requestBody:    "invalid json",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				UpdateTicketFunc: func(ctx context.Context, publicID string, req *UpdateTicketRequest) (*TicketListResponse, error) {
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

			req := httptest.NewRequest(http.MethodPut, "/tickets/"+tt.ticketID, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_DeleteTicket(t *testing.T) {
	tests := []struct {
		name           string
		ticketID       string
		mockError      error
		expectedStatus int
	}{
		{
			name:           "successful delete",
			ticketID:       "01912345-6789-7abc-def0-123456789abc",
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "ticket not found",
			ticketID:       "non-existent-id",
			mockError:      ErrTicketNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				DeleteTicketFunc: func(ctx context.Context, publicID string) error {
					return tt.mockError
				},
			}

			handler := NewHandler(mockService)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			req := httptest.NewRequest(http.MethodDelete, "/tickets/"+tt.ticketID, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_SearchTickets(t *testing.T) {
	now := time.Now()
	query := "bug"
	tests := []struct {
		name           string
		queryParams    string
		requestBody    interface{}
		mockReturn     *TicketListResponseWrapper
		mockError      error
		expectedStatus int
	}{
		{
			name:        "successful search",
			queryParams: "",
			requestBody: SearchTicketRequest{
				Query: &query,
			},
			mockReturn: &TicketListResponseWrapper{
				Data: []TicketListResponse{
					{
						ID:          "01912345-6789-7abc-def0-123456789abc",
						Title:       "Bug in login",
						Status:      TicketStatusOpen,
						Priority:    TicketPriorityHigh,
						RequestType: TicketRequestTypeBug,
						CreatedAt:   now,
						UpdatedAt:   now,
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
				SearchTicketsFunc: func(ctx context.Context, criteria *SearchTicketRequest, page, limit int) (*TicketListResponseWrapper, error) {
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

			req := httptest.NewRequest(http.MethodPost, "/tickets/search"+tt.queryParams, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_CreateEntry(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name           string
		ticketID       string
		requestBody    interface{}
		mockReturn     *EntryDetailResponse
		mockError      error
		expectedStatus int
	}{
		{
			name:     "successful creation",
			ticketID: "01912345-6789-7abc-def0-123456789abc",
			requestBody: CreateEntryRequest{
				EntryType: EntryTypeComment,
				Body:      ptrString("New comment"),
			},
			mockReturn: &EntryDetailResponse{
				ID:        1,
				TicketID:  "01912345-6789-7abc-def0-123456789abc",
				EntryType: EntryTypeComment,
				Format:    ContentFormatNone,
				Body:      ptrString("New comment"),
				Payload:   json.RawMessage("{}"),
				Tags:      []TagResponse{},
				References: []ReferenceResponse{},
				CreatedAt: now,
				UpdatedAt: now,
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:     "ticket not found",
			ticketID: "non-existent-id",
			requestBody: CreateEntryRequest{
				EntryType: EntryTypeComment,
			},
			mockReturn:     nil,
			mockError:      ErrTicketNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				CreateEntryFunc: func(ctx context.Context, ticketPublicID string, req *CreateEntryRequest, authorUserID *int64) (*EntryDetailResponse, error) {
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

			req := httptest.NewRequest(http.MethodPost, "/tickets/"+tt.ticketID+"/entries", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_GetEntryByID(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name           string
		entryID        string
		mockReturn     *EntryDetailResponse
		mockError      error
		expectedStatus int
	}{
		{
			name:    "successful get",
			entryID: "1",
			mockReturn: &EntryDetailResponse{
				ID:        1,
				TicketID:  "01912345-6789-7abc-def0-123456789abc",
				EntryType: EntryTypeComment,
				Format:    ContentFormatMarkdown,
				Body:      ptrString("Test entry"),
				Payload:   json.RawMessage("{}"),
				Tags:      []TagResponse{},
				References: []ReferenceResponse{},
				CreatedAt: now,
				UpdatedAt: now,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "entry not found",
			entryID:        "999",
			mockReturn:     nil,
			mockError:      ErrEntryNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid entry ID",
			entryID:        "invalid",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				GetEntryByIDFunc: func(ctx context.Context, entryID int64) (*EntryDetailResponse, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			req := httptest.NewRequest(http.MethodGet, "/entries/"+tt.entryID, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_ListTags(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockReturn     *TagListResponseWrapper
		mockError      error
		expectedStatus int
	}{
		{
			name:        "successful list",
			queryParams: "",
			mockReturn: &TagListResponseWrapper{
				Data: []TagResponse{
					{
						ID:        1,
						Name:      "urgent",
						ColorCode: ptrString("#FF0000"),
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
				ListTagsFunc: func(ctx context.Context, page, limit int) (*TagListResponseWrapper, error) {
					return tt.mockReturn, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			req := httptest.NewRequest(http.MethodGet, "/tags"+tt.queryParams, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_CreateTag(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockReturn     *TagResponse
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful creation",
			requestBody: CreateTagRequest{
				Name:      "urgent",
				ColorCode: ptrString("#FF0000"),
			},
			mockReturn: &TagResponse{
				ID:        1,
				Name:      "urgent",
				ColorCode: ptrString("#FF0000"),
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid name - empty",
			requestBody: CreateTagRequest{
				Name: "",
			},
			mockReturn:     nil,
			mockError:      ErrInvalidTagName,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				CreateTagFunc: func(ctx context.Context, req *CreateTagRequest) (*TagResponse, error) {
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

			req := httptest.NewRequest(http.MethodPost, "/tags", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_AddTagsToTicket(t *testing.T) {
	tests := []struct {
		name           string
		ticketID       string
		requestBody    interface{}
		mockError      error
		expectedStatus int
	}{
		{
			name:     "successful add",
			ticketID: "01912345-6789-7abc-def0-123456789abc",
			requestBody: AddTagRequest{
				TagIDs: []int64{1, 2},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:     "ticket not found",
			ticketID: "non-existent-id",
			requestBody: AddTagRequest{
				TagIDs: []int64{1},
			},
			mockError:      ErrTicketNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				AddTagsToTicketFunc: func(ctx context.Context, ticketPublicID string, req *AddTagRequest) error {
					return tt.mockError
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

			req := httptest.NewRequest(http.MethodPost, "/tickets/"+tt.ticketID+"/tags", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_RemoveTagFromTicket(t *testing.T) {
	tests := []struct {
		name           string
		ticketID       string
		tagID          string
		mockError      error
		expectedStatus int
	}{
		{
			name:           "successful remove",
			ticketID:       "01912345-6789-7abc-def0-123456789abc",
			tagID:          "1",
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "ticket not found",
			ticketID:       "non-existent-id",
			tagID:          "1",
			mockError:      ErrTicketNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "tag not found",
			ticketID:       "01912345-6789-7abc-def0-123456789abc",
			tagID:          "999",
			mockError:      ErrTagNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				RemoveTagFromTicketFunc: func(ctx context.Context, ticketPublicID string, tagID int64) error {
					return tt.mockError
				},
			}

			handler := NewHandler(mockService)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			req := httptest.NewRequest(http.MethodDelete, "/tickets/"+tt.ticketID+"/tags/"+tt.tagID, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

// Helper function to create string pointers
func ptrString(s string) *string {
	return &s
}
