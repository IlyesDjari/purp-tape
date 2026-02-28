#!/bin/bash

# PurpTape Production Deployment Setup Script
# This script helps you set up GitHub Actions and production deployment

set -e

echo "🚀 PurpTape Production Deployment Setup"
echo "========================================"
echo ""

# Check if we're in the right directory
if [ ! -f "go.mod" ] && [ ! -f "backend/go.mod" ]; then
    echo "❌ Error: Please run this script from the project root directory"
    exit 1
fi

echo "Available deployment options:"
echo "1. Fly.io (Recommended - Global edge deployment, best for audio)"
echo "2. Railway (Simplest setup, single region)"
echo "3. Google Cloud Run (Serverless, most cost-efficient)"
echo "4. Manual SSH deployment"
echo ""

read -p "Choose deployment platform (1-4): " choice

case $choice in
    1)
        echo "📍 Setting up Fly.io deployment (RECOMMENDED)"
        echo ""
        echo "Prerequisites:"
        echo "1. Create account at https://fly.io"
        echo "2. Install flyctl: brew install flyctl"
        echo "3. Authenticate: flyctl auth login"
        echo ""
        read -p "Enter your Fly.io API Token (from 'flyctl tokens create deploy'): " flyio_token
        
        echo ""
        echo "Adding GitHub Secrets..."
        echo "gh secret set FLY_API_TOKEN --body '${flyio_token}'"
        echo ""
        echo "Create Fly app (if not already done):"
        echo "  flyctl apps create purptape-api"
        echo ""
        echo "Then set environment variables:"
        echo "  flyctl secrets set DATABASE_URL=postgres://..."
        echo "  flyctl secrets set SUPABASE_URL=https://..."
        echo "  flyctl secrets set SUPABASE_ANON_KEY=..."
        echo "  flyctl secrets set R2_ACCESS_KEY_ID=..."
        echo "  flyctl secrets set R2_SECRET_ACCESS_KEY=..."
        echo "  flyctl secrets set R2_ENDPOINT=..."
        echo "  flyctl secrets set R2_BUCKET_NAME=..."
        echo ""
        echo "Deploy: Just push to main branch!"
        echo ""
        echo "Why Fly.io?"
        echo "  ✅ Global edge deployment (30+ regions, <50ms latency)"
        echo "  ✅ Auto-scales infinitely"
        echo "  ✅ Perfect for audio streaming"
        echo "  ✅ Enterprise-grade security"
        ;;
        
    2)
        echo "📍 Setting up Railway deployment"
        echo ""
        echo "Prerequisites:"
        echo "1. Create account at https://railway.app"
        echo "2. Create a new project in Railway Dashboard"
        echo "3. Generate API token: Settings → Tokens"
        echo ""
        read -p "Enter your Railway API Token: " railway_token
        read -p "Enter your Railway Project ID: " railway_project_id
        
        echo ""
        echo "Adding GitHub Secrets..."
        echo "gh secret set RAILWAY_TOKEN --body '${railway_token}'"
        echo "gh secret set RAILWAY_PROJECT_ID --body '${railway_project_id}'"
        echo ""
        echo "Environment Variables to set in Railway:"
        echo "  • DATABASE_URL (from Supabase)"
        echo "  • SUPABASE_URL"
        echo "  • SUPABASE_ANON_KEY"
        echo "  • R2_ACCESS_KEY_ID"
        echo "  • R2_SECRET_ACCESS_KEY"
        echo "  • R2_ENDPOINT"
        echo "  • R2_BUCKET_NAME"
        echo ""
        echo "Update .github/workflows/deploy-production.yml"
        echo "to enable the deploy-railway job"
        ;;
        
    3)
        echo "📍 Setting up Google Cloud Run deployment"
        echo ""
        echo "Prerequisites:"
        echo "1. Create GCP project"
        echo "2. Enable Cloud Run and Container Registry APIs"
        echo "3. Create service account with Editor role"
        echo "4. Download JSON key file"
        echo ""
        read -p "Path to GCP service account JSON: " gcp_json
        
        if [ ! -f "$gcp_json" ]; then
            echo "❌ File not found: $gcp_json"
            exit 1
        fi
        
        credentials=$(base64 < "$gcp_json" | tr -d '\n')
        echo ""
        echo "Adding GitHub Secret..."
        echo "gh secret set GCP_CREDENTIALS --body '${credentials}'"
        echo ""
        echo "Then update workflow to uncomment deploy-google-cloud-run job"
        ;;
        
    4)
        echo "📍 Setting up SSH deployment"
        echo ""
        echo "Prerequisites:"
        echo "1. VPS with Docker and Docker Compose installed"
        echo "2. SSH access configured"
        echo ""
        read -p "Server hostname/IP: " server_host
        read -p "SSH username: " ssh_user
        read -p "Path to SSH private key (~/.ssh/id_rsa): " ssh_key_path
        
        if [ ! -f "$ssh_key_path" ]; then
            echo "❌ SSH key not found: $ssh_key_path"
            echo "Generate one with: ssh-keygen -t rsa -b 4096"
            exit 1
        fi
        
        ssh_key=$(cat "$ssh_key_path")
        
        echo ""
        echo "Adding GitHub Secrets..."
        echo "gh secret set DEPLOY_HOST --body '${server_host}'"
        echo "gh secret set DEPLOY_USER --body '${ssh_user}'"
        echo "gh secret set DEPLOY_KEY --body '$(cat "$ssh_key_path")'"
        echo ""
        echo "Add public key to server:"
        echo "  ssh-copy-id -i ${ssh_key_path}.pub ${ssh_user}@${server_host}"
        echo ""
        echo "Then update workflow to uncomment deploy-ssh job"
        ;;
        
    *)
        echo "❌ Invalid choice"
        exit 1
        ;;
esac

echo ""
echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "1. Add required GitHub secrets (see instructions above)"
echo "2. Set environment variables in your deployment platform"
echo "3. Push to main branch to trigger deployment"
echo ""
echo "For detailed instructions, see: DEPLOYMENT.md"
