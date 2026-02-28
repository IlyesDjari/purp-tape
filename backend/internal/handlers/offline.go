package handlers

import (
	"log/slog"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/storage"
)

// OfflineHandlers manages offline download functionality.
type OfflineHandlers struct {
	db  *db.Database
	r2  *storage.R2Client
	log *slog.Logger
}

// NewOfflineHandlers creates offline handler.
func NewOfflineHandlers(database *db.Database, r2Client *storage.R2Client, log *slog.Logger) *OfflineHandlers {
	return &OfflineHandlers{db: database, r2: r2Client, log: log}
}
