package main

import (
	"log/slog"
	"net/http"

	"github.com/IlyesDjari/purp-tape/backend/internal/audit"
	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/graphql"
	"github.com/IlyesDjari/purp-tape/backend/internal/handlers"
	"github.com/IlyesDjari/purp-tape/backend/internal/notifications"
	"github.com/IlyesDjari/purp-tape/backend/internal/storage"
)

type appHandlers struct {
	OpsConsole          http.HandlerFunc
	OpsEndpointCatalog  http.HandlerFunc
	health        *handlers.HealthHandlers
	project       *handlers.ProjectHandlers
	track         *handlers.TrackHandlers
	rollback      *handlers.TrackRollbackHandlers
	image         *handlers.ImageHandlers
	payment       *handlers.PaymentHandlers
	offline       *handlers.OfflineHandlers
	compliance    *handlers.ComplianceHandlers
	collaboration *handlers.CollaborationHandlers
	download      *handlers.DownloadHandlers
	share         *handlers.ShareHandlers
	analytics     *handlers.AnalyticsHandlers
	search        *handlers.SearchHandlers
	finops        *handlers.FinOpsHandlers
	notifications *handlers.NotificationHandlers
	graphql       *graphql.Resolver
}

func newAppHandlers(
	database *db.Database,
	r2Client *storage.R2Client,
	notifSvc *notifications.NotificationService,
	pushSvc *notifications.PushNotificationService,
	prefsSvc *notifications.PreferencesService,
	log *slog.Logger,
) appHandlers {
	auditLogger := audit.NewLogger(database, log)

	return appHandlers{
		OpsConsole:          handlers.OpsConsole,
		OpsEndpointCatalog:  handlers.OpsEndpointCatalog,
		health:        handlers.NewHealthHandlers(database, r2Client, log),
		project:       handlers.NewProjectHandlers(database, log),
		track:         handlers.NewTrackHandlers(database, r2Client, log),
		rollback:      handlers.NewTrackRollbackHandlers(database, log),
		image:         handlers.NewImageHandlers(database, r2Client, log),
		payment:       handlers.NewPaymentHandlers(database, log),
		offline:       handlers.NewOfflineHandlers(database, r2Client, log),
		compliance:    handlers.NewComplianceHandlers(database, auditLogger, log),
		collaboration: handlers.NewCollaborationHandlers(database, log),
		download:      handlers.NewDownloadHandlers(database, r2Client, log),
		share:         handlers.NewShareHandlers(database, log),
		analytics:     handlers.NewAnalyticsHandlers(database, log),
		search:        handlers.NewSearchHandlers(database, log),
		finops:        handlers.NewFinOpsHandlers(database, log),
		notifications: handlers.NewNotificationHandlers(database, notifSvc, pushSvc, prefsSvc, log),
		graphql:       graphql.NewResolver(database, notifSvc, pushSvc, prefsSvc, log),
	}
}

func registerRoutes(mux *http.ServeMux, handlers appHandlers, withAuth func(http.HandlerFunc) http.HandlerFunc) {
	registerPublicRoutes(mux, handlers)
	registerProtectedRoutes(mux, handlers, withAuth)
}

func registerPublicRoutes(mux *http.ServeMux, handlers appHandlers) {
	mux.HandleFunc("GET /ops", handlers.OpsConsole)
	mux.HandleFunc("GET /ops/endpoints", handlers.OpsEndpointCatalog)
	mux.HandleFunc("GET /health", handlers.health.GetHealth)
	mux.HandleFunc("GET /health/deep", handlers.health.GetDeepHealth)
	mux.HandleFunc("GET /readiness", handlers.health.GetReadiness)
	mux.HandleFunc("GET /metrics", handlers.health.GetMetrics)
	mux.HandleFunc("GET /search", handlers.search.SearchAll)
	mux.HandleFunc("GET /discover/trending", handlers.search.GetTrending)
	mux.HandleFunc("GET /discover/public", handlers.search.GetPublicProjects)
	mux.HandleFunc("GET /share/{share_hash}", handlers.share.GetProjectByShareHash)
	mux.HandleFunc("POST /share/{share_hash}/verify", handlers.share.VerifySharePassword)
	mux.HandleFunc("POST /webhooks/stripe", handlers.payment.StripeWebhook)
	mux.HandleFunc("POST /webhooks/revenuecat", handlers.payment.RevenueCatWebhook)
	mux.HandleFunc("GET /pricing/tiers", handlers.payment.GetPricingTiers)
	mux.HandleFunc("POST /finops/cost-events", handlers.finops.IngestCostEvent)
	
	// GraphQL endpoint (public, but auth is checked in resolvers)
	mux.HandleFunc("POST /graphql", handlers.graphql.GraphQLHandler)
	mux.HandleFunc("GET /graphql", handlers.graphql.GraphQLHandler)
}

func registerProtectedRoutes(mux *http.ServeMux, handlers appHandlers, withAuth func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("GET /projects", withAuth(handlers.project.ListProjects))
	mux.HandleFunc("POST /projects", withAuth(handlers.project.CreateProject))
	mux.HandleFunc("GET /projects/{id}", withAuth(handlers.project.GetProject))
	mux.HandleFunc("PATCH /projects/{project_id}/privacy", withAuth(handlers.collaboration.UpdateProjectPrivacy))
	mux.HandleFunc("POST /projects/{project_id}/collaborators", withAuth(handlers.collaboration.AddCollaborator))
	mux.HandleFunc("POST /projects/{project_id}/cover", withAuth(handlers.image.UploadCover))
	mux.HandleFunc("GET /projects/{project_id}/tracks", withAuth(handlers.track.ListTracks))
	mux.HandleFunc("POST /projects/{project_id}/tracks", withAuth(handlers.track.CreateTrack))
	mux.HandleFunc("GET /projects/{project_id}/analytics", withAuth(handlers.analytics.GetProjectAnalytics))
	mux.HandleFunc("POST /projects/{project_id}/share-link", withAuth(handlers.share.GenerateShareLink))
	mux.HandleFunc("DELETE /projects/{project_id}/share-link/{share_hash}", withAuth(handlers.share.RevokeShareLink))
	mux.HandleFunc("POST /projects/{project_id}/share-link/{share_hash}/regenerate", withAuth(handlers.share.RegenerateShareHash))

	mux.HandleFunc("GET /tracks/{track_id}/versions", withAuth(handlers.track.ListTrackVersions))
	mux.HandleFunc("POST /tracks/{track_id}/versions", withAuth(handlers.track.UploadTrackVersion))
	mux.HandleFunc("GET /tracks/{track_id}/versions/{version_id}/download", withAuth(handlers.download.DownloadTrackVersion))
	mux.HandleFunc("POST /tracks/{track_id}/versions/{version_number}/rollback", withAuth(handlers.rollback.RollbackTrackVersion))
	mux.HandleFunc("GET /tracks/{track_id}/versions/history", withAuth(handlers.rollback.GetTrackVersionHistory))
	mux.HandleFunc("GET /tracks/{track_id}/play", withAuth(handlers.track.GetSignedPlayURL))
	mux.HandleFunc("POST /tracks/{track_id}/play-start", withAuth(handlers.analytics.RecordPlay))
	mux.HandleFunc("GET /tracks/{track_id}/stats", withAuth(handlers.analytics.GetTrackStats))
	mux.HandleFunc("POST /tracks/{track_id}/like", withAuth(handlers.collaboration.LikeTrack))
	mux.HandleFunc("DELETE /tracks/{track_id}/like", withAuth(handlers.collaboration.UnlikeTrack))
	mux.HandleFunc("POST /tracks/{track_id}/offline/download", withAuth(handlers.offline.InitiateDownload))

	mux.HandleFunc("POST /track-versions/{track_version_id}/comments", withAuth(handlers.collaboration.AddComment))
	mux.HandleFunc("GET /track-versions/{track_version_id}/comments", withAuth(handlers.collaboration.GetComments))

	mux.HandleFunc("GET /images/{image_id}/url", withAuth(handlers.image.GetCoverSignedURL))
	mux.HandleFunc("POST /checkout/session", withAuth(handlers.payment.CreateCheckoutSession))
	mux.HandleFunc("POST /plays/{play_id}/complete", withAuth(handlers.analytics.CompletePlay))
	mux.HandleFunc("GET /search/projects", withAuth(handlers.search.SearchProjects))

	mux.HandleFunc("POST /users/{user_id}/follow", withAuth(handlers.collaboration.FollowUser))
	mux.HandleFunc("DELETE /users/{user_id}/follow", withAuth(handlers.collaboration.UnfollowUser))
	mux.HandleFunc("GET /user/following", withAuth(handlers.collaboration.GetFollowing))
	mux.HandleFunc("GET /user/followers", withAuth(handlers.collaboration.GetFollowers))
	mux.HandleFunc("GET /user/notifications", withAuth(handlers.collaboration.GetNotifications))
	mux.HandleFunc("PATCH /notifications/{notification_id}/read", withAuth(handlers.collaboration.MarkNotificationAsRead))
	mux.HandleFunc("GET /user/stats", withAuth(handlers.analytics.GetUserStats))

	mux.HandleFunc("POST /offline/downloads/{download_id}/confirm", withAuth(handlers.offline.ConfirmDownload))
	mux.HandleFunc("GET /offline/downloads", withAuth(handlers.offline.GetOfflineDownloads))
	mux.HandleFunc("GET /offline/downloads/{download_id}/file", withAuth(handlers.download.DownloadOfflineFile))
	mux.HandleFunc("GET /offline/storage", withAuth(handlers.offline.GetOfflineStorageStatus))
	mux.HandleFunc("POST /offline/play-log", withAuth(handlers.offline.LogOfflinePlay))
	mux.HandleFunc("POST /offline/reconcile", withAuth(handlers.offline.ReconcileOfflineData))
	mux.HandleFunc("POST /offline/projects/{project_id}/sync", withAuth(handlers.offline.SyncDownloadProject))
	mux.HandleFunc("DELETE /offline/downloads/{download_id}", withAuth(handlers.offline.DeleteOfflineDownload))
	mux.HandleFunc("DELETE /offline/downloads/expired", withAuth(handlers.offline.DeleteAllExpiredDownloads))
	mux.HandleFunc("GET /offline/storage/info", withAuth(handlers.offline.GetOfflineStorageInfo))

	mux.HandleFunc("GET /compliance/data-export", withAuth(handlers.compliance.ExportUserData))
	mux.HandleFunc("DELETE /compliance/delete-account", withAuth(handlers.compliance.DeleteUserData))
	mux.HandleFunc("GET /compliance/privacy-settings", withAuth(handlers.compliance.GetPrivacySettings))
	mux.HandleFunc("PATCH /compliance/privacy-settings", withAuth(handlers.compliance.UpdatePrivacySettings))
	mux.HandleFunc("GET /finops/summary", withAuth(handlers.finops.GetSummary))
	
	// Notification endpoints (REST + GraphQL available)
	mux.HandleFunc("GET /notifications", withAuth(handlers.notifications.GetNotifications))
	mux.HandleFunc("POST /notifications/device-token", withAuth(handlers.notifications.RegisterDeviceToken))
	mux.HandleFunc("PATCH /notifications/preferences", withAuth(handlers.notifications.UpdatePreferences))
	mux.HandleFunc("POST /notifications/{notification_id}/read", withAuth(handlers.notifications.MarkAsRead))
	mux.HandleFunc("POST /notifications/read-all", withAuth(handlers.notifications.MarkAllAsRead))
	mux.HandleFunc("GET /notifications/unread-count", withAuth(handlers.notifications.GetUnreadCount))
}