package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// ============================================
// gRPC SERVICE DEFINITIONS & IMPLEMENTATIONS
// ============================================

// ProjectService provides internal gRPC API for project operations
type ProjectService struct {
	db  interface{}
	log *slog.Logger
}

// ProjectProto messages
type GetProjectRequest struct {
	ProjectId string
}

type ProjectProto struct {
	Id          string
	Name        string
	Description string
	OwnerId     string
	IsPrivate   bool
	TrackCount  int32
	CreatedAt   int64 // Unix timestamp
	UpdatedAt   int64
}

type ListProjectsRequest struct {
	OwnerId string
	Limit   int32
	Offset  int32
}

type ListProjectsResponse struct {
	Projects   []*ProjectProto
	TotalCount int32
}

type CreateProjectRequest struct {
	OwnerId     string
	Name        string
	Description string
	IsPrivate   bool
}

// NewProjectService creates project service
func NewProjectService(db interface{}, log *slog.Logger) *ProjectService {
	return &ProjectService{db: db, log: log}
}

// GetProject (unary RPC)
func (ps *ProjectService) GetProject(ctx context.Context, req *GetProjectRequest) (*ProjectProto, error) {
	ps.log.InfoContext(ctx, "grpc: get project", "project_id", req.ProjectId)

	project := &ProjectProto{
		Id:          req.ProjectId,
		Name:        "Summer Vibes",
		Description: "Collection of summer tracks",
		OwnerId:     "user123",
		IsPrivate:   false,
		TrackCount:  5,
		CreatedAt:   time.Now().AddDate(0, -3, 0).Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	return project, nil
}

// ListProjects (unary RPC)
func (ps *ProjectService) ListProjects(ctx context.Context, req *ListProjectsRequest) (*ListProjectsResponse, error) {
	ps.log.InfoContext(ctx, "grpc: list projects",
		"owner_id", req.OwnerId,
		"limit", req.Limit)

	projects := make([]*ProjectProto, 0, req.Limit)

	for i := int32(0); i < req.Limit; i++ {
		projects = append(projects, &ProjectProto{
			Id:        fmt.Sprintf("project_%d", req.Offset+i),
			Name:      fmt.Sprintf("Project %d", req.Offset+i),
			OwnerId:   req.OwnerId,
			TrackCount: int32((req.Offset + i) % 10),
		})
	}

	return &ListProjectsResponse{
		Projects:   projects,
		TotalCount: 1000,
	}, nil
}

// CreateProject (unary RPC)
func (ps *ProjectService) CreateProject(ctx context.Context, req *CreateProjectRequest) (*ProjectProto, error) {
	ps.log.InfoContext(ctx, "grpc: create project", "name", req.Name)

	project := &ProjectProto{
		Id:          fmt.Sprintf("proj_%d", time.Now().Unix()),
		Name:        req.Name,
		Description: req.Description,
		OwnerId:     req.OwnerId,
		IsPrivate:   req.IsPrivate,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	return project, nil
}

// ============================================
// TRACK SERVICE
// ============================================

type TrackService struct {
	db  interface{}
	log *slog.Logger
}

type TrackProto struct {
	Id          string
	ProjectId   string
	Name        string
	Duration    int32
	FileSize    int64
	R2ObjectKey string
	CreatedAt   int64
	UpdatedAt   int64
}

type GetTrackRequest struct {
	TrackId string
}

type ListTracksRequest struct {
	ProjectId string
	Limit     int32
	Offset    int32
}

type ListTracksResponse struct {
	Tracks     []*TrackProto
	TotalCount int32
}

type UploadTrackRequest struct {
	ProjectId string
	Name      string
	FileSize  int64
	FileHash  string
}

type UploadTrackResponse struct {
	TrackId        string
	PresignedUrl   string
	PresignedExpiry int64
}

// NewTrackService creates track service
func NewTrackService(db interface{}, log *slog.Logger) *TrackService {
	return &TrackService{db: db, log: log}
}

// GetTrack (unary RPC)
func (ts *TrackService) GetTrack(ctx context.Context, req *GetTrackRequest) (*TrackProto, error) {
	ts.log.InfoContext(ctx, "grpc: get track", "track_id", req.TrackId)

	track := &TrackProto{
		Id:          req.TrackId,
		ProjectId:   "project123",
		Name:        "Summer Song",
		Duration:    180,
		FileSize:    5000000,
		R2ObjectKey: "tracks/project123/track123.mp3",
		CreatedAt:   time.Now().AddDate(0, -1, 0).Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	return track, nil
}

// ListTracks (unary RPC)
func (ts *TrackService) ListTracks(ctx context.Context, req *ListTracksRequest) (*ListTracksResponse, error) {
	ts.log.InfoContext(ctx, "grpc: list tracks",
		"project_id", req.ProjectId,
		"limit", req.Limit)

	tracks := make([]*TrackProto, 0, req.Limit)

	for i := int32(0); i < req.Limit; i++ {
		tracks = append(tracks, &TrackProto{
			Id:        fmt.Sprintf("track_%d", req.Offset+i),
			ProjectId: req.ProjectId,
			Name:      fmt.Sprintf("Track %d", req.Offset+i),
			Duration:  180 + (i * 30),
		})
	}

	return &ListTracksResponse{
		Tracks:     tracks,
		TotalCount: 100,
	}, nil
}

// UploadTrack (unary RPC)
func (ts *TrackService) UploadTrack(ctx context.Context, req *UploadTrackRequest) (*UploadTrackResponse, error) {
	ts.log.InfoContext(ctx, "grpc: upload track",
		"project_id", req.ProjectId,
		"name", req.Name)

	trackId := fmt.Sprintf("track_%d", time.Now().Unix())

	return &UploadTrackResponse{
		TrackId:        trackId,
		PresignedUrl:   fmt.Sprintf("https://r2.example.com/upload/%s", trackId),
		PresignedExpiry: time.Now().Add(1 * time.Hour).Unix(),
	}, nil
}

// ============================================
// ANALYTICS SERVICE (gRPC Streaming Example)
// ============================================

type AnalyticsService struct {
	db  interface{}
	log *slog.Logger
}

type PlayEvent struct {
	ProjectId      string
	ListenerUserId string
	DurationPlayed int32
	Timestamp      int64
}

type PlayHistoryRequest struct {
	ProjectId string
	StartTime int64
	EndTime   int64
}

type PlayStatistics struct {
	ProjectId    string
	TotalPlays   int32
	UniquePlays  int32
	AverageDuration int32
	TopListener  string
}

// NewAnalyticsService creates analytics service
func NewAnalyticsService(db interface{}, log *slog.Logger) *AnalyticsService {
	return &AnalyticsService{db: db, log: log}
}

// StreamPlayHistory (server streaming RPC)
// Client sends request, server streams back play events
func (as *AnalyticsService) StreamPlayHistory(ctx context.Context, req *PlayHistoryRequest, handler func(*PlayEvent) error) error {
	as.log.InfoContext(ctx, "grpc: stream play history", "project_id", req.ProjectId)

	// Example: stream 10 events
	for i := 0; i < 10; i++ {
		event := &PlayEvent{
			ProjectId:       req.ProjectId,
			ListenerUserId:  fmt.Sprintf("user_%d", i),
			DurationPlayed:  180,
			Timestamp:       time.Now().Add(-time.Duration(i*24) * time.Hour).Unix(),
		}

		if err := handler(event); err != nil {
			return err
		}
	}

	return nil
}

// ============================================
// COLLABORATION SERVICE
// ============================================

type CollaborationService struct {
	db  interface{}
	log *slog.Logger
}

type ShareProjectRequest struct {
	ProjectId   string
	SharedWith  string
	Permissions []string
}

type ShareProjectResponse struct {
	ProjectId string
	SharedWith string
	ExpiresAt int64
}

type CommentProto struct {
	Id        string
	ProjectId string
	UserId    string
	Content   string
	CreatedAt int64
}

type AddCommentRequest struct {
	ProjectId string
	UserId    string
	Content   string
}

// NewCollaborationService creates collaboration service
func NewCollaborationService(db interface{}, log *slog.Logger) *CollaborationService {
	return &CollaborationService{db: db, log: log}
}

// ShareProject
func (cs *CollaborationService) ShareProject(ctx context.Context, req *ShareProjectRequest) (*ShareProjectResponse, error) {
	cs.log.InfoContext(ctx, "grpc: share project",
		"project_id", req.ProjectId,
		"shared_with", req.SharedWith)

	return &ShareProjectResponse{
		ProjectId:  req.ProjectId,
		SharedWith: req.SharedWith,
		ExpiresAt:  time.Now().Add(30 * 24 * time.Hour).Unix(),
	}, nil
}

// AddComment
func (cs *CollaborationService) AddComment(ctx context.Context, req *AddCommentRequest) (*CommentProto, error) {
	cs.log.InfoContext(ctx, "grpc: add comment",
		"project_id", req.ProjectId,
		"content", req.Content)

	comment := &CommentProto{
		Id:        fmt.Sprintf("comment_%d", time.Now().Unix()),
		ProjectId: req.ProjectId,
		UserId:    req.UserId,
		Content:   req.Content,
		CreatedAt: time.Now().Unix(),
	}

	return comment, nil
}

// ============================================
// gRPC PROTO DEFINITIONS
// ============================================

const ProtoDefinitions = `
syntax = "proto3";

package purptape;

service ProjectService {
  rpc GetProject(GetProjectRequest) returns (ProjectProto);
  rpc ListProjects(ListProjectsRequest) returns (ListProjectsResponse);
  rpc CreateProject(CreateProjectRequest) returns (ProjectProto);
  rpc UpdateProject(UpdateProjectRequest) returns (ProjectProto);
  rpc DeleteProject(DeleteProjectRequest) returns (DeleteProjectResponse);
}

service TrackService {
  rpc GetTrack(GetTrackRequest) returns (TrackProto);
  rpc ListTracks(ListTracksRequest) returns (ListTracksResponse);
  rpc UploadTrack(UploadTrackRequest) returns (UploadTrackResponse);
  rpc DeleteTrack(DeleteTrackRequest) returns (DeleteTrackResponse);
}

service AnalyticsService {
  // Server streaming: server sends play history events
  rpc StreamPlayHistory(PlayHistoryRequest) returns (stream PlayEvent);
  
  // Server streaming: server streams live analytics
  rpc StreamLiveAnalytics(AnalyticsRequest) returns (stream PlayEvent);
}

service CollaborationService {
  rpc ShareProject(ShareProjectRequest) returns (ShareProjectResponse);
  rpc AddComment(AddCommentRequest) returns (CommentProto);
  rpc GetComments(GetCommentsRequest) returns (stream CommentProto);
}

message ProjectProto {
  string id = 1;
  string name = 2;
  string description = 3;
  string owner_id = 4;
  bool is_private = 5;
  int32 track_count = 6;
  int64 created_at = 7;
  int64 updated_at = 8;
}

message TrackProto {
  string id = 1;
  string project_id = 2;
  string name = 3;
  int32 duration = 4;
  int64 file_size = 5;
  string r2_object_key = 6;
  int64 created_at = 7;
  int64 updated_at = 8;
}

message PlayEvent {
  string project_id = 1;
  string listener_user_id = 2;
  int32 duration_played = 3;
  int64 timestamp = 4;
}

message GetProjectRequest { string project_id = 1; }
message ListProjectsRequest { string owner_id = 1; int32 limit = 2; int32 offset = 3; }
message ListProjectsResponse { repeated ProjectProto projects = 1; int32 total_count = 2; }
message CreateProjectRequest { string owner_id = 1; string name = 2; string description = 3; bool is_private = 4; }
message UpdateProjectRequest { string id = 1; string name = 2; }
message DeleteProjectRequest { string id = 1; }
message DeleteProjectResponse { bool success = 1; }

message GetTrackRequest { string track_id = 1; }
message ListTracksRequest { string project_id = 1; int32 limit = 2; int32 offset = 3; }
message ListTracksResponse { repeated TrackProto tracks = 1; int32 total_count = 2; }
message UploadTrackRequest { string project_id = 1; string name = 2; int64 file_size = 3; string file_hash = 4; }
message UploadTrackResponse { string track_id = 1; string presigned_url = 2; int64 presigned_expiry = 3; }
message DeleteTrackRequest { string id = 1; }
message DeleteTrackResponse { bool success = 1; }

message PlayHistoryRequest { string project_id = 1; int64 start_time = 2; int64 end_time = 3; }
message AnalyticsRequest { string project_id = 1; }

message ShareProjectRequest { string project_id = 1; string shared_with = 2; repeated string permissions = 3; }
message ShareProjectResponse { string project_id = 1; string shared_with = 2; int64 expires_at = 3; }
message CommentProto { string id = 1; string project_id = 2; string user_id = 3; string content = 4; int64 created_at = 5; }
message AddCommentRequest { string project_id = 1; string user_id = 2; string content = 3; }
message GetCommentsRequest { string project_id = 1; int32 limit = 2; }
`
