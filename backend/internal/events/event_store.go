package events

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ============================================
// EVENT SOURCING SYSTEM
// ============================================

// Event represents an immutable fact about what happened
type Event interface {
	EventType() string
	AggregateID() string
	Timestamp() time.Time
	Payload() map[string]interface{}
}

// BaseEvent provides common event fields
type BaseEvent struct {
	Type        string                 `json:"type"`
	AggregateIDValue string            `json:"aggregate_id"`
	Version     int64                  `json:"version"`
	TimestampValue time.Time           `json:"timestamp"`
	UserID      string                 `json:"user_id"`
	Data        map[string]interface{} `json:"data"`
}

func (e *BaseEvent) EventType() string    { return e.Type }
func (e *BaseEvent) AggregateID() string  { return e.AggregateIDValue }
func (e *BaseEvent) Timestamp() time.Time { return e.TimestampValue }
func (e *BaseEvent) Payload() map[string]interface{} { return e.Data }

// ============================================
// DOMAIN EVENTS
// ============================================

// ProjectCreatedEvent
type ProjectCreatedEvent struct {
	*BaseEvent
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
}

// ProjectUpdatedEvent
type ProjectUpdatedEvent struct {
	*BaseEvent
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
}

// ProjectDeletedEvent
type ProjectDeletedEvent struct {
	*BaseEvent
	Reason string `json:"reason"`
}

// TrackUploadedEvent
type TrackUploadedEvent struct {
	*BaseEvent
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	FileSize    int64  `json:"file_size"`
	Duration    int32  `json:"duration"`
	R2ObjectKey string `json:"r2_object_key"`
}

// PlayRecordedEvent
type PlayRecordedEvent struct {
	*BaseEvent
	ProjectID        string `json:"project_id"`
	ListenerUserID   string `json:"listener_user_id"`
	DurationListened int32  `json:"duration_listened"`
}

// CommentAddedEvent
type CommentAddedEvent struct {
	*BaseEvent
	ProjectID string `json:"project_id"`
	Content   string `json:"content"`
	ParentID  string `json:"parent_id"`
}

// ProjectSharedEvent
type ProjectSharedEvent struct {
	*BaseEvent
	ProjectID  string   `json:"project_id"`
	SharedWith string   `json:"shared_with"`
	Permissions []string `json:"permissions"`
}

// ============================================
// EVENT STORE
// ============================================

// EventStore persists and retrieves events (immutable)
type EventStore struct {
	events    []Event
	mu        sync.RWMutex
	log       *slog.Logger
	subscribers map[string][]EventHandler
}

// EventHandler is called when event occurs
type EventHandler func(ctx context.Context, event Event) error

// NewEventStore creates event store
func NewEventStore(log *slog.Logger) *EventStore {
	return &EventStore{
		events:      make([]Event, 0, 10000),
		log:         log,
		subscribers: make(map[string][]EventHandler),
	}
}

// AppendEvent adds immutable event to store
func (es *EventStore) AppendEvent(ctx context.Context, event Event) error {
	es.mu.Lock()
	es.events = append(es.events, event)
	versionNum := int64(len(es.events))
	es.mu.Unlock()

	es.log.InfoContext(ctx, "event appended",
		"type", event.EventType(),
		"aggregate_id", event.AggregateID(),
		"version", versionNum)

	// Publish to subscribers (async)
	es.publishEvent(ctx, event)

	return nil
}

// GetEvents retrieves all events for aggregate
func (es *EventStore) GetEvents(ctx context.Context, aggregateID string) ([]Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	var events []Event
	for _, e := range es.events {
		if e.AggregateID() == aggregateID {
			events = append(events, e)
		}
	}

	es.log.InfoContext(ctx, "events retrieved",
		"aggregate_id", aggregateID,
		"count", len(events))

	return events, nil
}

// GetEventsSince retrieves events since version
func (es *EventStore) GetEventsSince(ctx context.Context, aggregateID string, version int64) ([]Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	var events []Event
	count := 0
	for _, e := range es.events {
		if e.AggregateID() == aggregateID && int64(count) >= version {
			events = append(events, e)
		}
		if e.AggregateID() == aggregateID {
			count++
		}
	}

	return events, nil
}

// GetAllEvents retrieves all events (for replay/migration)
func (es *EventStore) GetAllEvents(ctx context.Context) ([]Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	return append([]Event{}, es.events...), nil
}

// ============================================
// EVENT SOURCING AGGREGATE
// ============================================

// ProjectAggregate reconstructs project state from events (event sourcing)
type ProjectAggregate struct {
	ID          string
	Name        string
	Description string
	IsPrivate   bool
	OwnerID     string
	Version     int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Events      []Event
}

// NewProjectAggregate creates aggregate from events
func NewProjectAggregate(events []Event) *ProjectAggregate {
	agg := &ProjectAggregate{
		Events: events,
	}

	// Replay events to reconstruct state
	for _, e := range events {
		agg.applyEvent(e)
	}

	return agg
}

// applyEvent applies event to reconstruct state
func (pa *ProjectAggregate) applyEvent(event Event) {
	switch e := event.(type) {
	case *ProjectCreatedEvent:
		pa.ID = e.AggregateIDValue
		pa.Name = e.Name
		pa.Description = e.Description
		pa.IsPrivate = e.IsPrivate
		pa.OwnerID = e.UserID
		pa.CreatedAt = e.TimestampValue
		pa.UpdatedAt = e.TimestampValue
		pa.Version++

	case *ProjectUpdatedEvent:
		pa.Name = e.Name
		pa.Description = e.Description
		pa.IsPrivate = e.IsPrivate
		pa.UpdatedAt = e.TimestampValue
		pa.Version++

	case *ProjectDeletedEvent:
		// Mark as deleted (soft delete via events)
		pa.UpdatedAt = e.TimestampValue
	}
}

// ============================================
// SUBSCRIPTIONS & EVENT HANDLERS
// ============================================

// Subscribe registers handler for event type
func (es *EventStore) Subscribe(eventType string, handler EventHandler) {
	es.mu.Lock()
	defer es.mu.Unlock()

	es.subscribers[eventType] = append(es.subscribers[eventType], handler)
	es.log.Info("subscriber registered", "event_type", eventType)
}

// publishEvent publishes to all subscribers (async)
func (es *EventStore) publishEvent(ctx context.Context, event Event) {
	handlers := es.subscribers[event.EventType()]

	go func() {
		for _, handler := range handlers {
			if err := handler(ctx, event); err != nil {
				es.log.Error("handler error",
					"event_type", event.EventType(),
					"error", err)
			}
		}
	}()
}

// ============================================
// EVENT HANDLERS (PROJECTIONS)
// ============================================

// ProjectionHandler builds read models from events
type ProjectionHandler struct {
	cache map[string]interface{}
	mu    sync.RWMutex
	log   *slog.Logger
}

// NewProjectionHandler creates projection handler
func NewProjectionHandler(log *slog.Logger) *ProjectionHandler {
	return &ProjectionHandler{
		cache: make(map[string]interface{}),
		log:   log,
	}
}

// Handle builds/updates read model
func (ph *ProjectionHandler) Handle(ctx context.Context, event Event) error {
	ph.mu.Lock()
	defer ph.mu.Unlock()

	cacheKey := fmt.Sprintf("%s:%s", event.EventType(), event.AggregateID())

	switch e := event.(type) {
	case *ProjectCreatedEvent:
		ph.cache[cacheKey] = map[string]interface{}{
			"id":          e.AggregateIDValue,
			"name":        e.Name,
			"description": e.Description,
			"is_private":  e.IsPrivate,
			"owner_id":    e.UserID,
			"created_at":  e.TimestampValue,
		}
		ph.log.InfoContext(ctx, "projection updated", "event", e.EventType())

	case *ProjectUpdatedEvent:
		ph.cache[cacheKey] = map[string]interface{}{
			"id":          e.AggregateIDValue,
			"name":        e.Name,
			"description": e.Description,
			"is_private":  e.IsPrivate,
			"updated_at":  e.TimestampValue,
		}
		ph.log.InfoContext(ctx, "projection updated", "event", e.EventType())

	case *TrackUploadedEvent:
		key := fmt.Sprintf("tracks:%s", e.Data["project_id"])
		ph.cache[key] = map[string]interface{}{
			"track_id":    e.AggregateIDValue,
			"project_id":  e.Data["project_id"],
			"name":        e.Name,
			"file_size":   e.FileSize,
			"uploaded_at": e.TimestampValue,
		}

	case *PlayRecordedEvent:
		key := fmt.Sprintf("plays:%s", e.Data["project_id"])
		ph.cache[key] = map[string]interface{}{
			"project_id":         e.Data["project_id"],
			"listener_user_id":   e.Data["listener_user_id"],
			"duration_listened":  e.Data["duration_listened"],
			"played_at":          e.TimestampValue,
		}
	}

	return nil
}

// GetProjection retrieves cached read model
func (ph *ProjectionHandler) GetProjection(key string) interface{} {
	ph.mu.RLock()
	defer ph.mu.RUnlock()

	return ph.cache[key]
}

// ============================================
// EVENT REPLAY (Disaster Recovery / Migration)
// ============================================

// EventReplayer replays all events to rebuild state
type EventReplayer struct {
	eventStore *EventStore
	log        *slog.Logger
}

// NewEventReplayer creates replayer
func NewEventReplayer(eventStore *EventStore, log *slog.Logger) *EventReplayer {
	return &EventReplayer{eventStore: eventStore, log: log}
}

// ReplayAll replays all events (useful after crashes)
func (er *EventReplayer) ReplayAll(ctx context.Context) error {
	er.log.InfoContext(ctx, "replaying all events")

	events, err := er.eventStore.GetAllEvents(ctx)
	if err != nil {
		return err
	}

	// Publish each event as if it just happened
	for _, e := range events {
		er.eventStore.publishEvent(ctx, e)
	}

	er.log.InfoContext(ctx, "event replay completed", "count", len(events))
	return nil
}

// ReplayAggregate replays events for specific aggregate
func (er *EventReplayer) ReplayAggregate(ctx context.Context, aggregateID string) error {
	er.log.InfoContext(ctx, "replaying events", "aggregate_id", aggregateID)

	events, err := er.eventStore.GetEvents(ctx, aggregateID)
	if err != nil {
		return err
	}

	for _, e := range events {
		er.eventStore.publishEvent(ctx, e)
	}

	er.log.InfoContext(ctx, "replay completed", "count", len(events))
	return nil
}

// ============================================
// SNAPSHOT SYSTEM (Performance Optimization)
// ============================================

// Snapshot captures state at point in time
type Snapshot struct {
	AggregateID string                 `json:"aggregate_id"`
	Version     int64                  `json:"version"`
	State       map[string]interface{} `json:"state"`
	CreatedAt   time.Time              `json:"created_at"`
}

// SnapshotStore manages snapshots
type SnapshotStore struct {
	snapshots map[string]*Snapshot
	mu        sync.RWMutex
	log       *slog.Logger
}

// NewSnapshotStore creates snapshot store
func NewSnapshotStore(log *slog.Logger) *SnapshotStore {
	return &SnapshotStore{
		snapshots: make(map[string]*Snapshot),
		log:       log,
	}
}

// CreateSnapshot creates snapshot of aggregate state
func (ss *SnapshotStore) CreateSnapshot(ctx context.Context, aggregateID string, version int64, state interface{}) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	stateMap, _ := state.(map[string]interface{})

	snapshot := &Snapshot{
		AggregateID: aggregateID,
		Version:     version,
		State:       stateMap,
		CreatedAt:   time.Now(),
	}

	ss.snapshots[aggregateID] = snapshot

	ss.log.InfoContext(ctx, "snapshot created",
		"aggregate_id", aggregateID,
		"version", version)

	return nil
}

// GetSnapshot retrieves latest snapshot
func (ss *SnapshotStore) GetSnapshot(ctx context.Context, aggregateID string) (*Snapshot, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	return ss.snapshots[aggregateID], nil
}
