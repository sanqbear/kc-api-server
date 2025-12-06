package aiqueue

import (
	"context"
	"errors"
	"fmt"
)

var (
	// ErrTaskNotFound is returned when a task is not found
	ErrTaskNotFound = errors.New("task not found")
	// ErrInvalidInput is returned when input validation fails
	ErrInvalidInput = errors.New("invalid input")
	// ErrClientNotInitialized is returned when the client is not initialized
	ErrClientNotInitialized = errors.New("AI queue client not initialized")
)

// Service provides business logic for AI task queue operations
type Service interface {
	// Summarize submits a text summarization task
	Summarize(ctx context.Context, text string, maxLength int) (string, error)

	// ExtractKeywords submits a keyword extraction task
	ExtractKeywords(ctx context.Context, text string, maxKeywords int) (string, error)

	// NormalizeRequest submits a JSON normalization task
	NormalizeRequest(ctx context.Context, request string, schema map[string]interface{}) (string, error)

	// GetTaskResult retrieves the result of a task
	GetTaskResult(ctx context.Context, taskID string) (*TaskResult, error)

	// GetTaskStatus retrieves only the status of a task
	GetTaskStatus(ctx context.Context, taskID string) (TaskStatus, error)

	// DeleteTaskResult deletes a task result
	DeleteTaskResult(ctx context.Context, taskID string) error
}

type service struct {
	client *Client
}

// NewService creates a new AI queue service
func NewService(client *Client) Service {
	return &service{
		client: client,
	}
}

// Summarize submits a text summarization task to the AI worker
func (s *service) Summarize(ctx context.Context, text string, maxLength int) (string, error) {
	if s.client == nil {
		return "", ErrClientNotInitialized
	}

	if text == "" {
		return "", fmt.Errorf("%w: text is required", ErrInvalidInput)
	}

	// Prepare task kwargs
	kwargs := map[string]interface{}{
		"text": text,
	}
	if maxLength > 0 {
		kwargs["max_length"] = maxLength
	}

	// Submit task to Celery queue
	taskID, err := s.client.SubmitTask(ctx, "ai_worker.tasks.summarize", kwargs)
	if err != nil {
		return "", fmt.Errorf("failed to submit summarization task: %w", err)
	}

	return taskID, nil
}

// ExtractKeywords submits a keyword extraction task to the AI worker
func (s *service) ExtractKeywords(ctx context.Context, text string, maxKeywords int) (string, error) {
	if s.client == nil {
		return "", ErrClientNotInitialized
	}

	if text == "" {
		return "", fmt.Errorf("%w: text is required", ErrInvalidInput)
	}

	// Prepare task kwargs
	kwargs := map[string]interface{}{
		"text": text,
	}
	if maxKeywords > 0 {
		kwargs["max_keywords"] = maxKeywords
	}

	// Submit task to Celery queue
	taskID, err := s.client.SubmitTask(ctx, "ai_worker.tasks.extract_keywords", kwargs)
	if err != nil {
		return "", fmt.Errorf("failed to submit keyword extraction task: %w", err)
	}

	return taskID, nil
}

// NormalizeRequest submits a JSON normalization task to the AI worker
func (s *service) NormalizeRequest(ctx context.Context, request string, schema map[string]interface{}) (string, error) {
	if s.client == nil {
		return "", ErrClientNotInitialized
	}

	if request == "" {
		return "", fmt.Errorf("%w: request is required", ErrInvalidInput)
	}
	if schema == nil || len(schema) == 0 {
		return "", fmt.Errorf("%w: schema is required", ErrInvalidInput)
	}

	// Prepare task kwargs
	kwargs := map[string]interface{}{
		"request": request,
		"schema":  schema,
	}

	// Submit task to Celery queue
	taskID, err := s.client.SubmitTask(ctx, "ai_worker.tasks.normalize_request", kwargs)
	if err != nil {
		return "", fmt.Errorf("failed to submit normalization task: %w", err)
	}

	return taskID, nil
}

// GetTaskResult retrieves the full result of a task from Redis
func (s *service) GetTaskResult(ctx context.Context, taskID string) (*TaskResult, error) {
	if s.client == nil {
		return nil, ErrClientNotInitialized
	}

	if taskID == "" {
		return nil, fmt.Errorf("%w: task ID is required", ErrInvalidInput)
	}

	result, err := s.client.GetTaskResult(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task result: %w", err)
	}

	return result, nil
}

// GetTaskStatus retrieves only the status of a task (lightweight operation)
func (s *service) GetTaskStatus(ctx context.Context, taskID string) (TaskStatus, error) {
	if s.client == nil {
		return "", ErrClientNotInitialized
	}

	if taskID == "" {
		return "", fmt.Errorf("%w: task ID is required", ErrInvalidInput)
	}

	status, err := s.client.GetTaskStatus(ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("failed to get task status: %w", err)
	}

	return status, nil
}

// DeleteTaskResult deletes a task result from Redis
func (s *service) DeleteTaskResult(ctx context.Context, taskID string) error {
	if s.client == nil {
		return ErrClientNotInitialized
	}

	if taskID == "" {
		return fmt.Errorf("%w: task ID is required", ErrInvalidInput)
	}

	if err := s.client.DeleteTaskResult(ctx, taskID); err != nil {
		return fmt.Errorf("failed to delete task result: %w", err)
	}

	return nil
}
