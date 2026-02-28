package handlers

import (
	"log/slog"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
)

// PaymentHandlers handles payment webhooks (Stripe, RevenueCat, etc)
type PaymentHandlers struct {
	db  *db.Database
	log *slog.Logger
}

// NewPaymentHandlers creates payment handler
func NewPaymentHandlers(database *db.Database, log *slog.Logger) *PaymentHandlers {
	return &PaymentHandlers{db: database, log: log}
}
