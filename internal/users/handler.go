package users

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// Handler handles HTTP requests for user operations
type Handler struct {
	service Service
}

// NewHandler creates a new user handler with the given service
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers user routes on the given router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/users", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Post("/search", h.Search)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})
}

// List godoc
// @Summary      List users
// @Description  Retrieves a paginated list of users with simplified response
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        page   query     int  false  "Page number"     default(1)
// @Param        limit  query     int  false  "Items per page"  default(10)
// @Success      200    {object}  UserListResponseWrapper
// @Failure      500    {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /users [get]
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
		respondError(w, http.StatusInternalServerError, "Failed to retrieve users", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// Create godoc
// @Summary      Create a new user
// @Description  Creates a new user with the provided data. Email and name are required. If login_id is not provided, email is used as login_id.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request  body      CreateUserRequest  true  "User data"
// @Success      201      {object}  UserListResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      409      {object}  ErrorResponse  "Email or login_id already exists"
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /users [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.Create(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidEmail):
			respondError(w, http.StatusBadRequest, "Bad Request", "Invalid email format")
		case errors.Is(err, ErrInvalidName):
			respondError(w, http.StatusBadRequest, "Bad Request", "Name must have at least one locale value")
		case errors.Is(err, ErrEmailExists):
			respondError(w, http.StatusConflict, "Conflict", "Email already exists")
		case errors.Is(err, ErrLoginIDExists):
			respondError(w, http.StatusConflict, "Conflict", "Login ID already exists")
		default:
			respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		}
		return
	}

	respondJSON(w, http.StatusCreated, result)
}

// GetByID godoc
// @Summary      Get user by ID
// @Description  Retrieves detailed user information by their public ID
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User Public ID (UUID)"
// @Success      200  {object}  UserDetailResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /users/{id} [get]
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "User ID is required")
		return
	}

	result, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "User not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// Update godoc
// @Summary      Update user
// @Description  Updates an existing user with the provided data
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id       path      string             true  "User Public ID (UUID)"
// @Param        request  body      UpdateUserRequest  true  "User data to update"
// @Success      200      {object}  UserListResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Failure      409      {object}  ErrorResponse  "Email or login_id already exists"
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /users/{id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "User ID is required")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			respondError(w, http.StatusNotFound, "Not Found", "User not found")
		case errors.Is(err, ErrInvalidEmail):
			respondError(w, http.StatusBadRequest, "Bad Request", "Invalid email format")
		case errors.Is(err, ErrInvalidName):
			respondError(w, http.StatusBadRequest, "Bad Request", "Name must have at least one locale value")
		case errors.Is(err, ErrEmailExists):
			respondError(w, http.StatusConflict, "Conflict", "Email already exists")
		case errors.Is(err, ErrLoginIDExists):
			respondError(w, http.StatusConflict, "Conflict", "Login ID already exists")
		default:
			respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		}
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// Delete godoc
// @Summary      Delete user
// @Description  Performs a soft delete on a user (sets is_deleted flag to true)
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User Public ID (UUID)"
// @Success      200  {object}  SuccessResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /users/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "User ID is required")
		return
	}

	err := h.service.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "User not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "User deleted successfully"})
}

// Search godoc
// @Summary      Search users
// @Description  Searches for users based on various criteria (name, email, mobile number, office number)
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        page     query     int                false  "Page number"     default(1)
// @Param        limit    query     int                false  "Items per page"  default(10)
// @Param        request  body      SearchUserRequest  true   "Search criteria"
// @Success      200      {object}  UserListResponseWrapper
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /users/search [post]
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	var req SearchUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.Search(r.Context(), &req, page, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to search users", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// respondError writes an error response
func respondError(w http.ResponseWriter, status int, errType, message string) {
	respondJSON(w, status, ErrorResponse{Error: errType, Message: message})
}
