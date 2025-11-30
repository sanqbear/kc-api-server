package tickets

import (
	"database/sql"
	"encoding/json"
	"time"
)

// TicketStatus represents the status of a ticket
type TicketStatus string

const (
	TicketStatusOpen           TicketStatus = "OPEN"
	TicketStatusWaitingForInfo TicketStatus = "WAITING_FOR_INFO"
	TicketStatusInProgress     TicketStatus = "IN_PROGRESS"
	TicketStatusResolved       TicketStatus = "RESOLVED"
	TicketStatusClosed         TicketStatus = "CLOSED"
	TicketStatusReopened       TicketStatus = "REOPENED"
)

// TicketPriority represents the priority level of a ticket
type TicketPriority string

const (
	TicketPriorityLow      TicketPriority = "LOW"
	TicketPriorityMedium   TicketPriority = "MEDIUM"
	TicketPriorityHigh     TicketPriority = "HIGH"
	TicketPriorityCritical TicketPriority = "CRITICAL"
)

// TicketRequestType represents the type of request
type TicketRequestType string

const (
	TicketRequestTypeBug            TicketRequestType = "BUG"
	TicketRequestTypeMaintenance    TicketRequestType = "MAINTENANCE"
	TicketRequestTypeFeatureRequest TicketRequestType = "FEATURE_REQUEST"
	TicketRequestTypeGeneralInquiry TicketRequestType = "GENERAL_INQUIRY"
)

// EntryType represents the type of a ticket entry
type EntryType string

const (
	EntryTypeComment  EntryType = "COMMENT"
	EntryTypeFile     EntryType = "FILE"
	EntryTypeSchedule EntryType = "SCHEDULE"
	EntryTypeEvent    EntryType = "EVENT"
)

// ContentFormat represents the format of entry content
type ContentFormat string

const (
	ContentFormatPlainText ContentFormat = "PLAIN_TEXT"
	ContentFormatMarkdown  ContentFormat = "MARKDOWN"
	ContentFormatHTML      ContentFormat = "HTML"
	ContentFormatNone      ContentFormat = "NONE"
)

// Ticket represents the internal ticket entity in the database
type Ticket struct {
	ID             int64          `json:"-"`
	PublicID       string         `json:"id"`
	Title          string         `json:"title"`
	AssignedUserID sql.NullInt64  `json:"-"`
	Status         TicketStatus   `json:"status"`
	Priority       TicketPriority `json:"priority"`
	RequestType    TicketRequestType `json:"request_type"`
	DueDate        sql.NullTime   `json:"-"`
	CreatedAt      time.Time      `json:"-"`
	UpdatedAt      time.Time      `json:"-"`
}

// TicketEntry represents the internal ticket entry entity in the database
type TicketEntry struct {
	ID            int64          `json:"-"`
	TicketID      int64          `json:"-"`
	AuthorUserID  sql.NullInt64  `json:"-"`
	ParentEntryID sql.NullInt64  `json:"-"`
	EntryType     EntryType      `json:"entry_type"`
	Format        ContentFormat  `json:"format"`
	Body          sql.NullString `json:"body"`
	Payload       json.RawMessage `json:"payload"`
	IsDeleted     bool           `json:"-"`
	CreatedAt     time.Time      `json:"-"`
	UpdatedAt     time.Time      `json:"-"`
}

// Tag represents the internal tag entity in the database
type Tag struct {
	ID        int64          `json:"-"`
	Name      string         `json:"name"`
	ColorCode sql.NullString `json:"-"`
	IsDeleted bool           `json:"-"`
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
}

// EntryReference represents a reference from an entry to other entities
type EntryReference struct {
	SourceEntryID int64         `json:"-"`
	TargetEntryID sql.NullInt64 `json:"-"`
	TargetTicketID sql.NullInt64 `json:"-"`
	TargetUserID  sql.NullInt64 `json:"-"`
	CreatedAt     time.Time     `json:"-"`
}

// TicketTag represents the association between a ticket and a tag
type TicketTag struct {
	TicketID  int64          `json:"-"`
	TagID     int64          `json:"-"`
	Category  sql.NullString `json:"-"`
	CreatedAt time.Time      `json:"-"`
}

// EntryTag represents the association between an entry and a tag
type EntryTag struct {
	EntryID   int64          `json:"-"`
	TagID     int64          `json:"-"`
	Category  sql.NullString `json:"-"`
	CreatedAt time.Time      `json:"-"`
}

// -------------------- Response DTOs --------------------

// TicketListResponse represents a simplified ticket response for list queries
type TicketListResponse struct {
	ID          string            `json:"id" example:"01912345-6789-7abc-def0-123456789abc"`
	Title       string            `json:"title" example:"Bug in login page"`
	Status      TicketStatus      `json:"status" example:"OPEN"`
	Priority    TicketPriority    `json:"priority" example:"HIGH"`
	RequestType TicketRequestType `json:"request_type" example:"BUG"`
	DueDate     *time.Time        `json:"due_date,omitempty" example:"2024-12-31T23:59:59Z"`
	CreatedAt   time.Time         `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt   time.Time         `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// TicketDetailResponse represents a detailed ticket response
type TicketDetailResponse struct {
	ID               string            `json:"id" example:"01912345-6789-7abc-def0-123456789abc"`
	Title            string            `json:"title" example:"Bug in login page"`
	Status           TicketStatus      `json:"status" example:"OPEN"`
	Priority         TicketPriority    `json:"priority" example:"HIGH"`
	RequestType      TicketRequestType `json:"request_type" example:"BUG"`
	AssignedUserID   *string           `json:"assigned_user_id,omitempty" example:"01912345-6789-7abc-def0-123456789abc"`
	AssignedUserName json.RawMessage   `json:"assigned_user_name,omitempty" swaggertype:"object"`
	DueDate          *time.Time        `json:"due_date,omitempty" example:"2024-12-31T23:59:59Z"`
	Tags             []TagResponse     `json:"tags"`
	Entries          []EntryListResponse `json:"entries"`
	CreatedAt        time.Time         `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt        time.Time         `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// EntryListResponse represents a simplified entry response for list queries
type EntryListResponse struct {
	ID             int64           `json:"id" example:"1"`
	EntryType      EntryType       `json:"entry_type" example:"COMMENT"`
	Format         ContentFormat   `json:"format" example:"MARKDOWN"`
	Body           *string         `json:"body,omitempty" example:"This is the entry content"`
	AuthorUserID   *string         `json:"author_user_id,omitempty" example:"01912345-6789-7abc-def0-123456789abc"`
	AuthorUserName json.RawMessage `json:"author_user_name,omitempty" swaggertype:"object"`
	ParentEntryID  *int64          `json:"parent_entry_id,omitempty" example:"0"`
	CreatedAt      time.Time       `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt      time.Time       `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// EntryDetailResponse represents a detailed entry response
type EntryDetailResponse struct {
	ID             int64               `json:"id" example:"1"`
	TicketID       string              `json:"ticket_id" example:"01912345-6789-7abc-def0-123456789abc"`
	EntryType      EntryType           `json:"entry_type" example:"COMMENT"`
	Format         ContentFormat       `json:"format" example:"MARKDOWN"`
	Body           *string             `json:"body,omitempty" example:"This is the entry content"`
	Payload        json.RawMessage     `json:"payload" swaggertype:"object"`
	AuthorUserID   *string             `json:"author_user_id,omitempty" example:"01912345-6789-7abc-def0-123456789abc"`
	AuthorUserName json.RawMessage     `json:"author_user_name,omitempty" swaggertype:"object"`
	ParentEntryID  *int64              `json:"parent_entry_id,omitempty" example:"0"`
	Tags           []TagResponse       `json:"tags"`
	References     []ReferenceResponse `json:"references"`
	CreatedAt      time.Time           `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt      time.Time           `json:"updated_at" example:"2024-01-01T00:00:00Z"`
}

// TagResponse represents a tag response
type TagResponse struct {
	ID        int64   `json:"id" example:"1"`
	Name      string  `json:"name" example:"urgent"`
	ColorCode *string `json:"color_code,omitempty" example:"#FF0000"`
	Category  *string `json:"category,omitempty" example:"priority"`
}

// ReferenceResponse represents a reference response
type ReferenceResponse struct {
	TargetType     string          `json:"target_type" example:"entry"`
	TargetEntryID  *int64          `json:"target_entry_id,omitempty" example:"1"`
	TargetTicketID *string         `json:"target_ticket_id,omitempty" example:"01912345-6789-7abc-def0-123456789abc"`
	TargetUserID   *string         `json:"target_user_id,omitempty" example:"01912345-6789-7abc-def0-123456789abc"`
	TargetUserName json.RawMessage `json:"target_user_name,omitempty" swaggertype:"object"`
	CreatedAt      time.Time       `json:"created_at" example:"2024-01-01T00:00:00Z"`
}

// -------------------- Request DTOs --------------------

// CreateTicketRequest represents the request body for creating a ticket
type CreateTicketRequest struct {
	Title          string            `json:"title" example:"Bug in login page"`
	Status         *TicketStatus     `json:"status,omitempty" example:"OPEN"`
	Priority       *TicketPriority   `json:"priority,omitempty" example:"MEDIUM"`
	RequestType    *TicketRequestType `json:"request_type,omitempty" example:"BUG"`
	AssignedUserID *string           `json:"assigned_user_id,omitempty" example:"01912345-6789-7abc-def0-123456789abc"`
	DueDate        *time.Time        `json:"due_date,omitempty" example:"2024-12-31T23:59:59Z"`
	TagIDs         []int64           `json:"tag_ids,omitempty"`
	// Initial entry (required)
	InitialEntry   CreateEntryRequest `json:"initial_entry"`
}

// UpdateTicketRequest represents the request body for updating a ticket
type UpdateTicketRequest struct {
	Title          *string            `json:"title,omitempty" example:"Updated title"`
	Status         *TicketStatus      `json:"status,omitempty" example:"IN_PROGRESS"`
	Priority       *TicketPriority    `json:"priority,omitempty" example:"HIGH"`
	RequestType    *TicketRequestType `json:"request_type,omitempty" example:"FEATURE_REQUEST"`
	AssignedUserID *string            `json:"assigned_user_id,omitempty" example:"01912345-6789-7abc-def0-123456789abc"`
	DueDate        *time.Time         `json:"due_date,omitempty" example:"2024-12-31T23:59:59Z"`
}

// CreateEntryRequest represents the request body for creating an entry
type CreateEntryRequest struct {
	EntryType     EntryType       `json:"entry_type" example:"COMMENT"`
	Format        *ContentFormat  `json:"format,omitempty" example:"MARKDOWN"`
	Body          *string         `json:"body,omitempty" example:"This is the entry content"`
	Payload       json.RawMessage `json:"payload,omitempty" swaggertype:"object"`
	ParentEntryID *int64          `json:"parent_entry_id,omitempty" example:"0"`
	TagIDs        []int64         `json:"tag_ids,omitempty"`
	References    []CreateReferenceRequest `json:"references,omitempty"`
}

// UpdateEntryRequest represents the request body for updating an entry
type UpdateEntryRequest struct {
	Format  *ContentFormat  `json:"format,omitempty" example:"MARKDOWN"`
	Body    *string         `json:"body,omitempty" example:"Updated content"`
	Payload json.RawMessage `json:"payload,omitempty" swaggertype:"object"`
}

// CreateTagRequest represents the request body for creating a tag
type CreateTagRequest struct {
	Name      string  `json:"name" example:"urgent"`
	ColorCode *string `json:"color_code,omitempty" example:"#FF0000"`
}

// UpdateTagRequest represents the request body for updating a tag
type UpdateTagRequest struct {
	Name      *string `json:"name,omitempty" example:"very-urgent"`
	ColorCode *string `json:"color_code,omitempty" example:"#FF5500"`
}

// CreateReferenceRequest represents a reference to be created
type CreateReferenceRequest struct {
	TargetEntryID  *int64  `json:"target_entry_id,omitempty" example:"1"`
	TargetTicketID *string `json:"target_ticket_id,omitempty" example:"01912345-6789-7abc-def0-123456789abc"`
	TargetUserID   *string `json:"target_user_id,omitempty" example:"01912345-6789-7abc-def0-123456789abc"`
}

// AddTagRequest represents the request to add tags to a ticket or entry
type AddTagRequest struct {
	TagIDs   []int64 `json:"tag_ids"`
	Category *string `json:"category,omitempty" example:"priority"`
}

// SearchTicketRequest represents the search criteria for tickets
type SearchTicketRequest struct {
	Query       *string            `json:"query,omitempty" example:"login bug"`
	Status      []TicketStatus     `json:"status,omitempty"`
	Priority    []TicketPriority   `json:"priority,omitempty"`
	RequestType []TicketRequestType `json:"request_type,omitempty"`
	TagIDs      []int64            `json:"tag_ids,omitempty"`
	AssignedUserID *string         `json:"assigned_user_id,omitempty" example:"01912345-6789-7abc-def0-123456789abc"`
	DueDateFrom *time.Time         `json:"due_date_from,omitempty" example:"2024-01-01T00:00:00Z"`
	DueDateTo   *time.Time         `json:"due_date_to,omitempty" example:"2024-12-31T23:59:59Z"`
}

// -------------------- Wrapper Responses --------------------

// TicketListResponseWrapper wraps the list response with pagination info
type TicketListResponseWrapper struct {
	Data       []TicketListResponse `json:"data"`
	Page       int                  `json:"page" example:"1"`
	Limit      int                  `json:"limit" example:"10"`
	TotalCount int                  `json:"total_count" example:"100"`
	TotalPages int                  `json:"total_pages" example:"10"`
}

// TagListResponseWrapper wraps the tag list response with pagination info
type TagListResponseWrapper struct {
	Data       []TagResponse `json:"data"`
	Page       int           `json:"page" example:"1"`
	Limit      int           `json:"limit" example:"10"`
	TotalCount int           `json:"total_count" example:"100"`
	TotalPages int           `json:"total_pages" example:"10"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error" example:"Bad Request"`
	Message string `json:"message" example:"Invalid request body"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

// -------------------- Conversion Methods --------------------

// ToListResponse converts a Ticket to TicketListResponse
func (t *Ticket) ToListResponse() TicketListResponse {
	resp := TicketListResponse{
		ID:          t.PublicID,
		Title:       t.Title,
		Status:      t.Status,
		Priority:    t.Priority,
		RequestType: t.RequestType,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
	if t.DueDate.Valid {
		resp.DueDate = &t.DueDate.Time
	}
	return resp
}

// ToListResponse converts a TicketEntry to EntryListResponse
func (e *TicketEntry) ToListResponse() EntryListResponse {
	resp := EntryListResponse{
		ID:        e.ID,
		EntryType: e.EntryType,
		Format:    e.Format,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
	if e.Body.Valid {
		resp.Body = &e.Body.String
	}
	if e.ParentEntryID.Valid {
		resp.ParentEntryID = &e.ParentEntryID.Int64
	}
	return resp
}

// ToResponse converts a Tag to TagResponse
func (t *Tag) ToResponse(category *string) TagResponse {
	resp := TagResponse{
		ID:       t.ID,
		Name:     t.Name,
		Category: category,
	}
	if t.ColorCode.Valid {
		resp.ColorCode = &t.ColorCode.String
	}
	return resp
}
