package files

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
)

// Repository defines the interface for file data access operations
type Repository interface {
	// Storage operations
	GetDefaultStorage(ctx context.Context) (*FileStorage, error)
	GetStorageByID(ctx context.Context, id int64) (*FileStorage, error)

	// File operations
	CreateFile(ctx context.Context, file *File) error
	GetFileByPublicID(ctx context.Context, publicID string) (*File, error)
	GetFileByID(ctx context.Context, id int64) (*File, error)
	GetFileDetailByPublicID(ctx context.Context, publicID string) (*FileResponse, error)
	ListFilesByUploader(ctx context.Context, uploaderID int64, page, limit int) ([]FileResponse, int, error)
	UpdateFileMetadata(ctx context.Context, publicID string, metadata json.RawMessage) error
	IncrementDownloadCount(ctx context.Context, publicID string) error
	SoftDeleteFile(ctx context.Context, publicID string) error

	// Helper operations
	GetUserInternalID(ctx context.Context, publicID string) (int64, error)
}

type repository struct {
	db *sql.DB
}

// NewRepository creates a new file repository
func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// -------------------- Storage Operations --------------------

func (r *repository) GetDefaultStorage(ctx context.Context) (*FileStorage, error) {
	query := `
		SELECT id, public_id, name, storage_type, base_path, config, status,
		       max_file_size, allowed_mime_types, is_default, is_deleted, created_at, updated_at
		FROM managements.file_storages
		WHERE is_default = true AND is_deleted = false
		LIMIT 1`

	storage := &FileStorage{}
	var allowedMimeTypes pq.StringArray

	err := r.db.QueryRowContext(ctx, query).Scan(
		&storage.ID,
		&storage.PublicID,
		&storage.Name,
		&storage.StorageType,
		&storage.BasePath,
		&storage.Config,
		&storage.Status,
		&storage.MaxFileSize,
		&allowedMimeTypes,
		&storage.IsDefault,
		&storage.IsDeleted,
		&storage.CreatedAt,
		&storage.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	storage.AllowedMimeTypes = allowedMimeTypes

	return storage, nil
}

func (r *repository) GetStorageByID(ctx context.Context, id int64) (*FileStorage, error) {
	query := `
		SELECT id, public_id, name, storage_type, base_path, config, status,
		       max_file_size, allowed_mime_types, is_default, is_deleted, created_at, updated_at
		FROM managements.file_storages
		WHERE id = $1 AND is_deleted = false`

	storage := &FileStorage{}
	var allowedMimeTypes pq.StringArray

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&storage.ID,
		&storage.PublicID,
		&storage.Name,
		&storage.StorageType,
		&storage.BasePath,
		&storage.Config,
		&storage.Status,
		&storage.MaxFileSize,
		&allowedMimeTypes,
		&storage.IsDefault,
		&storage.IsDeleted,
		&storage.CreatedAt,
		&storage.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	storage.AllowedMimeTypes = allowedMimeTypes

	return storage, nil
}

// -------------------- File Operations --------------------

func (r *repository) CreateFile(ctx context.Context, file *File) error {
	query := `
		INSERT INTO managements.files (
			storage_id, relative_path, original_filename, mime_type, file_size,
			checksum_sha256, uploaded_by, metadata, is_public
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, public_id, download_count, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		file.StorageID,
		file.RelativePath,
		file.OriginalFilename,
		file.MimeType,
		file.FileSize,
		file.ChecksumSHA256,
		file.UploadedBy,
		file.Metadata,
		file.IsPublic,
	).Scan(&file.ID, &file.PublicID, &file.DownloadCount, &file.CreatedAt, &file.UpdatedAt)
}

func (r *repository) GetFileByPublicID(ctx context.Context, publicID string) (*File, error) {
	query := `
		SELECT id, public_id, storage_id, relative_path, original_filename, mime_type, file_size,
		       checksum_sha256, uploaded_by, metadata, download_count, last_accessed_at,
		       is_public, is_deleted, deleted_at, created_at, updated_at
		FROM managements.files
		WHERE public_id = $1 AND is_deleted = false`

	file := &File{}
	err := r.db.QueryRowContext(ctx, query, publicID).Scan(
		&file.ID,
		&file.PublicID,
		&file.StorageID,
		&file.RelativePath,
		&file.OriginalFilename,
		&file.MimeType,
		&file.FileSize,
		&file.ChecksumSHA256,
		&file.UploadedBy,
		&file.Metadata,
		&file.DownloadCount,
		&file.LastAccessedAt,
		&file.IsPublic,
		&file.IsDeleted,
		&file.DeletedAt,
		&file.CreatedAt,
		&file.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (r *repository) GetFileByID(ctx context.Context, id int64) (*File, error) {
	query := `
		SELECT id, public_id, storage_id, relative_path, original_filename, mime_type, file_size,
		       checksum_sha256, uploaded_by, metadata, download_count, last_accessed_at,
		       is_public, is_deleted, deleted_at, created_at, updated_at
		FROM managements.files
		WHERE id = $1 AND is_deleted = false`

	file := &File{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&file.ID,
		&file.PublicID,
		&file.StorageID,
		&file.RelativePath,
		&file.OriginalFilename,
		&file.MimeType,
		&file.FileSize,
		&file.ChecksumSHA256,
		&file.UploadedBy,
		&file.Metadata,
		&file.DownloadCount,
		&file.LastAccessedAt,
		&file.IsPublic,
		&file.IsDeleted,
		&file.DeletedAt,
		&file.CreatedAt,
		&file.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (r *repository) GetFileDetailByPublicID(ctx context.Context, publicID string) (*FileResponse, error) {
	query := `
		SELECT
			f.public_id, f.original_filename, f.mime_type, f.file_size, f.checksum_sha256,
			f.download_count, f.is_public, f.metadata, f.last_accessed_at, f.created_at, f.updated_at,
			u.public_id, u.name
		FROM managements.files f
		LEFT JOIN organizations.users u ON f.uploaded_by = u.id
		WHERE f.public_id = $1 AND f.is_deleted = false`

	var uploaderPublicID sql.NullString
	var uploaderName sql.NullString
	var lastAccessedAt sql.NullTime
	resp := &FileResponse{}

	err := r.db.QueryRowContext(ctx, query, publicID).Scan(
		&resp.ID,
		&resp.OriginalFilename,
		&resp.MimeType,
		&resp.FileSize,
		&resp.ChecksumSHA256,
		&resp.DownloadCount,
		&resp.IsPublic,
		&resp.Metadata,
		&lastAccessedAt,
		&resp.CreatedAt,
		&resp.UpdatedAt,
		&uploaderPublicID,
		&uploaderName,
	)
	if err != nil {
		return nil, err
	}

	resp.DownloadURL = "/files/" + resp.ID + "/download"

	if uploaderPublicID.Valid {
		resp.UploadedBy = &uploaderPublicID.String
		if uploaderName.Valid {
			resp.UploadedByName = json.RawMessage(uploaderName.String)
		}
	}

	if lastAccessedAt.Valid {
		resp.LastAccessedAt = &lastAccessedAt.Time
	}

	return resp, nil
}

func (r *repository) ListFilesByUploader(ctx context.Context, uploaderID int64, page, limit int) ([]FileResponse, int, error) {
	offset := (page - 1) * limit

	// Count total files
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM managements.files WHERE uploaded_by = $1 AND is_deleted = false`
	if err := r.db.QueryRowContext(ctx, countQuery, uploaderID).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// Query files with user information
	query := `
		SELECT
			f.public_id, f.original_filename, f.mime_type, f.file_size, f.checksum_sha256,
			f.download_count, f.is_public, f.metadata, f.last_accessed_at, f.created_at, f.updated_at,
			u.public_id, u.name
		FROM managements.files f
		LEFT JOIN organizations.users u ON f.uploaded_by = u.id
		WHERE f.uploaded_by = $1 AND f.is_deleted = false
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, uploaderID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var files []FileResponse
	for rows.Next() {
		var resp FileResponse
		var uploaderPublicID sql.NullString
		var uploaderName sql.NullString
		var lastAccessedAt sql.NullTime

		if err := rows.Scan(
			&resp.ID,
			&resp.OriginalFilename,
			&resp.MimeType,
			&resp.FileSize,
			&resp.ChecksumSHA256,
			&resp.DownloadCount,
			&resp.IsPublic,
			&resp.Metadata,
			&lastAccessedAt,
			&resp.CreatedAt,
			&resp.UpdatedAt,
			&uploaderPublicID,
			&uploaderName,
		); err != nil {
			return nil, 0, err
		}

		resp.DownloadURL = "/files/" + resp.ID + "/download"

		if uploaderPublicID.Valid {
			resp.UploadedBy = &uploaderPublicID.String
			if uploaderName.Valid {
				resp.UploadedByName = json.RawMessage(uploaderName.String)
			}
		}

		if lastAccessedAt.Valid {
			resp.LastAccessedAt = &lastAccessedAt.Time
		}

		files = append(files, resp)
	}

	return files, totalCount, rows.Err()
}

func (r *repository) UpdateFileMetadata(ctx context.Context, publicID string, metadata json.RawMessage) error {
	query := `
		UPDATE managements.files
		SET metadata = $1
		WHERE public_id = $2 AND is_deleted = false`

	result, err := r.db.ExecContext(ctx, query, metadata, publicID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *repository) IncrementDownloadCount(ctx context.Context, publicID string) error {
	query := `
		UPDATE managements.files
		SET download_count = download_count + 1,
		    last_accessed_at = CURRENT_TIMESTAMP
		WHERE public_id = $1 AND is_deleted = false`

	result, err := r.db.ExecContext(ctx, query, publicID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *repository) SoftDeleteFile(ctx context.Context, publicID string) error {
	query := `
		UPDATE managements.files
		SET is_deleted = true,
		    deleted_at = CURRENT_TIMESTAMP
		WHERE public_id = $1 AND is_deleted = false`

	result, err := r.db.ExecContext(ctx, query, publicID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("file not found or already deleted")
	}

	return nil
}

// -------------------- Helper Operations --------------------

func (r *repository) GetUserInternalID(ctx context.Context, publicID string) (int64, error) {
	var id int64
	query := `SELECT id FROM organizations.users WHERE public_id = $1`
	err := r.db.QueryRowContext(ctx, query, publicID).Scan(&id)
	return id, err
}
