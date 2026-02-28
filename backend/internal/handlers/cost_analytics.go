package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/helpers"
)

// CostAnalyticsHandlers provides cost analysis and attribution endpoints
type CostAnalyticsHandlers struct {
	db  *db.Database
	log *slog.Logger
}

// NewCostAnalyticsHandlers creates cost analytics handlers
func NewCostAnalyticsHandlers(database *db.Database, log *slog.Logger) *CostAnalyticsHandlers {
	return &CostAnalyticsHandlers{
		db:  database,
		log: log,
	}
}

// ============================================================================
// ENDPOINT: GET /users/{user_id}/costs/current
// Returns user's cost for current month
// ============================================================================
func (h *CostAnalyticsHandlers) GetCurrentMonthCost(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	if userID == "" {
		helpers.WriteBadRequest(w, "missing user_id")
		return
	}

	// Verify user can only see their own costs (unless admin)
	authUserID, _ := helpers.GetUserID(r)
	if authUserID != userID {
		helpers.WriteForbidden(w, "cannot view other user's costs")
		return
	}

	cost, err := h.db.GetUserCurrentMonthCost(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to get user current month cost", "error", err, "user_id", userID)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	response := map[string]interface{}{
		"user_id": userID,
		"month":   time.Now().Format("2006-01"),
		"cost_usd": fmt.Sprintf("%.2f", cost),
		"status": "active",
	}

	helpers.WriteJSON(w, http.StatusOK, response)
}

// ============================================================================
// ENDPOINT: GET /users/{user_id}/costs/history?months=3
// Returns user's monthly cost history
// ============================================================================
func (h *CostAnalyticsHandlers) GetCostHistory(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	if userID == "" {
		helpers.WriteBadRequest(w, "missing user_id")
		return
	}

	authUserID, _ := helpers.GetUserID(r)
	if authUserID != userID {
		helpers.WriteForbidden(w, "cannot view other user's costs")
		return
	}

	months := 6
	if m := r.URL.Query().Get("months"); m != "" {
		_, _ = fmt.Sscanf(m, "%d", &months)
	}
	if months < 1 || months > 24 {
		months = 6
	}

	history, err := h.db.GetUserMonthlyCosts(r.Context(), userID, months)
	if err != nil {
		h.log.Error("failed to get user cost history", "error", err, "user_id", userID)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	type HistoryResponse struct {
		Month                string  `json:"month"`
		StorageCostUSD       float64 `json:"storage_cost_usd"`
		APICostUSD           float64 `json:"api_cost_usd"`
		TransferCostUSD      float64 `json:"transfer_cost_usd"`
		TotalCostUSD         float64 `json:"total_cost_usd"`
		ProjectsCount        int     `json:"projects_count"`
		LastActivityDate     string  `json:"last_activity_date"`
	}

	var response []HistoryResponse
	for _, h := range history {
		response = append(response, HistoryResponse{
			Month:           h.BillingMonth.Format("2006-01"),
			StorageCostUSD:  h.TotalStorageCostUSD,
			APICostUSD:      h.TotalAPICostUSD,
			TransferCostUSD: h.TotalTransferCostUSD,
			TotalCostUSD:    h.TotalCostUSD,
			ProjectsCount:   h.ProjectsCount,
			LastActivityDate: h.LastActivityDate.Format("2006-01-02"),
		})
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"user_id": userID,
		"history": response,
	})
}

// ============================================================================
// ENDPOINT: GET /users/{user_id}/costs/breakdown
// Returns per-project cost breakdown for current month
// ============================================================================
func (h *CostAnalyticsHandlers) GetCostBreakdown(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	if userID == "" {
		helpers.WriteBadRequest(w, "missing user_id")
		return
	}

	authUserID, _ := helpers.GetUserID(r)
	if authUserID != userID {
		helpers.WriteForbidden(w, "cannot view other user's costs")
		return
	}

	breakdown, err := h.db.GetUserCostBreakdown(r.Context(), userID, time.Now())
	if err != nil {
		h.log.Error("failed to get user cost breakdown", "error", err, "user_id", userID)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	type BreakdownResponse struct {
		ProjectID       string  `json:"project_id"`
		ProjectName     string  `json:"project_name"`
		StorageCostUSD  float64 `json:"storage_cost_usd"`
		APICostUSD      float64 `json:"api_cost_usd"`
		TransferCostUSD float64 `json:"transfer_cost_usd"`
		TotalCostUSD    float64 `json:"total_cost_usd"`
		CostPercentage  float64 `json:"cost_percentage"`
	}

	var response []BreakdownResponse
	for _, b := range breakdown {
		response = append(response, BreakdownResponse{
			ProjectID:       b.ProjectID,
			ProjectName:     b.ProjectName,
			StorageCostUSD:  b.StorageCostUSD,
			APICostUSD:      b.APICostUSD,
			TransferCostUSD: b.TransferCostUSD,
			TotalCostUSD:    b.TotalCostUSD,
			CostPercentage:  b.CostPercentage,
		})
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":   userID,
		"month":     time.Now().Format("2006-01"),
		"breakdown": response,
	})
}

// ============================================================================
// ENDPOINT: GET /users/{user_id}/invoice?month=2026-02
// Generates invoice for specified month
// ============================================================================
func (h *CostAnalyticsHandlers) GetUserInvoice(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	if userID == "" {
		helpers.WriteBadRequest(w, "missing user_id")
		return
	}

	authUserID, _ := helpers.GetUserID(r)
	if authUserID != userID {
		helpers.WriteForbidden(w, "cannot view other user's invoice")
		return
	}

	monthStr := r.URL.Query().Get("month")
	if monthStr == "" {
		monthStr = time.Now().Format("2006-01")
	}

	month, err := time.Parse("2006-01", monthStr)
	if err != nil {
		helpers.WriteBadRequest(w, "invalid month format (use YYYY-MM)")
		return
	}

	invoice, err := h.db.GenerateUserInvoice(r.Context(), userID, month)
	if err != nil {
		h.log.Error("failed to generate user invoice", "error", err, "user_id", userID)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	if invoice == nil {
		helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"user_id": userID,
			"month":   monthStr,
			"message": "no charges for this period",
		})
		return
	}

	// Parse breakdown JSON if available
	var breakdown interface{}
	if invoice.BreakdownJSON != "" {
		_ = json.Unmarshal([]byte(invoice.BreakdownJSON), &breakdown)
	}

	response := map[string]interface{}{
		"user_id":           userID,
		"invoice_month":     invoice.InvoiceMonth,
		"total_amount_usd":  fmt.Sprintf("%.2f", invoice.TotalAmountUSD),
		"storage_amount_usd": fmt.Sprintf("%.2f", invoice.StorageAmountUSD),
		"api_amount_usd":    fmt.Sprintf("%.2f", invoice.APIAmountUSD),
		"transfer_amount_usd": fmt.Sprintf("%.2f", invoice.TransferAmountUSD),
		"due_date":          month.AddDate(0, 1, 0).Format("2006-01-02"),
		"status":            "pending",
	}

	if breakdown != nil {
		response["breakdown"] = breakdown
	}

	helpers.WriteJSON(w, http.StatusOK, response)
}

// ============================================================================
// ENDPOINT: GET /projects/{project_id}/costs
// Returns current month cost for a project
// ============================================================================
func (h *CostAnalyticsHandlers) GetProjectCost(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	if projectID == "" {
		helpers.WriteBadRequest(w, "missing project_id")
		return
	}

	cost, err := h.db.GetProjectCurrentMonthCost(r.Context(), projectID)
	if err != nil {
		h.log.Error("failed to get project cost", "error", err, "project_id", projectID)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	response := map[string]interface{}{
		"project_id": projectID,
		"month":      time.Now().Format("2006-01"),
		"cost_usd":   fmt.Sprintf("%.2f", cost),
	}

	helpers.WriteJSON(w, http.StatusOK, response)
}

// ============================================================================
// ENDPOINT: GET /cost-analysis/high-cost-projects?user_id=xxx
// Admin endpoint: identify high-cost projects for a user
// ============================================================================
func (h *CostAnalyticsHandlers) AnalyzeHighCostProjects(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		helpers.WriteBadRequest(w, "missing user_id query parameter")
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		_, _ = fmt.Sscanf(l, "%d", &limit)
	}

	projects, err := h.db.IdentifyHighCostProjects(r.Context(), userID, limit)
	if err != nil {
		h.log.Error("failed to analyze high cost projects", "error", err, "user_id", userID)
		helpers.WriteInternalError(w, h.log, err)
		return
	}

	type ProjectAnalysis struct {
		ProjectID       string  `json:"project_id"`
		ProjectName     string  `json:"project_name"`
		TotalCostUSD    float64 `json:"total_cost_usd"`
		CostPercentage  float64 `json:"cost_percentage"`
		OptimizationTip string  `json:"optimization_tip"`
	}

	var response []ProjectAnalysis
	for i, p := range projects {
		tip := ""
		if p.CostPercentage > 50 {
			tip = "This project consumes over 50% of your usage - consider optimizing storage or archiving old files"
		}

		response = append(response, ProjectAnalysis{
			ProjectID:       p.ProjectID,
			ProjectName:     p.ProjectName,
			TotalCostUSD:    p.TotalCostUSD,
			CostPercentage:  p.CostPercentage,
			OptimizationTip: tip,
		})

		if i >= limit-1 {
			break
		}
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":   userID,
		"month":     time.Now().Format("2006-01"),
		"projects":  response,
	})
}
