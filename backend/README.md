# PurpTape Backend API

A high-performance, secure audio streaming backend built with Go, PostgreSQL, and Cloudflare R2.

## Tech Stack

- **Language**: Go 1.22
- **Database**: PostgreSQL 16
- **Object Storage**: Cloudflare R2
- **Authentication**: Supabase Auth
- **Deployment**: Docker + Docker Compose

## Architecture Overview

### Database Schema

- **users**: User accounts and profiles
- **projects**: "Vaults" containing tracks
- **tracks**: Individual audio tracks
- **track_versions**: Version control for tracks (v1, v2, etc.)
- **project_shares**: Access control and sharing permissions
- **audit_logs**: Compliance and debugging logs

### API Structure

```
internal/
├── auth/        # JWT validation & Supabase Auth
├── config/      # Configuration management
├── db/          # Database layer (queries, connection pooling)
├── handlers/    # HTTP endpoint handlers
├── middleware/  # Auth, logging, CORS
└── models/      # Domain entities

cmd/api/        # Main server entrypoint
migrations/     # SQL migration files
```

## Getting Started

### Prerequisites

- Go 1.22+
- PostgreSQL 16
- Docker & Docker Compose (optional, for containerized dev)

### Local Development

1. **Clone the repository**:
   ```bash
   cd backend
   ```

2. **Copy environment variables**:
   ```bash
   cp .env.example .env
   ```
   Then fill in your Supabase and Cloudflare R2 credentials.

3. **Install dependencies**:
   ```bash
   go mod tidy
   ```

4. **Run PostgreSQL** (using Docker):
   ```bash
   docker run -d \
     --name purptape-postgres \
     -e POSTGRES_USER=purptape \
     -e POSTGRES_PASSWORD=devpassword123 \
     -e POSTGRES_DB=purptape \
     -p 5432:5432 \
     postgres:16-alpine
   ```

5. **Run migrations**:
   ```bash
   # Using psql
   psql -h localhost -U purptape -d purptape < migrations/001_create_users_table.sql
   psql -h localhost -U purptape -d purptape < migrations/002_create_projects_table.sql
   psql -h localhost -U purptape -d purptape < migrations/003_create_tracks_table.sql
   psql -h localhost -U purptape -d purptape < migrations/004_create_track_versions_table.sql
   psql -h localhost -U purptape -d purptape < migrations/005_create_project_shares_table.sql
   psql -h localhost -U purptape -d purptape < migrations/006_create_audit_logs_table.sql
   ```

6. **Start the API**:
   ```bash
   go run ./cmd/api
   ```

   The server will start at `http://localhost:8080`

### Docker Development

Run everything with Docker Compose:

```bash
docker compose up --build
```

The API will be available at `http://localhost:8080` and PostgreSQL at `localhost:5432`.

Run the strict local quality gate:

```bash
make quality-gate
```

This replays every SQL migration on a clean PostgreSQL and then runs `go test ./...`.

Run the full local release gate (migrations + static checks + startup smoke probe):

```bash
make release-gate
```

Run the strict S-tier production-readiness gate:

```bash
DATABASE_URL='postgres://purptape:devpassword123@localhost:5432/purptape?sslmode=disable' \
SUPABASE_URL='http://localhost:54321' \
SUPABASE_ANON_KEY='smoke-key' \
SUPABASE_SECRET_KEY='smoke-secret' \
R2_ACCESS_KEY_ID='smoke-r2-key' \
R2_SECRET_ACCESS_KEY='smoke-r2-secret' \
R2_ENDPOINT='http://localhost:9000' \
R2_BUCKET_NAME='smoke-bucket' \
R2_ACCOUNT_ID='smoke-account' \
FRONTEND_URL='http://localhost:3000' \
make s-tier-gate
```

`s-tier-gate` fails if the workspace is dirty. Use `ALLOW_DIRTY=1` only for local diagnostics.

## API Endpoints

### Health Check
- `GET /health` - Returns `{"status":"ok"}`

### Projects (Protected)
- `GET /projects` - List all projects for the user
- `POST /projects` - Create a new project
- `GET /projects/{id}` - Get a specific project

### Tracks (Protected)
- `GET /projects/{project_id}/tracks` - List tracks in a project
- `POST /projects/{project_id}/tracks` - Create a new track
- `GET /tracks/{track_id}/versions` - List all versions of a track
- `POST /tracks/{track_id}/versions` - Upload a new track version

## Authentication

All protected endpoints require a Bearer token from Supabase Auth:

```bash
curl -H "Authorization: Bearer <your-token>" http://localhost:8080/projects
```

## Configuration

See `.env.example` for all available configuration options:

- `PORT` - Server port (default: 8080)
- `DATABASE_URL` - PostgreSQL connection string
- `SUPABASE_URL` - Supabase project URL
- `SUPABASE_ANON_KEY` - Supabase anonymous key
- `R2_*` - Cloudflare R2 credentials
- `JOB_WORKER_CONCURRENCY` - Number of concurrent background workers per instance
- `JOB_BATCH_SIZE` - Number of jobs atomically claimed per poll cycle
- `FINOPS_STORAGE_COST_PER_GB_MONTH` - Storage cost assumption used for `/metrics` cost gauges
- `FINOPS_MONTHLY_BUDGET_USD` - Monthly budget threshold used for FinOps utilization/degraded health status
- `FINOPS_BUDGET_GUARD_ENABLED` - Skips expensive background jobs when budget guard threshold is reached
- `FINOPS_UPLOAD_BLOCK_ENABLED` - Blocks new track/cover uploads when projected monthly storage cost exceeds threshold
- `FINOPS_BUDGET_GUARD_RATIO` - Guard activation ratio relative to budget (e.g. `1.0` = 100%, `0.9` = 90%)
- `FINOPS_ENFORCE_R2_LIFECYCLE` - Applies bucket lifecycle rules directly on startup
- `FINOPS_R2_LIFECYCLE_STRICT` - Fails startup when lifecycle enforcement is enabled but policy application fails
- `FINOPS_COST_INGEST_TOKEN` - Bearer token for `POST /finops/cost-events` to ingest actual cloud billing entries

## Next Steps

1. Implement signed URL generation for Cloudflare R2
2. Add HLS transcoding pipeline with FFmpeg
3. Implement audio upload validation and checksum verification
4. Add WebSocket support for real-time collaboration
5. Implement audit logging for compliance

## Development Philosophy

- **Security First**: JWT validation on all protected routes
- **Performance**: Connection pooling, efficient database indexes
- **Scalability**: Stateless API design, cloud-native storage
- **Maintainability**: Clear separation of concerns, structured logging
- **Future-Proof**: Uses Go 1.22 features, modern PostgreSQL patterns
