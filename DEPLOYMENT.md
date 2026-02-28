# Production Deployment Guide

This guide explains how to set up and configure your GitHub Actions deployment pipeline for PurpTape.

## Overview

The deployment workflow (`deploy-production.yml`) automatically:
1. ✅ Tests and builds your Go backend
2. 🏗️ Builds a Docker image
3. 📦 Pushes the image to GitHub Container Registry (GHCR)
4. 🚀 Deploys to your chosen platform

## Prerequisites

- GitHub repository with Actions enabled
- Supabase project with PostgreSQL database (already set up in your `.env`)
- Cloudflare R2 bucket configured
- Choose your deployment platform (Railway, Fly.io, Cloud Run, SSH, etc.)

## Deployment Platform Options

### Option 1: Railway (Recommended - Easiest)

Railway is the simplest option for Go backends. It handles Docker automatically, migrations, and scales easily.

**Setup steps:**

1. Go to [railway.app](https://railway.app) and create an account
2. Create a new project
3. Generate a Railway API token: Railway Dashboard → Account Settings → Tokens
4. Add GitHub Secrets:
   ```
   RAILWAY_TOKEN: <your-token>
   RAILWAY_PROJECT_ID: <your-project-id>
   ```
5. Create a `railway.json` at the root of your repo:
   ```json
   {
     "build": {
       "builder": "dockerfile",
       "dockerfilePath": "backend/Dockerfile"
     },
     "deploy": {
       "numReplicas": 1,
       "restartPolicyMaxRetries": 5,
       "healthchecks": {
         "readiness": "/health",
         "startup": "/ready"
       }
     }
   }
   ```

### Option 2: Fly.io (Great for Global Apps)

Fly.io provides edge deployment with automatic TLS and easy PostgreSQL integration.

**Setup steps:**

1. Go to [fly.io](https://fly.io) and sign up
2. Install flyctl: `brew install flyctl` (on Mac)
3. Initialize: `flyctl apps create purptape-api`
4. Generate auth token: `flyctl auth token`
5. Add GitHub Secrets:
   ```
   FLY_API_TOKEN: <your-token>
   ```
6. Create `fly.toml` at project root:
   ```toml
   app = "purptape-api"
   primary_region = "sea"

   [build]
     dockerfile = "backend/Dockerfile"

   [env]
     PORT = "8080"

   [[services]]
     protocol = "tcp"
     internal_port = 8080
     processes = ["app"]

     [[services.ports]]
       port = 80
       handlers = ["http"]
       force_https = true

     [[services.ports]]
       port = 443
       handlers = ["tls", "http"]
   ```
7. Set environment variables:
   ```bash
   flyctl secrets set \
     DATABASE_URL=postgres://... \
     SUPABASE_URL=... \
     SUPABASE_ANON_KEY=... \
     R2_ACCESS_KEY_ID=... \
     R2_SECRET_ACCESS_KEY=... \
     R2_ENDPOINT=... \
     R2_BUCKET_NAME=...
   ```

### Option 3: Google Cloud Run (Serverless)

Cloud Run is great if you want serverless with auto-scaling.

**Setup steps:**

1. Create a GCP project
2. Enable Container Registry and Cloud Run APIs
3. Create a service account with appropriate permissions
4. Download service account JSON key
5. Add GitHub Secrets:
   ```
   GCP_CREDENTIALS: <base64-encoded-json>
   ```
6. Uncomment the `deploy-google-cloud-run` job in the workflow

### Option 4: Self-Hosted (SSH to Your Server)

Deploy to your own VPS or dedicated server.

**Setup steps:**

1. Prepare your server with Docker and Docker Compose
2. Generate SSH key pair:
   ```bash
   ssh-keygen -t rsa -b 4096 -f ~/.ssh/deploy_key
   ```
3. Add public key to server: `ssh-copy-id -i ~/.ssh/deploy_key.pub user@yourserver.com`
4. Add GitHub Secrets:
   ```
   DEPLOY_HOST: your.server.com
   DEPLOY_USER: deploy_user
   DEPLOY_KEY: <private-key-content>
   ```
5. Uncomment the `deploy-ssh` job in the workflow

## Required GitHub Secrets

Add these to your GitHub repository (Settings → Secrets → New Repository Secret):

```
# Supabase
SUPABASE_DATABASE_URL=postgres://...
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your-anon-key
SUPABASE_SECRET_KEY=your-secret-key

# Cloudflare R2
R2_ACCESS_KEY_ID=your-r2-access-key
R2_SECRET_ACCESS_KEY=your-r2-secret-key
R2_ENDPOINT=https://your-account-id.r2.cloudflarestorage.com
R2_BUCKET_NAME=purptape-audio
R2_ACCOUNT_ID=your-account-id

# FinOps (if enabled)
FINOPS_COST_INGEST_TOKEN=your-token
```

## Database Migrations

The workflow runs, but migrations need to be executed on your Supabase database:

### Option A: Automatic via Cloud Run/Render
Deploy an init container that runs migrations before the app starts.

### Option B: Manual via Supabase
1. Go to Supabase Dashboard
2. Open SQL Editor
3. Copy migrations from `backend/migrations/` and run them in order

### Option C: Using Supabase CLI
```bash
supabase db push  # After setting up supabase CLI
```

## Monitoring Deployments

1. Go to your GitHub repository
2. Click "Actions" tab
3. View the deployment workflow runs
4. Check logs for each step

## Rollback

To rollback to a previous version:
1. The Docker images are tagged with Git SHA, so you can easily reference previous versions
2. Redeploy the previous Docker image tag to your platform

## Troubleshooting

### Build fails with "go mod download" error
- Ensure `go.sum` is committed to the repository
- Run `go mod tidy` locally and push

### Docker image push fails
- Ensure GitHub Token has write access to Container Registry
- Use `${{ secrets.GITHUB_TOKEN }}` (auto-provided by GitHub)

### Database connection fails in production
- Verify `DATABASE_URL` secret is correct
- Use `psql` to test connection locally
- Check Supabase RLS policies aren't blocking queries

### Migrations fail
- Check migration SQL syntax
- Ensure migrations are run in order
- Use Supabase's SQL editor for debugging

## Next Steps

1. Choose your deployment platform
2. Update `.github/workflows/deploy-production.yml` to enable the correct job
3. Add required secrets to GitHub
4. Push changes to main branch
5. Watch the deployment workflow run

For questions, check the individual platform's documentation or test locally with Docker Compose first.
