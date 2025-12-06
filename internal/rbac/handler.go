package rbac

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"kc-api/internal/utils"
)

// Handler handles HTTP requests for RBAC operations
type Handler struct {
	permissionManager *PermissionManager
}

// NewHandler creates a new RBAC handler
func NewHandler(pm *PermissionManager) *Handler {
	return &Handler{permissionManager: pm}
}

// RegisterRoutes registers RBAC admin routes on the given router
// These routes should only be accessible by sysadmin users
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/admin/refresh-permissions", h.RefreshPermissions)
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message" example:"Permissions refreshed successfully"`
}

// RefreshPermissions godoc
// @Summary      Refresh API permissions cache
// @Description  Reloads API permissions from the database into the in-memory cache. This endpoint allows hot-reloading of permission rules without restarting the server.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Success      200  {object}  SuccessResponse
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      403  {object}  ErrorResponse  "Forbidden - requires sysadmin role"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Security     BearerAuth
// @Router       /admin/refresh-permissions [post]
func (h *Handler) RefreshPermissions(w http.ResponseWriter, r *http.Request) {
	if err := h.permissionManager.LoadPermissions(r.Context()); err != nil {
		utils.RespondInternalError(w, r, err, "Failed to refresh permissions")
		return
	}

	utils.RespondJSON(w, http.StatusOK, SuccessResponse{Message: "Permissions refreshed successfully"})
}
