package finops

import (
	"testing"
)

func TestLoadSettingsFromEnv(t *testing.T) {
	settings := LoadSettingsFromEnv()

	if settings.StorageCostPerGBMonth <= 0 {
		t.Errorf("expected positive StorageCostPerGBMonth, got %v", settings.StorageCostPerGBMonth)
	}

	if settings.MonthlyBudgetUSD <= 0 {
		t.Errorf("expected positive MonthlyBudgetUSD, got %v", settings.MonthlyBudgetUSD)
	}

	if settings.BudgetGuardRatio <= 0 {
		t.Errorf("expected positive BudgetGuardRatio, got %v", settings.BudgetGuardRatio)
	}
}

func TestBudgetDecision_Fields(t *testing.T) {
	decision := &BudgetDecision{
		Block:              true,
		Reason:             "budget exceeded",
		ProjectedCostUSD:   50.0,
		UtilizationRatio:   1.2,
	}

	if !decision.Block {
		t.Errorf("expected Block=true, got false")
	}

	if decision.Reason != "budget exceeded" {
		t.Errorf("expected reason 'budget exceeded', got %s", decision.Reason)
	}

	if decision.ProjectedCostUSD != 50.0 {
		t.Errorf("expected cost 50.0, got %v", decision.ProjectedCostUSD)
	}

	if decision.UtilizationRatio != 1.2 {
		t.Errorf("expected ratio 1.2, got %v", decision.UtilizationRatio)
	}
}

func TestBudgetDecision_Allowed(t *testing.T) {
	decision := &BudgetDecision{
		Block:              false,
		ProjectedCostUSD:   10.0,
		UtilizationRatio:   0.5,
	}

	if decision.Block {
		t.Errorf("expected Block=false for low utilization")
	}

	if decision.UtilizationRatio > 1.0 {
		t.Errorf("utilization should be under 100 percent")
	}
}
