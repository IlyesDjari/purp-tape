package handlers

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
)

func TestProjectHandlers_ListProjects_Unauthorized(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	var mockDB *db.Database // Would be mocked in real tests
	handlers := NewProjectHandlers(mockDB, logger)

	req := httptest.NewRequest("GET", "/projects", nil)
	w := httptest.NewRecorder()

	handlers.ListProjects(w, req)

	if w.Code != http.StatusUnauthorized && w.Code != http.StatusBadRequest {
		t.Errorf("expected 401 or 400, got %d", w.Code)
	}
}

func TestProjectHandlers_CreateProject_InvalidRequest(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	var mockDB *db.Database
	handlers := NewProjectHandlers(mockDB, logger)

	// Empty body
	req := httptest.NewRequest("POST", "/projects", bytes.NewReader([]byte("")))
	w := httptest.NewRecorder()

	handlers.CreateProject(w, req)

	if w.Code != http.StatusBadRequest && w.Code != http.StatusUnauthorized {
		t.Errorf("expected error status, got %d", w.Code)
	}
}

func TestHealthHandlers_GetHealth(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	var mockDB *db.Database

	handlers := NewHealthHandlers(mockDB, nil, logger)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handlers.GetHealth(w, req)

	// Health check should not require auth
	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 200 or 503, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") == "" {
		t.Errorf("expected Content-Type header")
	}
}

func TestShareHandlers_GenerateShareLink_InvalidProject(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	var mockDB *db.Database
	handlers := NewShareHandlers(mockDB, logger)

	req := httptest.NewRequest("POST", "/projects/invalid-id/shares", nil)
	w := httptest.NewRecorder()

	handlers.GenerateShareLink(w, req)

	// Should require auth or return error
	if w.Code < 400 {
		t.Errorf("expected error status, got %d", w.Code)
	}
}
