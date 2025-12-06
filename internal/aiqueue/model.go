package aiqueue

import "time"

// TaskType represents the type of AI task
type TaskType string

const (
	TaskTypeSummarize TaskType = "summarize"
	TaskTypeKeywords  TaskType = "keywords"
	TaskTypeNormalize TaskType = "normalize"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending TaskStatus = "PENDING"
	TaskStatusStarted TaskStatus = "STARTED"
	TaskStatusSuccess TaskStatus = "SUCCESS"
	TaskStatusFailure TaskStatus = "FAILURE"
	TaskStatusRetry   TaskStatus = "RETRY"
	TaskStatusRevoked TaskStatus = "REVOKED"
)

// TaskRequest represents a request to the AI worker
type TaskRequest struct {
	ID       string                 `json:"id"`
	Type     TaskType               `json:"type"`
	Input    map[string]interface{} `json:"input"`
	Priority int                    `json:"priority,omitempty"`
}

// TaskResult represents the result from AI worker
type TaskResult struct {
	ID          string                 `json:"id"`
	Status      TaskStatus             `json:"status"` // PENDING, STARTED, SUCCESS, FAILURE
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// SummarizeInput for summarization task
type SummarizeInput struct {
	Text      string `json:"text" validate:"required"`
	MaxLength int    `json:"max_length,omitempty"`
}

// SummarizeRequest represents the HTTP request for summarization
type SummarizeRequest struct {
	Text      string `json:"text" validate:"required"`
	MaxLength int    `json:"max_length,omitempty"`
}

// SummarizeResponse represents the HTTP response for task submission
type SummarizeResponse struct {
	TaskID  string `json:"task_id"`
	Message string `json:"message"`
}

// KeywordsInput for keyword extraction task
type KeywordsInput struct {
	Text        string `json:"text" validate:"required"`
	MaxKeywords int    `json:"max_keywords,omitempty"`
}

// KeywordsRequest represents the HTTP request for keyword extraction
type KeywordsRequest struct {
	Text        string `json:"text" validate:"required"`
	MaxKeywords int    `json:"max_keywords,omitempty"`
}

// KeywordsResponse represents the HTTP response for task submission
type KeywordsResponse struct {
	TaskID  string `json:"task_id"`
	Message string `json:"message"`
}

// NormalizeInput for JSON normalization task
type NormalizeInput struct {
	Request string                 `json:"request" validate:"required"`
	Schema  map[string]interface{} `json:"schema" validate:"required"`
}

// NormalizeRequest represents the HTTP request for normalization
type NormalizeRequest struct {
	Request string                 `json:"request" validate:"required"`
	Schema  map[string]interface{} `json:"schema" validate:"required"`
}

// NormalizeResponse represents the HTTP response for task submission
type NormalizeResponse struct {
	TaskID  string `json:"task_id"`
	Message string `json:"message"`
}

// TaskStatusResponse represents the HTTP response for task status
type TaskStatusResponse struct {
	TaskID      string                 `json:"task_id"`
	Status      TaskStatus             `json:"status"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}
