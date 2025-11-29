# Authentication Domain

This document describes the Authentication domain implementation in the Knowledge Center API server.

## Overview

The Authentication domain provides user registration, login, and token management functionality. It implements JWT-based authentication with access and refresh tokens, following security best practices.

## Architecture

The Auth domain follows Domain-Driven Design (DDD) principles with Dependency Injection (DI):

```
internal/auth/
├── model.go         # Data structures and DTOs
├── repository.go    # Database access layer
├── service.go       # Business logic layer
├── handler.go       # HTTP handlers (Controller)
├── middleware.go    # JWT authentication middleware
└── handler_test.go  # Handler unit tests
```

### Dependency Flow

```
Handler → Service → Repository → Database
                ↓
           Middleware (for protected routes)
```

All dependencies are injected in `internal/server/server.go`.

## Token Architecture

### Access Token (JWT)

- **Format**: JSON Web Token (JWT) signed with HS256
- **Duration**: 15 minutes
- **Transmission**: Response body (JSON)
- **Storage**: Client-side (memory or secure storage)
- **Claims**:
  - `user_id`: Public UUID of the user
  - `login_id`: User's login identifier
  - `email`: User's email address
  - `roles`: Array of unique role names (from user and group assignments)
  - `jti`: Unique token identifier
  - `iat`: Issued at timestamp
  - `exp`: Expiration timestamp
  - `iss`: Token issuer (`knowledgecenter-api`)

### Refresh Token

- **Format**: Cryptographically random 32-byte string (Base64 URL encoded)
- **Duration**: 7 days
- **Transmission**: HTTP-only cookie
- **Storage**: Database (hashed with SHA-256)
- **Security Features**:
  - Token rotation on each refresh
  - Token lineage tracking (parent/child relationships)
  - Automatic revocation on token reuse detection
  - Client IP and User-Agent tracking

## Database Schema

### user_tokens Table

| Column | Type | Description |
|--------|------|-------------|
| id | BIGINT | Primary key (auto-increment) |
| user_id | INT | Foreign key to users table |
| token_hash | VARCHAR(255) | SHA-256 hash of refresh token |
| expires_at | TIMESTAMPTZ | Token expiration time |
| is_revoked | BOOLEAN | Revocation flag |
| replaced_by_token_id | BIGINT | ID of the replacement token |
| parent_token_id | BIGINT | ID of the parent token |
| client_ip | INET | Client IP address |
| user_agent | VARCHAR(1024) | Client user agent |
| created_at | TIMESTAMPTZ | Creation timestamp |
| updated_at | TIMESTAMPTZ | Update timestamp |

## API Endpoints

### Register

```http
POST /auth/register
```

Creates a new user account and automatically adds them to the 'public' group.

**Request:**
```json
{
  "email": "john.doe@example.com",
  "password": "securePassword123",
  "name": {"en-US": "John Doe"},
  "login_id": "john.doe"  // Optional, defaults to email
}
```

**Response (201 Created):**
```json
{
  "user": {
    "id": "01912345-6789-7abc-def0-123456789abc",
    "login_id": "john.doe",
    "name": {"en-US": "John Doe"},
    "email": "john.doe@example.com"
  },
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900
  },
  "message": "User registered successfully"
}
```

**Cookie Set:**
```
Set-Cookie: refresh_token=<token>; Path=/api/auth; HttpOnly; Secure; SameSite=Strict; Max-Age=604800
```

### Login

```http
POST /auth/login
```

Authenticates a user and returns tokens.

**Request:**
```json
{
  "login_id": "john.doe@example.com",
  "password": "securePassword123"
}
```

**Response (200 OK):**
```json
{
  "user": {
    "id": "01912345-6789-7abc-def0-123456789abc",
    "login_id": "john.doe",
    "name": {"en-US": "John Doe"},
    "email": "john.doe@example.com"
  },
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

### Refresh

```http
POST /auth/refresh
```

Generates new access and refresh tokens using the refresh token from the cookie. Implements token rotation.

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

### Logout

```http
POST /auth/logout
```

Revokes the current refresh token and clears the cookie.

**Response (200 OK):**
```json
{
  "message": "Logged out successfully"
}
```

### Logout All (Protected)

```http
POST /auth/logout-all
Authorization: Bearer <access_token>
```

Revokes all refresh tokens for the current user across all devices.

**Response (200 OK):**
```json
{
  "message": "Logged out from all devices successfully"
}
```

### Get Me (Protected)

```http
GET /auth/me
Authorization: Bearer <access_token>
```

Returns the current user's information and roles.

**Response (200 OK):**
```json
{
  "user": {
    "id": "01912345-6789-7abc-def0-123456789abc",
    "login_id": "john.doe",
    "name": {"en-US": "John Doe"},
    "email": "john.doe@example.com"
  },
  "roles": ["user", "member"]
}
```

## Role System

Roles can be assigned to users through two mechanisms:

1. **Direct User Roles**: Roles assigned directly to a user via `user_roles` table
2. **Group Roles**: Roles assigned to groups via `group_roles` table, inherited by all group members

The token's `roles` claim contains a deduplicated list of all roles from both sources.

### Default Group Assignment

When a user registers, they are automatically added to the `public` group. This group should exist in the database with the `public_id` of `"public"`.

## Security Features

### Password Hashing

Passwords are hashed using **Argon2id** algorithm:
- Time: 1 iteration
- Memory: 64 MB
- Threads: 4
- Key Length: 32 bytes
- Salt Length: 16 bytes

### Token Security

1. **Short-lived Access Tokens**: 15-minute expiration limits exposure
2. **Refresh Token Rotation**: New refresh token issued on each refresh
3. **Token Reuse Detection**: If a revoked token is used, all user tokens are revoked
4. **Secure Cookie Settings**:
   - `HttpOnly`: Prevents JavaScript access
   - `Secure`: HTTPS only (in production)
   - `SameSite=Strict`: CSRF protection

### Bearer Authentication

Protected endpoints require the `Authorization` header:
```
Authorization: Bearer <access_token>
```

## Middleware

### Authenticate Middleware

Validates the JWT access token and adds user information to the request context:

```go
r.Use(authMiddleware.Authenticate)
```

Context values set:
- `userID`: User's public UUID
- `userRoles`: Array of role names
- `claims`: Full token claims

### RequireRoles Middleware

Requires the user to have at least one of the specified roles:

```go
r.Use(authMiddleware.RequireRoles("admin", "moderator"))
```

### RequireAllRoles Middleware

Requires the user to have all of the specified roles:

```go
r.Use(authMiddleware.RequireAllRoles("admin", "superuser"))
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| JWT_SECRET | Secret key for signing JWT tokens | (required in production) |

## Error Responses

| Status Code | Error | Description |
|-------------|-------|-------------|
| 400 | Bad Request | Invalid input or validation error |
| 401 | Unauthorized | Invalid credentials or token |
| 403 | Forbidden | Insufficient permissions |
| 409 | Conflict | Email or login_id already exists |
| 500 | Internal Server Error | Server-side error |

**Error Response Format:**
```json
{
  "error": "Unauthorized",
  "message": "Invalid credentials"
}
```

## Extensibility

The authentication system is designed for future extensibility:

1. **OAuth/SSO Integration**: The service interface can be extended to support external authentication providers
2. **Multi-factor Authentication**: The login flow can be enhanced with MFA steps
3. **Session Management**: Additional session tracking features can be added to the token storage

## Testing

Run the handler tests:

```bash
go test ./internal/auth/... -v
```

The tests use mock service implementation to test HTTP handlers and middleware in isolation.
