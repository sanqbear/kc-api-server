package aiqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Client handles communication with Redis for Celery task queue
type Client struct {
	redisClient *redis.Client
	queueName   string
	resultTTL   time.Duration
}

// CeleryTask represents a Celery task in the format expected by the worker
type CeleryTask struct {
	ID      string                   `json:"id"`
	Task    string                   `json:"task"`
	Args    []interface{}            `json:"args"`
	Kwargs  map[string]interface{}   `json:"kwargs"`
	Retries int                      `json:"retries,omitempty"`
	ETA     *time.Time               `json:"eta,omitempty"`
	Expires *time.Time               `json:"expires,omitempty"`
	Headers map[string]interface{}   `json:"headers,omitempty"`
	UTCTime *time.Time               `json:"utctime,omitempty"`
}

// CeleryTaskMessage is the wrapper format for Celery tasks in Redis
type CeleryTaskMessage struct {
	Body            string                 `json:"body"`
	ContentEncoding string                 `json:"content-encoding"`
	ContentType     string                 `json:"content-type"`
	Headers         map[string]interface{} `json:"headers"`
	Properties      map[string]interface{} `json:"properties"`
}

// CeleryResult represents the result stored by Celery in Redis
type CeleryResult struct {
	Status    string                 `json:"status"`
	Result    map[string]interface{} `json:"result,omitempty"`
	Traceback string                 `json:"traceback,omitempty"`
	Children  []interface{}          `json:"children,omitempty"`
	DateDone  *time.Time             `json:"date_done,omitempty"`
}

// NewClient creates a new AI queue client connected to Redis
func NewClient(redisAddr, redisPassword string, redisDB int, queueName string) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Default result TTL to 1 hour
	resultTTL := 1 * time.Hour

	return &Client{
		redisClient: rdb,
		queueName:   queueName,
		resultTTL:   resultTTL,
	}, nil
}

// Close closes the Redis client connection
func (c *Client) Close() error {
	return c.redisClient.Close()
}

// SubmitTask submits a task to the Celery queue via Redis
func (c *Client) SubmitTask(ctx context.Context, taskName string, kwargs map[string]interface{}) (string, error) {
	// Generate unique task ID
	taskID := uuid.New().String()

	// Create Celery task structure
	now := time.Now().UTC()
	celeryTask := CeleryTask{
		ID:      taskID,
		Task:    taskName,
		Args:    []interface{}{},
		Kwargs:  kwargs,
		Retries: 0,
		UTCTime: &now,
		Headers: map[string]interface{}{
			"lang":      "go",
			"task":      taskName,
			"id":        taskID,
			"root_id":   taskID,
			"parent_id": nil,
			"group":     nil,
		},
	}

	// Serialize task to JSON
	taskBody, err := json.Marshal(celeryTask)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Celery task: %w", err)
	}

	// Create Celery message wrapper
	message := CeleryTaskMessage{
		Body:            string(taskBody),
		ContentEncoding: "utf-8",
		ContentType:     "application/json",
		Headers: map[string]interface{}{
			"lang":      "go",
			"task":      taskName,
			"id":        taskID,
			"root_id":   taskID,
			"parent_id": nil,
			"group":     nil,
		},
		Properties: map[string]interface{}{
			"correlation_id":  taskID,
			"reply_to":        uuid.New().String(),
			"delivery_mode":   2,
			"delivery_info":   map[string]interface{}{"exchange": "", "routing_key": c.queueName},
			"priority":        0,
			"body_encoding":   "base64",
			"delivery_tag":    uuid.New().String(),
		},
	}

	// Serialize message wrapper to JSON
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Celery message: %w", err)
	}

	// Push task to Redis list (Celery queue)
	if err := c.redisClient.LPush(ctx, c.queueName, messageJSON).Err(); err != nil {
		return "", fmt.Errorf("failed to push task to Redis queue: %w", err)
	}

	return taskID, nil
}

// GetTaskResult retrieves the result of a task from Redis
func (c *Client) GetTaskResult(ctx context.Context, taskID string) (*TaskResult, error) {
	// Celery stores results with key pattern: celery-task-meta-{task_id}
	resultKey := fmt.Sprintf("celery-task-meta-%s", taskID)

	// Get result from Redis
	resultJSON, err := c.redisClient.Get(ctx, resultKey).Result()
	if err != nil {
		if err == redis.Nil {
			// Result not found - task is still pending or doesn't exist
			return &TaskResult{
				ID:     taskID,
				Status: TaskStatusPending,
			}, nil
		}
		return nil, fmt.Errorf("failed to get task result from Redis: %w", err)
	}

	// Parse Celery result
	var celeryResult CeleryResult
	if err := json.Unmarshal([]byte(resultJSON), &celeryResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Celery result: %w", err)
	}

	// Convert Celery result to TaskResult
	taskResult := &TaskResult{
		ID:     taskID,
		Status: TaskStatus(celeryResult.Status),
		Result: celeryResult.Result,
	}

	// Handle error status
	if celeryResult.Status == "FAILURE" {
		taskResult.Error = celeryResult.Traceback
	}

	// Set completion time
	if celeryResult.DateDone != nil {
		taskResult.CompletedAt = celeryResult.DateDone
	}

	return taskResult, nil
}

// GetTaskStatus retrieves only the status of a task (lightweight version of GetTaskResult)
func (c *Client) GetTaskStatus(ctx context.Context, taskID string) (TaskStatus, error) {
	result, err := c.GetTaskResult(ctx, taskID)
	if err != nil {
		return "", err
	}
	return result.Status, nil
}

// DeleteTaskResult removes a task result from Redis
func (c *Client) DeleteTaskResult(ctx context.Context, taskID string) error {
	resultKey := fmt.Sprintf("celery-task-meta-%s", taskID)
	return c.redisClient.Del(ctx, resultKey).Err()
}

// SetResultTTL sets the time-to-live for task results
func (c *Client) SetResultTTL(ttl time.Duration) {
	c.resultTTL = ttl
}
