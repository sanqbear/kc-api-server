package users

import (
	"database/sql"
	"encoding/json"
	"time"
)

// User represents the internal user entity in the database
type User struct {
	ID        int            `json:"-"`
	PublicID  string         `json:"id"`
	LoginID   string         `json:"login_id"`
	Name      json.RawMessage `json:"name"`
	Email     string         `json:"email"`

	DeptID     sql.NullInt64 `json:"-"`
	RankID     sql.NullInt64 `json:"-"`
	DutyID     sql.NullInt64 `json:"-"`
	TitleID    sql.NullInt64 `json:"-"`
	PositionID sql.NullInt64 `json:"-"`
	LocationID sql.NullInt64 `json:"-"`

	ContactMobile     sql.NullString `json:"-"`
	ContactMobileHash sql.NullString `json:"-"`
	ContactMobileID   sql.NullString `json:"-"`

	ContactOffice     sql.NullString `json:"-"`
	ContactOfficeHash sql.NullString `json:"-"`
	ContactOfficeID   sql.NullString `json:"-"`

	CreatedAt    time.Time      `json:"-"`
	UpdatedAt    time.Time      `json:"-"`
	PasswordHash sql.NullString `json:"-"`
	IsVisible    bool           `json:"-"`
	IsDeleted    bool           `json:"-"`
}

// UserListResponse represents a simplified user response for list queries
type UserListResponse struct {
	ID      string          `json:"id" example:"01912345-6789-7abc-def0-123456789abc"`
	LoginID string          `json:"login_id" example:"john.doe"`
	Name    json.RawMessage `json:"name" swaggertype:"object"`
	Email   string          `json:"email" example:"john.doe@example.com"`
}

// UserDetailResponse represents a detailed user response
type UserDetailResponse struct {
	ID            string          `json:"id" example:"01912345-6789-7abc-def0-123456789abc"`
	LoginID       string          `json:"login_id" example:"john.doe"`
	Name          json.RawMessage `json:"name" swaggertype:"object"`
	Email         string          `json:"email" example:"john.doe@example.com"`
	DeptName      json.RawMessage `json:"dept_name" swaggertype:"object"`
	RankName      json.RawMessage `json:"rank_name" swaggertype:"object"`
	DutyName      json.RawMessage `json:"duty_name" swaggertype:"object"`
	TitleName     json.RawMessage `json:"title_name" swaggertype:"object"`
	PositionName  json.RawMessage `json:"position_name" swaggertype:"object"`
	LocationName  json.RawMessage `json:"location_name" swaggertype:"object"`
	ContactMobile string          `json:"contact_mobile" example:"***-****-1234"`
	ContactOffice string          `json:"contact_office" example:"***-****-5678"`
}

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	Email         string          `json:"email" example:"john.doe@example.com"`
	Name          json.RawMessage `json:"name" swaggertype:"object"`
	LoginID       *string         `json:"login_id,omitempty" example:"john.doe"`
	Password      *string         `json:"password,omitempty"`
	DeptID        *int64          `json:"dept_id,omitempty"`
	RankID        *int64          `json:"rank_id,omitempty"`
	DutyID        *int64          `json:"duty_id,omitempty"`
	TitleID       *int64          `json:"title_id,omitempty"`
	PositionID    *int64          `json:"position_id,omitempty"`
	LocationID    *int64          `json:"location_id,omitempty"`
	ContactMobile *string         `json:"contact_mobile,omitempty" example:"010-1234-5678"`
	ContactOffice *string         `json:"contact_office,omitempty" example:"02-1234-5678"`
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	LoginID       *string          `json:"login_id,omitempty" example:"john.doe"`
	Name          *json.RawMessage `json:"name,omitempty" swaggertype:"object"`
	Email         *string          `json:"email,omitempty" example:"john.doe@example.com"`
	Password      *string          `json:"password,omitempty"`
	DeptID        *int64           `json:"dept_id,omitempty"`
	RankID        *int64           `json:"rank_id,omitempty"`
	DutyID        *int64           `json:"duty_id,omitempty"`
	TitleID       *int64           `json:"title_id,omitempty"`
	PositionID    *int64           `json:"position_id,omitempty"`
	LocationID    *int64           `json:"location_id,omitempty"`
	ContactMobile *string          `json:"contact_mobile,omitempty" example:"010-1234-5678"`
	ContactOffice *string          `json:"contact_office,omitempty" example:"02-1234-5678"`
	IsVisible     *bool            `json:"is_visible,omitempty"`
}

// SearchUserRequest represents the search criteria for users
type SearchUserRequest struct {
	Name           *string `json:"name,omitempty" example:"John"`
	Email          *string `json:"email,omitempty" example:"john@example.com"`
	MobileFull     *string `json:"mobile_full,omitempty" example:"010-1234-5678"`
	OfficeFull     *string `json:"office_full,omitempty" example:"02-1234-5678"`
	MobileLast4    *string `json:"mobile_last4,omitempty" example:"5678"`
	OfficeLast4    *string `json:"office_last4,omitempty" example:"5678"`
}

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page  int `json:"page" example:"1"`
	Limit int `json:"limit" example:"10"`
}

// UserListResponseWrapper wraps the list response with pagination info
type UserListResponseWrapper struct {
	Data       []UserListResponse `json:"data"`
	Page       int                `json:"page" example:"1"`
	Limit      int                `json:"limit" example:"10"`
	TotalCount int                `json:"total_count" example:"100"`
	TotalPages int                `json:"total_pages" example:"10"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error" example:"Bad Request"`
	Message string `json:"message" example:"Invalid email format"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string `json:"message" example:"User deleted successfully"`
}
