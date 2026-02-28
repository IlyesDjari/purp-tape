#!/bin/bash

# PurpTape Backend Setup Script
# Automates environment setup for local development

set -e

echo "🎙️  PurpTape Backend Setup"
echo "=============================="
echo ""

# Check if Docker is installed
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose not found. Install from: https://docs.docker.com/compose/install/"
    exit 1
fi

# Check if we're in the backend directory
if [ ! -f "go.mod" ]; then
    echo "❌ go.mod not found. Please run this script from the backend directory:"
    echo "   cd backend && ./setup.sh"
    exit 1
fi

# Create .env from template if it doesn't exist
if [ ! -f .env ]; then
    echo "📋 Creating .env from template..."
    cp .env.example .env
    echo "✅ .env created"
    echo ""
    echo "📝 Edit .env with your credentials:"
    echo "   - SUPABASE_URL (from https://supabase.com)"
    echo "   - SUPABASE_ANON_KEY"
    echo "   - SUPABASE_SECRET_KEY"
    echo "   - R2_* credentials (from Cloudflare)"
    echo ""
    echo "Then run this script again."
    echo ""
else
    echo "✅ .env already exists"
fi

# Start services
echo "🐳 Starting Docker Compose services..."
echo ""

docker-compose up --build

echo ""
echo "✅ Backend is running at http://localhost:8080"
echo ""
echo "Next steps:"
echo "1. Test health:    curl http://localhost:8080/health"
echo "2. Check logs:     docker-compose logs -f api"
echo "3. Stop services:  docker-compose down"

