package main

import (
	"context"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/DiegoGarciaCo/websitesAPI/internal/auth"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type contextKey string

const (
	UserKey      contextKey = "user" // Changed to store full User
	requestIDKey contextKey = "requestID"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Generate and set request ID before calling the next handler
		requestID := uuid.New().String()
		r = r.WithContext(context.WithValue(r.Context(), requestIDKey, requestID))

		// Call the next handler
		next.ServeHTTP(wrappedWriter, r)

		duration := time.Since(start)

		// Handle client IP with load balancer
		clientIP := r.Header.Get("X-Forwarded-For")
		if clientIP == "" {
			clientIP = r.Header.Get("X-Real-IP")
		}
		if clientIP == "" {
			clientIP = r.RemoteAddr
		}

		// Build log fields
		fields := logrus.Fields{
			"request_id": requestID,
			"method":     r.Method,
			"path":       r.URL.Path, // or r.URL.String() for full URL
			"status":     wrappedWriter.statusCode,
			"duration":   duration.Milliseconds(),
			"client_ip":  clientIP,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		}

		// Log based on status code
		if wrappedWriter.statusCode >= 400 {
			logrus.WithFields(fields).Error("Request failed")
		} else {
			logrus.WithFields(fields).Info("Request processed")
		}
	})
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logrus.WithFields(logrus.Fields{
					"request_id": r.Context().Value(requestIDKey),
				}).Errorf("Recovery: %v, stack trace: %s", err, string(debug.Stack()))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Authentication middleware with role support
func (cfg *apiCfg) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		token, err := req.Cookie("token")
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid token", err)
			return
		}

		csrf := req.Header.Get("X-CSRF-TOKEN")
		if csrf == "" {
			respondWithError(w, http.StatusUnauthorized, "Invalid csrf token", nil)
			return
		}

		dbCsrf, err := cfg.DB.GetCsfToken(req.Context(), csrf)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid csrf token", err)
			return
		}

		userID, err := auth.ValidateJWT(token.Value, cfg.secret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid token", err)
			return
		}

		if dbCsrf.UserID != userID {
			respondWithError(w, http.StatusUnauthorized, "Invalid csrf token", nil)
			return
		}

		_, err = cfg.DB.GetUserByID(req.Context(), userID)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "User not found", err)
			return
		}

		next(w, req)
	}
}
