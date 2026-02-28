package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// RevenueCatWebhook handles POST /webhooks/revenuecat.
// Verifies webhook signature with HMAC-SHA256.
func (h *PaymentHandlers) RevenueCatWebhook(w http.ResponseWriter, r *http.Request) {
	endpointSecret := os.Getenv("REVENUECAT_WEBHOOK_SECRET")
	if endpointSecret == "" {
		h.log.Error("revenuecat webhook secret not configured")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("failed to read revenuecat webhook body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if !isValidRevenueCatSignature(r.Header.Get("X-Revenuecat-Signature"), body, endpointSecret) {
		h.log.Warn("revenuecat webhook signature verification failed")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		h.log.Error("failed to parse revenuecat webhook", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	eventData, ok := payload["event"].(map[string]interface{})
	if !ok {
		h.log.Warn("revenuecat webhook missing event field")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	eventType, _ := eventData["type"].(string)
	if strings.TrimSpace(eventType) == "" {
		h.log.Warn("revenuecat webhook event missing type")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.applyRevenueCatEvent(r.Context(), eventType, payload); err != nil {
		h.log.Warn("revenuecat event ignored", "type", eventType, "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}

func isValidRevenueCatSignature(providedSignature string, body []byte, endpointSecret string) bool {
	providedSignature = strings.TrimSpace(providedSignature)
	if providedSignature == "" {
		return false
	}

	h256 := hmac.New(sha256.New, []byte(endpointSecret))
	_, _ = h256.Write(body)
	expectedBytes := h256.Sum(nil)
	expectedHex := hex.EncodeToString(expectedBytes)
	expectedBase64 := base64.StdEncoding.EncodeToString(expectedBytes)

	return hmac.Equal([]byte(expectedHex), []byte(providedSignature)) ||
		hmac.Equal([]byte(expectedBase64), []byte(providedSignature))
}

func (h *PaymentHandlers) applyRevenueCatEvent(ctx context.Context, eventType string, payload map[string]interface{}) error {
	userID, tier, err := extractRevenueCatUserAndTier(payload)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	switch eventType {
	case "INITIAL_PURCHASE", "SUBSCRIPTION_PURCHASE", "RENEWAL", "SUBSCRIPTION_RENEWAL":
		if err := h.db.UpsertSubscriptionFromPayment(ctx, userID, true, tier, "", "", &now, nil); err != nil {
			return err
		}
		h.log.Info("revenuecat premium access applied", "user_id", userID, "tier", tier, "event", eventType)
		return nil
	case "SUBSCRIPTION_PAUSED", "EXPIRATION", "CANCELLATION", "SUBSCRIPTION_CANCEL", "SUBSCRIPTION_EXPIRED":
		if err := h.db.UpsertSubscriptionFromPayment(ctx, userID, false, "free", "", "", nil, &now); err != nil {
			return err
		}
		h.log.Info("revenuecat premium access removed", "user_id", userID, "event", eventType)
		return nil
	default:
		h.log.Debug("unhandled revenuecat webhook event", "type", eventType)
		return nil
	}
}

func extractRevenueCatUserAndTier(payload map[string]interface{}) (string, string, error) {
	event, ok := payload["event"].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("missing event payload")
	}

	appUserID, _ := event["app_user_id"].(string)
	if strings.TrimSpace(appUserID) == "" {
		if fallback, ok := event["original_app_user_id"].(string); ok {
			appUserID = fallback
		}
	}
	appUserID = strings.TrimSpace(appUserID)
	if appUserID == "" {
		return "", "", fmt.Errorf("missing app_user_id")
	}

	productID, _ := event["product_id"].(string)
	tier := tierFromRevenueCatProductID(productID)

	return appUserID, tier, nil
}

func tierFromRevenueCatProductID(productID string) string {
	productID = strings.ToLower(strings.TrimSpace(productID))
	switch {
	case strings.Contains(productID, "pro_plus"), strings.Contains(productID, "proplus"):
		return "pro_plus"
	case strings.Contains(productID, "unlimited"):
		return "unlimited"
	case productID == "":
		return "pro"
	default:
		return "pro"
	}
}
