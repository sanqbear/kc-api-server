package tickets

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"kc-api/internal/auth"
)

// Handler handles HTTP requests for ticket operations
type Handler struct {
	service Service
}

// NewHandler creates a new ticket handler with the given service
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers ticket routes on the given router
func (h *Handler) RegisterRoutes(r chi.Router) {
	// Ticket routes
	r.Route("/tickets", func(r chi.Router) {
		r.Get("/", h.ListTickets)
		r.Post("/", h.CreateTicket)
		r.Post("/search", h.SearchTickets)
		r.Get("/{id}", h.GetTicketByID)
		r.Put("/{id}", h.UpdateTicket)
		r.Delete("/{id}", h.DeleteTicket)

		// Ticket-Tag routes
		r.Post("/{id}/tags", h.AddTagsToTicket)
		r.Delete("/{id}/tags/{tagId}", h.RemoveTagFromTicket)

		// Entry routes within ticket context
		r.Post("/{id}/entries", h.CreateEntry)
	})

	// Entry routes
	r.Route("/entries", func(r chi.Router) {
		r.Get("/{id}", h.GetEntryByID)
		r.Put("/{id}", h.UpdateEntry)
		r.Delete("/{id}", h.DeleteEntry)

		// Entry-Tag routes
		r.Post("/{id}/tags", h.AddTagsToEntry)
		r.Delete("/{id}/tags/{tagId}", h.RemoveTagFromEntry)
	})

	// Tag routes
	r.Route("/tags", func(r chi.Router) {
		r.Get("/", h.ListTags)
		r.Post("/", h.CreateTag)
		r.Get("/{id}", h.GetTagByID)
		r.Put("/{id}", h.UpdateTag)
		r.Delete("/{id}", h.DeleteTag)
	})
}

// -------------------- Ticket Handlers --------------------

// ListTickets godoc
// @Summary      List tickets
// @Description  Retrieves a paginated list of tickets with simplified response
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        page   query     int  false  "Page number"     default(1)
// @Param        limit  query     int  false  "Items per page"  default(10)
// @Success      200    {object}  TicketListResponseWrapper
// @Failure      500    {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /tickets [get]
func (h *Handler) ListTickets(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	result, err := h.service.ListTickets(r.Context(), page, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to retrieve tickets", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// CreateTicket godoc
// @Summary      Create a new ticket
// @Description  Creates a new ticket with an initial entry. Title and initial_entry are required.
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        request  body      CreateTicketRequest  true  "Ticket data with initial entry"
// @Success      201      {object}  TicketDetailResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /tickets [post]
func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	var req CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Get current user ID from context
	authorUserID := auth.GetUserIDFromContext(r.Context())

	result, err := h.service.CreateTicket(r.Context(), &req, authorUserID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidTitle):
			respondError(w, http.StatusBadRequest, "Bad Request", "Title is required")
		default:
			respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		}
		return
	}

	respondJSON(w, http.StatusCreated, result)
}

// GetTicketByID godoc
// @Summary      Get ticket by ID
// @Description  Retrieves detailed ticket information including entries and tags
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Ticket Public ID (UUID)"
// @Success      200  {object}  TicketDetailResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /tickets/{id} [get]
func (h *Handler) GetTicketByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "Ticket ID is required")
		return
	}

	result, err := h.service.GetTicketByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrTicketNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "Ticket not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// UpdateTicket godoc
// @Summary      Update ticket
// @Description  Updates an existing ticket
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        id       path      string              true  "Ticket Public ID (UUID)"
// @Param        request  body      UpdateTicketRequest true  "Ticket data to update"
// @Success      200      {object}  TicketListResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /tickets/{id} [put]
func (h *Handler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "Ticket ID is required")
		return
	}

	var req UpdateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.UpdateTicket(r.Context(), id, &req)
	if err != nil {
		if errors.Is(err, ErrTicketNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "Ticket not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// DeleteTicket godoc
// @Summary      Delete ticket
// @Description  Deletes a ticket and all its entries
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Ticket Public ID (UUID)"
// @Success      200  {object}  SuccessResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /tickets/{id} [delete]
func (h *Handler) DeleteTicket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "Ticket ID is required")
		return
	}

	err := h.service.DeleteTicket(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrTicketNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "Ticket not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Ticket deleted successfully"})
}

// SearchTickets godoc
// @Summary      Search tickets
// @Description  Searches for tickets based on various criteria
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        page     query     int                  false  "Page number"     default(1)
// @Param        limit    query     int                  false  "Items per page"  default(10)
// @Param        request  body      SearchTicketRequest  true   "Search criteria"
// @Success      200      {object}  TicketListResponseWrapper
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /tickets/search [post]
func (h *Handler) SearchTickets(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	var req SearchTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.SearchTickets(r.Context(), &req, page, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to search tickets", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// AddTagsToTicket godoc
// @Summary      Add tags to ticket
// @Description  Adds tags to a ticket
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        id       path      string         true  "Ticket Public ID (UUID)"
// @Param        request  body      AddTagRequest  true  "Tags to add"
// @Success      200      {object}  SuccessResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /tickets/{id}/tags [post]
func (h *Handler) AddTagsToTicket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "Ticket ID is required")
		return
	}

	var req AddTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	err := h.service.AddTagsToTicket(r.Context(), id, &req)
	if err != nil {
		if errors.Is(err, ErrTicketNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "Ticket not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Tags added successfully"})
}

// RemoveTagFromTicket godoc
// @Summary      Remove tag from ticket
// @Description  Removes a tag from a ticket
// @Tags         tickets
// @Accept       json
// @Produce      json
// @Param        id     path      string  true  "Ticket Public ID (UUID)"
// @Param        tagId  path      int     true  "Tag ID"
// @Success      200    {object}  SuccessResponse
// @Failure      404    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /tickets/{id}/tags/{tagId} [delete]
func (h *Handler) RemoveTagFromTicket(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "Ticket ID is required")
		return
	}

	tagID, err := strconv.ParseInt(chi.URLParam(r, "tagId"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request", "Invalid tag ID")
		return
	}

	err = h.service.RemoveTagFromTicket(r.Context(), id, tagID)
	if err != nil {
		if errors.Is(err, ErrTicketNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "Ticket not found")
			return
		}
		if errors.Is(err, ErrTagNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "Tag not found on ticket")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Tag removed successfully"})
}

// -------------------- Entry Handlers --------------------

// CreateEntry godoc
// @Summary      Create a new entry
// @Description  Creates a new entry for a ticket
// @Tags         entries
// @Accept       json
// @Produce      json
// @Param        id       path      string              true  "Ticket Public ID (UUID)"
// @Param        request  body      CreateEntryRequest  true  "Entry data"
// @Success      201      {object}  EntryDetailResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /tickets/{id}/entries [post]
func (h *Handler) CreateEntry(w http.ResponseWriter, r *http.Request) {
	ticketID := chi.URLParam(r, "id")
	if ticketID == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "Ticket ID is required")
		return
	}

	var req CreateEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Get current user ID from context
	authorUserID := auth.GetUserIDFromContext(r.Context())

	result, err := h.service.CreateEntry(r.Context(), ticketID, &req, authorUserID)
	if err != nil {
		if errors.Is(err, ErrTicketNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "Ticket not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, result)
}

// GetEntryByID godoc
// @Summary      Get entry by ID
// @Description  Retrieves detailed entry information including tags and references
// @Tags         entries
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Entry ID"
// @Success      200  {object}  EntryDetailResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /entries/{id} [get]
func (h *Handler) GetEntryByID(w http.ResponseWriter, r *http.Request) {
	entryID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request", "Invalid entry ID")
		return
	}

	result, err := h.service.GetEntryByID(r.Context(), entryID)
	if err != nil {
		if errors.Is(err, ErrEntryNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "Entry not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// UpdateEntry godoc
// @Summary      Update entry
// @Description  Updates an existing entry
// @Tags         entries
// @Accept       json
// @Produce      json
// @Param        id       path      int                 true  "Entry ID"
// @Param        request  body      UpdateEntryRequest  true  "Entry data to update"
// @Success      200      {object}  EntryListResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /entries/{id} [put]
func (h *Handler) UpdateEntry(w http.ResponseWriter, r *http.Request) {
	entryID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request", "Invalid entry ID")
		return
	}

	var req UpdateEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.UpdateEntry(r.Context(), entryID, &req)
	if err != nil {
		if errors.Is(err, ErrEntryNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "Entry not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// DeleteEntry godoc
// @Summary      Delete entry
// @Description  Performs a soft delete on an entry
// @Tags         entries
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Entry ID"
// @Success      200  {object}  SuccessResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /entries/{id} [delete]
func (h *Handler) DeleteEntry(w http.ResponseWriter, r *http.Request) {
	entryID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request", "Invalid entry ID")
		return
	}

	err = h.service.DeleteEntry(r.Context(), entryID)
	if err != nil {
		if errors.Is(err, ErrEntryNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "Entry not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Entry deleted successfully"})
}

// AddTagsToEntry godoc
// @Summary      Add tags to entry
// @Description  Adds tags to an entry
// @Tags         entries
// @Accept       json
// @Produce      json
// @Param        id       path      int            true  "Entry ID"
// @Param        request  body      AddTagRequest  true  "Tags to add"
// @Success      200      {object}  SuccessResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /entries/{id}/tags [post]
func (h *Handler) AddTagsToEntry(w http.ResponseWriter, r *http.Request) {
	entryID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request", "Invalid entry ID")
		return
	}

	var req AddTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	err = h.service.AddTagsToEntry(r.Context(), entryID, &req)
	if err != nil {
		if errors.Is(err, ErrEntryNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "Entry not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Tags added successfully"})
}

// RemoveTagFromEntry godoc
// @Summary      Remove tag from entry
// @Description  Removes a tag from an entry
// @Tags         entries
// @Accept       json
// @Produce      json
// @Param        id     path      int  true  "Entry ID"
// @Param        tagId  path      int  true  "Tag ID"
// @Success      200    {object}  SuccessResponse
// @Failure      404    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /entries/{id}/tags/{tagId} [delete]
func (h *Handler) RemoveTagFromEntry(w http.ResponseWriter, r *http.Request) {
	entryID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request", "Invalid entry ID")
		return
	}

	tagID, err := strconv.ParseInt(chi.URLParam(r, "tagId"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request", "Invalid tag ID")
		return
	}

	err = h.service.RemoveTagFromEntry(r.Context(), entryID, tagID)
	if err != nil {
		if errors.Is(err, ErrEntryNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "Entry not found")
			return
		}
		if errors.Is(err, ErrTagNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "Tag not found on entry")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Tag removed successfully"})
}

// -------------------- Tag Handlers --------------------

// ListTags godoc
// @Summary      List tags
// @Description  Retrieves a paginated list of tags
// @Tags         tags
// @Accept       json
// @Produce      json
// @Param        page   query     int  false  "Page number"     default(1)
// @Param        limit  query     int  false  "Items per page"  default(10)
// @Success      200    {object}  TagListResponseWrapper
// @Failure      500    {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /tags [get]
func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	result, err := h.service.ListTags(r.Context(), page, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to retrieve tags", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// CreateTag godoc
// @Summary      Create a new tag
// @Description  Creates a new tag
// @Tags         tags
// @Accept       json
// @Produce      json
// @Param        request  body      CreateTagRequest  true  "Tag data"
// @Success      201      {object}  TagResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /tags [post]
func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	var req CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.CreateTag(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidTagName):
			respondError(w, http.StatusBadRequest, "Bad Request", "Tag name is required")
		default:
			respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		}
		return
	}

	respondJSON(w, http.StatusCreated, result)
}

// GetTagByID godoc
// @Summary      Get tag by ID
// @Description  Retrieves a tag by its ID
// @Tags         tags
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Tag ID"
// @Success      200  {object}  TagResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /tags/{id} [get]
func (h *Handler) GetTagByID(w http.ResponseWriter, r *http.Request) {
	tagID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request", "Invalid tag ID")
		return
	}

	result, err := h.service.GetTagByID(r.Context(), tagID)
	if err != nil {
		if errors.Is(err, ErrTagNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "Tag not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// UpdateTag godoc
// @Summary      Update tag
// @Description  Updates an existing tag
// @Tags         tags
// @Accept       json
// @Produce      json
// @Param        id       path      int               true  "Tag ID"
// @Param        request  body      UpdateTagRequest  true  "Tag data to update"
// @Success      200      {object}  TagResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /tags/{id} [put]
func (h *Handler) UpdateTag(w http.ResponseWriter, r *http.Request) {
	tagID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request", "Invalid tag ID")
		return
	}

	var req UpdateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	result, err := h.service.UpdateTag(r.Context(), tagID, &req)
	if err != nil {
		if errors.Is(err, ErrTagNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "Tag not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// DeleteTag godoc
// @Summary      Delete tag
// @Description  Performs a soft delete on a tag
// @Tags         tags
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "Tag ID"
// @Success      200  {object}  SuccessResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /tags/{id} [delete]
func (h *Handler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	tagID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request", "Invalid tag ID")
		return
	}

	err = h.service.DeleteTag(r.Context(), tagID)
	if err != nil {
		if errors.Is(err, ErrTagNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "Tag not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "Tag deleted successfully"})
}

// -------------------- Helper Functions --------------------

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func respondError(w http.ResponseWriter, status int, errType, message string) {
	respondJSON(w, status, ErrorResponse{Error: errType, Message: message})
}
