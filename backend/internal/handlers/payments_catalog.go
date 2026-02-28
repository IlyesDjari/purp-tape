package handlers

import (
	"encoding/json"
	"net/http"
)

// GetPricingTiers returns available tiers for client purchase flows.
func (h *PaymentHandlers) GetPricingTiers(w http.ResponseWriter, r *http.Request) {
	tiers := []map[string]interface{}{
		{
			"id":              "free",
			"name":            "Free",
			"price_monthly":   0,
			"storage_gb":      10,
			"features":        []string{"Upload 10 projects", "Basic analytics"},
			"stripe_price_id": "",
		},
		{
			"id":              "pro",
			"name":            "Pro",
			"price_monthly":   4.99,
			"storage_gb":      100,
			"features":        []string{"Unlimited projects", "Advanced analytics", "Collaborate with 5 people"},
			"stripe_price_id": "price_pro_monthly",
		},
		{
			"id":              "pro_plus",
			"name":            "Pro+",
			"price_monthly":   9.99,
			"storage_gb":      500,
			"features":        []string{"Unlimited everything", "Premium support", "Advanced collaboration"},
			"stripe_price_id": "price_proplus_monthly",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"tiers": tiers})
}
