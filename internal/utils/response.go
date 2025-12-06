package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
)

// ErrorResponse represents an error response structure
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// RespondJSON writes a JSON response
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// If encoding fails, log the error and try to send a plain text response
		log.Printf("[ERROR] Failed to encode JSON response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// RespondError writes an error response with logging for server errors (5xx)
func RespondError(w http.ResponseWriter, r *http.Request, status int, errType, message string) {
	// Log detailed error information for server errors (500-599)
	if status >= 500 {
		log.Printf("[ERROR 500] %s %s - Error: %s - Message: %s - RemoteAddr: %s - UserAgent: %s",
			r.Method, r.URL.Path, errType, message, r.RemoteAddr, r.UserAgent())
	}

	RespondJSON(w, status, ErrorResponse{
		Error:   errType,
		Message: message,
	})
}

// RespondInternalError is a convenience function for 500 Internal Server Error with detailed logging
func RespondInternalError(w http.ResponseWriter, r *http.Request, err error, userMessage string) {
	// Log the full error with stack trace for internal server errors
	log.Printf("[ERROR 500] %s %s - Internal Error: %v - UserMessage: %s - RemoteAddr: %s - UserAgent: %s",
		r.Method, r.URL.Path, err, userMessage, r.RemoteAddr, r.UserAgent())

	// In development, you might want to include the stack trace
	// Uncomment the following line if needed for debugging:
	// log.Printf("Stack trace:\n%s", string(debug.Stack()))

	// Don't expose internal error details to the client in production
	message := userMessage
	if message == "" {
		message = "An internal error occurred"
	}

	RespondJSON(w, http.StatusInternalServerError, ErrorResponse{
		Error:   "Internal Server Error",
		Message: message,
	})
}

// RespondInternalErrorWithStack is similar to RespondInternalError but always includes stack trace
// Use this for critical errors where you need maximum debugging information
func RespondInternalErrorWithStack(w http.ResponseWriter, r *http.Request, err error, userMessage string) {
	// Log the full error with stack trace
	log.Printf("[ERROR 500 WITH STACK] %s %s - Internal Error: %v - UserMessage: %s - RemoteAddr: %s - UserAgent: %s",
		r.Method, r.URL.Path, err, userMessage, r.RemoteAddr, r.UserAgent())
	log.Printf("Stack trace:\n%s", string(debug.Stack()))

	// Don't expose internal error details to the client in production
	message := userMessage
	if message == "" {
		message = "An internal error occurred"
	}

	RespondJSON(w, http.StatusInternalServerError, ErrorResponse{
		Error:   "Internal Server Error",
		Message: message,
	})
}

// LogError logs an error without sending a response (useful when response is already sent)
func LogError(r *http.Request, status int, err error, context string) {
	level := "ERROR"
	if status >= 500 {
		level = "ERROR 500"
	} else if status >= 400 {
		level = "WARN"
	}

	log.Printf("[%s] %s %s - Status: %d - Error: %v - Context: %s - RemoteAddr: %s",
		level, r.Method, r.URL.Path, status, err, context, r.RemoteAddr)
}

// LogErrorf logs a formatted error message
func LogErrorf(r *http.Request, status int, format string, args ...interface{}) {
	level := "ERROR"
	if status >= 500 {
		level = "ERROR 500"
	} else if status >= 400 {
		level = "WARN"
	}

	message := fmt.Sprintf(format, args...)
	log.Printf("[%s] %s %s - Status: %d - %s - RemoteAddr: %s",
		level, r.Method, r.URL.Path, status, message, r.RemoteAddr)
}
