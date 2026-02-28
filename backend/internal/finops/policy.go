package finops

import (
	"context"
	"os"
	"strconv"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
)

type Settings struct {
	StorageCostPerGBMonth float64
	MonthlyBudgetUSD      float64
	BudgetGuardEnabled    bool
	UploadBlockEnabled    bool
	BudgetGuardRatio      float64
}

type BudgetDecision struct {
	Block              bool
	Reason             string
	ProjectedCostUSD   float64
	UtilizationRatio   float64
	Snapshot           *db.FinOpsSnapshot
}

func LoadSettingsFromEnv() Settings {
	return Settings{
		StorageCostPerGBMonth: getEnvFloat("FINOPS_STORAGE_COST_PER_GB_MONTH", 0.015),
		MonthlyBudgetUSD:      getEnvFloat("FINOPS_MONTHLY_BUDGET_USD", 25.0),
		BudgetGuardEnabled:    getEnvBool("FINOPS_BUDGET_GUARD_ENABLED", false),
		UploadBlockEnabled:    getEnvBool("FINOPS_UPLOAD_BLOCK_ENABLED", false),
		BudgetGuardRatio:      getEnvFloat("FINOPS_BUDGET_GUARD_RATIO", 1.0),
	}
}

func EvaluateUploadGuard(ctx context.Context, database *db.Database, incomingBytes int64) (*BudgetDecision, error) {
	settings := LoadSettingsFromEnv()
	if !settings.UploadBlockEnabled || settings.MonthlyBudgetUSD <= 0 || settings.BudgetGuardRatio <= 0 {
		return &BudgetDecision{}, nil
	}

	snapshot, err := database.GetFinOpsSnapshot(ctx, settings.StorageCostPerGBMonth)
	if err != nil {
		return nil, err
	}

	projectedEstimatedCost := (float64(snapshot.TotalActiveStorageBytes+incomingBytes) / (1024 * 1024 * 1024)) * settings.StorageCostPerGBMonth
	projectedCost := projectedEstimatedCost
	if snapshot.ActualMonthlyCostUSD > projectedCost {
		projectedCost = snapshot.ActualMonthlyCostUSD
	}

	utilization := projectedCost / settings.MonthlyBudgetUSD
	decision := &BudgetDecision{
		ProjectedCostUSD: projectedCost,
		UtilizationRatio: utilization,
		Snapshot:         snapshot,
	}
	if utilization >= settings.BudgetGuardRatio {
		decision.Block = true
		decision.Reason = "projected cost exceeds budget threshold"
	}

	return decision, nil
}

func EvaluateExpensiveJobGuard(ctx context.Context, database *db.Database) (*BudgetDecision, error) {
	settings := LoadSettingsFromEnv()
	if !settings.BudgetGuardEnabled || settings.MonthlyBudgetUSD <= 0 || settings.BudgetGuardRatio <= 0 {
		return &BudgetDecision{}, nil
	}

	snapshot, err := database.GetFinOpsSnapshot(ctx, settings.StorageCostPerGBMonth)
	if err != nil {
		return nil, err
	}

	utilization := snapshot.GoverningMonthlyCostUSD / settings.MonthlyBudgetUSD
	decision := &BudgetDecision{
		ProjectedCostUSD: snapshot.GoverningMonthlyCostUSD,
		UtilizationRatio: utilization,
		Snapshot:         snapshot,
	}

	if utilization >= settings.BudgetGuardRatio {
		decision.Block = true
		decision.Reason = "budget threshold reached"
		return decision, nil
	}

	if snapshot.FailedCleanupJobs > 0 && snapshot.PendingCleanupBytes > 5*1024*1024*1024 {
		decision.Block = true
		decision.Reason = "cleanup backlog too high"
	}

	return decision, nil
}

func getEnvFloat(key string, defaultValue float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
