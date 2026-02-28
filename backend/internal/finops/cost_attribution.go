package finops

import (
	"context"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
)

// ============================================================================
// COST ANALYSIS HELPERS
// ============================================================================

// EstimateMonthlyCost projects user's monthly cost based on current usage
func EstimateMonthlyCost(settings Settings, snapshot *db.FinOpsSnapshot) float64 {
	daysElapsed := time.Now().Day()
	if daysElapsed == 0 {
		daysElapsed = 1
	}

	dailyStorageCost := (float64(snapshot.TotalActiveStorageBytes) / (1024 * 1024 * 1024)) * settings.StorageCostPerGBMonth / 30
	estimatedMonthlyStorage := dailyStorageCost * float64(30)

	return estimatedMonthlyStorage
}

// CheckCostThreshold determines if user is approaching budget limit
func CheckCostThreshold(settings Settings, currentCostUSD float64) (percentage float64, approachingLimit bool) {
	if settings.MonthlyBudgetUSD <= 0 {
		return 0, false
	}

	percentage = (currentCostUSD / settings.MonthlyBudgetUSD) * 100
	// Warn when approaching 80% of budget
	approachingLimit = percentage >= 80

	return percentage, approachingLimit
}

// ============================================================================
// INTEGRATION WITH FINOPS POLICY
// ============================================================================

// EnhancedBudgetCheck integrates cost attribution data with upload guard
// Returns blocking decision based on detailed cost tracking
func EnhancedBudgetCheck(ctx context.Context, database *db.Database, userID string, settings Settings) (*BudgetDecision, error) {
	// Get current month cost from attribution
	currentCost, err := database.GetUserCurrentMonthCost(ctx, userID)
	if err != nil {
		return nil, err
	}

	utilization := currentCost / settings.MonthlyBudgetUSD
	decision := &BudgetDecision{
		ProjectedCostUSD: currentCost,
		UtilizationRatio: utilization,
	}

	if utilization >= settings.BudgetGuardRatio {
		decision.Block = true
		decision.Reason = "user cost allocation exceeds budget threshold"
	}

	return decision, nil
}
