# 🎙️ PurpTape Backend API

**Enterprise-Grade Audio Streaming Platform**

A production-ready, high-performance backend API for secure audio streaming and project management. Built with Go, PostgreSQL, and Cloudflare R2 for global distribution.

**Status**: ✅ Production Ready | **Version**: 1.0 | **Go**: 1.24+ | **License**: MIT

---

## 📋 Table of Contents

1. [Quick Start](#quick-start)
2. [Architecture](#architecture)
3. [Data Models](#data-models)
4. [API Reference](#api-reference)
5. [Authentication & Security](#authentication--security)
6. [Environment Configuration](#environment-configuration)
7. [Setup & Development](#setup--development)
8. [Deployment](#deployment)
9. [CI/CD Pipeline](#cicd-pipeline)
10. [Monitoring & Troubleshooting](#monitoring--troubleshooting)
11. [Contributing](#contributing)

---

## 🚀 Quick Start

### For Mobile/Web Developers

1. **Get the API URL**: `https://purptape-api.fly.dev` (production)
2. **Authenticate**: Via Supabase (see [Authentication](#authentication))
3. **Make requests**: Use Bearer tokens in `Authorization` header
4. **Check health**: `GET /health` (no auth required)

```bash
# Test the API
curl -X GET https://purptape-api.fly.dev/health
# Response: {"status":"ok"}
```

### For Backend Developers

```bash
cd backend
./setup.sh  # Start local dev environment
curl http://localhost:8080/health
```

---

## 🏗️ Architecture

### High-Level System Design

```
┌─────────────────────────────────────────────────────────────┐
│                      Mobile/Web Clients                      │
└──────────────────────────┬──────────────────────────────────┘
                           │ HTTPS (TLS 1.3)
                           ↓
┌─────────────────────────────────────────────────────────────┐
│              PurpTape API (Fly.io Global Edge)              │
│  ┌──────────┬──────────┬──────────┬──────────┐              │
│  │ JWT Auth │  Routes  │ Handlers │ Logging  │              │
│  └──────────┴──────────┴──────────┴──────────┘              │
└──────────────────────────┬──────────────────────────────────┘
           │               │               │
           ↓               ↓               ↓
    ┌────────────┐  ┌──────────────┐  ┌──────────┐
    │ PostgreSQL │  │ Cloudflare   │  │ Supabase │
    │ (RLS)      │  │ R2 Storage   │  │ Auth     │
    └────────────┘  └──────────────┘  └──────────┘
         (Supabase)   (Global CDN)    (User Auth)
```

### Code Organization

```
backend/
├── cmd/api/                  # Application entrypoint
│   └── main.go              # Server initialization
├── internal/
│   ├── auth/                # JWT validation & Supabase Auth
│   ├── handlers/            # HTTP endpoint handlers
│   ├── middleware/          # CORS, auth checks, logging
│   ├── db/                  # Database queries & connection pool
│   ├── models/              # Domain entities (User, Project, Track)
│   ├── config/              # Configuration from env vars
│   ├── storage/             # Cloudflare R2 integration
│   ├── notifications/       # Event notifications
│   ├── audit/               # Compliance & audit logging
│   └── errors/              # Error types & handling
├── migrations/              # SQL migration files (numbered: 001_, 002_, etc.)
├── scripts/                 # Helper scripts for testing
├── Dockerfile               # Production container image
├── docker-compose.yml       # Local development services
└── go.mod / go.sum          # Dependency management
```

### Database Schema (Entity Relationship)

```
users (from Supabase Auth)
  ├─── projects (1:N)
  │    ├─── tracks (1:N)
  │    │    └─── track_versions (1:N)
  │    └─── project_shares (1:N) ──→ users (shared_with)
  ├─── audit_logs (1:N)
  └─── notifications (1:N)
```

---

## 📊 Data Models

### Users

**Source**: Supabase Auth (JWT claims)

```typescript
User {
  id: UUID                    // From Supabase Auth
  email: string              // Unique email
  display_name: string       // User's name
  avatar_url: string         // Profile picture URL
  created_at: timestamp      // Account creation time
  updated_at: timestamp      // Last profile update
  metadata: jsonb            // Custom user fields
}
```

### Projects (Audio Vaults)

**Table**: `projects`

```typescript
Project {
  id: UUID                   // Unique project ID
  user_id: UUID             // Owner (FK → users)
  name: string              // Project name (max 255 chars)
  description: text         // Project description
  is_public: boolean        // Public or private
  cover_image_url: string   // Thumbnail URL
  created_at: timestamp
  updated_at: timestamp
  metadata: jsonb           // Custom fields (genre, tags, etc.)
}
```

**Example**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "user_id": "550e8400-e29b-41d4-a716-446655440002",
  "name": "Summer Vibes 2026",
  "description": "Collection of chilled beats",
  "is_public": true,
  "cover_image_url": "https://r2.purptape.com/covers/summer-vibes.jpg",
  "created_at": "2026-02-28T10:00:00Z"
}
```

### Tracks

**Table**: `tracks`

```typescript
Track {
  id: UUID                   // Unique track ID
  project_id: UUID          // Parent project (FK → projects)
  title: string             // Track title
  description: text         // Notes about the track
  duration_seconds: integer // Length of track (current version)
  current_version_id: UUID  // Latest version (FK → track_versions)
  created_at: timestamp
  updated_at: timestamp
  metadata: jsonb           // BPM, key, producer, etc.
}
```

**Example**:
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440003",
  "project_id": "550e8400-e29b-41d4-a716-446655440001",
  "title": "Midnight Groove",
  "description": "Smooth lo-fi beat with chill vibes",
  "duration_seconds": 245,
  "current_version_id": "770e8400-e29b-41d4-a716-446655440004",
  "metadata": {
    "bpm": 85,
    "key": "C minor",
    "producer": "DJ Cool",
    "tags": ["lo-fi", "chill", "2026"]
  }
}
```

### Track Versions

**Table**: `track_versions`

```typescript
TrackVersion {
  id: UUID                   // Unique version ID
  track_id: UUID            // Parent track (FK → tracks)
  version_number: integer   // v1, v2, v3, etc.
  file_url: string          // Signed R2 URL (expires in 1 hour)
  file_size_bytes: bigint   // File size for bandwidth tracking
  duration_seconds: integer // Length of this version
  file_hash: string         // SHA-256 for integrity checks
  uploaded_by: UUID         // User who uploaded (FK → users)
  created_at: timestamp
  notes: text               // What changed in this version
  metadata: jsonb           // Audio format, bitrate, codec
}
```

**Example**:
```json
{
  "id": "770e8400-e29b-41d4-a716-446655440004",
  "track_id": "660e8400-e29b-41d4-a716-446655440003",
  "version_number": 2,
  "file_url": "https://r2.purptape.com/tracks/midnight-groove-v2.mp3?token=...",
  "file_size_bytes": 4856832,
  "duration_seconds": 245,
  "file_hash": "abc123def456...",
  "uploaded_by": "550e8400-e29b-41d4-a716-446655440002",
  "created_at": "2026-02-28T12:30:00Z",
  "notes": "Added reverb, fixed clipping at 1:30",
  "metadata": {
    "codec": "mp3",
    "bitrate": "320kbps",
    "sample_rate": 44100,
    "channels": 2
  }
}
```

### Project Shares

**Table**: `project_shares`

```typescript
ProjectShare {
  id: UUID                   // Unique share ID
  project_id: UUID          // Shared project (FK → projects)
  shared_with_user_id: UUID // Recipient user (FK → users)
  permission_level: enum    // 'view' | 'comment' | 'edit'
  created_at: timestamp
  expires_at: timestamp     // Optional: access expires
}
```

---

## 🔌 API Reference

### Base URL

| Environment | URL |
|:-----------|-----|
| **Production** | `https://purptape-api.fly.dev` |
| **Staging** | `https://purptape-api-staging.fly.dev` |
| **Development** | `http://localhost:8080` |

### Response Format

All responses use JSON with consistent structure:

**Success (2xx)**:
```json
{
  "data": { /* response body */ },
  "status": "success",
  "timestamp": "2026-02-28T18:00:00Z"
}
```

**Error (4xx/5xx)**:
```json
{
  "error": "Descriptive error message",
  "code": "error_code",
  "status": "error",
  "timestamp": "2026-02-28T18:00:00Z",
  "request_id": "req-123456"
}
```

### HTTP Status Codes

| Code | Meaning | Example |
|------|---------|---------|
| `200` | Success | GET request completed |
| `201` | Created | POST created new resource |
| `204` | No Content | DELETE successful |
| `400` | Bad Request | Invalid parameters |
| `401` | Unauthorized | Missing/invalid token |
| `403` | Forbidden | No permission |
| `404` | Not Found | Resource doesn't exist |
| `409` | Conflict | Duplicate name/email |
| `422` | Unprocessable | Invalid field values |
| `429` | Rate Limited | Too many requests |
| `500` | Server Error | Internal error |
| `503` | Unavailable | Service maintenance |

### Endpoints

---

#### 🏥 **Health Check** (No Auth Required)

```http
GET /health
```

**Description**: Verify API is running. Used by load balancers and monitoring.

**Response** (200):
```json
{
  "status": "ok",
  "timestamp": "2026-02-28T18:00:00Z",
  "database": "connected",
  "version": "1.0.0"
}
```

---

#### 📦 **Projects**

##### Get All Projects

```http
GET /projects
Authorization: Bearer {token}
```

**Description**: List all projects for authenticated user.

**Response** (200):
```json
{
  "data": [
    {
      "id": "550e8400...",
      "name": "Summer Vibes 2026",
      "description": "Collection of chilled beats",
      "is_public": true,
      "track_count": 12,
      "created_at": "2026-02-28T10:00:00Z"
    }
  ],
  "status": "success",
  "pagination": {
    "total": 1,
    "page": 1,
    "limit": 50
  }
}
```

**Query Parameters**:
- `page?: number` (default: 1)
- `limit?: number` (default: 50, max: 100)
- `sort?: 'created' | 'updated' | 'name'` (default: 'updated')

##### Create Project

```http
POST /projects
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "New Project",
  "description": "Project description",
  "is_public": false,
  "metadata": {
    "genre": "electronic",
    "tags": ["house", "2026"]
  }
}
```

**Response** (201):
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "name": "New Project",
    "created_at": "2026-02-28T18:00:00Z"
  },
  "status": "success"
}
```

**Validation**:
- `name` required, 1-255 chars
- `description` max 5000 chars
- User can create max 20 projects (free tier)

**Errors**:
- `401 Unauthorized` - Invalid token
- `422 Unprocessable` - Invalid field values
- `409 Conflict` - Project name already exists

##### Get Project

```http
GET /projects/{project_id}
Authorization: Bearer {token}
```

**Response** (200):
```json
{
  "data": {
    "id": "550e8400...",
    "name": "Summer Vibes 2026",
    "track_count": 12,
    "shared_with": 3,
    "created_at": "2026-02-28T10:00:00Z"
  },
  "status": "success"
}
```

**Errors**:
- `404 Not Found` - Project doesn't exist
- `403 Forbidden` - No access to project

##### Update Project

```http
PATCH /projects/{project_id}
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "Updated Name",
  "description": "New description",
  "is_public": true
}
```

**Response** (200): Updated project object

**Errors**:
- `403 Forbidden` - Only owner can edit
- `409 Conflict` - Name already taken

##### Delete Project

```http
DELETE /projects/{project_id}
Authorization: Bearer {token}
```

**Response** (204): No content

**Warning**: Deletion is permanent and cascades to all tracks/versions.

---

#### 🎵 **Tracks**

##### List Tracks in Project

```http
GET /projects/{project_id}/tracks
Authorization: Bearer {token}
```

**Query Parameters**:
- `page?: number` (default: 1)
- `limit?: number` (default: 20, max: 100)
- `sort?: 'created' | 'duration' | 'title'`

**Response** (200):
```json
{
  "data": [
    {
      "id": "660e8400...",
      "title": "Midnight Groove",
      "duration_seconds": 245,
      "current_version": 2,
      "created_at": "2026-02-28T10:00:00Z"
    }
  ],
  "status": "success"
}
```

##### Create Track

```http
POST /projects/{project_id}/tracks
Authorization: Bearer {token}
Content-Type: application/json

{
  "title": "New Song",
  "description": "Song description",
  "metadata": {
    "bpm": 120,
    "key": "D minor"
  }
}
```

**Response** (201):
```json
{
  "data": {
    "id": "660e8400...",
    "title": "New Song",
    "created_at": "2026-02-28T18:00:00Z"
  },
  "status": "success"
}
```

**Validation**:
- `title` required, 1-255 chars
- `description` max 5000 chars
- Max 500 tracks per project

##### Get Track

```http
GET /tracks/{track_id}
Authorization: Bearer {token}
```

**Response** (200):
```json
{
  "data": {
    "id": "660e8400...",
    "title": "Midnight Groove",
    "duration_seconds": 245,
    "created_at": "2026-02-28T10:00:00Z"
  },
  "status": "success"
}
```

##### Update Track

```http
PATCH /tracks/{track_id}
Authorization: Bearer {token}

{
  "title": "Updated Title",
  "description": "Updated description"
}
```

##### Delete Track

```http
DELETE /tracks/{track_id}
Authorization: Bearer {token}
```

**Response** (204): No content

**Warning**: Deletes all versions of this track.

---

#### 📝 **Track Versions**

##### List All Versions

```http
GET /tracks/{track_id}/versions
Authorization: Bearer {token}
```

**Response** (200):
```json
{
  "data": [
    {
      "id": "770e8400...",
      "version_number": 2,
      "file_url": "https://r2.purptape.com/...",
      "duration_seconds": 245,
      "file_size_bytes": 4856832,
      "created_at": "2026-02-28T12:30:00Z",
      "notes": "Fixed clipping"
    },
    {
      "id": "880e8400...",
      "version_number": 1,
      "file_url": "https://r2.purptape.com/...",
      "duration_seconds": 240,
      "file_size_bytes": 4700000,
      "created_at": "2026-02-28T10:00:00Z"
    }
  ],
  "status": "success"
}
```

##### Upload New Version

```http
POST /tracks/{track_id}/versions
Authorization: Bearer {token}
Content-Type: multipart/form-data

file: <audio file (mp3, wav, flac)>
notes: "Added reverb, fixed drums"
```

**Response** (201):
```json
{
  "data": {
    "id": "770e8400...",
    "version_number": 3,
    "file_url": "https://r2.purptape.com/...",
    "duration_seconds": 250,
    "file_size_bytes": 5120000,
    "created_at": "2026-02-28T18:00:00Z"
  },
  "status": "success"
}
```

**Validation**:
- File format: `.mp3`, `.wav`, `.flac`, `.m4a`
- Max file size: 500 MB
- Max 100 versions per track
- Checks sufficient storage budget (FinOps)

**Errors**:
- `413 Payload Too Large` - File exceeds 500 MB
- `422 Unprocessable` - Invalid audio format
- `429 Rate Limited` - Storage budget exceeded

##### Download Track Version

```http
GET /tracks/{track_id}/versions/{version_id}/download
Authorization: Bearer {token}
```

**Response** (302 Redirect):
Redirects to signed R2 URL (valid for 1 hour)

---

#### 👥 **Sharing & Collaboration**

##### Share Project

```http
POST /projects/{project_id}/share
Authorization: Bearer {token}
Content-Type: application/json

{
  "user_email": "colleague@company.com",
  "permission": "edit",
  "expires_in_days": 30
}
```

**Response** (201):
```json
{
  "data": {
    "id": "990e8400...",
    "shared_with_email": "colleague@company.com",
    "permission": "edit",
    "created_at": "2026-02-28T18:00:00Z",
    "expires_at": "2026-03-30T18:00:00Z"
  },
  "status": "success"
}
```

**Permission Levels**:
- `view` - Read-only access
- `comment` - Can view + leave comments
- `edit` - Full edit access

**Errors**:
- `404 Not Found` - User not found
- `403 Forbidden` - Only owner can share
- `409 Conflict` - Already shared with user

##### List Shared Projects

```http
GET /projects/shared
Authorization: Bearer {token}
```

**Response** (200):
```json
{
  "data": [
    {
      "id": "550e8400...",
      "name": "Team Project",
      "owner": "colleague@company.com",
      "permission": "edit",
      "shared_at": "2026-02-28T10:00:00Z"
    }
  ],
  "status": "success"
}
```

##### Revoke Access

```http
DELETE /projects/{project_id}/shares/{share_id}
Authorization: Bearer {token}
```

**Response** (204): No content

---

#### 🔐 **Authentication & Users**

##### Get Current User

```http
GET /auth/me
Authorization: Bearer {token}
```

**Response** (200):
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440002",
    "email": "user@example.com",
    "display_name": "John Doe",
    "created_at": "2026-01-15T10:00:00Z"
  },
  "status": "success"
}
```

##### Update Profile

```http
PATCH /auth/me
Authorization: Bearer {token}

{
  "display_name": "Jane Doe",
  "avatar_url": "https://example.com/avatar.jpg"
}
```

---

## 🔐 Authentication & Security

### Supabase Auth Flow

1. **User signs up/logs in** via Supabase (handled by mobile/web client)
2. **Receives JWT token** from Supabase
3. **Sends token** in `Authorization: Bearer {token}` header
4. **API validates** token with Supabase public key
5. **Access granted** to protected endpoints

### Bearer Token Example

```bash
curl -X GET https://purptape-api.fly.dev/projects \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### Security Features

| Feature | Description |
|---------|-------------|
| **TLS 1.3** | All traffic encrypted in transit |
| **JWT Validation** | Every protected request verified |
| **Row-Level Security (RLS)** | Database level access control |
| **CORS** | Restricted to known domains |
| **Rate Limiting** | 100 requests/min per IP |
| **Audit Logging** | All actions logged for compliance |
| **Signed URLs** | R2 downloads expire after 1 hour |

### Error Codes

| Code | Meaning | Solution |
|------|---------|----------|
| `invalid_token` | Token missing/malformed | Refresh token via Supabase |
| `token_expired` | Token older than 1 hour | Request new token |
| `insufficient_permissions` | Action not allowed | Check project shares |
| `user_not_found` | Sharing with non-existent user | Verify email address |
| `storage_quota_exceeded` | Ran out of storage | Upgrade plan or delete files |

---

## ⚙️ Environment Configuration

### Required Variables (Production)

```bash
# Server
PORT=8080
ENV=production

# Database (Supabase)
DATABASE_URL=postgres://user:pass@db.supabase.co:5432/postgres?sslmode=require

# Authentication (Supabase)
SUPABASE_URL=https://xxx.supabase.co
SUPABASE_ANON_KEY=eyJhbGc...  # Public key
SUPABASE_SECRET_KEY=eyJhbGc... # Service role

# Storage (Cloudflare R2)
R2_ACCESS_KEY_ID=xxx
R2_SECRET_ACCESS_KEY=xxx
R2_ENDPOINT=https://xxx.r2.cloudflarestorage.com
R2_BUCKET_NAME=purptape
R2_ACCOUNT_ID=xxx

# FinOps (Cost Control)
FINOPS_STORAGE_COST_PER_GB_MONTH=0.015
FINOPS_MONTHLY_BUDGET_USD=1000
FINOPS_BUDGET_GUARD_RATIO=0.9
```

### Optional Variables (Tuning)

```bash
# Job Processing
JOB_WORKER_CONCURRENCY=4      # Concurrent background jobs
JOB_BATCH_SIZE=32             # Jobs per poll cycle

# Cost Controls
FINOPS_BUDGET_GUARD_ENABLED=true       # Pause jobs at 90% budget
FINOPS_UPLOAD_BLOCK_ENABLED=true       # Block uploads over budget
FINOPS_ENFORCE_R2_LIFECYCLE=true       # Auto-delete old versions

# Performance
DB_MAX_CONNS=25               # Max database connections
DB_MIN_CONNS=5                # Min idle connections
CACHE_TTL_SECONDS=3600        # Cache freshness
```

See [.env.example](.env.example) for complete reference.

---

## 🛠️ Setup & Development

### Option 1: Docker Compose (Recommended)

**Best for**: Quick local development, testing integrations

```bash
cd backend
cp .env.example .env
# Edit .env with Supabase & R2 credentials
docker compose up --build
```

**Services started**:
- API at `http://localhost:8080`
- PostgreSQL at `localhost:5432`
- Adminer (DB UI) at `http://localhost:8080/adminer`

### Option 2: Manual Setup

**Best for**: Native development, debugging

```bash
# Clone & dependencies
cd backend
cp .env.example .env
go mod download
go mod verify

# Database
docker run -d \
  --name purptape-postgres \
  -e POSTGRES_USER=purptape \
  -e POSTGRES_PASSWORD=devpassword123 \
  -e POSTGRES_DB=purptape \
  -p 5432:5432 \
  postgres:16-alpine

# Migrations
for f in migrations/*.sql; do
  psql postgres://purptape:devpassword123@localhost:5432/purptape < $f
done

# Run
go run ./cmd/api
```

### Option 3: VS Code Dev Container

**Best for**: Integrated IDE development

```bash
# Open in VS Code
# Press Shift+Cmd+P → "Dev Containers: Reopen in Container"
# Wait for setup, then:
make run
```

---

## 📦 Deployment

###

 Cloud Platforms

| Platform | Status | Region | Auto-Scale |
|----------|--------|--------|-----------|
| **Fly.io (Production)** | ✅ Active | CDG (Paris) | 2-10 machines |
| **Fly.io (Staging)** | ✅ Active | CDG (Paris) | 1-3 machines |
| **Fly.io (Dev)** | ✅ Active | CDG (Paris) | 1 machine |

### Deploy to Production

```bash
# Automatically triggered on push to main branch
# (See CI/CD Pipeline section)

# Manual deploy (if needed)
flyctl deploy -c fly.toml
```

### Deploy to Staging

```bash
# Automatically triggered on push to staging branch
flyctl deploy -c fly.staging.toml
```

### Deploy to Development

```bash
# Automatically triggered on push to develop branch
flyctl deploy -c fly.dev.toml
```

### Database Migrations

**Automatic**: Migrations run in GitHub Actions before deployment

**Manual** (if needed):
```bash
# Get DATABASE_URL from Fly secrets
flyctl secrets list -a purptape-api

# Run migrations
PGPASSWORD=xxx psql postgres://user:pass@host/db -f migrations/001_*.sql
```

**Important**: Always test migrations locally before production push!

---

## 🔄 CI/CD Pipeline

### Automated Process

```
Push to develop/staging/main
  │
  ├─→ Run Tests (go test -v -race)
  ├─→ Lint Code (golangci-lint)
  ├─→ Security Scan (gosec)
  ├─→ Build Docker Image
  ├─→ Push to GitHub Container Registry
  └─→ Deploy to Environment
          │
          ├─ develop → purptape-api-dev.fly.dev
          ├─ staging → purptape-api-staging.fly.dev
          └─ main → purptape-api.fly.dev (⏸️ needs approval)
```

### Workflows

| Branch | Workflow | Trigger | Status |
|--------|----------|---------|--------|
| `develop` | ci-tests, deploy-dev | Tests + Docker build + Deploy | Auto |
| `staging` | ci-tests, deploy-staging | Tests + Docker build + Deploy | Auto |
| `main` | ci-tests, deploy-prod | Tests + Docker build + Deploy | ⏸️ Manual Approval |

**Check Status**: https://github.com/IlyesDjari/purp-tape/actions

---

## 📊 Monitoring & Logging

### Health Endpoint

```bash
GET /health
```

**Response**:
```json
{
  "status": "ok",
  "database": "connected",
  "version": "1.0.0",
  "timestamp": "2026-02-28T18:00:00Z"
}
```

**Used by**: Fly.io load balancers to check pod health

### Application Logs

**View live logs**:
```bash
flyctl logs -a purptape-api        # Production
flyctl logs -a purptape-api-staging # Staging
flyctl logs -a purptape-api-dev     # Development
```

**Log Format**:
```
2026-02-28T18:00:00Z [INFO] user=abc123 action=project_created project_id=xyz789
2026-02-28T18:00:01Z [ERROR] user=abc123 action=upload_failed error="file_too_large"
```

### Monitoring Dashboards

| Tool | Purpose | URL |
|------|---------|-----|
| **Fly.io** | Infrastructure metrics | https://fly.io/apps/purptape-api |
| **Supabase** | Database performance | https://app.supabase.com |
| **Cloudflare** | R2 storage + API | https://dash.cloudflare.com |

### Error Tracking

**Rate your app**: If errors spike above 1%, Fly.io will notify via email.

---

## 🐛 Troubleshooting

### Common Issues

#### "Database Connection Failed"

**Symptom**: `ERROR: failed to connect to database`

**Causes**:
- DATABASE_URL is wrong
- PostgreSQL is down
- Network connectivity issue

**Fix**:
```bash
# Check DATABASE_URL
flyctl secrets list -a purptape-api | grep DATABASE_URL

# Verify database
psql {DATABASE_URL}

# Reset connection pool
flyctl restart -a purptape-api
```

#### "Unauthorized" on API Requests

**Symptom**: `401 Unauthorized` on all requests

**Causes**:
- Token is expired
- Token is malformed
- Supabase keys are wrong

**Fix**:
```bash
# Get fresh token from Supabase
# Check SUPABASE_URL and SUPABASE_SECRET_KEY are correct
flyctl secrets list -a purptape-api
```

#### "File Upload Fails"

**Symptom**: `POST /tracks/{id}/versions` returns 413 or 422

**Causes**:
- File exceeds 500 MB
- Audio format not supported
- Storage quota exceeded

**Fix**:
```bash
# Check file size
ls -lh tracks/my-song.mp3  # Should be < 500 MB

# Check quota
GET /auth/me  # Look at storage_used vs storage_limit

# Supported formats: mp3, wav, flac, m4a
file tracks/my-song.unknown
```

#### "Builds Failing"

**Symptom**: GitHub Actions shows ❌ on commits

**Check**: https://github.com/IlyesDjari/purp-tape/actions

**Common fixes**:
- Go version mismatch → Update Dockerfile
- Test failure → Run `go test ./...` locally
- Docker build error → Check Dockerfile syntax

---

## 📖 Contributing

### Code Style

- **Go**: Follow [Google Go Style Guide](https://google.github.io/styleguide/go/)
- **Comments**: Explain *why*, not *what*
- **Tests**: Aim for 80%+ coverage
- **SQL**: Use parameterized queries (always, never string concat)

### Development Workflow

1. **Create feature branch**:
   ```bash
   git checkout develop
   git pull origin develop
   git checkout -b feature/my-feature
   ```

2. **Commit messages** (Conventional Commits):
   ```
   feat: add audio validation endpoint
   fix: resolve race condition in track upload
   docs: update API reference
   chore: update dependencies
   ```

3. **Push & create PR**:
   ```bash
   git push origin feature/my-feature
   ```
   → GitHub shows test results automatically

4. **Code review**:
   - At least 1 approval required
   - All tests must pass
   - No merge conflicts

5. **Merge to develop**:
   - Auto-deploys to staging
   - Test in staging first
   - Then merge staging → main for production

### Testing

```bash
# Unit tests
go test ./...

# With coverage
go test -cover ./...

# Race condition detection
go test -race ./...

# Specific package
go test -v ./internal/handlers
```

### Database Changes

**Every change needs a migration**:

1. Create `migrations/NNN_description.sql`:
   ```sql
   -- Add new column
   ALTER TABLE tracks ADD COLUMN genre VARCHAR(50);
   
   -- Create index for performance
   CREATE INDEX idx_tracks_genre ON tracks(genre);
   ```

2. Test locally:
   ```bash
   docker compose down -v
   docker compose up --build
   ```

3. Verify migrations ran:
   ```bash
   psql localhost:5432 -c "SELECT * FROM migrations"
   ```

---

## 📱 For Mobile Developers

### Quick Integration Guide

#### Setup

1. Install Supabase SDK for iOS/Android
2. Initialize with your project URL & key:
   ```swift
   let supabase = SupabaseClient(
     url: "https://xxx.supabase.co",
     key: "your-anon-key"
   )
   ```

3. Sign up user:
   ```swift
   try await supabase.auth.signUp(email: "user@example.com", password: "...")
   ```

4. Get JWT token for API calls:
   ```swift
   let session = try await supabase.auth.session
   let token = session.accessToken  // Use in Authorization header
   ```

#### Making API Calls

```swift
// Example: Get all projects
var request = URLRequest(url: URL(string: "https://purptape-api.fly.dev/projects")!)
request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

let (data, _) = try await URLSession.shared.data(for: request)
let projects = try JSONDecoder().decode([Project].self, from: data)
```

#### Error Handling

```swift
do {
  let response = try await URLSession.shared.data(for: request)
} catch {
  if let httpResponse = response as? HTTPURLResponse {
    switch httpResponse.statusCode {
    case 401: print("Token expired - refresh auth")
    case 403: print("No permission")
    case 429: print("Rate limited - wait before retry")
    default: print("Error: \(httpResponse.statusCode)")
    }
  }
}
```

#### Best Practices

- Always cache auth token in Keychain
- Refresh token before expiry (1 hour)
- Implement retry logic for network requests
- Use URLSession background tasks for large uploads
- Validate responses before parsing
- Handle errors gracefully in UI

---

## 🤝 Support & Questions

- **Documentation**: This README
- **Issues**: https://github.com/IlyesDjari/purp-tape/issues
- **Discussions**: https://github.com/IlyesDjari/purp-tape/discussions
- **Email**: dev@purptape.com

---

## 📜 License

MIT License - See LICENSE file

---

**Last Updated**: February 28, 2026  
**Maintainer**: Ilyes Djari  
**Status**: ✅ Production Ready
