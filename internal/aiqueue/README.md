# AI Queue Module

The AI queue module provides integration with an AI worker queue system using Redis and Celery protocol. It allows the Go API server to submit AI tasks (summarization, keyword extraction, JSON normalization) to a Python-based AI worker and retrieve results asynchronously.

## Architecture

```
Go API Server (aiqueue module)
    ↓ (submit tasks)
Redis (Celery queue)
    ↓ (process tasks)
Python AI Worker (Celery)
    ↓ (store results)
Redis (result backend)
    ↑ (retrieve results)
Go API Server (aiqueue module)
```

## Components

### 1. model.go
Defines data structures for tasks and results:
- `TaskType`: Type of AI task (summarize, keywords, normalize)
- `TaskStatus`: Task execution status (PENDING, STARTED, SUCCESS, FAILURE)
- `TaskRequest`: Request structure for submitting tasks
- `TaskResult`: Result structure from completed tasks
- HTTP request/response DTOs for each task type

### 2. client.go
Redis client for Celery task queue:
- `NewClient()`: Creates a new Redis client connection
- `SubmitTask()`: Submits tasks in Celery-compatible format
- `GetTaskResult()`: Retrieves task results from Redis backend
- `GetTaskStatus()`: Lightweight status check
- `DeleteTaskResult()`: Cleanup completed task results

**Celery Format**: Tasks are submitted using Celery's message protocol with proper JSON serialization and message wrapper structure.

### 3. service.go
Business logic layer:
- `Summarize()`: Submit text summarization task
- `ExtractKeywords()`: Submit keyword extraction task
- `NormalizeRequest()`: Submit JSON normalization task
- `GetTaskResult()`: Get full task result
- `GetTaskStatus()`: Get task status only
- `DeleteTaskResult()`: Delete task result

Input validation and error handling are performed at this layer.

### 4. handler.go
HTTP handlers with Swagger documentation:
- `POST /api/ai/summarize`: Submit summarization task
- `POST /api/ai/keywords`: Submit keyword extraction task
- `POST /api/ai/normalize`: Submit JSON normalization task
- `GET /api/ai/tasks/{id}`: Get task status and result
- `DELETE /api/ai/tasks/{id}`: Delete task result

All endpoints require authentication (BearerAuth) and return proper HTTP status codes.

## Configuration

Add to `.env` file:

```bash
# Redis Configuration for AI Worker Queue (Optional)
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_QUEUE_NAME=celery
```

If `REDIS_ADDR` is not set, the AI queue module will not be initialized (graceful degradation).

## Dependencies

This module requires:
- `github.com/redis/go-redis/v9`: Redis client for Go
- `github.com/google/uuid`: UUID generation for task IDs

Add to `go.mod`:
```bash
go get github.com/redis/go-redis/v9
```

## Usage Example

### 1. Submit a Summarization Task

```bash
curl -X POST http://localhost:8080/api/ai/summarize \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Long text to summarize...",
    "max_length": 100
  }'
```

Response:
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Summarization task submitted successfully"
}
```

### 2. Check Task Status

```bash
curl -X GET http://localhost:8080/api/ai/tasks/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

Response (pending):
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "PENDING"
}
```

Response (success):
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "SUCCESS",
  "result": {
    "summary": "This is the summarized text."
  },
  "completed_at": "2025-12-06T10:30:00Z"
}
```

Response (failure):
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "FAILURE",
  "error": "Error traceback from worker",
  "completed_at": "2025-12-06T10:30:00Z"
}
```

### 3. Submit Keyword Extraction Task

```bash
curl -X POST http://localhost:8080/api/ai/keywords \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Text to extract keywords from...",
    "max_keywords": 5
  }'
```

### 4. Submit Normalization Task

```bash
curl -X POST http://localhost:8080/api/ai/normalize \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "request": "Create a ticket for network issue",
    "schema": {
      "title": "string",
      "description": "string",
      "category": "string"
    }
  }'
```

## Error Handling

The module handles errors at multiple levels:

1. **Input Validation**: Returns `400 Bad Request` for invalid input
2. **Service Unavailable**: Returns `503 Service Unavailable` if Redis is not configured
3. **Internal Errors**: Returns `500 Internal Server Error` for Redis connection issues
4. **Task Not Found**: Returns `404 Not Found` if task ID doesn't exist

## Integration with Python AI Worker

The Python AI worker should implement Celery tasks matching these task names:

```python
# ai_worker/tasks.py
from celery import Celery

app = Celery('ai_worker',
             broker='redis://localhost:6379/0',
             backend='redis://localhost:6379/0')

@app.task(name='ai_worker.tasks.summarize')
def summarize(text, max_length=None):
    # Implement summarization logic
    return {'summary': 'Summarized text'}

@app.task(name='ai_worker.tasks.extract_keywords')
def extract_keywords(text, max_keywords=None):
    # Implement keyword extraction logic
    return {'keywords': ['keyword1', 'keyword2']}

@app.task(name='ai_worker.tasks.normalize_request')
def normalize_request(request, schema):
    # Implement JSON normalization logic
    return {'title': 'Network Issue', 'description': '...', 'category': 'IT'}
```

## Security Considerations

1. **Authentication Required**: All endpoints require JWT authentication
2. **Authorization**: Can be further restricted using RBAC middleware
3. **Rate Limiting**: Consider implementing rate limiting for AI tasks (not included)
4. **Input Validation**: All inputs are validated before submission
5. **Redis Security**: Use Redis password authentication in production
6. **Result Cleanup**: Implement automatic cleanup of old task results

## Testing

To test the integration:

1. Start Redis:
   ```bash
   docker run -d -p 6379:6379 redis:latest
   ```

2. Start Python AI Worker:
   ```bash
   celery -A ai_worker worker --loglevel=info
   ```

3. Configure `.env`:
   ```bash
   REDIS_ADDR=localhost:6379
   REDIS_QUEUE_NAME=celery
   ```

4. Start Go API server:
   ```bash
   make run
   ```

5. Submit test tasks using curl or Swagger UI at `/swagger/index.html`

## Performance Considerations

- **Connection Pooling**: The Redis client uses connection pooling automatically
- **Result TTL**: Task results have a 1-hour TTL by default (configurable)
- **Async Processing**: All tasks are processed asynchronously
- **Polling**: Client applications should poll for results rather than blocking
- **Cleanup**: Delete task results after retrieval to free Redis memory

## Future Enhancements

Potential improvements:
1. WebSocket support for real-time task updates
2. Task priority queuing
3. Task cancellation support
4. Batch task submission
5. Task retry configuration
6. Metrics and monitoring integration
7. Rate limiting per user
8. Task result pagination
