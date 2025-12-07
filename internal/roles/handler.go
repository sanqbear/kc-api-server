package roles

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"kc-api/internal/utils"
)

// Handler handles HTTP requests for role operations
type Handler struct {
	service Service
}

// NewHandler creates a new role handler with the given service
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers role routes on the given router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/roles", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Post("/search", h.Search)
		r.Post("/batch", h.BatchCreate)
		r.Put("/batch", h.BatchUpdate)
		r.Delete("/batch", h.BatchDelete)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Get("/{id}/users", h.GetUsersWithRole)
	})

	r.Route("/users/{userId}/roles", func(r chi.Router) {
		r.Get("/", h.GetUserRoles)
		r.Put("/", h.AssignUserRoles)
		r.Delete("/{roleId}", h.RemoveUserRole)
	})
}

// List godoc
// @Summary      List roles
// @Description  Retrieves a paginated list of roles
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        page   query     int  false  "Page number"     default(1)
// @Param        limit  query     int  false  "Items per page"  default(10)
// @Success      200    {object}  RoleListResponseWrapper
// @Failure      500    {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /roles [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	result, err := h.service.List(r.Context(), page, limit)
	if err != nil {
		utils.RespondInternalError(w, r, err, "Failed to retrieve roles")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// Create godoc
// @Summary      Create a new role
// @Description  Creates a new role with unique name
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        request  body      CreateRoleRequest  true  "Role data"
// @Success      201      {object}  RoleResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      409      {object}  ErrorResponse  "Role name already exists"
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /roles [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.Create(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRoleName):
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Role name is required")
		case errors.Is(err, ErrRoleNameExists):
			utils.RespondError(w, r, http.StatusConflict, "Conflict", "Role name already exists")
		default:
			utils.RespondInternalError(w, r, err, "Failed to create role")
		}
		return
	}

	utils.RespondJSON(w, http.StatusCreated, result)
}

// GetByID godoc
// @Summary      Get role by ID
// @Description  Retrieves a role by its ID
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Role ID"
// @Success      200  {object}  RoleResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /roles/{id} [get]
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid ID")
		return
	}

	result, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Role not found")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to retrieve role")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// Update godoc
// @Summary      Update role
// @Description  Updates an existing role
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        id       path      int                true  "Role ID"
// @Param        request  body      UpdateRoleRequest  true  "Role data to update"
// @Success      200      {object}  RoleResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Failure      409      {object}  ErrorResponse  "Role name already exists"
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /roles/{id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid ID")
		return
	}

	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrRoleNotFound):
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Role not found")
		case errors.Is(err, ErrRoleNameExists):
			utils.RespondError(w, r, http.StatusConflict, "Conflict", "Role name already exists")
		default:
			utils.RespondInternalError(w, r, err, "Failed to update role")
		}
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// Delete godoc
// @Summary      Delete role
// @Description  Deletes a role by ID
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Role ID"
// @Success      200  {object}  SuccessResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /roles/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid ID")
		return
	}

	err = h.service.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Role not found")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to delete role")
		return
	}

	utils.RespondJSON(w, http.StatusOK, SuccessResponse{Message: "Role deleted successfully"})
}

// Search godoc
// @Summary      Search roles
// @Description  Searches for roles based on name or description
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        page     query     int                false  "Page number"     default(1)
// @Param        limit    query     int                false  "Items per page"  default(10)
// @Param        request  body      SearchRoleRequest  true   "Search criteria"
// @Success      200      {object}  RoleListResponseWrapper
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /roles/search [post]
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	var req SearchRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.Search(r.Context(), &req, page, limit)
	if err != nil {
		utils.RespondInternalError(w, r, err, "Failed to search roles")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// BatchCreate godoc
// @Summary      Batch create roles
// @Description  Creates multiple roles in a single transaction
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        request  body      BatchCreateRoleRequest  true  "Batch create request"
// @Success      201      {object}  BatchOperationResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /roles/batch [post]
func (h *Handler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	var req BatchCreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.BatchCreate(r.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrEmptyBatchRequest) {
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Batch request cannot be empty")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to batch create roles")
		return
	}

	utils.RespondJSON(w, http.StatusCreated, result)
}

// BatchUpdate godoc
// @Summary      Batch update roles
// @Description  Updates multiple roles in a single transaction
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        request  body      BatchUpdateRoleRequest  true  "Batch update request"
// @Success      200      {object}  BatchOperationResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /roles/batch [put]
func (h *Handler) BatchUpdate(w http.ResponseWriter, r *http.Request) {
	var req BatchUpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.BatchUpdate(r.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrEmptyBatchRequest) {
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Batch request cannot be empty")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to batch update roles")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// BatchDelete godoc
// @Summary      Batch delete roles
// @Description  Deletes multiple roles in a single transaction
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        request  body      BatchDeleteRoleRequest  true  "Batch delete request"
// @Success      200      {object}  BatchOperationResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /roles/batch [delete]
func (h *Handler) BatchDelete(w http.ResponseWriter, r *http.Request) {
	var req BatchDeleteRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.BatchDelete(r.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrEmptyBatchRequest) {
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Batch request cannot be empty")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to batch delete roles")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// GetUserRoles godoc
// @Summary      Get user roles
// @Description  Retrieves all roles assigned to a user
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        userId  path      int  true  "User ID"
// @Success      200     {array}   UserRoleResponse
// @Failure      500     {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /users/{userId}/roles [get]
func (h *Handler) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userId")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid user ID")
		return
	}

	result, err := h.service.GetUserRoles(r.Context(), userID)
	if err != nil {
		utils.RespondInternalError(w, r, err, "Failed to get user roles")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// AssignUserRoles godoc
// @Summary      Assign roles to user
// @Description  Assigns roles to a user (replaces existing roles)
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        userId   path      int                     true  "User ID"
// @Param        request  body      AssignUserRolesRequest  true  "Role IDs to assign"
// @Success      200      {object}  SuccessResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /users/{userId}/roles [put]
func (h *Handler) AssignUserRoles(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userId")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid user ID")
		return
	}

	var req AssignUserRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	err = h.service.AssignUserRoles(r.Context(), userID, &req)
	if err != nil {
		utils.RespondInternalError(w, r, err, "Failed to assign user roles")
		return
	}

	utils.RespondJSON(w, http.StatusOK, SuccessResponse{Message: "Roles assigned successfully"})
}

// RemoveUserRole godoc
// @Summary      Remove role from user
// @Description  Removes a specific role from a user
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        userId  path      int  true  "User ID"
// @Param        roleId  path      int  true  "Role ID"
// @Success      200     {object}  SuccessResponse
// @Failure      404     {object}  ErrorResponse
// @Failure      500     {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /users/{userId}/roles/{roleId} [delete]
func (h *Handler) RemoveUserRole(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userId")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid user ID")
		return
	}

	roleIDStr := chi.URLParam(r, "roleId")
	roleID, err := strconv.Atoi(roleIDStr)
	if err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid role ID")
		return
	}

	err = h.service.RemoveUserRole(r.Context(), userID, roleID)
	if err != nil {
		if errors.Is(err, ErrUserRoleNotFound) {
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "User role not found")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to remove user role")
		return
	}

	utils.RespondJSON(w, http.StatusOK, SuccessResponse{Message: "Role removed successfully"})
}

// GetUsersWithRole godoc
// @Summary      Get users with role
// @Description  Retrieves all user IDs that have a specific role
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Role ID"
// @Success      200  {array}   int
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /roles/{id}/users [get]
func (h *Handler) GetUsersWithRole(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid ID")
		return
	}

	result, err := h.service.GetUsersWithRole(r.Context(), id)
	if err != nil {
		utils.RespondInternalError(w, r, err, "Failed to get users with role")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}
