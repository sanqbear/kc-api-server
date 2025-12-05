package files

import "errors"

var (
	// ErrFileNotFound is returned when a file is not found
	ErrFileNotFound = errors.New("file not found")

	// ErrFileTooLarge is returned when a file exceeds the maximum allowed size
	ErrFileTooLarge = errors.New("file size exceeds maximum allowed size")

	// ErrInvalidMimeType is returned when a file's MIME type is not allowed
	ErrInvalidMimeType = errors.New("file mime type is not allowed")

	// ErrUnauthorized is returned when a user attempts to perform an action they are not authorized for
	ErrUnauthorized = errors.New("unauthorized action")

	// ErrStorageNotAvailable is returned when the storage backend is not available
	ErrStorageNotAvailable = errors.New("storage backend is not available")
)
