package commoncodes

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"kc-api/internal/utils"
)

// Handler handles HTTP requests for common code operations
type Handler struct {
	service Service
}

// NewHandler creates a new common code handler with the given service
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers common code routes on the given router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/common-codes", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Post("/search", h.Search)
		r.Post("/batch", h.BatchCreate)
		r.Put("/batch", h.BatchUpdate)
		r.Delete("/batch", h.BatchDelete)
		r.Get("/categories", h.ListCategories)
		r.Get("/categories/{category}", h.GetByCategory)
		r.Put("/categories/{category}/reorder", h.Reorder)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})
}

// List godoc
// @Summary      List common codes
// @Description  Retrieves a paginated list of common codes ordered by category and sort_order
// @Tags         common-codes
// @Accept       json
// @Produce      json
// @Param        page   query     int  false  "Page number"     default(1)
// @Param        limit  query     int  false  "Items per page"  default(10)
// @Success      200    {object}  CommonCodeListResponseWrapper
// @Failure      500    {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /common-codes [get]
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
		utils.RespondInternalError(w, r, err, "Failed to retrieve common codes")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// Create godoc
// @Summary      Create a new common code
// @Description  Creates a new common code. Category, code, and name are required. Category + code must be unique.
// @Tags         common-codes
// @Accept       json
// @Produce      json
// @Param        request  body      CreateCommonCodeRequest  true  "Common code data"
// @Success      201      {object}  CommonCodeResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      409      {object}  ErrorResponse  "Category and code combination already exists"
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /common-codes [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCommonCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.Create(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCategory):
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Category is required")
		case errors.Is(err, ErrInvalidCode):
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Code is required")
		case errors.Is(err, ErrInvalidName):
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Name must have at least one locale value")
		case errors.Is(err, ErrCategoryCodeExists):
			utils.RespondError(w, r, http.StatusConflict, "Conflict", "Category and code combination already exists")
		default:
			utils.RespondInternalError(w, r, err, "Failed to create common code")
		}
		return
	}

	utils.RespondJSON(w, http.StatusCreated, result)
}

// GetByID godoc
// @Summary      Get common code by ID
// @Description  Retrieves a common code by its ID
// @Tags         common-codes
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Common Code ID"
// @Success      200  {object}  CommonCodeResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /common-codes/{id} [get]
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid ID")
		return
	}

	result, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrCommonCodeNotFound) {
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Common code not found")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to retrieve common code")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// Update godoc
// @Summary      Update common code
// @Description  Updates an existing common code. Category + code must be unique.
// @Tags         common-codes
// @Accept       json
// @Produce      json
// @Param        id       path      int                      true  "Common Code ID"
// @Param        request  body      UpdateCommonCodeRequest  true  "Common code data to update"
// @Success      200      {object}  CommonCodeResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Failure      409      {object}  ErrorResponse  "Category and code combination already exists"
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /common-codes/{id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid ID")
		return
	}

	var req UpdateCommonCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrCommonCodeNotFound):
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Common code not found")
		case errors.Is(err, ErrInvalidName):
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Name must have at least one locale value")
		case errors.Is(err, ErrCategoryCodeExists):
			utils.RespondError(w, r, http.StatusConflict, "Conflict", "Category and code combination already exists")
		default:
			utils.RespondInternalError(w, r, err, "Failed to update common code")
		}
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// Delete godoc
// @Summary      Delete common code
// @Description  Deletes a common code by ID
// @Tags         common-codes
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Common Code ID"
// @Success      200  {object}  SuccessResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /common-codes/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid ID")
		return
	}

	err = h.service.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrCommonCodeNotFound) {
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Common code not found")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to delete common code")
		return
	}

	utils.RespondJSON(w, http.StatusOK, SuccessResponse{Message: "Common code deleted successfully"})
}

// Search godoc
// @Summary      Search common codes
// @Description  Searches for common codes based on category, code, name, or description
// @Tags         common-codes
// @Accept       json
// @Produce      json
// @Param        page     query     int                      false  "Page number"     default(1)
// @Param        limit    query     int                      false  "Items per page"  default(10)
// @Param        request  body      SearchCommonCodeRequest  true   "Search criteria"
// @Success      200      {object}  CommonCodeListResponseWrapper
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /common-codes/search [post]
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	var req SearchCommonCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.Search(r.Context(), &req, page, limit)
	if err != nil {
		utils.RespondInternalError(w, r, err, "Failed to search common codes")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// BatchCreate godoc
// @Summary      Batch create common codes
// @Description  Creates multiple common codes in a single transaction
// @Tags         common-codes
// @Accept       json
// @Produce      json
// @Param        request  body      BatchCreateRequest  true  "Batch create request"
// @Success      201      {object}  BatchOperationResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /common-codes/batch [post]
func (h *Handler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	var req BatchCreateRequest
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
		utils.RespondInternalError(w, r, err, "Failed to batch create common codes")
		return
	}

	utils.RespondJSON(w, http.StatusCreated, result)
}

// BatchUpdate godoc
// @Summary      Batch update common codes
// @Description  Updates multiple common codes in a single transaction
// @Tags         common-codes
// @Accept       json
// @Produce      json
// @Param        request  body      BatchUpdateRequest  true  "Batch update request"
// @Success      200      {object}  BatchOperationResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /common-codes/batch [put]
func (h *Handler) BatchUpdate(w http.ResponseWriter, r *http.Request) {
	var req BatchUpdateRequest
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
		utils.RespondInternalError(w, r, err, "Failed to batch update common codes")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// BatchDelete godoc
// @Summary      Batch delete common codes
// @Description  Deletes multiple common codes in a single transaction
// @Tags         common-codes
// @Accept       json
// @Produce      json
// @Param        request  body      BatchDeleteRequest  true  "Batch delete request"
// @Success      200      {object}  BatchOperationResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /common-codes/batch [delete]
func (h *Handler) BatchDelete(w http.ResponseWriter, r *http.Request) {
	var req BatchDeleteRequest
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
		utils.RespondInternalError(w, r, err, "Failed to batch delete common codes")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// ListCategories godoc
// @Summary      List all categories
// @Description  Retrieves all unique categories from common codes
// @Tags         common-codes
// @Accept       json
// @Produce      json
// @Success      200  {array}   CategoryResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /common-codes/categories [get]
func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.ListCategories(r.Context())
	if err != nil {
		utils.RespondInternalError(w, r, err, "Failed to list categories")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// GetByCategory godoc
// @Summary      Get codes by category
// @Description  Retrieves all common codes in a specific category, ordered by sort_order
// @Tags         common-codes
// @Accept       json
// @Produce      json
// @Param        category  path      string  true  "Category name"
// @Success      200       {array}   CommonCodeResponse
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /common-codes/categories/{category} [get]
func (h *Handler) GetByCategory(w http.ResponseWriter, r *http.Request) {
	category := chi.URLParam(r, "category")
	if category == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Category is required")
		return
	}

	result, err := h.service.GetByCategory(r.Context(), category)
	if err != nil {
		utils.RespondInternalError(w, r, err, "Failed to get codes by category")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// Reorder godoc
// @Summary      Reorder codes in a category
// @Description  Updates the sort_order of multiple codes in a category
// @Tags         common-codes
// @Accept       json
// @Produce      json
// @Param        category  path      string          true  "Category name"
// @Param        request   body      ReorderRequest  true  "Reorder request"
// @Success      200       {object}  SuccessResponse
// @Failure      400       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /common-codes/categories/{category}/reorder [put]
func (h *Handler) Reorder(w http.ResponseWriter, r *http.Request) {
	category := chi.URLParam(r, "category")
	if category == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Category is required")
		return
	}

	var req ReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	err := h.service.Reorder(r.Context(), category, &req)
	if err != nil {
		if errors.Is(err, ErrInvalidReorderRequest) {
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid reorder request")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to reorder codes")
		return
	}

	utils.RespondJSON(w, http.StatusOK, SuccessResponse{Message: "Codes reordered successfully"})
}
