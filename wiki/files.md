# File Management Domain

The file management domain provides secure file upload, storage, and retrieval functionality for the Knowledge Center API.

## Architecture

The files domain follows the standard DDD pattern:

```
internal/files/
  model.go        # Domain models and DTOs
  errors.go       # Domain-specific errors
  repository.go   # Database operations
  service.go      # Business logic and storage abstraction
  handler.go      # HTTP handlers
```

## Features

### File Upload
- **Endpoint**: `POST /files`
- **Max Size**: 100 MB (configurable via storage configuration)
- **Storage**: Local filesystem (extensible to S3, Azure Blob, GCS, NFS)
- **Security**:
  - File size validation
  - MIME type detection and validation
  - Filename sanitization
  - SHA-256 checksum calculation
  - User authentication required

### File Download
- **Endpoint**: `GET /files/{id}/download`
- **Features**:
  - Streaming download
  - Download counter tracking
  - Last accessed timestamp
  - Proper Content-Disposition headers

### File Information
- **Endpoint**: `GET /files/{id}`
- **Returns**: File metadata without downloading the file

### File Listing
- **Endpoint**: `GET /files`
- **Features**:
  - Paginated results (default 20 per page)
  - Shows only files uploaded by the authenticated user
  - Includes uploader information

### Metadata Management
- **Endpoint**: `PUT /files/{id}/metadata`
- **Features**:
  - Update custom JSON metadata
  - Owner-only access

### File Deletion
- **Endpoint**: `DELETE /files/{id}`
- **Features**:
  - Soft delete (sets is_deleted flag)
  - Owner-only access
  - Physical file deletion (best effort)

## Database Schema

### managements.file_storages
Configuration for storage backends:
- `id`: Internal ID (BIGINT)
- `public_id`: UUID v7 for external reference
- `name`: Storage backend name
- `storage_type`: LOCAL, S3, AZURE_BLOB, GCS, NFS
- `base_path`: Base path or bucket name
- `config`: JSON configuration for storage-specific settings
- `status`: ACTIVE, READONLY, DISABLED
- `max_file_size`: Maximum file size in bytes (nullable)
- `allowed_mime_types`: Array of allowed MIME types (nullable)
- `is_default`: Whether this is the default storage

### managements.files
File metadata and tracking:
- `id`: Internal ID (BIGINT)
- `public_id`: UUID v7 for external reference
- `storage_id`: Foreign key to file_storages
- `relative_path`: Path within storage (e.g., 2024/12/05/{uuid}.pdf)
- `original_filename`: Original filename from upload
- `mime_type`: Detected MIME type
- `file_size`: File size in bytes
- `checksum_sha256`: SHA-256 checksum for integrity
- `uploaded_by`: Foreign key to users (nullable)
- `metadata`: Custom JSON metadata
- `download_count`: Number of downloads
- `last_accessed_at`: Last download timestamp
- `is_public`: Public access flag (default: false)
- `is_deleted`: Soft delete flag
- `deleted_at`: Deletion timestamp

## File Storage

### Path Generation
Files are stored with the following path structure:
```
{base_path}/{year}/{month}/{day}/{uuid}{ext}
```

Example: `./uploads/2024/12/05/550e8400-e29b-41d4-a716-446655440000.pdf`

### Storage Abstraction
The `Storage` interface allows for multiple backend implementations:

```go
type Storage interface {
    Save(ctx context.Context, reader io.Reader, relativePath string) error
    Get(ctx context.Context, relativePath string) (io.ReadCloser, error)
    Delete(ctx context.Context, relativePath string) error
}
```

### Local Storage Implementation
The default `LocalStorage` implementation:
- Creates directories automatically
- Stores files in the filesystem
- Handles file cleanup on deletion

### Future Storage Backends
The architecture supports:
- **S3**: Amazon S3 or S3-compatible storage
- **Azure Blob**: Azure Blob Storage
- **GCS**: Google Cloud Storage
- **NFS**: Network File System

## Security

### Authentication & Authorization
- All endpoints require JWT authentication
- File operations are restricted to the file owner
- Admin override could be added via RBAC middleware

### Input Validation
- **File Size**: Validated against storage configuration
- **MIME Type**: Optional whitelist validation
- **Filename**: Sanitized to prevent path traversal
  - Removes path separators
  - Replaces dangerous characters
  - Limits length to 255 characters

### Integrity
- **SHA-256 Checksum**: Calculated during upload
- **Returned in Headers**: X-Content-SHA256 header on download
- **Database Storage**: Stored for verification

### File Sanitization
Filenames are sanitized using the following rules:
- Only alphanumeric characters, spaces, dots, dashes, and underscores are allowed
- Multiple spaces are collapsed to a single space
- Path separators are removed
- Leading/trailing spaces and dots are trimmed
- Maximum length of 255 characters

## Configuration

### Environment Variables
```bash
# File storage path for uploaded files
FILE_STORAGE_PATH=./uploads
```

### Storage Configuration
Storage backends are configured in the database (`file_storages` table). The system uses the storage marked as `is_default=true`.

Example storage configuration:
```sql
INSERT INTO managements.file_storages (
    name, storage_type, base_path, status, is_default, max_file_size
) VALUES (
    'default-local', 'LOCAL', './uploads', 'ACTIVE', true, 104857600
);
```

## API Examples

### Upload a File
```bash
curl -X POST http://localhost:8080/files \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@document.pdf" \
  -F 'metadata={"category":"invoice","year":2024}'
```

Response:
```json
{
  "id": "01912345-6789-7abc-def0-123456789abc",
  "original_filename": "document.pdf",
  "mime_type": "application/pdf",
  "file_size": 1048576,
  "checksum_sha256": "abc123...",
  "download_url": "/files/01912345-6789-7abc-def0-123456789abc/download",
  "message": "File uploaded successfully"
}
```

### Download a File
```bash
curl -X GET http://localhost:8080/files/01912345-6789-7abc-def0-123456789abc/download \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -o downloaded_file.pdf
```

### Get File Information
```bash
curl -X GET http://localhost:8080/files/01912345-6789-7abc-def0-123456789abc \
  -H "Authorization: Bearer YOUR_TOKEN"
```

Response:
```json
{
  "id": "01912345-6789-7abc-def0-123456789abc",
  "original_filename": "document.pdf",
  "mime_type": "application/pdf",
  "file_size": 1048576,
  "checksum_sha256": "abc123...",
  "download_url": "/files/01912345-6789-7abc-def0-123456789abc/download",
  "download_count": 5,
  "is_public": false,
  "metadata": {"category":"invoice","year":2024},
  "uploaded_by": "01912345-6789-7abc-def0-987654321abc",
  "uploaded_by_name": "John Doe",
  "created_at": "2024-12-05T10:30:00Z",
  "updated_at": "2024-12-05T10:30:00Z",
  "last_accessed_at": "2024-12-05T14:22:00Z"
}
```

### List My Files
```bash
curl -X GET "http://localhost:8080/files?page=1&limit=20" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

Response:
```json
{
  "data": [
    {
      "id": "01912345-6789-7abc-def0-123456789abc",
      "original_filename": "document.pdf",
      "mime_type": "application/pdf",
      "file_size": 1048576,
      "checksum_sha256": "abc123...",
      "download_url": "/files/01912345-6789-7abc-def0-123456789abc/download",
      "download_count": 5,
      "is_public": false,
      "created_at": "2024-12-05T10:30:00Z",
      "updated_at": "2024-12-05T10:30:00Z"
    }
  ],
  "page": 1,
  "limit": 20,
  "total_count": 42,
  "total_pages": 3
}
```

### Update File Metadata
```bash
curl -X PUT http://localhost:8080/files/01912345-6789-7abc-def0-123456789abc/metadata \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"metadata":{"category":"invoice","year":2024,"processed":true}}'
```

### Delete a File
```bash
curl -X DELETE http://localhost:8080/files/01912345-6789-7abc-def0-123456789abc \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Error Handling

The files domain uses custom error types:
- `ErrFileNotFound`: File not found in database
- `ErrFileTooLarge`: File exceeds maximum size
- `ErrInvalidMimeType`: MIME type not allowed
- `ErrUnauthorized`: User not authorized for operation
- `ErrStorageNotAvailable`: Storage backend not available

HTTP status codes:
- `201 Created`: File uploaded successfully
- `200 OK`: Successful operation
- `400 Bad Request`: Invalid request (file missing, invalid metadata)
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Not the file owner
- `404 Not Found`: File not found
- `413 Payload Too Large`: File too large
- `500 Internal Server Error`: Server error

## Performance Considerations

### Streaming
- Files are streamed during upload and download
- SHA-256 calculation uses `io.TeeReader` for single-pass processing
- No full file buffering in memory

### Async Operations
- Download counter is incremented asynchronously (best effort)
- Physical file deletion is attempted but doesn't fail the soft delete

### Database Indexes
Recommended indexes for optimal performance:
```sql
-- Query by public_id (most common)
CREATE INDEX idx_files_public_id ON managements.files(public_id) WHERE is_deleted = false;

-- List files by uploader
CREATE INDEX idx_files_uploaded_by ON managements.files(uploaded_by, created_at DESC) WHERE is_deleted = false;

-- Storage lookup
CREATE INDEX idx_file_storages_default ON managements.file_storages(is_default) WHERE is_deleted = false;
```

## Future Enhancements

### Potential Features
1. **Public File Access**: Support for `is_public` files without authentication
2. **File Sharing**: Generate temporary signed URLs
3. **File Versioning**: Track file versions and revisions
4. **Thumbnails**: Auto-generate thumbnails for images
5. **Virus Scanning**: Integration with ClamAV or similar
6. **Compression**: Automatic compression for eligible file types
7. **CDN Integration**: Serve files through CDN
8. **Quota Management**: Per-user storage quotas
9. **Batch Operations**: Upload/download multiple files
10. **Search**: Full-text search on filename and metadata

### Storage Backends
Future implementations could add:
- S3-compatible storage (AWS S3, MinIO, etc.)
- Azure Blob Storage
- Google Cloud Storage
- Network File System (NFS)
- SFTP/FTP storage

## Testing

### Integration Tests
Create integration tests similar to other domains:
```go
// Example test structure
func TestFileUpload(t *testing.T) {
    // Setup test database
    // Create test file
    // Upload file
    // Verify file exists in storage
    // Verify database record
    // Cleanup
}
```

### Test Coverage
Key areas to test:
- File upload with valid/invalid files
- MIME type validation
- File size limits
- Filename sanitization
- Download functionality
- Ownership validation
- Soft delete behavior
- Checksum calculation
- Storage backend operations

## Monitoring & Logging

### Metrics to Track
- Upload success/failure rate
- Download frequency
- Storage usage by user
- Average file size
- MIME type distribution
- Error rates by type

### Logging
Key events to log:
- File uploads (user, file size, MIME type)
- File deletions (user, file ID)
- Storage errors
- Large file uploads (above threshold)
- Failed authorization attempts

## Maintenance

### Cleanup Tasks
Consider implementing scheduled tasks:
1. **Hard Delete**: Remove soft-deleted files after retention period
2. **Orphan Cleanup**: Remove files without database records
3. **Storage Verification**: Verify checksums periodically
4. **Statistics**: Generate usage reports
