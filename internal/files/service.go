package files

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Storage defines the interface for file storage operations
type Storage interface {
	Save(ctx context.Context, reader io.Reader, relativePath string) error
	Get(ctx context.Context, relativePath string) (io.ReadCloser, error)
	Delete(ctx context.Context, relativePath string) error
}

// Service defines the interface for file business logic operations
type Service interface {
	UploadFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, uploaderID *string, metadata json.RawMessage) (*FileUploadResponse, error)
	GetFileForDownload(ctx context.Context, publicID string) (io.ReadCloser, *File, error)
	GetFileInfo(ctx context.Context, publicID string) (*FileResponse, error)
	ListMyFiles(ctx context.Context, uploaderID string, page, limit int) (*FileListResponse, error)
	UpdateFileMetadata(ctx context.Context, publicID string, uploaderID string, metadata json.RawMessage) error
	DeleteFile(ctx context.Context, publicID string, requesterID string) error
}

type service struct {
	repo    Repository
	storage Storage
}

// NewService creates a new file service
func NewService(repo Repository, storageBasePath string) Service {
	return &service{
		repo:    repo,
		storage: NewLocalStorage(storageBasePath),
	}
}

// -------------------- Storage Implementations --------------------

// LocalStorage implements the Storage interface for local filesystem
type LocalStorage struct {
	basePath string
}

// NewLocalStorage creates a new local storage implementation
func NewLocalStorage(basePath string) Storage {
	return &LocalStorage{basePath: basePath}
}

func (s *LocalStorage) Save(ctx context.Context, reader io.Reader, relativePath string) error {
	fullPath := filepath.Join(s.basePath, relativePath)

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the file
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy data to file
	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (s *LocalStorage) Get(ctx context.Context, relativePath string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, relativePath)

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

func (s *LocalStorage) Delete(ctx context.Context, relativePath string) error {
	fullPath := filepath.Join(s.basePath, relativePath)

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// -------------------- Service Methods --------------------

func (s *service) UploadFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, uploaderID *string, metadata json.RawMessage) (*FileUploadResponse, error) {
	// Get default storage configuration
	storage, err := s.repo.GetDefaultStorage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get default storage: %w", err)
	}

	// Check if storage is active
	if storage.Status != StorageStatusActive {
		return nil, ErrStorageNotAvailable
	}

	// Validate file size
	if storage.MaxFileSize.Valid && header.Size > storage.MaxFileSize.Int64 {
		return nil, ErrFileTooLarge
	}

	// Detect MIME type
	mimeType, err := detectMimeType(file, header)
	if err != nil {
		return nil, fmt.Errorf("failed to detect mime type: %w", err)
	}

	// Validate MIME type if allowed types are configured
	if len(storage.AllowedMimeTypes) > 0 {
		allowed := false
		for _, allowedType := range storage.AllowedMimeTypes {
			if mimeType == allowedType {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, ErrInvalidMimeType
		}
	}

	// Reset file pointer to beginning
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to reset file pointer: %w", err)
	}

	// Generate relative path: {year}/{month}/{day}/{uuid}{ext}
	now := time.Now()
	ext := filepath.Ext(header.Filename)
	fileUUID := uuid.New().String()
	relativePath := fmt.Sprintf("%04d/%02d/%02d/%s%s",
		now.Year(), now.Month(), now.Day(), fileUUID, ext)

	// Calculate checksum while saving
	checksum, err := s.saveWithChecksum(ctx, file, relativePath)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Sanitize original filename
	sanitizedFilename := sanitizeFilename(header.Filename)

	// Convert uploader public ID to internal ID
	var uploaderInternalID sql.NullInt64
	if uploaderID != nil {
		internalID, err := s.repo.GetUserInternalID(ctx, *uploaderID)
		if err != nil {
			// If user not found, still allow upload but without uploader reference
			uploaderInternalID = sql.NullInt64{Valid: false}
		} else {
			uploaderInternalID = sql.NullInt64{Int64: internalID, Valid: true}
		}
	}

	// Set default metadata if nil
	if metadata == nil {
		metadata = json.RawMessage("{}")
	}

	// Create file record in database
	fileRecord := &File{
		StorageID:        storage.ID,
		RelativePath:     relativePath,
		OriginalFilename: sanitizedFilename,
		MimeType:         mimeType,
		FileSize:         header.Size,
		ChecksumSHA256:   checksum,
		UploadedBy:       uploaderInternalID,
		Metadata:         metadata,
		IsPublic:         false, // Default to private
	}

	if err := s.repo.CreateFile(ctx, fileRecord); err != nil {
		// Rollback: delete the saved file
		_ = s.storage.Delete(ctx, relativePath)
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}

	return &FileUploadResponse{
		ID:               fileRecord.PublicID,
		OriginalFilename: fileRecord.OriginalFilename,
		MimeType:         fileRecord.MimeType,
		FileSize:         fileRecord.FileSize,
		ChecksumSHA256:   fileRecord.ChecksumSHA256,
		DownloadURL:      "/files/" + fileRecord.PublicID + "/download",
		Message:          "File uploaded successfully",
	}, nil
}

func (s *service) GetFileForDownload(ctx context.Context, publicID string) (io.ReadCloser, *File, error) {
	// Get file metadata
	file, err := s.repo.GetFileByPublicID(ctx, publicID)
	if err != nil {
		return nil, nil, ErrFileNotFound
	}

	// Get file from storage
	reader, err := s.storage.Get(ctx, file.RelativePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve file: %w", err)
	}

	// Increment download count asynchronously (best effort)
	go func() {
		_ = s.repo.IncrementDownloadCount(context.Background(), publicID)
	}()

	return reader, file, nil
}

func (s *service) GetFileInfo(ctx context.Context, publicID string) (*FileResponse, error) {
	resp, err := s.repo.GetFileDetailByPublicID(ctx, publicID)
	if err != nil {
		return nil, ErrFileNotFound
	}

	return resp, nil
}

func (s *service) ListMyFiles(ctx context.Context, uploaderID string, page, limit int) (*FileListResponse, error) {
	// Convert uploader public ID to internal ID
	internalID, err := s.repo.GetUserInternalID(ctx, uploaderID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Get files
	files, totalCount, err := s.repo.ListFilesByUploader(ctx, internalID, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	// Calculate total pages
	totalPages := (totalCount + limit - 1) / limit

	return &FileListResponse{
		Data:       files,
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

func (s *service) UpdateFileMetadata(ctx context.Context, publicID string, uploaderID string, metadata json.RawMessage) error {
	// Get file to check ownership
	file, err := s.repo.GetFileByPublicID(ctx, publicID)
	if err != nil {
		return ErrFileNotFound
	}

	// Check ownership
	if file.UploadedBy.Valid {
		uploaderInternalID, err := s.repo.GetUserInternalID(ctx, uploaderID)
		if err != nil {
			return fmt.Errorf("user not found: %w", err)
		}

		if file.UploadedBy.Int64 != uploaderInternalID {
			return ErrUnauthorized
		}
	}

	// Update metadata
	if err := s.repo.UpdateFileMetadata(ctx, publicID, metadata); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return nil
}

func (s *service) DeleteFile(ctx context.Context, publicID string, requesterID string) error {
	// Get file to check ownership
	file, err := s.repo.GetFileByPublicID(ctx, publicID)
	if err != nil {
		return ErrFileNotFound
	}

	// Check ownership
	if file.UploadedBy.Valid {
		requesterInternalID, err := s.repo.GetUserInternalID(ctx, requesterID)
		if err != nil {
			return fmt.Errorf("user not found: %w", err)
		}

		if file.UploadedBy.Int64 != requesterInternalID {
			return ErrUnauthorized
		}
	}

	// Soft delete in database
	if err := s.repo.SoftDeleteFile(ctx, publicID); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Optionally delete from storage (best effort, file is already soft-deleted in DB)
	// We don't fail the operation if physical deletion fails
	_ = s.storage.Delete(ctx, file.RelativePath)

	return nil
}

// -------------------- Helper Functions --------------------

// saveWithChecksum saves the file and calculates SHA-256 checksum simultaneously
func (s *service) saveWithChecksum(ctx context.Context, reader io.Reader, relativePath string) (string, error) {
	// Create a SHA-256 hasher
	hasher := sha256.New()

	// Use TeeReader to calculate checksum while reading
	teeReader := io.TeeReader(reader, hasher)

	// Save the file
	if err := s.storage.Save(ctx, teeReader, relativePath); err != nil {
		return "", err
	}

	// Get the checksum
	checksum := hex.EncodeToString(hasher.Sum(nil))

	return checksum, nil
}

// detectMimeType detects the MIME type of the uploaded file
func detectMimeType(file multipart.File, header *multipart.FileHeader) (string, error) {
	// Read first 512 bytes for content type detection
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	// Detect content type
	mimeType := ""
	if n > 0 {
		mimeType = http.DetectContentType(buffer[:n])
	}

	// Fallback to header content type if detection failed or returned generic type
	if mimeType == "" || mimeType == "application/octet-stream" {
		headerMimeType := header.Header.Get("Content-Type")
		if headerMimeType != "" {
			mimeType = headerMimeType
		}
	}

	// If still unknown, use application/octet-stream
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return mimeType, nil
}

// sanitizeFilename removes potentially dangerous characters from filename
func sanitizeFilename(filename string) string {
	// Remove any path separators
	filename = filepath.Base(filename)

	// Replace multiple spaces with single space
	re := regexp.MustCompile(`\s+`)
	filename = re.ReplaceAllString(filename, " ")

	// Remove or replace potentially dangerous characters
	// Keep alphanumeric, spaces, dots, dashes, underscores
	re = regexp.MustCompile(`[^a-zA-Z0-9\s.\-_]`)
	filename = re.ReplaceAllString(filename, "_")

	// Trim spaces and dots from beginning and end
	filename = strings.Trim(filename, " .")

	// Limit length to 255 characters (common filesystem limit)
	if len(filename) > 255 {
		ext := filepath.Ext(filename)
		nameWithoutExt := filename[:len(filename)-len(ext)]
		if len(nameWithoutExt) > 255-len(ext) {
			nameWithoutExt = nameWithoutExt[:255-len(ext)]
		}
		filename = nameWithoutExt + ext
	}

	// If filename is empty after sanitization, use a default
	if filename == "" {
		filename = "unnamed_file"
	}

	return filename
}
