#!/bin/bash

set -e

echo "🚀 Setting up NeotexAI MVP..."

# Check for required tools
command -v docker >/dev/null 2>&1 || { echo "❌ Docker is required but not installed."; exit 1; }
command -v docker-compose >/dev/null 2>&1 || { echo "❌ Docker Compose is required but not installed."; exit 1; }
command -v go >/dev/null 2>&1 || { echo "❌ Go is required but not installed."; exit 1; }

echo "✅ All required tools found"

# Start Docker Compose
echo "📦 Starting Docker Compose services..."
docker-compose up -d

# Wait for PostgreSQL to be ready
echo "⏳ Waiting for PostgreSQL to be ready..."
sleep 5

# Run migrations
echo "🗄️  Running database migrations..."
make migrate-up

# Build binaries
echo "🔨 Building binaries..."
make build

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "📝 Creating .env file..."
    cp .env.example .env
    echo "⚠️  Remember to update OPENAI_API_KEY in .env file"
fi

echo ""
echo "✅ Setup complete!"
echo ""
echo "To start the server:"
echo "  ./bin/neotexd serve"
echo ""
echo "To bootstrap (create org and API key):"
echo "  ORG_ID=\$(./bin/neotexd org create \"MyOrg\" --output=json | jq -r '.id')"
echo "  API_KEY=\$(./bin/neotexd apikey create --org \$ORG_ID --name \"dev-key\" --output=json | jq -r '.token')"
echo "  echo \"NEOTEX_API_KEY=\$API_KEY\" >> .env"
echo ""
echo "Note: .env file is loaded automatically - no 'source' needed!"
echo ""
