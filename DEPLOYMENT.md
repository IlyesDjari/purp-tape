# Production Deployment Guide

This guide explains how to set up and configure your GitHub Actions deployment pipeline for PurpTape using **Fly.io** as the primary deployment target.

## Overview

The deployment workflow (`deploy-production.yml`) automatically:
1. ✅ Tests and builds your Go backend
2. 🏗️ Builds a Docker image  
3. 📦 Pushes the image to GitHub Container Registry (GHCR)
4. 🚀 Deploys to Fly.io (30+ global edge regions)

## Why Fly.io?

This is an **optimal architecture for a market-leading audio streaming backend**:

| Factor | Rating | Why |
|--------|--------|-----|
| **Global Performance** | ⭐⭐⭐⭐⭐ | 30+ edge regions = <50ms latency worldwide |
| **Cost Efficiency** | ⭐⭐⭐⭐ | Start at 1 machine (~$0), scales infinitely |
| **Scalability** | ⭐⭐⭐⭐⭐ | Auto-scales from 0 to millions of requests |
| **Security** | ⭐⭐⭐⭐⭐ | Enterprise-grade, automatic TLS, DDoS protection |
| **Developer Experience** | ⭐⭐⭐⭐⭐ | Simple CLI, great dashboards, fast deploys |
| **Future-Proof** | ⭐⭐⭐⭐⭐ | Can add WebSockets, Postgres standby, etc. |

**Your Full Stack (World-Class)**:
- API: Go on Fly.io (edge-deployed) ✅
- Database: PostgreSQL on Supabase ✅
- Storage: Cloudflare R2 (global CDN) ✅
- DDoS/WAF: Cloudflare ✅

This is what enterprise audio companies use.

## Prerequisites

- GitHub repository with Actions enabled
- Supabase project with PostgreSQL database
- Cloudflare R2 bucket configured
- Fly.io account (free tier available)

## Quick Start (10 minutes)

### Step 1: Create Fly.io Account

Go to [fly.io](https://fly.io) and sign up (takes 2 minutes).

### Step 2: Install flyctl

```bash
# Mac
brew install flyctl

# Linux/Windows - see https://fly.io/docs/hands-on/install/
```

### Step 3: Authenticate

```bash
flyctl auth login
# This opens your browser to log in
```

### Step 4: Generate GitHub Token

```bash
flyctl tokens create deploy --name github-actions
# Copy the token
```

### Step 5: Add GitHub Secret

```bash
# Using GitHub CLI
gh secret set FLY_API_TOKEN --body "your-token-here"

# Or manually:
# 1. Go to GitHub repo Settings → Secrets and Variables → Actions
# 2. Click "New repository secret"
# 3. Name: FLY_API_TOKEN
# 4. Value: Paste your token
```

### Step 6: Create Fly App

```bash
flyctl apps create purptape-api
# Takes 10 seconds
```

### Step 7: Set Environment Variables

```bash
flyctl secrets set \
  DATABASE_URL="postgres://user:password@db.supabase.co:5432/postgres" \
  SUPABASE_URL="https://xxx.supabase.co" \
  SUPABASE_ANON_KEY="your-anon-key-here" \
  SUPABASE_SECRET_KEY="your-secret-key-here" \
  R2_ACCESS_KEY_ID="your-r2-access-key" \
  R2_SECRET_ACCESS_KEY="your-r2-secret-key" \
  R2_ENDPOINT="https://xxx.r2.cloudflarestorage.com" \
  R2_BUCKET_NAME="purptape-audio" \
  R2_ACCOUNT_ID="your-account-id" \
  JOB_WORKER_CONCURRENCY="4" \
  JOB_BATCH_SIZE="32" \
  ENV="production"
```

### Step 8: Deploy

Just push to main branch:

```bash
git push origin main
```

GitHub Actions will automatically:
1. Build and test your code
2. Build Docker image
3. Push to GitHub Container Registry
4. Deploy to Fly.io

**That's it!** Your app is now live at: `https://purptape-api.fly.dev`

## Configuration

The `fly.toml` file controls your deployment:

```toml
app = "purptape-api"
primary_region = "sea"  # Seattle - change if needed

[autoscaling]
enabled = true
min_machines = 1        # Minimum: 1 machine
max_machines = 10       # Maximum: 10 machines (auto-scales)

[[vm]]
cpu_kind = "performance"
cpus = 1
memory_mb = 512
```

### Global Regions

Fly.io has 30+ regions worldwide. For best performance, place your primary region near your largest user base:

- **sea**: Seattle (US West)
- **sjc**: San Jose (US West)
- **lax**: Los Angeles (US West)
- **ord**: Chicago (US Midwest)
- **iad**: Virginia (US East)
- **yyz**: Toronto (Canada)
- **lhr**: London (EU)
- **ams**: Amsterdam (EU)
- **fra**: Frankfurt (EU)
- **sin**: Singapore (Asia)
- **syd**: Sydney (Australia)
- **nrt**: Tokyo (Asia)
- **sao**: São Paulo (Brazil)

For global coverage, Fly.io automatically replicates your app across regions.

## Database Migrations

### Option A: Supabase Dashboard
1. Go to your Supabase project
2. SQL Editor → New Query
3. Copy migrations from `backend/migrations/` in order
4. Run them

### Option B: Supabase CLI
```bash
# Install Supabase CLI
npm install -g supabase

# Set up
supabase init

# Deploy migrations
supabase db push
```

### Option C: Automatic (via init container)
Modify the workflow to run migrations automatically before app starts.

## Monitoring

### View Logs

```bash
flyctl logs --app purptape-api
```

### View Metrics

```bash
flyctl status --app purptape-api
```

### Dashboard

Go to [fly.io/apps/purptape-api](https://fly.io/apps) to see:
- Real-time CPU/memory usage
- Request latency by region
- Error rates
- Deployment history

## Scaling

### Manual Scaling

```bash
# Scale to 5 machines
flyctl scale count 5 --app purptape-api

# Scale down
flyctl scale count 1 --app purptape-api
```

### Auto-scaling

Already configured in `fly.toml`:
- Starts with 1 machine
- Automatically scales to 10 machines if CPU/memory usage spikes
- Scales back down when traffic decreases

## Deployment History

View all deployments:

```bash
flyctl releases list --app purptape-api
```

Rollback to previous version:

```bash
flyctl releases rollback --app purptape-api
```

## Cost Estimation

**For small-to-medium apps:**
- 1 shared-cpu machine: ~$5/month
- 1 performance machine: ~$15/month
- Postgres add-on: $5-50/month depending on size
- Storage/bandwidth: Included

**Under free tier?**
- First 3 shared-cpu-1x 256MB machines free
- 3GB Postgres database free
- Perfect for early development

## Troubleshooting

### App won't start

```bash
flyctl logs --app purptape-api
# Check for errors in logs
```

### Database connection fails

```bash
# Check DATABASE_URL is correct
flyctl secrets list --app purptape-api

# Test connection locally
psql $DATABASE_URL
```

### Cold starts / high latency

- Enable autoscaling to keep machines warm
- Use performance VMs instead of shared-cpu
- Check Fly.io metrics dashboard

### Health check failing

The workflow includes a health check on `/health` endpoint. Ensure your API has this endpoint implemented.

## Switching Deployment Platforms

To switch to a different platform:

1. Edit `.github/workflows/deploy-production.yml`
2. Change which job is enabled (Fly.io is default)
3. Add required secrets for new platform
4. Deploy

Available alternatives:
- **Railway**: Easiest setup, single region
- **Cloud Run**: Most cost-efficient for bursty traffic
- **Self-hosted SSH**: Maximum control

See comments in the workflow file for other platform configurations.

## Next Steps

1. ✅ Complete the 10-minute Quick Start above
2. Set up database migrations in Supabase
3. Push to main branch to trigger deployment
4. Monitor logs: `flyctl logs --app purptape-api`
5. Set custom domain (optional): `flyctl certs add yourdomain.com`

## Support

- Fly.io Docs: [fly.io/docs](https://fly.io/docs)
- Supabase Docs: [supabase.com/docs](https://supabase.com/docs)
- GitHub Actions: [github.com/features/actions](https://github.com/features/actions)

You now have a world-class production deployment pipeline. 🚀
