// Package graphql implements a GraphQL API for PurpTape
//
// This package provides a complete GraphQL interface alongside the REST API,
// enabling advanced client capabilities like:
//
// QUERIES
// - User information and search
// - Projects with full relationship loading
// - Tracks and versions with metadata
// - Notifications with pagination
// - Analytics and statistics
// - Subscriptions and cost data
//
// MUTATIONS
// - Create/update/delete projects and tracks
// - Collaboration management
// - Sharing and access control
// - Comment and like operations
// - Notification preference management
// - Device token registration for push notifications
//
// SUBSCRIPTIONS
// - Real-time notification delivery (WebSocket)
// - Live project updates during collaborative editing
// - Collaborator presence tracking
//
// USAGE:
//
// The GraphQL endpoint is served at POST /graphql
// GraphQL schema is defined in schema.graphql
//
// Example query:
//
//	query {
//	  me {
//	    id
//	    username
//	    projects(limit: 10) {
//	      edges { id name }
//	      pageInfo { totalCount hasMore }
//	    }
//	  }
//	  notifications(limit: 20) {
//	    edges {
//	      id
//	      type
//	      isRead
//	      createdAt
//	    }
//	    unreadCount
//	  }
//	}
//
// Example mutation:
//
//	mutation {
//	  registerDeviceToken(token: "FCM_TOKEN", platform: IOS) {
//	    id
//	    platform
//	  }
//	  updateNotificationPreferences(input: {
//	    pushEnabled: true
//	    pushLikes: true
//	    pushComments: true
//	    quietHoursEnabled: true
//	    quietHoursStart: "22:00"
//	    quietHoursEnd: "09:00"
//	  }) {
//	    pushEnabled
//	    quietHoursEnabled
//	  }
//	}
//
// INTEGRATION:
//
// The resolver integrates with:
// - NotificationService: Multi-channel notification delivery
// - PushNotificationService: Firebase Cloud Messaging
// - PreferencesService: User notification preferences
// - Database layer: All data persistence
//
// For production use, consider integrating with gqlgen or graphql-go
// for automatic schema validation and resolver generation
package graphql
