package departments

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"kc-api/internal/utils"
)

// Handler handles HTTP requests for department operations
type Handler struct {
	service Service
}

// NewHandler creates a new department handler with the given service
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers department routes on the given router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/departments", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Post("/search", h.Search)
		r.Post("/batch", h.BatchCreate)
		r.Put("/batch", h.BatchUpdate)
		r.Delete("/batch", h.BatchDelete)
		r.Get("/tree", h.GetTree)
		r.Get("/{publicId}", h.GetByPublicID)
		r.Put("/{publicId}", h.Update)
		r.Delete("/{publicId}", h.Delete)
		r.Get("/{publicId}/tree", h.GetSubtree)
		r.Put("/{publicId}/move", h.Move)
		r.Get("/{publicId}/children", h.GetChildren)
	})
}

// List godoc
// @Summary      List departments
// @Description  Retrieves a paginated list of departments
// @Tags         departments
// @Accept       json
// @Produce      json
// @Param        page   query     int  false  "Page number"     default(1)
// @Param        limit  query     int  false  "Items per page"  default(10)
// @Success      200    {object}  DepartmentListResponseWrapper
// @Failure      500    {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /departments [get]
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
		utils.RespondInternalError(w, r, err, "Failed to retrieve departments")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// Create godoc
// @Summary      Create a new department
// @Description  Creates a new department with JSONB name and description
// @Tags         departments
// @Accept       json
// @Produce      json
// @Param        request  body      CreateDepartmentRequest  true  "Department data"
// @Success      201      {object}  DepartmentResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /departments [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.Create(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidName):
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Name must have at least one locale value")
		case errors.Is(err, ErrParentDepartmentNotFound):
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Parent department not found")
		default:
			utils.RespondInternalError(w, r, err, "Failed to create department")
		}
		return
	}

	utils.RespondJSON(w, http.StatusCreated, result)
}

// GetByPublicID godoc
// @Summary      Get department by public ID
// @Description  Retrieves a department by its public ID
// @Tags         departments
// @Accept       json
// @Produce      json
// @Param        publicId  path      string  true  "Department Public ID"
// @Success      200       {object}  DepartmentResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /departments/{publicId} [get]
func (h *Handler) GetByPublicID(w http.ResponseWriter, r *http.Request) {
	publicID := chi.URLParam(r, "publicId")
	if publicID == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Public ID is required")
		return
	}

	result, err := h.service.GetByPublicID(r.Context(), publicID)
	if err != nil {
		if errors.Is(err, ErrDepartmentNotFound) {
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Department not found")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to retrieve department")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// Update godoc
// @Summary      Update department
// @Description  Updates an existing department
// @Tags         departments
// @Accept       json
// @Produce      json
// @Param        publicId  path      string                   true  "Department Public ID"
// @Param        request   body      UpdateDepartmentRequest  true  "Department data to update"
// @Success      200       {object}  DepartmentResponse
// @Failure      400       {object}  ErrorResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /departments/{publicId} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	publicID := chi.URLParam(r, "publicId")
	if publicID == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Public ID is required")
		return
	}

	var req UpdateDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.Update(r.Context(), publicID, &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrDepartmentNotFound):
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Department not found")
		case errors.Is(err, ErrInvalidName):
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Name must have at least one locale value")
		case errors.Is(err, ErrCircularReference):
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Circular reference detected")
		case errors.Is(err, ErrParentDepartmentNotFound):
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Parent department not found")
		default:
			utils.RespondInternalError(w, r, err, "Failed to update department")
		}
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// Delete godoc
// @Summary      Delete department
// @Description  Soft-deletes a department by public ID
// @Tags         departments
// @Accept       json
// @Produce      json
// @Param        publicId  path      string  true  "Department Public ID"
// @Success      200       {object}  SuccessResponse
// @Failure      400       {object}  ErrorResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /departments/{publicId} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	publicID := chi.URLParam(r, "publicId")
	if publicID == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Public ID is required")
		return
	}

	err := h.service.Delete(r.Context(), publicID)
	if err != nil {
		switch {
		case errors.Is(err, ErrDepartmentNotFound):
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Department not found")
		case errors.Is(err, ErrHasChildren):
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Department has children and cannot be deleted")
		default:
			utils.RespondInternalError(w, r, err, "Failed to delete department")
		}
		return
	}

	utils.RespondJSON(w, http.StatusOK, SuccessResponse{Message: "Department deleted successfully"})
}

// Search godoc
// @Summary      Search departments
// @Description  Searches for departments based on name or description
// @Tags         departments
// @Accept       json
// @Produce      json
// @Param        page     query     int                      false  "Page number"     default(1)
// @Param        limit    query     int                      false  "Items per page"  default(10)
// @Param        request  body      SearchDepartmentRequest  true   "Search criteria"
// @Success      200      {object}  DepartmentListResponseWrapper
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /departments/search [post]
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	var req SearchDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.Search(r.Context(), &req, page, limit)
	if err != nil {
		utils.RespondInternalError(w, r, err, "Failed to search departments")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// BatchCreate godoc
// @Summary      Batch create departments
// @Description  Creates multiple departments in a single transaction
// @Tags         departments
// @Accept       json
// @Produce      json
// @Param        request  body      BatchCreateDepartmentRequest  true  "Batch create request"
// @Success      201      {object}  BatchOperationResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /departments/batch [post]
func (h *Handler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	var req BatchCreateDepartmentRequest
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
		utils.RespondInternalError(w, r, err, "Failed to batch create departments")
		return
	}

	utils.RespondJSON(w, http.StatusCreated, result)
}

// BatchUpdate godoc
// @Summary      Batch update departments
// @Description  Updates multiple departments in a single transaction
// @Tags         departments
// @Accept       json
// @Produce      json
// @Param        request  body      BatchUpdateDepartmentRequest  true  "Batch update request"
// @Success      200      {object}  BatchOperationResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /departments/batch [put]
func (h *Handler) BatchUpdate(w http.ResponseWriter, r *http.Request) {
	var req BatchUpdateDepartmentRequest
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
		utils.RespondInternalError(w, r, err, "Failed to batch update departments")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// BatchDelete godoc
// @Summary      Batch delete departments
// @Description  Soft-deletes multiple departments in a single transaction
// @Tags         departments
// @Accept       json
// @Produce      json
// @Param        request  body      BatchDeleteDepartmentRequest  true  "Batch delete request"
// @Success      200      {object}  BatchOperationResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /departments/batch [delete]
func (h *Handler) BatchDelete(w http.ResponseWriter, r *http.Request) {
	var req BatchDeleteDepartmentRequest
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
		utils.RespondInternalError(w, r, err, "Failed to batch delete departments")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// GetTree godoc
// @Summary      Get department tree
// @Description  Retrieves the full department tree structure
// @Tags         departments
// @Accept       json
// @Produce      json
// @Success      200  {array}   DepartmentTreeResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /departments/tree [get]
func (h *Handler) GetTree(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.GetTree(r.Context())
	if err != nil {
		utils.RespondInternalError(w, r, err, "Failed to get department tree")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// GetSubtree godoc
// @Summary      Get department subtree
// @Description  Retrieves a subtree starting from a specific department
// @Tags         departments
// @Accept       json
// @Produce      json
// @Param        publicId  path      string  true  "Department Public ID"
// @Success      200       {object}  DepartmentTreeResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /departments/{publicId}/tree [get]
func (h *Handler) GetSubtree(w http.ResponseWriter, r *http.Request) {
	publicID := chi.URLParam(r, "publicId")
	if publicID == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Public ID is required")
		return
	}

	result, err := h.service.GetSubtree(r.Context(), publicID)
	if err != nil {
		if errors.Is(err, ErrDepartmentNotFound) {
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Department not found")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to get subtree")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// Move godoc
// @Summary      Move department
// @Description  Moves a department to a new parent
// @Tags         departments
// @Accept       json
// @Produce      json
// @Param        publicId  path      string                 true  "Department Public ID"
// @Param        request   body      MoveDepartmentRequest  true  "Move request"
// @Success      200       {object}  SuccessResponse
// @Failure      400       {object}  ErrorResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /departments/{publicId}/move [put]
func (h *Handler) Move(w http.ResponseWriter, r *http.Request) {
	publicID := chi.URLParam(r, "publicId")
	if publicID == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Public ID is required")
		return
	}

	var req MoveDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	err := h.service.Move(r.Context(), publicID, &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrDepartmentNotFound):
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Department not found")
		case errors.Is(err, ErrCircularReference):
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Circular reference detected")
		case errors.Is(err, ErrParentDepartmentNotFound):
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Parent department not found")
		default:
			utils.RespondInternalError(w, r, err, "Failed to move department")
		}
		return
	}

	utils.RespondJSON(w, http.StatusOK, SuccessResponse{Message: "Department moved successfully"})
}

// GetChildren godoc
// @Summary      Get department children
// @Description  Retrieves direct children of a department
// @Tags         departments
// @Accept       json
// @Produce      json
// @Param        publicId  path      string  true  "Department Public ID"
// @Success      200       {array}   DepartmentResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /departments/{publicId}/children [get]
func (h *Handler) GetChildren(w http.ResponseWriter, r *http.Request) {
	publicID := chi.URLParam(r, "publicId")
	if publicID == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Public ID is required")
		return
	}

	result, err := h.service.GetChildren(r.Context(), publicID)
	if err != nil {
		if errors.Is(err, ErrDepartmentNotFound) {
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Department not found")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to get children")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}
