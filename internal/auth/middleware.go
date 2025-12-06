package auth

import (
	"context"
	"net/http"
	"strings"

	"kc-api/internal/utils"
)

// Context keys for storing user information
type contextKey string

const (
	userIDKey    contextKey = "userID"
	userRolesKey contextKey = "userRoles"
	claimsKey    contextKey = "claims"
)

// Middleware provides JWT authentication middleware
type Middleware struct {
	service Service
}

// NewMiddleware creates a new auth middleware
func NewMiddleware(service Service) *Middleware {
	return &Middleware{service: service}
}

// Authenticate is a middleware that validates JWT tokens from the Authorization header
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.RespondError(w, r, http.StatusUnauthorized, "Unauthorized", "Authorization header required")
			return
		}

		// Check Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			utils.RespondError(w, r, http.StatusUnauthorized, "Unauthorized", "Invalid authorization header format")
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := m.service.ValidateAccessToken(tokenString)
		if err != nil {
			utils.RespondError(w, r, http.StatusUnauthorized, "Unauthorized", "Invalid or expired token")
			return
		}

		// Add user info to context
		ctx := r.Context()
		ctx = context.WithValue(ctx, userIDKey, claims.UserID)
		ctx = context.WithValue(ctx, userRolesKey, claims.Roles)
		ctx = context.WithValue(ctx, claimsKey, claims)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuthenticate is a middleware that validates JWT tokens if present, but doesn't require them
func (m *Middleware) OptionalAuthenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				tokenString := parts[1]

				// Validate token
				claims, err := m.service.ValidateAccessToken(tokenString)
				if err == nil {
					// Add user info to context
					ctx := r.Context()
					ctx = context.WithValue(ctx, userIDKey, claims.UserID)
					ctx = context.WithValue(ctx, userRolesKey, claims.Roles)
					ctx = context.WithValue(ctx, claimsKey, claims)
					r = r.WithContext(ctx)
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// RequireRoles is a middleware that checks if the user has at least one of the required roles
func (m *Middleware) RequireRoles(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRoles := GetUserRolesFromContext(r.Context())
			if len(userRoles) == 0 {
				utils.RespondError(w, r, http.StatusForbidden, "Forbidden", "Access denied")
				return
			}

			// Check if user has any of the required roles
			hasRole := false
			for _, requiredRole := range roles {
				for _, userRole := range userRoles {
					if userRole == requiredRole {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}

			if !hasRole {
				utils.RespondError(w, r, http.StatusForbidden, "Forbidden", "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAllRoles is a middleware that checks if the user has all of the required roles
func (m *Middleware) RequireAllRoles(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRoles := GetUserRolesFromContext(r.Context())
			if len(userRoles) == 0 {
				utils.RespondError(w, r, http.StatusForbidden, "Forbidden", "Access denied")
				return
			}

			// Check if user has all required roles
			userRoleSet := make(map[string]bool)
			for _, role := range userRoles {
				userRoleSet[role] = true
			}

			for _, requiredRole := range roles {
				if !userRoleSet[requiredRole] {
					utils.RespondError(w, r, http.StatusForbidden, "Forbidden", "Insufficient permissions")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserIDFromContext retrieves the user ID from the context
func GetUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(userIDKey).(string); ok {
		return userID
	}
	return ""
}

// SetUserRolesInContext sets the user roles in the context (for testing purposes)
func SetUserRolesInContext(ctx context.Context, roles []string) context.Context {
	return context.WithValue(ctx, userRolesKey, roles)
}

// GetUserRolesFromContext retrieves the user roles from the context
func GetUserRolesFromContext(ctx context.Context) []string {
	if roles, ok := ctx.Value(userRolesKey).([]string); ok {
		return roles
	}
	return nil
}

// GetClaimsFromContext retrieves the full token claims from the context
func GetClaimsFromContext(ctx context.Context) *TokenClaims {
	if claims, ok := ctx.Value(claimsKey).(*TokenClaims); ok {
		return claims
	}
	return nil
}

// IsAuthenticated checks if the current request has a valid authenticated user
func IsAuthenticated(ctx context.Context) bool {
	return GetUserIDFromContext(ctx) != ""
}

// HasRole checks if the current user has a specific role
func HasRole(ctx context.Context, role string) bool {
	roles := GetUserRolesFromContext(ctx)
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the current user has any of the specified roles
func HasAnyRole(ctx context.Context, roles ...string) bool {
	userRoles := GetUserRolesFromContext(ctx)
	for _, role := range roles {
		for _, userRole := range userRoles {
			if userRole == role {
				return true
			}
		}
	}
	return false
}
