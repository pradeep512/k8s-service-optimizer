package api

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const requestIDKey contextKey = "requestID"

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Get request ID from context (added by requestIDMiddleware)
		requestID := getRequestID(r.Context())

		// Create a custom response writer to capture the status code
		lrw := &loggingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call the next handler
		next.ServeHTTP(lrw, r)

		// Log the request
		duration := time.Since(start)
		log.Printf("[%s] %s %s %d %s %s",
			requestID,
			r.Method,
			r.URL.Path,
			lrw.statusCode,
			duration,
			r.RemoteAddr,
		)
	})
}

// loggingResponseWriter wraps http.ResponseWriter to capture status code
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from localhost:3000 (React dashboard)
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requestIDMiddleware adds a unique request ID to each request
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request ID is already set (from header)
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			// Generate a new UUID
			requestID = uuid.New().String()
		}

		// Add to response header
		w.Header().Set("X-Request-ID", requestID)

		// Add to context
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// recoveryMiddleware recovers from panics and returns a 500 error
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				requestID := getRequestID(r.Context())
				log.Printf("[%s] PANIC: %v\n%s", requestID, err, debug.Stack())

				// Return 500 error
				respondWithError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An internal server error occurred")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// getRequestID retrieves the request ID from context
func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return "unknown"
}
