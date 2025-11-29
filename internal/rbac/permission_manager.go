package rbac

import (
	"context"
	"sync"
)

// PermissionManager holds permission rules in memory to avoid DB lookups on every request.
// It uses a map structure: map[method]map[path_pattern][]allowed_roles
type PermissionManager struct {
	mu          sync.RWMutex
	permissions map[string]map[string][]string // method -> path_pattern -> roles
	repository  Repository
}

// NewPermissionManager creates a new PermissionManager with the given repository
func NewPermissionManager(repo Repository) *PermissionManager {
	return &PermissionManager{
		permissions: make(map[string]map[string][]string),
		repository:  repo,
	}
}

// LoadPermissions fetches permission data from the database and replaces the in-memory map.
// This method is safe for concurrent access (Hot Reload).
func (pm *PermissionManager) LoadPermissions(ctx context.Context) error {
	// Fetch all permissions from database
	dbPermissions, err := pm.repository.GetAllPermissions(ctx)
	if err != nil {
		return err
	}

	// Build new permissions map
	newPermissions := make(map[string]map[string][]string)

	for _, perm := range dbPermissions {
		method := perm.Method
		pathPattern := perm.PathPattern
		roles := perm.RequiredRoles

		if _, exists := newPermissions[method]; !exists {
			newPermissions[method] = make(map[string][]string)
		}

		newPermissions[method][pathPattern] = roles
	}

	// Atomically replace the permissions map
	pm.mu.Lock()
	pm.permissions = newPermissions
	pm.mu.Unlock()

	return nil
}

// GetRequiredRoles retrieves the required roles for a given HTTP method and path pattern.
// Returns the roles and a boolean indicating whether the permission rule was found.
func (pm *PermissionManager) GetRequiredRoles(method, path string) ([]string, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// First, try exact method match
	if pathMap, methodExists := pm.permissions[method]; methodExists {
		if roles, pathExists := pathMap[path]; pathExists {
			return roles, true
		}
	}

	// Second, try wildcard method "*"
	if pathMap, wildcardExists := pm.permissions["*"]; wildcardExists {
		if roles, pathExists := pathMap[path]; pathExists {
			return roles, true
		}
	}

	return nil, false
}
