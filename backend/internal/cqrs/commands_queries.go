package cqrs

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ============================================
// CQRS PATTERN IMPLEMENTATION
// ============================================

// Command represents intent to change state
type Command interface {
	CommandType() string
}

// Query represents intent to read state
type Query interface {
	QueryType() string
}

// CommandResult is result of command execution
type CommandResult struct {
	Success bool
	Error   string
	Data    map[string]interface{}
}

// QueryResult is result of query execution
type QueryResult struct {
	Data  interface{}
	Error error
}

// ============================================
// COMMAND DEFINITIONS
// ============================================

// CreateProjectCommand
type CreateProjectCommand struct {
	ProjectID   string
	OwnerID     string
	Name        string
	Description string
	IsPrivate   bool
}

func (c *CreateProjectCommand) CommandType() string { return "CreateProject" }

// UpdateProjectCommand
type UpdateProjectCommand struct {
	ProjectID   string
	Name        string
	Description string
	IsPrivate   bool
}

func (c *UpdateProjectCommand) CommandType() string { return "UpdateProject" }

// DeleteProjectCommand
type DeleteProjectCommand struct {
	ProjectID string
	Reason    string
}

func (c *DeleteProjectCommand) CommandType() string { return "DeleteProject" }

// UploadTrackCommand
type UploadTrackCommand struct {
	TrackID     string
	ProjectID   string
	Name        string
	FileSize    int64
	Duration    int32
	R2ObjectKey string
}

func (c *UploadTrackCommand) CommandType() string { return "UploadTrack" }

// RecordPlayCommand
type RecordPlayCommand struct {
	ProjectID        string
	ListenerUserID   string
	DurationListened int32
}

func (c *RecordPlayCommand) CommandType() string { return "RecordPlay" }

// ShareProjectCommand
type ShareProjectCommand struct {
	ProjectID   string
	SharedWith  string
	Permissions []string
}

func (c *ShareProjectCommand) CommandType() string { return "ShareProject" }

// ============================================
// COMMAND HANDLERS
// ============================================

// CommandHandler executes commands and returns results
type CommandHandler struct {
	log       *slog.Logger
	eventBus  interface{} // Event bus to publish domain events
}

// NewCommandHandler creates command handler
func NewCommandHandler(log *slog.Logger) *CommandHandler {
	return &CommandHandler{log: log}
}

// HandleCommand executes command
func (ch *CommandHandler) HandleCommand(ctx context.Context, cmd Command) *CommandResult {
	ch.log.InfoContext(ctx, "handling command", "type", cmd.CommandType())

	switch c := cmd.(type) {
	case *CreateProjectCommand:
		return ch.handleCreateProject(ctx, c)
	case *UpdateProjectCommand:
		return ch.handleUpdateProject(ctx, c)
	case *DeleteProjectCommand:
		return ch.handleDeleteProject(ctx, c)
	case *UploadTrackCommand:
		return ch.handleUploadTrack(ctx, c)
	case *RecordPlayCommand:
		return ch.handleRecordPlay(ctx, c)
	case *ShareProjectCommand:
		return ch.handleShareProject(ctx, c)
	default:
		return &CommandResult{
			Success: false,
			Error:   "unknown command type",
		}
	}
}

func (ch *CommandHandler) handleCreateProject(ctx context.Context, cmd *CreateProjectCommand) *CommandResult {
	// Validate command
	if len(cmd.Name) == 0 {
		return &CommandResult{Success: false, Error: "name required"}
	}

	// Execute business logic (could involve event sourcing)
	ch.log.InfoContext(ctx, "creating project", "name", cmd.Name)

	// In real implementation:
	// 1. Validate permission (user owns this project?)
	// 2. Apply project creation business rules
	// 3. Store domain event (ProjectCreatedEvent)
	// 4. Return success

	return &CommandResult{
		Success: true,
		Data: map[string]interface{}{
			"project_id": cmd.ProjectID,
			"created_at": time.Now().String(),
		},
	}
}

func (ch *CommandHandler) handleUpdateProject(ctx context.Context, cmd *UpdateProjectCommand) *CommandResult {
	ch.log.InfoContext(ctx, "updating project", "id", cmd.ProjectID)

	return &CommandResult{
		Success: true,
		Data: map[string]interface{}{
			"updated_at": time.Now().String(),
		},
	}
}

func (ch *CommandHandler) handleDeleteProject(ctx context.Context, cmd *DeleteProjectCommand) *CommandResult {
	ch.log.InfoContext(ctx, "deleting project", "id", cmd.ProjectID, "reason", cmd.Reason)

	return &CommandResult{
		Success: true,
		Data: map[string]interface{}{
			"deleted_at": time.Now().String(),
		},
	}
}

func (ch *CommandHandler) handleUploadTrack(ctx context.Context, cmd *UploadTrackCommand) *CommandResult {
	ch.log.InfoContext(ctx, "uploading track", "name", cmd.Name, "size", cmd.FileSize)

	return &CommandResult{
		Success: true,
		Data: map[string]interface{}{
			"track_id": cmd.TrackID,
		},
	}
}

func (ch *CommandHandler) handleRecordPlay(ctx context.Context, cmd *RecordPlayCommand) *CommandResult {
	ch.log.InfoContext(ctx, "recording play",
		"project_id", cmd.ProjectID,
		"duration", cmd.DurationListened)

	return &CommandResult{
		Success: true,
	}
}

func (ch *CommandHandler) handleShareProject(ctx context.Context, cmd *ShareProjectCommand) *CommandResult {
	ch.log.InfoContext(ctx, "sharing project",
		"project_id", cmd.ProjectID,
		"shared_with", cmd.SharedWith)

	return &CommandResult{
		Success: true,
		Data: map[string]interface{}{
			"expires_at": time.Now().Add(30 * 24 * time.Hour).String(),
		},
	}
}

// ============================================
// QUERY DEFINITIONS
// ============================================

// GetProjectQuery
type GetProjectQuery struct {
	ProjectID string
}

func (q *GetProjectQuery) QueryType() string { return "GetProject" }

// ListProjectsQuery
type ListProjectsQuery struct {
	OwnerID string
	Limit   int32
	Offset  int32
}

func (q *ListProjectsQuery) QueryType() string { return "ListProjects" }

// GetProjectAnalyticsQuery
type GetProjectAnalyticsQuery struct {
	ProjectID string
	Period    string // "7d", "30d", "90d"
}

func (q *GetProjectAnalyticsQuery) QueryType() string { return "GetProjectAnalytics" }

// GetUserPlayHistoryQuery
type GetUserPlayHistoryQuery struct {
	UserID string
	Limit  int32
	Offset int32
}

func (q *GetUserPlayHistoryQuery) QueryType() string { return "GetUserPlayHistory" }

// SearchProjectsQuery
type SearchProjectsQuery struct {
	Query  string
	Limit  int32
	Offset int32
}

func (q *SearchProjectsQuery) QueryType() string { return "SearchProjects" }

// ============================================
// QUERY HANDLERS (READ MODEL)
// ============================================

// QueryHandler executes queries from read model (optimized for reads)
type QueryHandler struct {
	log     *slog.Logger
	cache   map[string]interface{}
	cacheMu sync.RWMutex
}

// NewQueryHandler creates query handler
func NewQueryHandler(log *slog.Logger) *QueryHandler {
	return &QueryHandler{
		log:   log,
		cache: make(map[string]interface{}),
	}
}

// HandleQuery executes query against read model
func (qh *QueryHandler) HandleQuery(ctx context.Context, query Query) *QueryResult {
	qh.log.InfoContext(ctx, "handling query", "type", query.QueryType())

	switch q := query.(type) {
	case *GetProjectQuery:
		return qh.handleGetProject(ctx, q)
	case *ListProjectsQuery:
		return qh.handleListProjects(ctx, q)
	case *GetProjectAnalyticsQuery:
		return qh.handleGetProjectAnalytics(ctx, q)
	case *GetUserPlayHistoryQuery:
		return qh.handleGetUserPlayHistory(ctx, q)
	case *SearchProjectsQuery:
		return qh.handleSearchProjects(ctx, q)
	default:
		return &QueryResult{Error: fmt.Errorf("unknown query type")}
	}
}

func (qh *QueryHandler) handleGetProject(ctx context.Context, query *GetProjectQuery) *QueryResult {
	qh.log.InfoContext(ctx, "getting project", "id", query.ProjectID)

	// Read from optimized read model (denormalized, fast)
	cacheKey := "project:" + query.ProjectID

	qh.cacheMu.RLock()
	cached := qh.cache[cacheKey]
	qh.cacheMu.RUnlock()

	if cached != nil {
		return &QueryResult{Data: cached}
	}

	// Simulate query from read model
	project := map[string]interface{}{
		"id":          query.ProjectID,
		"name":        "Summer Vibes",
		"track_count": 5,
		"plays":       1250,
		"followers":   42,
	}

	qh.cacheMu.Lock()
	qh.cache[cacheKey] = project
	qh.cacheMu.Unlock()

	return &QueryResult{Data: project}
}

func (qh *QueryHandler) handleListProjects(ctx context.Context, query *ListProjectsQuery) *QueryResult {
	qh.log.InfoContext(ctx, "listing projects",
		"owner_id", query.OwnerID,
		"limit", query.Limit)

	projects := []map[string]interface{}{
		{
			"id":         "proj1",
			"name":       "Project 1",
			"track_count": 5,
		},
		{
			"id":         "proj2",
			"name":       "Project 2",
			"track_count": 3,
		},
	}

	return &QueryResult{Data: projects}
}

func (qh *QueryHandler) handleGetProjectAnalytics(ctx context.Context, query *GetProjectAnalyticsQuery) *QueryResult {
	qh.log.InfoContext(ctx, "getting analytics",
		"project_id", query.ProjectID,
		"period", query.Period)

	// Query from read model optimized for analytics
	analytics := map[string]interface{}{
		"project_id":       query.ProjectID,
		"total_plays":      1250,
		"unique_listeners": 125,
		"avg_duration":     180,
		"top_referrer":     "search",
		"period":           query.Period,
	}

	return &QueryResult{Data: analytics}
}

func (qh *QueryHandler) handleGetUserPlayHistory(ctx context.Context, query *GetUserPlayHistoryQuery) *QueryResult {
	qh.log.InfoContext(ctx, "getting play history", "user_id", query.UserID)

	history := []map[string]interface{}{
		{
			"project_id": "proj1",
			"played_at":  time.Now().Add(-24 * time.Hour),
			"duration":   180,
		},
	}

	return &QueryResult{Data: history}
}

func (qh *QueryHandler) handleSearchProjects(ctx context.Context, query *SearchProjectsQuery) *QueryResult {
	qh.log.InfoContext(ctx, "searching projects", "query", query.Query)

	// Query from full-text search index (denormalized)
	results := []map[string]interface{}{
		{
			"id":   "proj_search_1",
			"name": query.Query + " - Result 1",
		},
	}

	return &QueryResult{Data: results}
}

// ============================================
// CQRS BUS
// ============================================

// CQRSBus handles commands and queries
type CQRSBus struct {
	commandHandler *CommandHandler
	queryHandler   *QueryHandler
	log            *slog.Logger
}

// NewCQRSBus creates CQRS bus
func NewCQRSBus(log *slog.Logger) *CQRSBus {
	return &CQRSBus{
		commandHandler: NewCommandHandler(log),
		queryHandler:   NewQueryHandler(log),
		log:            log,
	}
}

// ExecuteCommand sends command to command handler
func (cb *CQRSBus) ExecuteCommand(ctx context.Context, cmd Command) *CommandResult {
	return cb.commandHandler.HandleCommand(ctx, cmd)
}

// ExecuteQuery sends query to query handler
func (cb *CQRSBus) ExecuteQuery(ctx context.Context, query Query) *QueryResult {
	return cb.queryHandler.HandleQuery(ctx, query)
}

// ============================================
// CONSISTENCY: EVENTUAL CONSISTENCY
// ============================================

// Eventually the write model (commands) and read model (queries) sync via events

// SyncReadModel updates read model from domain events
func (qh *QueryHandler) SyncReadModel(projectionData map[string]interface{}) {
	qh.cacheMu.Lock()
	defer qh.cacheMu.Unlock()

	// Merge projection data into read model
	for key, value := range projectionData {
		qh.cache[key] = value
	}
}

// Example usage:
// 1. Command: CreateProject → ProjectCreatedEvent
// 2. Event is published
// 3. ProjectionHandler subscribes → updates denormalized read model
// 4. Query: GetProject now fetches optimized data from read model
//
// Benefits:
// - Commands are fast (just validate + append event)
// - Queries are fast (query pre-computed read model)
// - Decoupled: Can scale commands & queries independently
// - Event sourcing: Full audit trail
