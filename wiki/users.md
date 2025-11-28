# Users Domain

This document describes the User domain implementation in the Knowledge Center API server.

## Overview

The Users domain provides CRUD operations for managing user accounts in the system. It includes features for user creation, retrieval, updating, deletion (soft delete), and search functionality.

## Architecture

The Users domain follows Domain-Driven Design (DDD) principles with Dependency Injection (DI):

```
internal/users/
├── model.go       # Data structures and DTOs
├── repository.go  # Database access layer
├── service.go     # Business logic layer
├── handler.go     # HTTP handlers (Controller)
└── handler_test.go # Handler unit tests
```

### Dependency Flow

```
Handler → Service → Repository → Database
```

All dependencies are injected in `cmd/api/main.go` via `internal/server/server.go`.

## Data Model

### Database Schema

The `users` table contains the following columns:

| Column | Type | Description |
|--------|------|-------------|
| id | INT | Internal unique identifier (auto-increment) |
| public_id | UUID | Public unique identifier (UUIDv7, exposed via API) |
| login_id | VARCHAR(255) | User login identifier |
| name | JSONB | Multi-locale name (e.g., `{"en-US": "John", "ko-KR": "존"}`) |
| email | VARCHAR(255) | User email address |
| dept_id | INT | Foreign key to departments table |
| rank_id, duty_id, title_id, position_id, location_id | INT | Foreign keys to common_codes table |
| contact_mobile | VARCHAR(255) | Encrypted mobile number (AES-256-GCM, Base64) |
| contact_mobile_hash | VARCHAR(64) | SHA-256 hash of mobile number |
| contact_mobile_id | VARCHAR(4) | Last 4 digits of mobile number |
| contact_office | VARCHAR(255) | Encrypted office number (AES-256-GCM, Base64) |
| contact_office_hash | VARCHAR(64) | SHA-256 hash of office number |
| contact_office_id | VARCHAR(4) | Last 4 digits of office number |
| password_hash | VARCHAR(255) | Argon2id hashed password |
| is_visible | BOOLEAN | Visibility flag in organization chart |
| is_deleted | BOOLEAN | Soft delete flag |
| created_at | TIMESTAMPTZ | Record creation timestamp |
| updated_at | TIMESTAMPTZ | Record update timestamp |

## API Endpoints

### List Users

```http
GET /users?page=1&limit=10
```

Returns a paginated list of users with simplified response.

**Response:**
```json
{
  "data": [
    {
      "id": "01912345-6789-7abc-def0-123456789abc",
      "login_id": "john.doe",
      "name": {"en-US": "John Doe"},
      "email": "john.doe@example.com"
    }
  ],
  "page": 1,
  "limit": 10,
  "total_count": 100,
  "total_pages": 10
}
```

### Get User by ID

```http
GET /users/{id}
```

Returns detailed user information including related entity names.

**Response:**
```json
{
  "id": "01912345-6789-7abc-def0-123456789abc",
  "login_id": "john.doe",
  "name": {"en-US": "John Doe"},
  "email": "john.doe@example.com",
  "dept_name": {"en-US": "Engineering"},
  "rank_name": {},
  "duty_name": {},
  "title_name": {},
  "position_name": {},
  "location_name": {},
  "contact_mobile": "***-****-5678",
  "contact_office": "***-****-1234"
}
```

### Create User

```http
POST /users
```

Creates a new user. Required fields: `email`, `name` (with at least one locale).

**Request:**
```json
{
  "email": "john.doe@example.com",
  "name": {"en-US": "John Doe"},
  "login_id": "john.doe",
  "password": "securepassword",
  "contact_mobile": "010-1234-5678",
  "contact_office": "02-1234-5678"
}
```

If `login_id` is not provided, the `email` value is used.

### Update User

```http
PUT /users/{id}
```

Updates an existing user. All fields are optional.

**Request:**
```json
{
  "name": {"en-US": "John Updated"},
  "contact_mobile": "010-9999-8888"
}
```

### Delete User

```http
DELETE /users/{id}
```

Performs a soft delete (sets `is_deleted` to true).

### Search Users

```http
POST /users/search?page=1&limit=10
```

Searches for users based on various criteria.

**Request:**
```json
{
  "name": "John",
  "email": "john@example.com",
  "mobile_full": "010-1234-5678",
  "office_full": "02-1234-5678",
  "mobile_last4": "5678",
  "office_last4": "1234"
}
```

## Security Features

### Contact Information Encryption

Phone numbers are protected using multiple layers:

1. **AES-256-GCM Encryption**: Full phone number is encrypted before storage
2. **SHA-256 Hash**: Hash of the full phone number for exact match searching
3. **Last 4 Digits**: Stored separately for partial lookup

### Password Hashing

Passwords are hashed using **Argon2id** algorithm with the following parameters:
- Time: 1
- Memory: 64 MB
- Threads: 4
- Key Length: 32 bytes
- Salt Length: 16 bytes

### Internal ID Protection

The internal sequential ID (`id`) is never exposed via the API. Only the public UUID (`public_id`) is returned as `id` in responses.

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| ENCRYPTION_KEY | Key for AES-256-GCM encryption | (required in production) |

## Error Responses

| Status Code | Error | Description |
|-------------|-------|-------------|
| 400 | Bad Request | Invalid input (email format, empty name) |
| 404 | Not Found | User not found |
| 409 | Conflict | Email or login_id already exists |
| 500 | Internal Server Error | Server-side error |

**Error Response Format:**
```json
{
  "error": "Bad Request",
  "message": "Invalid email format"
}
```

## Testing

Run the handler tests:

```bash
go test ./internal/users/... -v
```

The tests use mock service implementation to test HTTP handlers in isolation.
