package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
	"github.com/stripe/stripe-go/v72/webhook"

	"github.com/IlyesDjari/purp-tape/backend/internal/helpers"
)

// StripeWebhook handles POST /webhooks/stripe
// Verifies webhook signature before processing events.
func (h *PaymentHandlers) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if endpointSecret == "" {
		h.log.Error("stripe webhook secret not configured")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("failed to read webhook body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	signatureHeader := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(body, signatureHeader, endpointSecret)
	if err != nil {
		h.log.Warn("webhook signature verification failed")
		helpers.WriteBadRequest(w, "invalid signature")
		return
	}

	h.log.Info("verified stripe webhook", "type", event.Type)

	switch event.Type {
	case "customer.subscription.created":
		h.handleSubscriptionCreated(r.Context(), event.Data.Object)
	case "customer.subscription.updated":
		h.handleSubscriptionUpdated(r.Context(), event.Data.Object)
	case "customer.subscription.deleted":
		h.handleSubscriptionDeleted(r.Context(), event.Data.Object)
	case "payment_intent.succeeded":
		h.handlePaymentSucceeded(r.Context(), event.Data.Object)
	case "payment_intent.payment_failed":
		h.handlePaymentFailed(r.Context(), event.Data.Object)
	default:
		h.log.Info("unhandled stripe event type", "type", event.Type)
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"received": "true"})
}

func (h *PaymentHandlers) handleSubscriptionCreated(ctx context.Context, event map[string]interface{}) {
	if event == nil {
		return
	}

	customerID, _ := event["customer"].(string)
	subscriptionID, _ := event["id"].(string)

	tier := "pro"
	var userID string
	if metadata, ok := event["metadata"].(map[string]interface{}); ok {
		if v, ok := metadata["user_id"].(string); ok {
			userID = strings.TrimSpace(v)
		}
		if v, ok := metadata["tier_id"].(string); ok {
			candidate := strings.TrimSpace(strings.ToLower(v))
			if candidate != "" {
				tier = candidate
			}
		}
	}

	if userID == "" && customerID != "" {
		user, err := h.db.GetUserByStripeCustomerID(ctx, customerID)
		if err != nil {
			h.log.Error("failed to resolve user by stripe customer", "error", err, "customer_id", customerID)
			return
		}
		if user != nil {
			userID = user.ID
		}
	}

	if userID == "" {
		h.log.Warn("unable to resolve user for stripe subscription created", "customer_id", customerID, "subscription_id", subscriptionID)
		return
	}

	now := time.Now().UTC()
	if err := h.db.UpsertSubscriptionFromPayment(ctx, userID, true, tier, customerID, subscriptionID, &now, nil); err != nil {
		h.log.Error("failed to upsert subscription after creation", "error", err, "user_id", userID)
		return
	}

	h.log.Info("subscription created", "user_id", userID, "tier", tier)
}

func (h *PaymentHandlers) handleSubscriptionUpdated(ctx context.Context, event map[string]interface{}) {
	if event == nil {
		return
	}

	customerID, _ := event["customer"].(string)
	subscriptionID, _ := event["id"].(string)
	status, _ := event["status"].(string)

	tier := "pro"
	if metadata, ok := event["metadata"].(map[string]interface{}); ok {
		if v, ok := metadata["tier_id"].(string); ok {
			candidate := strings.TrimSpace(strings.ToLower(v))
			if candidate != "" {
				tier = candidate
			}
		}
	}

	var userID string
	user, err := h.db.GetUserByStripeSubscriptionID(ctx, subscriptionID)
	if err != nil {
		h.log.Error("failed to resolve user by stripe subscription", "error", err, "subscription_id", subscriptionID)
		return
	}
	if user != nil {
		userID = user.ID
	}

	if userID == "" && customerID != "" {
		user, err = h.db.GetUserByStripeCustomerID(ctx, customerID)
		if err != nil {
			h.log.Error("failed to resolve user by stripe customer", "error", err, "customer_id", customerID)
			return
		}
		if user != nil {
			userID = user.ID
		}
	}

	if userID == "" {
		h.log.Warn("unable to resolve user for stripe subscription update", "subscription_id", subscriptionID)
		return
	}

	isPremium := status != "canceled" && status != "unpaid" && status != "incomplete_expired"
	var canceledAt *time.Time
	if !isPremium {
		now := time.Now().UTC()
		canceledAt = &now
		tier = "free"
	}

	if err := h.db.UpsertSubscriptionFromPayment(ctx, userID, isPremium, tier, customerID, subscriptionID, nil, canceledAt); err != nil {
		h.log.Error("failed to upsert subscription after update", "error", err, "user_id", userID)
		return
	}

	h.log.Info("subscription updated", "user_id", userID, "status", status)
}

func (h *PaymentHandlers) handleSubscriptionDeleted(ctx context.Context, event map[string]interface{}) {
	if event == nil {
		return
	}

	customerID, _ := event["customer"].(string)
	subscriptionID, _ := event["id"].(string)

	var userID string
	user, err := h.db.GetUserByStripeSubscriptionID(ctx, subscriptionID)
	if err != nil {
		h.log.Error("failed to resolve user by stripe subscription", "error", err, "subscription_id", subscriptionID)
		return
	}
	if user != nil {
		userID = user.ID
	}

	if userID == "" && customerID != "" {
		user, err = h.db.GetUserByStripeCustomerID(ctx, customerID)
		if err != nil {
			h.log.Error("failed to resolve user by stripe customer", "error", err, "customer_id", customerID)
			return
		}
		if user != nil {
			userID = user.ID
		}
	}

	if userID == "" {
		h.log.Warn("unable to resolve user for stripe subscription deletion", "subscription_id", subscriptionID)
		return
	}

	now := time.Now().UTC()
	if err := h.db.UpsertSubscriptionFromPayment(ctx, userID, false, "free", customerID, subscriptionID, nil, &now); err != nil {
		h.log.Error("failed to mark subscription canceled", "error", err, "user_id", userID)
		return
	}

	h.log.Info("subscription canceled", "user_id", userID)
}

func (h *PaymentHandlers) handlePaymentSucceeded(ctx context.Context, event map[string]interface{}) {
	h.log.Info("payment succeeded")
}

func (h *PaymentHandlers) handlePaymentFailed(ctx context.Context, event map[string]interface{}) {
	h.log.Warn("payment failed")
}

// CreateCheckoutSession creates Stripe checkout for iOS.
func (h *PaymentHandlers) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	userID, err := helpers.GetUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		TierID string `json:"tier_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	stripeSecret := strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY"))
	if stripeSecret == "" {
		h.log.Error("stripe secret key not configured")
		http.Error(w, "payment provider misconfigured", http.StatusInternalServerError)
		return
	}

	tierID := strings.ToLower(strings.TrimSpace(req.TierID))
	priceID, err := getStripePriceIDForTier(tierID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.db.GetUserByID(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to load user for checkout session", "error", err, "user_id", userID)
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	frontendURL := strings.TrimSpace(os.Getenv("FRONTEND_URL"))
	if frontendURL == "" {
		frontendURL = "https://purptape.com"
	}
	frontendURL = strings.TrimRight(frontendURL, "/")

	successURL := strings.TrimSpace(os.Getenv("STRIPE_CHECKOUT_SUCCESS_URL"))
	if successURL == "" {
		successURL = frontendURL + "/billing/success?session_id={CHECKOUT_SESSION_ID}"
	}

	cancelURL := strings.TrimSpace(os.Getenv("STRIPE_CHECKOUT_CANCEL_URL"))
	if cancelURL == "" {
		cancelURL = frontendURL + "/billing/cancel"
	}

	stripe.Key = stripeSecret
	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		ClientReferenceID: stripe.String(userID),
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"user_id": userID,
				"tier_id": tierID,
			},
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{{
			Price:    stripe.String(priceID),
			Quantity: stripe.Int64(1),
		}},
	}

	if strings.TrimSpace(user.Email) != "" {
		params.CustomerEmail = stripe.String(user.Email)
	}

	checkoutSession, err := session.New(params)
	if err != nil {
		h.log.Error("failed to create stripe checkout session", "error", err, "user_id", userID, "tier", tierID)
		http.Error(w, "failed to create checkout session", http.StatusInternalServerError)
		return
	}

	h.log.Info("checkout session created", "user_id", userID, "tier", tierID, "session_id", checkoutSession.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": checkoutSession.ID,
		"url":        checkoutSession.URL,
	})
}

func getStripePriceIDForTier(tierID string) (string, error) {
	switch tierID {
	case "pro":
		priceID := strings.TrimSpace(os.Getenv("STRIPE_PRICE_ID_PRO"))
		if priceID == "" {
			priceID = "price_pro_monthly"
		}
		return priceID, nil
	case "pro_plus":
		priceID := strings.TrimSpace(os.Getenv("STRIPE_PRICE_ID_PRO_PLUS"))
		if priceID == "" {
			priceID = "price_proplus_monthly"
		}
		return priceID, nil
	case "unlimited":
		priceID := strings.TrimSpace(os.Getenv("STRIPE_PRICE_ID_UNLIMITED"))
		if priceID == "" {
			return "", fmt.Errorf("tier 'unlimited' is not configured")
		}
		return priceID, nil
	case "free":
		return "", fmt.Errorf("free tier does not require checkout")
	default:
		return "", fmt.Errorf("unsupported tier_id")
	}
}
