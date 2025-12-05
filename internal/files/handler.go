package files

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"kc-api/internal/auth"
)

const (
	// MaxUploadSize is the maximum file size allowed for upload (100 MB)
	MaxUploadSize = 100 * 1024 * 1024 // 100 MB
)

// Handler handles HTTP requests for file operations
type Handler struct {
	service Service
}

// NewHandler creates a new file handler with the given service
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers file routes on the given router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/files", func(r chi.Router) {
		r.Post("/", h.UploadFile)
		r.Get("/", h.ListMyFiles)
		r.Get("/{id}", h.GetFileInfo)
		r.Get("/{id}/download", h.DownloadFile)
		r.Put("/{id}/metadata", h.UpdateFileMetadata)
		r.Delete("/{id}", h.DeleteFile)
	})
}

// -------------------- File Handlers --------------------

// UploadFile godoc
// @Summary      Upload a file
// @Description  Uploads a file to the default storage backend. Maximum file size is 100MB.
// @Tags         files
// @Accept       multipart/form-data
// @Produce      json
// @Param        file      formData  file    true   "File to upload"
// @Param        metadata  formData  string  false  "Optional metadata JSON object"
// @Success      201       {object}  FileUploadResponse
// @Failure      400       {object}  ErrorResponse
// @Failure      413       {object}  ErrorResponse  "File too large"
// @Failure      500       {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /files [post]
func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadSize)

	// Parse multipart form
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		respondError(w, http.StatusRequestEntityTooLarge, "File too large", "Maximum file size is 100MB")
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request", "File is required")
		return
	}
	defer file.Close()

	// Get optional metadata
	var metadata json.RawMessage
	metadataStr := r.FormValue("metadata")
	if metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			respondError(w, http.StatusBadRequest, "Bad Request", "Invalid metadata JSON")
			return
		}
	}

	// Get current user ID from context
	uploaderID := auth.GetUserIDFromContext(r.Context())
	var uploaderPtr *string
	if uploaderID != "" {
		uploaderPtr = &uploaderID
	}

	// Upload file
	result, err := h.service.UploadFile(r.Context(), file, header, uploaderPtr, metadata)
	if err != nil {
		if errors.Is(err, ErrFileTooLarge) {
			respondError(w, http.StatusRequestEntityTooLarge, "File too large", err.Error())
			return
		}
		if errors.Is(err, ErrInvalidMimeType) {
			respondError(w, http.StatusBadRequest, "Invalid MIME type", err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to upload file", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, result)
}

// GetFileInfo godoc
// @Summary      Get file information
// @Description  Retrieves detailed information about a file without downloading it
// @Tags         files
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "File Public ID (UUID)"
// @Success      200  {object}  FileResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /files/{id} [get]
func (h *Handler) GetFileInfo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "File ID is required")
		return
	}

	result, err := h.service.GetFileInfo(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrFileNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "File not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// DownloadFile godoc
// @Summary      Download a file
// @Description  Downloads the file content. Increments the download counter.
// @Tags         files
// @Produce      octet-stream
// @Param        id   path      string  true  "File Public ID (UUID)"
// @Success      200  {file}    binary
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /files/{id}/download [get]
func (h *Handler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "File ID is required")
		return
	}

	reader, file, err := h.service.GetFileForDownload(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrFileNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "File not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	defer reader.Close()

	// Set response headers
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+file.OriginalFilename+"\"")
	w.Header().Set("Content-Length", strconv.FormatInt(file.FileSize, 10))
	w.Header().Set("X-Content-SHA256", file.ChecksumSHA256)

	// Stream file to response
	if _, err := io.Copy(w, reader); err != nil {
		// Can't send error response here as headers are already sent
		// Log the error (in production, use proper logging)
		return
	}
}

// ListMyFiles godoc
// @Summary      List my uploaded files
// @Description  Retrieves a paginated list of files uploaded by the current user
// @Tags         files
// @Accept       json
// @Produce      json
// @Param        page   query     int  false  "Page number"     default(1)
// @Param        limit  query     int  false  "Items per page"  default(20)
// @Success      200    {object}  FileListResponse
// @Failure      401    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /files [get]
func (h *Handler) ListMyFiles(w http.ResponseWriter, r *http.Request) {
	// Get current user ID from context
	uploaderID := auth.GetUserIDFromContext(r.Context())
	if uploaderID == "" {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	result, err := h.service.ListMyFiles(r.Context(), uploaderID, page, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to retrieve files", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// UpdateFileMetadata godoc
// @Summary      Update file metadata
// @Description  Updates the metadata of a file. Only the file uploader can update metadata.
// @Tags         files
// @Accept       json
// @Produce      json
// @Param        id       path      string                    true  "File Public ID (UUID)"
// @Param        request  body      UpdateFileMetadataRequest true  "Metadata to update"
// @Success      200      {object}  SuccessResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse  "Forbidden - not the file owner"
// @Failure      404      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /files/{id}/metadata [put]
func (h *Handler) UpdateFileMetadata(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "File ID is required")
		return
	}

	// Get current user ID from context
	uploaderID := auth.GetUserIDFromContext(r.Context())
	if uploaderID == "" {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		return
	}

	var req UpdateFileMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := h.service.UpdateFileMetadata(r.Context(), id, uploaderID, req.Metadata); err != nil {
		if errors.Is(err, ErrFileNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "File not found")
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			respondError(w, http.StatusForbidden, "Forbidden", "You can only update your own files")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "File metadata updated successfully"})
}

// DeleteFile godoc
// @Summary      Delete a file
// @Description  Performs a soft delete on a file. Only the file uploader can delete the file.
// @Tags         files
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "File Public ID (UUID)"
// @Success      200  {object}  SuccessResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse  "Forbidden - not the file owner"
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /files/{id} [delete]
func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "File ID is required")
		return
	}

	// Get current user ID from context
	requesterID := auth.GetUserIDFromContext(r.Context())
	if requesterID == "" {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		return
	}

	if err := h.service.DeleteFile(r.Context(), id, requesterID); err != nil {
		if errors.Is(err, ErrFileNotFound) {
			respondError(w, http.StatusNotFound, "Not Found", "File not found")
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			respondError(w, http.StatusForbidden, "Forbidden", "You can only delete your own files")
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{Message: "File deleted successfully"})
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
