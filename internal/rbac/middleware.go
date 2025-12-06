package rbac

import (
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"

	"kc-api/internal/auth"
	"kc-api/internal/utils"
)

// Middleware provides RBAC authorization middleware
type Middleware struct {
	permissionManager *PermissionManager
}

// NewMiddleware creates a new RBAC middleware
func NewMiddleware(pm *PermissionManager) *Middleware {
	return &Middleware{permissionManager: pm}
}

// Authorize is a Chi middleware that intercepts requests and checks permissions.
// Prerequisite: A previous middleware must have already parsed the JWT and stored
// the user's roles in r.Context() with the key "user_roles".
func (m *Middleware) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user roles from context (set by auth.Authenticate middleware)
		userRoles := auth.GetUserRolesFromContext(r.Context())

		// 1. Sysadmin Bypass: If the user has the role "full_access", allow immediately
		if slices.Contains(userRoles, "full_access") {
			next.ServeHTTP(w, r)
			return
		}

		// 2. Path Matching: Use Chi RouteContext to get the registered route pattern
		// This returns patterns like "/users/{id}" instead of "/users/123"
		routeCtx := chi.RouteContext(r.Context())
		routePattern := routeCtx.RoutePattern()

		// If route pattern is empty, use the raw URL path
		if routePattern == "" {
			routePattern = r.URL.Path
		}

		method := r.Method

		// 3. Permission Check: Retrieve required roles from PermissionManager
		requiredRoles, found := m.permissionManager.GetRequiredRoles(method, routePattern)

		if !found {
			// Default policy: Allow requests to routes not defined in the permission manager.
			// This assumes that routes not explicitly restricted are public.
			// NOTE: To change this to a "deny by default" policy, return 403 here instead.
			next.ServeHTTP(w, r)
			return
		}

		// Check if user has at least one of the required roles
		if len(userRoles) == 0 {
			utils.RespondError(w, r, http.StatusForbidden, "Forbidden", "Access denied: authentication required")
			return
		}

		hasRequiredRole := false
		for _, requiredRole := range requiredRoles {
			if slices.Contains(userRoles, requiredRole) {
				hasRequiredRole = true
				break
			}
		}

		if !hasRequiredRole {
			utils.RespondError(w, r, http.StatusForbidden, "Forbidden", "Access denied: insufficient permissions")
			return
		}

		// Permission granted
		next.ServeHTTP(w, r)
	})
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
