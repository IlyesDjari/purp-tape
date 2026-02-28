#!/bin/bash

# PurpTape Backend Setup Script
# Automates environment setup for local development

set -e

echo "🎙️  PurpTape Backend Setup"
echo "=============================="
echo ""

# Navigate to backend directory
cd backend

# Check if .env exists
if [ ! -f .env ]; then
    echo "📋 Creating .env from template..."
    cp .env.example .env
    echo "✅ .env created. Please edit it with your credentials:"
    echo ""
    echo "   nano .env"
    echo ""
    echo "   Required:"
    echo "   - SUPABASE_URL (from https://supabase.com)"
    echo "   - SUPABASE_ANON_KEY"
    echo "   - R2_* credentials (from Cloudflare)"
else
    echo "✅ .env already exists"
fi

echo ""
echo "🐳 Starting Docker Compose..."
echo ""

# Check if Docker is installed
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose not found. Install from: https://docs.docker.com/compose/install/"
    exit 1
fi

# Start services
docker-compose up --build

echo ""
echo "✅ Backend is running at http://localhost:8080"
echo ""
echo "Next steps:"
echo "1. Test health: curl http://localhost:8080/health"
echo "2. Create a Supabase token and test the API"
echo "3. Check logs: docker-compose logs -f api"
