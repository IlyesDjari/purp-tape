package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/finops"
	"github.com/IlyesDjari/purp-tape/backend/internal/helpers"
)

type FinOpsHandlers struct {
	db  *db.Database
	log *slog.Logger
}

func NewFinOpsHandlers(database *db.Database, log *slog.Logger) *FinOpsHandlers {
	return &FinOpsHandlers{db: database, log: log}
}

func (h *FinOpsHandlers) IngestCostEvent(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(os.Getenv("FINOPS_COST_INGEST_TOKEN"))
	if token == "" {
		helpers.WriteForbidden(w, "finops ingestion disabled")
		return
	}

	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	expected := "Bearer " + token
	if authHeader != expected {
		helpers.WriteUnauthorized(w)
		return
	}

	var req struct {
		Source     string                 `json:"source"`
		Service    string                 `json:"service"`
		Category   string                 `json:"category"`
		AmountUSD  float64                `json:"amount_usd"`
		OccurredAt string                 `json:"occurred_at"`
		Metadata   map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteBadRequest(w, "invalid request body")
		return
	}

	if req.AmountUSD < 0 {
		helpers.WriteBadRequest(w, "amount_usd must be non-negative")
		return
	}

	occurredAt := time.Now().UTC()
	if strings.TrimSpace(req.OccurredAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.OccurredAt)
		if err != nil {
			helpers.WriteBadRequest(w, "occurred_at must be RFC3339")
			return
		}
		occurredAt = parsed.UTC()
	}

	event := &db.FinOpsCostEvent{
		Source:     strings.TrimSpace(req.Source),
		Service:    strings.TrimSpace(req.Service),
		Category:   strings.TrimSpace(req.Category),
		AmountUSD:  req.AmountUSD,
		OccurredAt: occurredAt,
		Metadata:   req.Metadata,
	}

	if err := h.db.CreateFinOpsCostEvent(r.Context(), event); err != nil {
		h.log.Error("failed to create FinOps cost event", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	helpers.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"id":          event.ID,
		"source":      event.Source,
		"service":     event.Service,
		"category":    event.Category,
		"amount_usd":  event.AmountUSD,
		"occurred_at": event.OccurredAt,
	})
}

func (h *FinOpsHandlers) GetSummary(w http.ResponseWriter, r *http.Request) {
	settings := finops.LoadSettingsFromEnv()

	days := 30
	if value := strings.TrimSpace(r.URL.Query().Get("days")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 || parsed > 365 {
			helpers.WriteBadRequest(w, "days must be an integer between 1 and 365")
			return
		}
		days = parsed
	}

	summary, err := h.db.GetFinOpsMonthlySummary(r.Context(), days, settings.StorageCostPerGBMonth, settings.MonthlyBudgetUSD)
	if err != nil {
		h.log.Error("failed to get FinOps summary", "error", err)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"days":                 summary.Days,
		"actual_cost_usd":      summary.ActualCostUSD,
		"estimated_storage_usd": summary.StorageEstimatedUSD,
		"governing_cost_usd":   summary.GoverningCostUSD,
		"budget_usd":           summary.BudgetUSD,
		"utilization_ratio":    summary.UtilizationRatio,
	})
}
