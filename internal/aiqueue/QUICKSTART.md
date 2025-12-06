# AI Queue Quick Start Guide

## Installation

1. Add Redis dependency to your project:
   ```bash
   cd server
   go get github.com/redis/go-redis/v9
   go mod tidy
   ```

2. Configure environment variables in `.env`:
   ```bash
   # Redis Configuration for AI Worker Queue
   REDIS_ADDR=localhost:6379
   REDIS_PASSWORD=
   REDIS_DB=0
   REDIS_QUEUE_NAME=celery
   ```

3. Start Redis server:
   ```bash
   # Using Docker
   docker run -d -p 6379:6379 --name redis redis:latest

   # Or using docker-compose (add to your docker-compose.yml)
   redis:
     image: redis:latest
     ports:
       - "6379:6379"
   ```

4. Rebuild and run the server:
   ```bash
   make build
   make run
   ```

## API Endpoints

All endpoints require JWT authentication via `Authorization: Bearer <token>` header.

### 1. Summarize Text
```http
POST /api/ai/summarize
Content-Type: application/json

{
  "text": "Your long text here...",
  "max_length": 100
}
```

**Response (202 Accepted):**
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Summarization task submitted successfully"
}
```

### 2. Extract Keywords
```http
POST /api/ai/keywords
Content-Type: application/json

{
  "text": "Your text here...",
  "max_keywords": 5
}
```

**Response (202 Accepted):**
```json
{
  "task_id": "660f9511-f3ac-52e5-b827-557766551111",
  "message": "Keyword extraction task submitted successfully"
}
```

### 3. Normalize Request
```http
POST /api/ai/normalize
Content-Type: application/json

{
  "request": "Create a ticket for the network issue in building A",
  "schema": {
    "title": "string",
    "description": "string",
    "category": "string",
    "priority": "string"
  }
}
```

**Response (202 Accepted):**
```json
{
  "task_id": "770g0622-g4bd-63f6-c938-668877662222",
  "message": "Normalization task submitted successfully"
}
```

### 4. Get Task Status
```http
GET /api/ai/tasks/{task_id}
```

**Response (200 OK) - Pending:**
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "PENDING"
}
```

**Response (200 OK) - Success:**
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "SUCCESS",
  "result": {
    "summary": "This is the summarized text."
  },
  "started_at": "2025-12-06T10:29:55Z",
  "completed_at": "2025-12-06T10:30:02Z"
}
```

**Response (200 OK) - Failure:**
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "FAILURE",
  "error": "Traceback (most recent call last)...",
  "started_at": "2025-12-06T10:29:55Z",
  "completed_at": "2025-12-06T10:30:00Z"
}
```

### 5. Delete Task Result
```http
DELETE /api/ai/tasks/{task_id}
```

**Response (200 OK):**
```json
{
  "message": "Task result deleted successfully"
}
```

## Task Status Values

- `PENDING`: Task is waiting in the queue
- `STARTED`: Worker has started processing the task
- `SUCCESS`: Task completed successfully, result is available
- `FAILURE`: Task failed, error message is available
- `RETRY`: Task is being retried after a failure
- `REVOKED`: Task was cancelled

## Error Responses

### 400 Bad Request
```json
{
  "error": "Bad Request",
  "message": "Text is required"
}
```

### 503 Service Unavailable
```json
{
  "error": "Service Unavailable",
  "message": "AI worker queue is not available"
}
```

### 500 Internal Server Error
```json
{
  "error": "Internal Server Error",
  "message": "Failed to submit summarization task"
}
```

## Testing with cURL

### Example: Complete Workflow

1. **Login to get JWT token:**
   ```bash
   curl -X POST http://localhost:8080/auth/login \
     -H "Content-Type: application/json" \
     -d '{
       "login_id": "admin",
       "password": "password123"
     }'
   ```

2. **Submit summarization task:**
   ```bash
   curl -X POST http://localhost:8080/api/ai/summarize \
     -H "Authorization: Bearer YOUR_JWT_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "text": "Artificial intelligence is the simulation of human intelligence processes by machines, especially computer systems. These processes include learning, reasoning, and self-correction. Particular applications of AI include expert systems, natural language processing, speech recognition and machine vision.",
       "max_length": 50
     }'
   ```

3. **Check task status (poll every 2-3 seconds):**
   ```bash
   curl -X GET http://localhost:8080/api/ai/tasks/TASK_ID \
     -H "Authorization: Bearer YOUR_JWT_TOKEN"
   ```

4. **Delete task result after retrieval:**
   ```bash
   curl -X DELETE http://localhost:8080/api/ai/tasks/TASK_ID \
     -H "Authorization: Bearer YOUR_JWT_TOKEN"
   ```

## Swagger UI

Access interactive API documentation at:
```
http://localhost:8080/swagger/index.html
```

1. Click "Authorize" button
2. Enter: `Bearer YOUR_JWT_TOKEN`
3. Try out the AI endpoints under the "ai" tag

## Integration Notes

### Optional Module
The AI queue module is optional. If Redis is not configured (`REDIS_ADDR` not set), the server will start normally without AI functionality. You'll see this log message:
```
AI queue integration not configured (REDIS_ADDR not set)
```

### Graceful Degradation
- If Redis connection fails during startup, the server continues running
- AI endpoints will return `503 Service Unavailable`
- Other API functionality remains unaffected

### Production Checklist

Before deploying to production:

1. Set Redis password:
   ```bash
   REDIS_PASSWORD=your-secure-password
   ```

2. Use Redis Sentinel or Cluster for high availability

3. Configure Redis persistence (AOF or RDB)

4. Set up monitoring for Redis and task queue

5. Implement rate limiting for AI endpoints

6. Configure appropriate timeouts for long-running tasks

7. Set up log aggregation for both Go server and Python workers

8. Test failover scenarios (Redis down, worker down, etc.)

## Troubleshooting

### "AI worker queue is not available" error
- Check if Redis is running: `redis-cli ping`
- Verify `REDIS_ADDR` in `.env`
- Check server logs for connection errors

### Task stays in PENDING status
- Ensure Python AI worker is running
- Check worker logs for errors
- Verify queue name matches in both Go and Python

### Task result not found
- Results may have expired (1-hour TTL)
- Task ID might be incorrect
- Check Redis for results: `redis-cli GET celery-task-meta-TASK_ID`

### Redis connection timeout
- Check network connectivity
- Verify Redis is listening on correct port
- Check firewall rules

## Development Tips

1. **Monitor Redis queue:**
   ```bash
   redis-cli LLEN celery
   ```

2. **View task results:**
   ```bash
   redis-cli KEYS "celery-task-meta-*"
   redis-cli GET "celery-task-meta-TASK_ID"
   ```

3. **Clear all tasks:**
   ```bash
   redis-cli FLUSHDB
   ```

4. **Watch Redis commands:**
   ```bash
   redis-cli MONITOR
   ```
