package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IlyesDjari/purp-tape/backend/internal/auth"
)

func TestAuthMiddleware_MissingAuthorizationHeader(t *testing.T) {
	validator := auth.NewValidator("", "", "", "")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := AuthMiddleware(validator, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "missing authorization header") {
		t.Fatalf("expected missing authorization header error, got: %s", rr.Body.String())
	}
}

func TestAuthMiddleware_InvalidAuthorizationHeader(t *testing.T) {
	validator := auth.NewValidator("", "", "", "")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := AuthMiddleware(validator, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	req.Header.Set("Authorization", "Token abc123")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "invalid authorization header") {
		t.Fatalf("expected invalid authorization header error, got: %s", rr.Body.String())
	}
}

func TestCORSMiddleware_AllowedOriginSetsHeaders(t *testing.T) {
	handler := CORSMiddleware([]string{"http://localhost:3000"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected Access-Control-Allow-Origin header to be set, got: %q", got)
	}
}

func TestCORSMiddleware_DisallowedOriginDoesNotSetHeaders(t *testing.T) {
	handler := CORSMiddleware([]string{"http://localhost:3000"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://evil.local")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected Access-Control-Allow-Origin header to be empty, got: %q", got)
	}
}

func TestCORSMiddleware_OptionsPreflightReturnsNoContent(t *testing.T) {
	handler := CORSMiddleware([]string{"http://localhost:3000"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/projects", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestSanitizePathForLogs_RedactsShareToken(t *testing.T) {
	path := "/share/very-secret-token"
	redacted := sanitizePathForLogs(path)

	if strings.Contains(redacted, "very-secret-token") {
		t.Fatalf("expected share token to be redacted, got: %s", redacted)
	}

	if redacted != "/share/[REDACTED]" {
		t.Fatalf("unexpected redacted path: %s", redacted)
	}
}
