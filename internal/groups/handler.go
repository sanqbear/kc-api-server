package groups

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"kc-api/internal/utils"
)

// Handler handles HTTP requests for group operations
type Handler struct {
	service Service
}

// NewHandler creates a new group handler with the given service
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers group routes on the given router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/groups", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Post("/search", h.Search)
		r.Post("/batch", h.BatchCreate)
		r.Put("/batch", h.BatchUpdate)
		r.Delete("/batch", h.BatchDelete)
		r.Get("/{publicId}", h.GetByPublicID)
		r.Put("/{publicId}", h.Update)
		r.Delete("/{publicId}", h.Delete)
		r.Get("/{publicId}/users", h.GetGroupUsers)
		r.Put("/{publicId}/users", h.AssignUsersToGroup)
		r.Delete("/{publicId}/users/{userId}", h.RemoveUserFromGroup)
		r.Get("/{publicId}/roles", h.GetGroupRoles)
		r.Put("/{publicId}/roles", h.AssignRolesToGroup)
		r.Delete("/{publicId}/roles/{roleId}", h.RemoveRoleFromGroup)
	})

	r.Route("/users/{userId}/groups", func(r chi.Router) {
		r.Get("/", h.GetUserGroups)
	})
}

// List godoc
// @Summary      List groups
// @Description  Retrieves a paginated list of groups
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        page   query     int  false  "Page number"     default(1)
// @Param        limit  query     int  false  "Items per page"  default(10)
// @Success      200    {object}  GroupListResponseWrapper
// @Failure      500    {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /groups [get]
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
		utils.RespondInternalError(w, r, err, "Failed to retrieve groups")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// Create godoc
// @Summary      Create a new group
// @Description  Creates a new group with JSONB name and description
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        request  body      CreateGroupRequest  true  "Group data"
// @Success      201      {object}  GroupResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /groups [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.Create(r.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrInvalidName) {
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Name must have at least one locale value")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to create group")
		return
	}

	utils.RespondJSON(w, http.StatusCreated, result)
}

// GetByPublicID godoc
// @Summary      Get group by public ID
// @Description  Retrieves a group by its public ID
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        publicId  path      string  true  "Group Public ID"
// @Success      200       {object}  GroupResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /groups/{publicId} [get]
func (h *Handler) GetByPublicID(w http.ResponseWriter, r *http.Request) {
	publicID := chi.URLParam(r, "publicId")
	if publicID == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Public ID is required")
		return
	}

	result, err := h.service.GetByPublicID(r.Context(), publicID)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Group not found")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to retrieve group")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// Update godoc
// @Summary      Update group
// @Description  Updates an existing group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        publicId  path      string              true  "Group Public ID"
// @Param        request   body      UpdateGroupRequest  true  "Group data to update"
// @Success      200       {object}  GroupResponse
// @Failure      400       {object}  ErrorResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /groups/{publicId} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	publicID := chi.URLParam(r, "publicId")
	if publicID == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Public ID is required")
		return
	}

	var req UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.Update(r.Context(), publicID, &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrGroupNotFound):
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Group not found")
		case errors.Is(err, ErrInvalidName):
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Name must have at least one locale value")
		default:
			utils.RespondInternalError(w, r, err, "Failed to update group")
		}
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// Delete godoc
// @Summary      Delete group
// @Description  Deletes a group by public ID
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        publicId  path      string  true  "Group Public ID"
// @Success      200       {object}  SuccessResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /groups/{publicId} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	publicID := chi.URLParam(r, "publicId")
	if publicID == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Public ID is required")
		return
	}

	err := h.service.Delete(r.Context(), publicID)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Group not found")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to delete group")
		return
	}

	utils.RespondJSON(w, http.StatusOK, SuccessResponse{Message: "Group deleted successfully"})
}

// Search godoc
// @Summary      Search groups
// @Description  Searches for groups based on name or description
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        page     query     int                 false  "Page number"     default(1)
// @Param        limit    query     int                 false  "Items per page"  default(10)
// @Param        request  body      SearchGroupRequest  true   "Search criteria"
// @Success      200      {object}  GroupListResponseWrapper
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /groups/search [post]
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	var req SearchGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.Search(r.Context(), &req, page, limit)
	if err != nil {
		utils.RespondInternalError(w, r, err, "Failed to search groups")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// BatchCreate godoc
// @Summary      Batch create groups
// @Description  Creates multiple groups in a single transaction
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        request  body      BatchCreateGroupRequest  true  "Batch create request"
// @Success      201      {object}  BatchOperationResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /groups/batch [post]
func (h *Handler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	var req BatchCreateGroupRequest
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
		utils.RespondInternalError(w, r, err, "Failed to batch create groups")
		return
	}

	utils.RespondJSON(w, http.StatusCreated, result)
}

// BatchUpdate godoc
// @Summary      Batch update groups
// @Description  Updates multiple groups in a single transaction
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        request  body      BatchUpdateGroupRequest  true  "Batch update request"
// @Success      200      {object}  BatchOperationResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /groups/batch [put]
func (h *Handler) BatchUpdate(w http.ResponseWriter, r *http.Request) {
	var req BatchUpdateGroupRequest
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
		utils.RespondInternalError(w, r, err, "Failed to batch update groups")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// BatchDelete godoc
// @Summary      Batch delete groups
// @Description  Deletes multiple groups in a single transaction
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        request  body      BatchDeleteGroupRequest  true  "Batch delete request"
// @Success      200      {object}  BatchOperationResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /groups/batch [delete]
func (h *Handler) BatchDelete(w http.ResponseWriter, r *http.Request) {
	var req BatchDeleteGroupRequest
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
		utils.RespondInternalError(w, r, err, "Failed to batch delete groups")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// GetGroupUsers godoc
// @Summary      Get group users
// @Description  Retrieves all users in a group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        publicId  path      string  true  "Group Public ID"
// @Success      200       {array}   int
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /groups/{publicId}/users [get]
func (h *Handler) GetGroupUsers(w http.ResponseWriter, r *http.Request) {
	publicID := chi.URLParam(r, "publicId")
	if publicID == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Public ID is required")
		return
	}

	result, err := h.service.GetGroupUsers(r.Context(), publicID)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Group not found")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to get group users")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// AssignUsersToGroup godoc
// @Summary      Assign users to group
// @Description  Assigns users to a group (replaces existing users)
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        publicId  path      string              true  "Group Public ID"
// @Param        request   body      AssignUsersRequest  true  "User IDs to assign"
// @Success      200       {object}  SuccessResponse
// @Failure      400       {object}  ErrorResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /groups/{publicId}/users [put]
func (h *Handler) AssignUsersToGroup(w http.ResponseWriter, r *http.Request) {
	publicID := chi.URLParam(r, "publicId")
	if publicID == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Public ID is required")
		return
	}

	var req AssignUsersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	err := h.service.AssignUsersToGroup(r.Context(), publicID, &req)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Group not found")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to assign users to group")
		return
	}

	utils.RespondJSON(w, http.StatusOK, SuccessResponse{Message: "Users assigned successfully"})
}

// RemoveUserFromGroup godoc
// @Summary      Remove user from group
// @Description  Removes a user from a group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        publicId  path      string  true  "Group Public ID"
// @Param        userId    path      int     true  "User ID"
// @Success      200       {object}  SuccessResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /groups/{publicId}/users/{userId} [delete]
func (h *Handler) RemoveUserFromGroup(w http.ResponseWriter, r *http.Request) {
	publicID := chi.URLParam(r, "publicId")
	if publicID == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Public ID is required")
		return
	}

	userIDStr := chi.URLParam(r, "userId")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid user ID")
		return
	}

	err = h.service.RemoveUserFromGroup(r.Context(), publicID, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrGroupNotFound):
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Group not found")
		case errors.Is(err, ErrGroupUserNotFound):
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "User not found in group")
		default:
			utils.RespondInternalError(w, r, err, "Failed to remove user from group")
		}
		return
	}

	utils.RespondJSON(w, http.StatusOK, SuccessResponse{Message: "User removed successfully"})
}

// GetUserGroups godoc
// @Summary      Get user groups
// @Description  Retrieves all groups a user belongs to
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        userId  path      int  true  "User ID"
// @Success      200     {array}   GroupResponse
// @Failure      500     {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /users/{userId}/groups [get]
func (h *Handler) GetUserGroups(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "userId")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid user ID")
		return
	}

	result, err := h.service.GetUserGroups(r.Context(), userID)
	if err != nil {
		utils.RespondInternalError(w, r, err, "Failed to get user groups")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// GetGroupRoles godoc
// @Summary      Get group roles
// @Description  Retrieves all roles assigned to a group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        publicId  path      string  true  "Group Public ID"
// @Success      200       {array}   GroupRoleResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /groups/{publicId}/roles [get]
func (h *Handler) GetGroupRoles(w http.ResponseWriter, r *http.Request) {
	publicID := chi.URLParam(r, "publicId")
	if publicID == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Public ID is required")
		return
	}

	result, err := h.service.GetGroupRoles(r.Context(), publicID)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Group not found")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to get group roles")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// AssignRolesToGroup godoc
// @Summary      Assign roles to group
// @Description  Assigns roles to a group (replaces existing roles)
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        publicId  path      string              true  "Group Public ID"
// @Param        request   body      AssignRolesRequest  true  "Role IDs to assign"
// @Success      200       {object}  SuccessResponse
// @Failure      400       {object}  ErrorResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /groups/{publicId}/roles [put]
func (h *Handler) AssignRolesToGroup(w http.ResponseWriter, r *http.Request) {
	publicID := chi.URLParam(r, "publicId")
	if publicID == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Public ID is required")
		return
	}

	var req AssignRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	err := h.service.AssignRolesToGroup(r.Context(), publicID, &req)
	if err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Group not found")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to assign roles to group")
		return
	}

	utils.RespondJSON(w, http.StatusOK, SuccessResponse{Message: "Roles assigned successfully"})
}

// RemoveRoleFromGroup godoc
// @Summary      Remove role from group
// @Description  Removes a role from a group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Param        publicId  path      string  true  "Group Public ID"
// @Param        roleId    path      int     true  "Role ID"
// @Success      200       {object}  SuccessResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /groups/{publicId}/roles/{roleId} [delete]
func (h *Handler) RemoveRoleFromGroup(w http.ResponseWriter, r *http.Request) {
	publicID := chi.URLParam(r, "publicId")
	if publicID == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Public ID is required")
		return
	}

	roleIDStr := chi.URLParam(r, "roleId")
	roleID, err := strconv.Atoi(roleIDStr)
	if err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid role ID")
		return
	}

	err = h.service.RemoveRoleFromGroup(r.Context(), publicID, roleID)
	if err != nil {
		switch {
		case errors.Is(err, ErrGroupNotFound):
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Group not found")
		case errors.Is(err, ErrGroupRoleNotFound):
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Role not found in group")
		default:
			utils.RespondInternalError(w, r, err, "Failed to remove role from group")
		}
		return
	}

	utils.RespondJSON(w, http.StatusOK, SuccessResponse{Message: "Role removed successfully"})
}
