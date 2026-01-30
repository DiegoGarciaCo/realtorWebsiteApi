package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

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

// key type for context
const userIDKey contextKey = "userID"

func VerifySignedCookie(cookieValue, secret string) (string, error) {
	parts := strings.Split(cookieValue, ".")
	if len(parts) != 2 {
		return "", errors.New("invalid cookie format")
	}

	payload := parts[0]
	signature := parts[1]

	// Step 1: URL-decode the signature (handles %2B, %3D, etc.)
	decodedSig, err := url.QueryUnescape(signature)
	if err != nil {
		decodedSig = signature // fallback to raw
	}

	// Step 2: Try standard Base64 decoding
	if sigBytes, err := base64.StdEncoding.DecodeString(decodedSig); err == nil {
		if verifyHMAC(payload, secret, sigBytes) {
			return payload, nil
		}
	}

	// Step 3: Try base64 URL encoding without padding
	if sigBytes, err := base64.RawURLEncoding.DecodeString(decodedSig); err == nil {
		if verifyHMAC(payload, secret, sigBytes) {
			return payload, nil
		}
	}

	return "", errors.New("invalid cookie signature")
}

func verifyHMAC(payload, secret string, signature []byte) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := mac.Sum(nil)
	return hmac.Equal(signature, expected)
}

// Authentication middleware with role support
func (cfg *apiCfg) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookieName := "__Secure-web.session_token"
		if cfg.Env != "Production" {
			cookieName = "web.session_token"
		}

		cookie, err := r.Cookie(cookieName)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Missing session cookie", err)
			return
		}

		// Verify the signed cookie
		token, err := VerifySignedCookie(cookie.Value, cfg.BetterAuthSecret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid session cookie", err)
			return
		}

		// Query the Better Auth sessions table
		dbToken, err := cfg.DB.CheckSessionByID(r.Context(), token)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid session", err)
			return
		}

		// Check if session expired
		if dbToken.ExpiresAt.Before(time.Now()) {
			respondWithError(w, http.StatusUnauthorized, "Session expired", nil)
			return
		}

		// Add userID to request context
		ctx := context.WithValue(r.Context(), userIDKey, dbToken.UserId.String())
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
