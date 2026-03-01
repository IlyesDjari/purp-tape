# Fly.io Deployment Guide

This document explains how to deploy PurpTape to Fly.io.

## Prerequisites

- Fly.io account (`flyctl` installed and authenticated)
- Database URL (Supabase or PostgreSQL)
- Supabase credentials (URL and ANON_KEY)
- R2/S3 credentials for storage
- Firebase Cloud Messaging (FCM) server key (optional)

## Setting Up Secrets

Before deploying, set the required environment secrets:

```bash
# Production/Staging deployment
flyctl secrets set \
  DATABASE_URL="postgresql://user:password@host:5432/dbname" \
  SUPABASE_URL="https://xxx.supabase.co" \
  SUPABASE_ANON_KEY="eyJxxx..." \
  SUPABASE_SECRET_KEY="eyJxxx..." \
  R2_ACCESS_KEY_ID="xxx" \
  R2_SECRET_ACCESS_KEY="xxx" \
  R2_ENDPOINT="https://xxx.r2.googleapis.com" \
  R2_BUCKET_NAME="purptape-assets" \
  R2_ACCOUNT_ID="xxx" \
  JWT_SECRET="your-jwt-secret-key-min-32-chars" \
  FRONTEND_URL="https://purptape.com" \
  --app purptape-api
```

### For Staging Environment

```bash
flyctl secrets set \
  DATABASE_URL="..." \
  SUPABASE_URL="..." \
  ... \
  FRONTEND_URL="https://staging.purptape.com" \
  --app purptape-api-staging
```

## Environment Variables

The following environment variables are required:

| Variable | Description | Required |
|----------|-------------|----------|
| `DATABASE_URL` | PostgreSQL connection string | ✅ Yes |
| `SUPABASE_URL` | Supabase project URL | ✅ Yes |
| `SUPABASE_ANON_KEY` | Supabase anonymous/public key | ✅ Yes |
| `SUPABASE_SECRET_KEY` | Supabase service role key | ✅ Yes |
| `R2_ACCESS_KEY_ID` | Cloudflare R2 access key | ✅ Yes |
| `R2_SECRET_ACCESS_KEY` | Cloudflare R2 secret key | ✅ Yes |
| `R2_ENDPOINT` | Cloudflare R2 endpoint URL | ✅ Yes |
| `R2_BUCKET_NAME` | R2 bucket name | ✅ Yes |
| `R2_ACCOUNT_ID` | Cloudflare account ID | ✅ Yes |
| `JWT_SECRET` | Secret key for JWT signing (min 32 chars) | ✅ Yes |
| `FRONTEND_URL` | Frontend application URL | ✅ Yes |
| `FCM_SERVER_KEY` | Firebase Cloud Messaging key | ❌ No |
| `PORT` | Server port (default: 8080) | ❌ No |
| `HOST` | Server host (default: 0.0.0.0) | ❌ No |

## Health Checks

The API exposes the following health check endpoints:

- `GET /health` - Basic liveness check (70 sec startup grace period)
- `GET /health/deep` - Comprehensive health check with component status
- `GET /readiness` - Readiness probe (checks all dependencies)

## Deployment

### Automatic (via GitHub Actions)

Push to the `staging` or `main` branch will trigger automatic deployment:

```bash
git push origin staging
```

### Manual Deployment

```bash
flyctl deploy --remote-only
```

## Troubleshooting

### Health Check Timeout

If the health check times out:

1. **Missing secrets**: Verify all required environment variables are set
   ```bash
   flyctl secrets list --app purptape-api
   ```

2. **Database connection**: Test the `DATABASE_URL`
   ```bash
   flyctl ssh console --app purptape-api
   # Inside the container:
   psql "$DATABASE_URL"
   ```

3. **Service initialization**: Check logs
   ```bash
   flyctl logs --app purptape-api
   ```

### Container Crashes

Check the application logs:

```bash
flyctl logs --app purptape-api -a purptape-api
```

Common issues:
- Invalid database connection string
- Missing Supabase credentials
- Invalid JWT_SECRET (too short)
- R2 bucket not accessible

## Configuration Files

- `fly.toml` - Production configuration
- `fly.staging.toml` - Staging environment
- `fly.dev.toml` - Development environment

All files specify:
- 60-second health check grace period
- HTTP service on ports 80/443 with HTTPS redirect
- Appropriate machine sizes and memory limits

## Scaling

### Production
- Machine size: `shared-cpu-1x`
- Memory: 512MB
- Autoscaling: Off (manual scaling)

### Staging
- Machine size: `shared-cpu-1x`
- Memory: 256MB
- Autoscaling: 1-3 machines

To scale:
```bash
flyctl scale count 2 --app purptape-api
```
