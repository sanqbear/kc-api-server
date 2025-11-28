# Swag API Comment Guidelines

This document provides guidelines for writing API documentation comments using the [swaggo/swag](https://github.com/swaggo/swag) library.

## General API Information

General API info is defined in `cmd/api/main.go` with the following annotations:

```go
// @title Knowledge Center API
// @version 1.0
// @description Knowledge Center REST API Server

// @contact.name API Support
// @contact.email support@knowledgecenter.io

// @license.name GNU General Public License v3.0
// @license.url https://www.gnu.org/licenses/gpl-3.0.html

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
```

## Handler Annotations

Each handler function should have swagger annotations. Place them directly above the function declaration.

### Basic Structure

```go
// HandlerName godoc
// @Summary      Short summary of the endpoint
// @Description  Detailed description of what the endpoint does
// @Tags         tag-name
// @Accept       json
// @Produce      json
// @Param        paramName  paramType  dataType  required  "description"
// @Success      200  {object}  ResponseType
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /path [method]
func (s *Server) HandlerName(w http.ResponseWriter, r *http.Request) {
    // implementation
}
```

## Common Annotations

### @Summary and @Description
- `@Summary` - Brief one-line description (shown in endpoint list)
- `@Description` - Detailed explanation (shown in endpoint details)

### @Tags
Groups endpoints in Swagger UI. Use lowercase with hyphens:
```go
// @Tags users
// @Tags user-management
```

### @Accept and @Produce
Specify content types:
```go
// @Accept json
// @Produce json
```

Common values: `json`, `xml`, `plain`, `html`, `mpfd` (multipart/form-data)

### @Param
Define request parameters:

```go
// Path parameter
// @Param id path int true "User ID"

// Query parameter
// @Param page query int false "Page number" default(1)

// Header parameter
// @Param Authorization header string true "Bearer token"

// Body parameter
// @Param request body CreateUserRequest true "User data"

// Form parameter
// @Param file formData file true "Upload file"
```

Format: `@Param name location type required "description" attributes`

Locations: `path`, `query`, `header`, `body`, `formData`

### @Success and @Failure
Define response types:

```go
// @Success 200 {object} User "Success response"
// @Success 201 {object} User "Created"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse "Bad Request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Not Found"
// @Failure 500 {object} ErrorResponse "Internal Server Error"
```

### @Router
Define the route path and HTTP method:
```go
// @Router /users/{id} [get]
// @Router /users [post]
// @Router /users/{id} [put]
// @Router /users/{id} [delete]
```

### @Security
Apply security definitions:
```go
// @Security BearerAuth
```

## Response Types

### Using Structs
Define response structs with JSON tags and examples:

```go
type User struct {
    ID    int    `json:"id" example:"1"`
    Name  string `json:"name" example:"John Doe"`
    Email string `json:"email" example:"john@example.com"`
}
```

### Using Maps
For simple key-value responses:
```go
// @Success 200 {object} map[string]string
```

### Using Arrays
```go
// @Success 200 {array} User
```

## Examples

### GET endpoint with path parameter
```go
// GetUser godoc
// @Summary      Get user by ID
// @Description  Retrieves a user by their unique identifier
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "User ID"
// @Success      200  {object}  User
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /users/{id} [get]
func (s *Server) GetUser(w http.ResponseWriter, r *http.Request) {}
```

### POST endpoint with body
```go
// CreateUser godoc
// @Summary      Create a new user
// @Description  Creates a new user with the provided data
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request  body      CreateUserRequest  true  "User data"
// @Success      201      {object}  User
// @Failure      400      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /users [post]
func (s *Server) CreateUser(w http.ResponseWriter, r *http.Request) {}
```

### GET endpoint with query parameters
```go
// ListUsers godoc
// @Summary      List users
// @Description  Retrieves a paginated list of users
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        page   query     int     false  "Page number"     default(1)
// @Param        limit  query     int     false  "Items per page"  default(10)
// @Param        q      query     string  false  "Search query"
// @Success      200    {array}   User
// @Security     BearerAuth
// @Router       /users [get]
func (s *Server) ListUsers(w http.ResponseWriter, r *http.Request) {}
```

## Generating Documentation

Run the following command to regenerate swagger docs after modifying annotations:

```bash
make swag
```

This generates files in the `docs/` directory:
- `docs.go` - Go file imported by the application
- `swagger.json` - OpenAPI 2.0 specification in JSON
- `swagger.yaml` - OpenAPI 2.0 specification in YAML

## Accessing Swagger UI

After starting the server, access Swagger UI at:
```
http://localhost:8080/swagger/index.html
```

## JWT Authentication in Swagger UI

1. Click the "Authorize" button in Swagger UI
2. Enter your JWT token in the format: `Bearer <your-token>`
3. Click "Authorize" to apply the token to all requests
