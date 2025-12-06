package server

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

// RecoverMiddleware recovers from panics and logs the error with stack trace
func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				log.Printf("[PANIC RECOVERED] %s %s - Error: %v\n", r.Method, r.URL.Path, err)
				log.Printf("Stack trace:\n%s", string(debug.Stack()))

				// Return 500 Internal Server Error
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"Internal Server Error","message":"An unexpected error occurred"}`))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// responseWriter is a wrapper around http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// DetailedLoggerMiddleware logs requests with additional details for error responses
func DetailedLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		// Log all requests, with extra detail for errors
		logMessage := fmt.Sprintf("%s %s - Status: %d - Duration: %v",
			r.Method, r.URL.Path, wrapped.statusCode, duration)

		// Add detailed logging for 4xx and 5xx errors
		if wrapped.statusCode >= 400 {
			logMessage += fmt.Sprintf(" - RemoteAddr: %s - UserAgent: %s",
				r.RemoteAddr, r.UserAgent())

			// Extra detail for server errors
			if wrapped.statusCode >= 500 {
				log.Printf("[ERROR] %s", logMessage)
			} else {
				log.Printf("[WARN] %s", logMessage)
			}
		} else {
			log.Printf("[INFO] %s", logMessage)
		}
	})
}
