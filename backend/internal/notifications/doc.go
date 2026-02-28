// Package notifications provides multi-channel notification delivery
//
// This package implements a comprehensive notification system with:
//
// 1. IN-APP NOTIFICATIONS
//    - Database-backed notification store
//    - Efficient pagination and querying
//    - Mark-as-read functionality
//    - Notification preferences per user
//
// 2. PUSH NOTIFICATIONS
//    - Firebase Cloud Messaging (FCM) integration
//    - Device token management (iOS, Android, Web)
//    - Automatic deactivation of invalid tokens
//    - Multi-device broadcasting
//
// 3. NOTIFICATION COORDINATION
//    - NotificationService orchestrates delivery across channels
//    - Respects user preferences for each channel
//    - Supports bulk notifications for announcements
//    - Async notification delivery to prevent blocking
//
// 4. USER PREFERENCES
//    - Per-notification-type enable/disable
//    - Quiet hours support
//    - Notification bundling options
//
// USAGE:
//
//	// Initialize services
//	pushSvc := NewPushNotificationService(db, fcmKey, log)
//	prefsSvc := NewPreferencesService(db, log)
//	notifSvc := NewNotificationService(db, pushSvc, prefsSvc, log)
//
//	// Send a notification
//	err := notifSvc.SendNotification(ctx, &NotificationRequest{
//		UserID:    "user-123",
//		Type:      "like",
//		ActorID:   ptr("user-456"),
//		TrackID:   ptr("track-789"),
//		Content:   "User456 liked your track",
//	})
//
// NOTIFICATION TYPES:
// - "like": Track or project liked
// - "comment": Comment added to track
// - "follow": User followed
// - "share": Project shared with user
// - "mention": User mentioned in comment
package notifications
