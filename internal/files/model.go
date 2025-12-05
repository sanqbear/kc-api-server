package files

import (
	"database/sql"
	"encoding/json"
	"time"
)

// StorageType represents the type of file storage backend
type StorageType string

const (
	StorageTypeLocal     StorageType = "LOCAL"
	StorageTypeS3        StorageType = "S3"
	StorageTypeAzureBlob StorageType = "AZURE_BLOB"
	StorageTypeGCS       StorageType = "GCS"
	StorageTypeNFS       StorageType = "NFS"
)

// StorageStatus represents the operational status of a storage backend
type StorageStatus string

const (
	StorageStatusActive   StorageStatus = "ACTIVE"
	StorageStatusReadonly StorageStatus = "READONLY"
	StorageStatusDisabled StorageStatus = "DISABLED"
)

// FileStorage represents a file storage backend configuration
type FileStorage struct {
	ID               int64          `json:"-"`
	PublicID         string         `json:"id"`
	Name             string         `json:"name"`
	StorageType      StorageType    `json:"storage_type"`
	BasePath         string         `json:"base_path"`
	Config           json.RawMessage `json:"config"`
	Status           StorageStatus  `json:"status"`
	MaxFileSize      sql.NullInt64  `json:"-"`
	AllowedMimeTypes []string       `json:"allowed_mime_types"`
	IsDefault        bool           `json:"is_default"`
	IsDeleted        bool           `json:"-"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// File represents an uploaded file in the system
type File struct {
	ID               int64          `json:"-"`
	PublicID         string         `json:"id"`
	StorageID        int64          `json:"-"`
	RelativePath     string         `json:"-"`
	OriginalFilename string         `json:"original_filename"`
	MimeType         string         `json:"mime_type"`
	FileSize         int64          `json:"file_size"`
	ChecksumSHA256   string         `json:"checksum_sha256"`
	UploadedBy       sql.NullInt64  `json:"-"`
	Metadata         json.RawMessage `json:"metadata"`
	DownloadCount    int64          `json:"download_count"`
	LastAccessedAt   sql.NullTime   `json:"-"`
	IsPublic         bool           `json:"is_public"`
	IsDeleted        bool           `json:"-"`
	DeletedAt        sql.NullTime   `json:"-"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// -------------------- Response DTOs --------------------

// FileResponse represents a file response for API
type FileResponse struct {
	ID               string          `json:"id" example:"01912345-6789-7abc-def0-123456789abc"`
	OriginalFilename string          `json:"original_filename" example:"document.pdf"`
	MimeType         string          `json:"mime_type" example:"application/pdf"`
	FileSize         int64           `json:"file_size" example:"1048576"`
	ChecksumSHA256   string          `json:"checksum_sha256" example:"abc123..."`
	DownloadURL      string          `json:"download_url" example:"/files/01912345-6789-7abc-def0-123456789abc/download"`
	DownloadCount    int64           `json:"download_count" example:"42"`
	IsPublic         bool            `json:"is_public" example:"false"`
	Metadata         json.RawMessage `json:"metadata,omitempty" swaggertype:"object"`
	UploadedBy       *string         `json:"uploaded_by,omitempty" example:"01912345-6789-7abc-def0-123456789abc"`
	UploadedByName   json.RawMessage `json:"uploaded_by_name,omitempty" swaggertype:"object"`
	CreatedAt        time.Time       `json:"created_at" example:"2024-12-05T00:00:00Z"`
	UpdatedAt        time.Time       `json:"updated_at" example:"2024-12-05T00:00:00Z"`
	LastAccessedAt   *time.Time      `json:"last_accessed_at,omitempty" example:"2024-12-05T00:00:00Z"`
}

// FileUploadResponse represents the response after successful file upload
type FileUploadResponse struct {
	ID               string `json:"id" example:"01912345-6789-7abc-def0-123456789abc"`
	OriginalFilename string `json:"original_filename" example:"document.pdf"`
	MimeType         string `json:"mime_type" example:"application/pdf"`
	FileSize         int64  `json:"file_size" example:"1048576"`
	ChecksumSHA256   string `json:"checksum_sha256" example:"abc123..."`
	DownloadURL      string `json:"download_url" example:"/files/01912345-6789-7abc-def0-123456789abc/download"`
	Message          string `json:"message" example:"File uploaded successfully"`
}

// FileListResponse represents a paginated list of files
type FileListResponse struct {
	Data       []FileResponse `json:"data"`
	Page       int            `json:"page" example:"1"`
	Limit      int            `json:"limit" example:"20"`
	TotalCount int            `json:"total_count" example:"100"`
	TotalPages int            `json:"total_pages" example:"5"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error" example:"Bad Request"`
	Message string `json:"message" example:"Invalid request"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

// -------------------- Request DTOs --------------------

// UpdateFileMetadataRequest represents the request to update file metadata
type UpdateFileMetadataRequest struct {
	Metadata json.RawMessage `json:"metadata" swaggertype:"object"`
}

// -------------------- Conversion Methods --------------------

// ToResponse converts a File to FileResponse
func (f *File) ToResponse(uploaderPublicID *string, uploaderName json.RawMessage) FileResponse {
	resp := FileResponse{
		ID:               f.PublicID,
		OriginalFilename: f.OriginalFilename,
		MimeType:         f.MimeType,
		FileSize:         f.FileSize,
		ChecksumSHA256:   f.ChecksumSHA256,
		DownloadURL:      "/files/" + f.PublicID + "/download",
		DownloadCount:    f.DownloadCount,
		IsPublic:         f.IsPublic,
		Metadata:         f.Metadata,
		CreatedAt:        f.CreatedAt,
		UpdatedAt:        f.UpdatedAt,
	}

	if uploaderPublicID != nil {
		resp.UploadedBy = uploaderPublicID
		if uploaderName != nil {
			resp.UploadedByName = uploaderName
		}
	}

	if f.LastAccessedAt.Valid {
		resp.LastAccessedAt = &f.LastAccessedAt.Time
	}

	return resp
}

// ToUploadResponse converts a File to FileUploadResponse
func (f *File) ToUploadResponse() FileUploadResponse {
	return FileUploadResponse{
		ID:               f.PublicID,
		OriginalFilename: f.OriginalFilename,
		MimeType:         f.MimeType,
		FileSize:         f.FileSize,
		ChecksumSHA256:   f.ChecksumSHA256,
		DownloadURL:      "/files/" + f.PublicID + "/download",
		Message:          "File uploaded successfully",
	}
}
