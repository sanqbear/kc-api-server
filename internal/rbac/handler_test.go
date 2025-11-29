package rbac

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"kc-api/internal/auth"
)

// MockRepository is a mock implementation of the Repository interface for testing
type MockRepository struct {
	GetAllPermissionsFunc func(ctx context.Context) ([]APIPermission, error)
}

func (m *MockRepository) GetAllPermissions(ctx context.Context) ([]APIPermission, error) {
	if m.GetAllPermissionsFunc != nil {
		return m.GetAllPermissionsFunc(ctx)
	}
	return nil, nil
}

func TestPermissionManager_LoadPermissions(t *testing.T) {
	tests := []struct {
		name        string
		permissions []APIPermission
		mockError   error
		expectError bool
	}{
		{
			name: "successful load",
			permissions: []APIPermission{
				{ID: 1, Method: "GET", PathPattern: "/users", RequiredRoles: []string{"admin", "user"}},
				{ID: 2, Method: "POST", PathPattern: "/users", RequiredRoles: []string{"admin"}},
				{ID: 3, Method: "*", PathPattern: "/public", RequiredRoles: []string{"public"}},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "empty permissions",
			permissions: []APIPermission{},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "database error",
			permissions: nil,
			mockError:   errors.New("database connection failed"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{
				GetAllPermissionsFunc: func(ctx context.Context) ([]APIPermission, error) {
					return tt.permissions, tt.mockError
				},
			}

			pm := NewPermissionManager(mockRepo)
			err := pm.LoadPermissions(context.Background())

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestPermissionManager_GetRequiredRoles(t *testing.T) {
	mockRepo := &MockRepository{
		GetAllPermissionsFunc: func(ctx context.Context) ([]APIPermission, error) {
			return []APIPermission{
				{ID: 1, Method: "GET", PathPattern: "/users", RequiredRoles: []string{"admin", "user"}},
				{ID: 2, Method: "POST", PathPattern: "/users", RequiredRoles: []string{"admin"}},
				{ID: 3, Method: "*", PathPattern: "/public", RequiredRoles: []string{"public"}},
				{ID: 4, Method: "GET", PathPattern: "/users/{id}", RequiredRoles: []string{"user"}},
			}, nil
		},
	}

	pm := NewPermissionManager(mockRepo)
	_ = pm.LoadPermissions(context.Background())

	tests := []struct {
		name          string
		method        string
		path          string
		expectedRoles []string
		expectedFound bool
	}{
		{
			name:          "exact method and path match",
			method:        "GET",
			path:          "/users",
			expectedRoles: []string{"admin", "user"},
			expectedFound: true,
		},
		{
			name:          "POST method match",
			method:        "POST",
			path:          "/users",
			expectedRoles: []string{"admin"},
			expectedFound: true,
		},
		{
			name:          "wildcard method match",
			method:        "DELETE",
			path:          "/public",
			expectedRoles: []string{"public"},
			expectedFound: true,
		},
		{
			name:          "parameterized path",
			method:        "GET",
			path:          "/users/{id}",
			expectedRoles: []string{"user"},
			expectedFound: true,
		},
		{
			name:          "unregistered path",
			method:        "GET",
			path:          "/unknown",
			expectedRoles: nil,
			expectedFound: false,
		},
		{
			name:          "unregistered method for path",
			method:        "DELETE",
			path:          "/users",
			expectedRoles: nil,
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles, found := pm.GetRequiredRoles(tt.method, tt.path)

			if found != tt.expectedFound {
				t.Errorf("expected found=%v, got found=%v", tt.expectedFound, found)
			}

			if tt.expectedFound {
				if len(roles) != len(tt.expectedRoles) {
					t.Errorf("expected %d roles, got %d", len(tt.expectedRoles), len(roles))
				}
			}
		})
	}
}

func TestMiddleware_Authorize(t *testing.T) {
	// Setup mock repository with test permissions
	mockRepo := &MockRepository{
		GetAllPermissionsFunc: func(ctx context.Context) ([]APIPermission, error) {
			return []APIPermission{
				{ID: 1, Method: "GET", PathPattern: "/users", RequiredRoles: []string{"admin", "user"}},
				{ID: 2, Method: "POST", PathPattern: "/users", RequiredRoles: []string{"admin"}},
				{ID: 3, Method: "GET", PathPattern: "/admin/settings", RequiredRoles: []string{"admin"}},
			}, nil
		},
	}

	pm := NewPermissionManager(mockRepo)
	_ = pm.LoadPermissions(context.Background())
	middleware := NewMiddleware(pm)

	tests := []struct {
		name           string
		method         string
		path           string
		routePattern   string
		userRoles      []string
		expectedStatus int
	}{
		{
			name:           "sysadmin bypass",
			method:         "POST",
			path:           "/users",
			routePattern:   "/users",
			userRoles:      []string{"sysadmin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user has required role",
			method:         "GET",
			path:           "/users",
			routePattern:   "/users",
			userRoles:      []string{"user"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user has admin role",
			method:         "POST",
			path:           "/users",
			routePattern:   "/users",
			userRoles:      []string{"admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user missing required role",
			method:         "POST",
			path:           "/users",
			routePattern:   "/users",
			userRoles:      []string{"user"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "no user roles - forbidden",
			method:         "GET",
			path:           "/users",
			routePattern:   "/users",
			userRoles:      nil,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "unregistered route - allow by default",
			method:         "GET",
			path:           "/unknown",
			routePattern:   "/unknown",
			userRoles:      []string{"user"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unregistered route - allow even without roles",
			method:         "GET",
			path:           "/public-endpoint",
			routePattern:   "/public-endpoint",
			userRoles:      nil,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Create chi router to set route context
			r := chi.NewRouter()
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Simulate user roles in context (normally set by auth middleware)
					ctx := r.Context()
					if tt.userRoles != nil {
						ctx = setUserRolesInContext(ctx, tt.userRoles)
					}
					next.ServeHTTP(w, r.WithContext(ctx))
				})
			})
			r.Use(middleware.Authorize)
			r.Method(tt.method, tt.routePattern, testHandler)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_RefreshPermissions(t *testing.T) {
	tests := []struct {
		name           string
		mockError      error
		expectedStatus int
	}{
		{
			name:           "successful refresh",
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "database error",
			mockError:      errors.New("database connection failed"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{
				GetAllPermissionsFunc: func(ctx context.Context) ([]APIPermission, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return []APIPermission{
						{ID: 1, Method: "GET", PathPattern: "/users", RequiredRoles: []string{"user"}},
					}, nil
				},
			}

			pm := NewPermissionManager(mockRepo)
			handler := NewHandler(pm)

			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			req := httptest.NewRequest(http.MethodPost, "/admin/refresh-permissions", nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

// setUserRolesInContext is a helper function for testing
// It uses auth.SetUserRolesInContext to set roles properly
func setUserRolesInContext(ctx context.Context, roles []string) context.Context {
	return auth.SetUserRolesInContext(ctx, roles)
}
