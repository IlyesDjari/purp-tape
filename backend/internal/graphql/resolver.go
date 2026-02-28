package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/handlers"
	"github.com/IlyesDjari/purp-tape/backend/internal/notifications"
)

// Resolver implements GraphQL resolvers
type Resolver struct {
	DB                *db.Database
	NotificationSvc   *notifications.NotificationService
	PushSvc           *notifications.PushNotificationService
	PreferencesSvc    *notifications.PreferencesService
	ProjectHandlers   *handlers.ProjectHandlers
	TrackHandlers     *handlers.TrackHandlers
	AnalyticsHandlers *handlers.AnalyticsHandlers
	Log               *slog.Logger
}

// NewResolver creates a GraphQL resolver
func NewResolver(
	database *db.Database,
	notifSvc *notifications.NotificationService,
	pushSvc *notifications.PushNotificationService,
	prefsSvc *notifications.PreferencesService,
	log *slog.Logger,
) *Resolver {
	return &Resolver{
		DB:              database,
		NotificationSvc: notifSvc,
		PushSvc:         pushSvc,
		PreferencesSvc:  prefsSvc,
		Log:             log,
	}
}

// Query resolvers

// User queries
func (r *Resolver) Me(ctx context.Context) (*User, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	user, err := r.DB.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return dbUserToGraphQL(user), nil
}

func (r *Resolver) User(ctx context.Context, id string) (*User, error) {
	user, err := r.DB.GetUserByID(ctx, id)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return dbUserToGraphQL(user), nil
}

// Project queries
func (r *Resolver) Projects(ctx context.Context, limit, offset int) (*ProjectConnection, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	projects, total, err := r.DB.GetUserProjectsPaginated(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	return &ProjectConnection{
		Edges:      projects,
		TotalCount: int(total),
		PageInfo: &PageInfo{
			Offset:    offset,
			Limit:     limit,
			HasMore:   int64(offset+limit) < total,
			TotalCount: int(total),
		},
	}, nil
}

// Notification queries
func (r *Resolver) Notifications(ctx context.Context, limit, offset int) (*NotificationConnection, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	notifications, total, err := r.NotificationSvc.GetNotifications(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}

	unread, err := r.NotificationSvc.GetUnreadCount(ctx, userID)
	if err != nil {
		unread = 0
	}

	return &NotificationConnection{
		Edges:       notifications,
		TotalCount:  int(total),
		UnreadCount: unread,
		PageInfo: &PageInfo{
			Offset:     offset,
			Limit:      limit,
			HasMore:    int64(offset+limit) < total,
			TotalCount: int(total),
		},
	}, nil
}

func (r *Resolver) UnreadNotificationCount(ctx context.Context) (int, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return 0, fmt.Errorf("unauthorized")
	}

	return r.NotificationSvc.GetUnreadCount(ctx, userID)
}

// Mutation resolvers

// RegisterDeviceToken registers a push notification device token
func (r *Resolver) RegisterDeviceToken(ctx context.Context, token, platform string) (*DeviceToken, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	if err := r.PushSvc.RegisterDeviceToken(ctx, userID, token, platform); err != nil {
		return nil, fmt.Errorf("failed to register device token: %w", err)
	}

	return &DeviceToken{
		Platform: platform,
	}, nil
}

// UpdateNotificationPreferences updates notification delivery preferences
func (r *Resolver) UpdateNotificationPreferences(ctx context.Context, input *NotificationPreferencesInput) (*NotificationPreferences, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	// Get current preferences
	prefs, err := r.PreferencesSvc.GetNotificationPreferences(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get preferences: %w", err)
	}

	// Update with provided values
	if input.PushEnabled != nil {
		prefs.PushEnabled = *input.PushEnabled
	}
	if input.PushLikes != nil {
		prefs.PushLikes = *input.PushLikes
	}
	if input.PushComments != nil {
		prefs.PushComments = *input.PushComments
	}
	if input.PushFollows != nil {
		prefs.PushFollows = *input.PushFollows
	}
	if input.PushShares != nil {
		prefs.PushShares = *input.PushShares
	}
	if input.PushMentions != nil {
		prefs.PushMentions = *input.PushMentions
	}
	if input.QuietHoursEnabled != nil {
		prefs.QuietHours = *input.QuietHoursEnabled
	}
	if input.QuietHoursStart != nil {
		prefs.QuietHoursStart = *input.QuietHoursStart
	}
	if input.QuietHoursEnd != nil {
		prefs.QuietHoursEnd = *input.QuietHoursEnd
	}
	if input.BundleByType != nil {
		prefs.BundleByType = *input.BundleByType
	}

	if err := r.PreferencesSvc.UpdateNotificationPreferences(ctx, userID, prefs); err != nil {
		return nil, fmt.Errorf("failed to update preferences: %w", err)
	}

	return &NotificationPreferences{
		PushEnabled:       prefs.PushEnabled,
		PushLikes:         prefs.PushLikes,
		PushComments:      prefs.PushComments,
		PushFollows:       prefs.PushFollows,
		PushShares:        prefs.PushShares,
		PushMentions:      prefs.PushMentions,
		QuietHoursEnabled: prefs.QuietHours,
		QuietHoursStart:   prefs.QuietHoursStart,
		QuietHoursEnd:     prefs.QuietHoursEnd,
		BundleByType:      prefs.BundleByType,
	}, nil
}

// MarkNotificationAsRead marks a single notification as read
func (r *Resolver) MarkNotificationAsRead(ctx context.Context, notificationID string) (interface{}, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	if err := r.NotificationSvc.MarkAsRead(ctx, notificationID, userID); err != nil {
		return nil, fmt.Errorf("failed to mark as read: %w", err)
	}

	return map[string]interface{}{"success": true}, nil
}

// MarkNotificationsAsRead marks multiple notifications as read
func (r *Resolver) MarkNotificationsAsRead(ctx context.Context, notificationIDs []string) (int, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return 0, fmt.Errorf("unauthorized")
	}

	count := 0
	for _, notificationID := range notificationIDs {
		if err := r.NotificationSvc.MarkAsRead(ctx, notificationID, userID); err != nil {
			r.Log.Warn("failed to mark notification as read", "notification_id", notificationID, "error", err)
			continue
		}
		count++
	}

	return count, nil
}

// MarkAllNotificationsAsRead marks all notifications as read
func (r *Resolver) MarkAllNotificationsAsRead(ctx context.Context) (bool, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return false, fmt.Errorf("unauthorized")
	}

	if err := r.NotificationSvc.MarkAllAsRead(ctx, userID); err != nil {
		return false, fmt.Errorf("failed to mark all as read: %w", err)
	}

	return true, nil
}

// Helper functions

func dbUserToGraphQL(user interface{}) *User {
	// Simplified conversion - in production, map all fields
	return &User{
		ID: "user-id",
	}
}

// ============================================================================
// GraphQL HTTP Handler
// ============================================================================

// GraphQLHandler exposes GraphQL endpoint with full query execution
func (r *Resolver) GraphQLHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost && req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract authorization from context or headers
	ctx := req.Context()
	if authHeader := req.Header.Get("Authorization"); authHeader != "" {
		// Bearer token handling would happen here via middleware
	}

	// Parse request body
	var queryReq QueryRequest
	if err := json.NewDecoder(req.Body).Decode(&queryReq); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(&QueryResponse{
			Errors: []string{"invalid request body: " + err.Error()},
		})
		return
	}

	// Validate query is not empty
	if strings.TrimSpace(queryReq.Query) == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(&QueryResponse{
			Errors: []string{"query cannot be empty"},
		})
		return
	}

	// Execute the GraphQL query
	response := r.ExecuteQuery(ctx, &queryReq)

	w.Header().Set("Content-Type", "application/json")
	wpayload, err := json.Marshal(response)
	if err != nil {
		r.Log.Error("failed to marshal GraphQL response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(wpayload)
}

// GraphQL input types

type NotificationPreferencesInput struct {
	PushEnabled       *bool
	PushLikes         *bool
	PushComments      *bool
	PushFollows       *bool
	PushShares        *bool
	PushMentions      *bool
	QuietHoursEnabled *bool
	QuietHoursStart   *string
	QuietHoursEnd     *string
	BundleByType      *bool
}

// GraphQL output types

type User struct {
	ID        string
	Username  string
	Email     string
	AvatarURL string
	Bio       string
}

type ProjectConnection struct {
	Edges      interface{}
	PageInfo   *PageInfo
	TotalCount int
}

type PageInfo struct {
	Offset     int
	Limit      int
	HasMore    bool
	TotalCount int
}

type NotificationConnection struct {
	Edges       interface{}
	PageInfo    *PageInfo
	TotalCount  int
	UnreadCount int
}

type DeviceToken struct {
	ID       string
	Platform string
}

type NotificationPreferences struct {
	PushEnabled       bool
	PushLikes         bool
	PushComments      bool
	PushFollows       bool
	PushShares        bool
	PushMentions      bool
	QuietHoursEnabled bool
	QuietHoursStart   string
	QuietHoursEnd     string
	BundleByType      bool
}
