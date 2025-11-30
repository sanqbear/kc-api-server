# RBAC (Role-Based Access Control) Domain

This document describes the RBAC domain implementation in the Knowledge Center API server.

## Overview

The RBAC domain provides dynamic, database-driven access control for API endpoints. It uses an in-memory permission cache to avoid database lookups on every request, while supporting hot-reload capability for runtime updates.

## Architecture

The RBAC domain follows Domain-Driven Design (DDD) principles with Dependency Injection (DI):

```
internal/rbac/
├── permission_manager.go  # In-memory permission cache with hot-reload
├── repository.go          # Database access layer
├── middleware.go          # Chi authorization middleware
├── handler.go             # HTTP handler for admin operations
└── handler_test.go        # Unit tests
```

### Dependency Flow

```
Handler → PermissionManager → Repository → Database
              ↓
         Middleware (for authorization checks)
```

All dependencies are injected in `internal/server/server.go`.

## Permission Manager

### Structure

The `PermissionManager` holds permission rules in memory using a nested map structure:

```go
map[method]map[path_pattern][]allowed_roles
```

For example:
```go
{
    "GET": {
        "/users": ["admin", "user"],
        "/users/{id}": ["user"],
    },
    "POST": {
        "/users": ["admin"],
    },
    "*": {
        "/public": ["public"],
    },
}
```

### Concurrency

- Uses `sync.RWMutex` for thread-safe access
- Multiple readers can access permissions simultaneously
- Write lock is acquired only during permission reload

### Methods

| Method | Description |
|--------|-------------|
| `LoadPermissions(ctx)` | Fetches permissions from DB and replaces the in-memory cache (Hot Reload) |
| `GetRequiredRoles(method, path)` | Retrieves required roles for a method/path combination |

## Authorization Middleware

### Logic Flow

1. **Full Access Bypass**: If the user has the `full_access` role, the request is allowed immediately without further checks.

2. **Path Matching**: Uses `chi.RouteContext(r.Context()).RoutePattern()` to get the registered route pattern (e.g., `/users/{id}`) instead of the raw URL path.

3. **Permission Check**:
   - Retrieves required roles from PermissionManager using method and route pattern
   - If route is NOT found in the manager, defaults to **allow** (assumes public route)
   - If found, checks if user has at least one of the required roles

4. **Response**: Returns `403 Forbidden` if permission is denied.

### Default Policy

Routes not defined in the `api_permissions` table are **allowed by default**. This assumes:
- Public routes don't need explicit permission entries
- Only restricted routes need to be added to the database

To change to a "deny by default" policy, modify the middleware to return 403 for unregistered routes.

## Database Schema

### api_permissions Table

| Column | Type | Description |
|--------|------|-------------|
| id | BIGINT | Primary key (auto-increment) |
| method | VARCHAR(10) | HTTP method (`GET`, `POST`, `PUT`, `DELETE`, `*`) |
| path_pattern | VARCHAR(255) | Chi router pattern (e.g., `/users/{id}`) |
| required_roles | TEXT[] | Array of allowed roles |
| description | JSONB | Multilingual description |
| created_at | TIMESTAMPTZ | Creation timestamp |
| updated_at | TIMESTAMPTZ | Update timestamp |

### Constraints

- Unique constraint on `(method, path_pattern)` combination
- Index on `path_pattern` for efficient lookups

### Example Data

```sql
INSERT INTO managements.api_permissions (method, path_pattern, required_roles, description) VALUES
('GET', '/users', ARRAY['admin', 'user'], '{"en-US": "List all users"}'),
('POST', '/users', ARRAY['admin'], '{"en-US": "Create a new user"}'),
('GET', '/users/{id}', ARRAY['admin', 'user'], '{"en-US": "Get user by ID"}'),
('PUT', '/users/{id}', ARRAY['admin'], '{"en-US": "Update user"}'),
('DELETE', '/users/{id}', ARRAY['admin'], '{"en-US": "Delete user"}'),
('POST', '/admin/refresh-permissions', ARRAY['full_access'], '{"en-US": "Refresh RBAC cache"}');
```

## API Endpoints

### Refresh Permissions

```http
POST /admin/refresh-permissions
Authorization: Bearer <access_token>
```

Reloads API permissions from the database into the in-memory cache. This endpoint allows hot-reloading of permission rules without restarting the server.

**Response (200 OK):**
```json
{
  "message": "Permissions refreshed successfully"
}
```

**Error Responses:**
- `401 Unauthorized`: Missing or invalid access token
- `403 Forbidden`: User doesn't have `full_access` role
- `500 Internal Server Error`: Database error

## Integration

### Middleware Chain

The RBAC middleware is applied after the authentication middleware:

```go
r.Group(func(r chi.Router) {
    r.Use(s.authMiddleware.Authenticate)  // First: Validate JWT
    r.Use(s.rbacMiddleware.Authorize)     // Second: Check permissions

    // Protected routes...
})
```

### Initial Load

Permissions are loaded from the database when the server starts:

```go
if err := permissionManager.LoadPermissions(context.Background()); err != nil {
    log.Printf("Warning: Failed to load initial permissions: %v", err)
}
```

## Special Roles

### full_access

The `full_access` role has special privileges:
- Bypasses all RBAC permission checks
- Can access any endpoint regardless of permission rules
- Can trigger permission hot-reload via `/admin/refresh-permissions`

## Hot Reload Workflow

1. Update permission rules in the database
2. Call `POST /admin/refresh-permissions` with a `full_access` token
3. The in-memory cache is atomically replaced with new permissions
4. New permission rules take effect immediately for subsequent requests

## Testing

Run the RBAC tests:

```bash
go test ./internal/rbac/... -v
```

The tests cover:
- PermissionManager loading and retrieval
- Middleware authorization logic
- Full access bypass functionality
- Default policy for unregistered routes
- Handler refresh-permissions endpoint

## Security Considerations

1. **Full Access Role**: Reserve this role for trusted administrators only
2. **Permission Updates**: Only full_access users can reload permissions
3. **Default Policy**: Consider changing to "deny by default" for high-security environments
4. **Audit Trail**: Consider logging permission changes for compliance
