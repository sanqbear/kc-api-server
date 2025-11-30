# Tickets Domain

This document describes the Tickets domain implementation in the Knowledge Center API server.

## Overview

The Tickets domain provides a complete ticket management system for tracking support requests and their resolution history. It includes four main entities:

- **Ticket**: A ticket object representing a request
- **Ticket Entry**: The body of a ticket in thread format (comments, files, schedules, events)
- **Tag**: Labels that can be added to tickets or entries
- **Entry Reference**: Backlinks tracking for entries

## Architecture

The Tickets domain follows Domain-Driven Design (DDD) principles with Dependency Injection (DI):

```
internal/tickets/
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

All dependencies are injected in `internal/server/server.go`.

## Data Model

### Ticket Table

| Column | Type | Description |
|--------|------|-------------|
| id | BIGINT | Internal unique identifier (auto-generated) |
| public_id | UUID | Public unique identifier (UUIDv7) |
| title | VARCHAR(255) | Title of the ticket |
| assigned_user_id | INT | Foreign key to users table |
| status | ENUM | OPEN, WAITING_FOR_INFO, IN_PROGRESS, RESOLVED, CLOSED, REOPENED |
| priority | ENUM | LOW, MEDIUM, HIGH, CRITICAL |
| request_type | ENUM | BUG, MAINTENANCE, FEATURE_REQUEST, GENERAL_INQUIRY |
| due_date | TIMESTAMPTZ | Due date for ticket resolution |
| created_at | TIMESTAMPTZ | Record creation timestamp |
| updated_at | TIMESTAMPTZ | Record update timestamp |

### Ticket Entry Table

| Column | Type | Description |
|--------|------|-------------|
| id | BIGINT | Internal unique identifier |
| ticket_id | BIGINT | Foreign key to tickets table |
| author_user_id | BIGINT | Foreign key to users table |
| parent_entry_id | BIGINT | Self-reference for hierarchical entries |
| entry_type | ENUM | COMMENT, FILE, SCHEDULE, EVENT |
| format | ENUM | PLAIN_TEXT, MARKDOWN, HTML, NONE |
| body | TEXT | Main content of the entry |
| payload | JSONB | Additional structured data |
| search_vector | TSVECTOR | Full-text search indexing |
| is_deleted | BOOLEAN | Soft delete flag |
| created_at | TIMESTAMPTZ | Record creation timestamp |
| updated_at | TIMESTAMPTZ | Record update timestamp |

### Tags Table

| Column | Type | Description |
|--------|------|-------------|
| id | BIGINT | Internal unique identifier |
| name | VARCHAR(255) | Tag name |
| color_code | VARCHAR(7) | Hex color code (e.g., #FF0000) |
| is_deleted | BOOLEAN | Soft delete flag |
| created_at | TIMESTAMPTZ | Record creation timestamp |
| updated_at | TIMESTAMPTZ | Record update timestamp |

### Entry References Table

| Column | Type | Description |
|--------|------|-------------|
| source_entry_id | BIGINT | Source entry reference |
| target_entry_id | BIGINT | Target entry reference (nullable) |
| target_ticket_id | BIGINT | Target ticket reference (nullable) |
| target_user_id | BIGINT | Target user reference (nullable) |
| created_at | TIMESTAMPTZ | Record creation timestamp |

## API Endpoints

### Ticket Endpoints

#### List Tickets

```http
GET /tickets?page=1&limit=10
```

Returns a paginated list of tickets with simplified response.

**Response:**
```json
{
  "data": [
    {
      "id": "01912345-6789-7abc-def0-123456789abc",
      "title": "Bug in login page",
      "status": "OPEN",
      "priority": "HIGH",
      "request_type": "BUG",
      "due_date": "2024-12-31T23:59:59Z",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "page": 1,
  "limit": 10,
  "total_count": 100,
  "total_pages": 10
}
```

#### Get Ticket by ID

```http
GET /tickets/{id}
```

Returns detailed ticket information including entries and tags.

**Response:**
```json
{
  "id": "01912345-6789-7abc-def0-123456789abc",
  "title": "Bug in login page",
  "status": "OPEN",
  "priority": "HIGH",
  "request_type": "BUG",
  "assigned_user_id": "01912345-6789-7abc-def0-123456789abc",
  "assigned_user_name": {"en-US": "John Doe"},
  "due_date": "2024-12-31T23:59:59Z",
  "tags": [
    {
      "id": 1,
      "name": "urgent",
      "color_code": "#FF0000",
      "category": "priority"
    }
  ],
  "entries": [
    {
      "id": 1,
      "entry_type": "COMMENT",
      "format": "MARKDOWN",
      "body": "Initial description of the issue",
      "author_user_id": "01912345-6789-7abc-def0-123456789abc",
      "author_user_name": {"en-US": "John Doe"},
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

#### Create Ticket

```http
POST /tickets
```

Creates a new ticket with an initial entry. When creating a ticket, an initial entry must be provided.

**Request:**
```json
{
  "title": "Bug in login page",
  "status": "OPEN",
  "priority": "HIGH",
  "request_type": "BUG",
  "assigned_user_id": "01912345-6789-7abc-def0-123456789abc",
  "due_date": "2024-12-31T23:59:59Z",
  "tag_ids": [1, 2],
  "initial_entry": {
    "entry_type": "COMMENT",
    "format": "MARKDOWN",
    "body": "Description of the issue",
    "payload": {},
    "tag_ids": [3],
    "references": [
      {"target_ticket_id": "01912345-6789-7abc-def0-987654321abc"}
    ]
  }
}
```

#### Update Ticket

```http
PUT /tickets/{id}
```

Updates an existing ticket. All fields are optional.

**Request:**
```json
{
  "title": "Updated title",
  "status": "IN_PROGRESS",
  "priority": "CRITICAL",
  "assigned_user_id": "01912345-6789-7abc-def0-123456789abc"
}
```

#### Delete Ticket

```http
DELETE /tickets/{id}
```

Deletes a ticket and all its entries (cascade delete).

#### Search Tickets

```http
POST /tickets/search?page=1&limit=10
```

Searches for tickets based on various criteria.

**Request:**
```json
{
  "query": "login bug",
  "status": ["OPEN", "IN_PROGRESS"],
  "priority": ["HIGH", "CRITICAL"],
  "request_type": ["BUG"],
  "tag_ids": [1, 2],
  "assigned_user_id": "01912345-6789-7abc-def0-123456789abc",
  "due_date_from": "2024-01-01T00:00:00Z",
  "due_date_to": "2024-12-31T23:59:59Z"
}
```

#### Add Tags to Ticket

```http
POST /tickets/{id}/tags
```

**Request:**
```json
{
  "tag_ids": [1, 2],
  "category": "priority"
}
```

#### Remove Tag from Ticket

```http
DELETE /tickets/{id}/tags/{tagId}
```

### Entry Endpoints

#### Create Entry

```http
POST /tickets/{id}/entries
```

Creates a new entry for a ticket.

**Request:**
```json
{
  "entry_type": "COMMENT",
  "format": "MARKDOWN",
  "body": "This is a comment",
  "payload": {},
  "parent_entry_id": 1,
  "tag_ids": [1],
  "references": [
    {"target_user_id": "01912345-6789-7abc-def0-123456789abc"}
  ]
}
```

#### Get Entry by ID

```http
GET /entries/{id}
```

Returns detailed entry information including tags and references.

**Response:**
```json
{
  "id": 1,
  "ticket_id": "01912345-6789-7abc-def0-123456789abc",
  "entry_type": "COMMENT",
  "format": "MARKDOWN",
  "body": "This is a comment",
  "payload": {},
  "author_user_id": "01912345-6789-7abc-def0-123456789abc",
  "author_user_name": {"en-US": "John Doe"},
  "parent_entry_id": null,
  "tags": [],
  "references": [
    {
      "target_type": "user",
      "target_user_id": "01912345-6789-7abc-def0-123456789abc",
      "target_user_name": {"en-US": "Jane Doe"},
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

#### Update Entry

```http
PUT /entries/{id}
```

**Request:**
```json
{
  "format": "MARKDOWN",
  "body": "Updated content",
  "payload": {"key": "value"}
}
```

#### Delete Entry

```http
DELETE /entries/{id}
```

Performs a soft delete on the entry.

#### Add Tags to Entry

```http
POST /entries/{id}/tags
```

#### Remove Tag from Entry

```http
DELETE /entries/{id}/tags/{tagId}
```

### Tag Endpoints

#### List Tags

```http
GET /tags?page=1&limit=10
```

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "name": "urgent",
      "color_code": "#FF0000"
    }
  ],
  "page": 1,
  "limit": 10,
  "total_count": 10,
  "total_pages": 1
}
```

#### Create Tag

```http
POST /tags
```

**Request:**
```json
{
  "name": "urgent",
  "color_code": "#FF0000"
}
```

#### Get Tag by ID

```http
GET /tags/{id}
```

#### Update Tag

```http
PUT /tags/{id}
```

**Request:**
```json
{
  "name": "very-urgent",
  "color_code": "#FF5500"
}
```

#### Delete Tag

```http
DELETE /tags/{id}
```

Performs a soft delete on the tag.

## Entry Types

| Type | Description | Payload Example |
|------|-------------|-----------------|
| COMMENT | Text comments on tickets | `{}` |
| FILE | File attachments | `{"file_url": "...", "file_name": "..."}` |
| SCHEDULE | Schedule/meeting entries | `{"start_time": "...", "end_time": "..."}` |
| EVENT | System events or status changes | `{"event_type": "status_change", "from": "OPEN", "to": "IN_PROGRESS"}` |

## Content Formats

| Format | Description |
|--------|-------------|
| PLAIN_TEXT | Plain text content |
| MARKDOWN | Markdown formatted content |
| HTML | HTML formatted content |
| NONE | No body content (used with payload-only entries) |

## Reference Types

References allow entries to link to:
- **Entry**: Other entries within the ticket system
- **Ticket**: Other tickets
- **User**: Users mentioned in the entry

## Error Responses

| Status Code | Error | Description |
|-------------|-------|-------------|
| 400 | Bad Request | Invalid input (empty title, invalid ID format) |
| 404 | Not Found | Ticket, entry, or tag not found |
| 500 | Internal Server Error | Server-side error |

**Error Response Format:**
```json
{
  "error": "Bad Request",
  "message": "Title is required"
}
```

## Testing

Run the handler tests:

```bash
go test ./internal/tickets/... -v
```

The tests use mock service implementation to test HTTP handlers in isolation.
