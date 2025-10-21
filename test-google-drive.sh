#!/bin/bash

# Test Google Drive functionality
echo "🔍 Testing Google Drive functionality..."
echo "========================================"

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "❌ .env file not found!"
    echo "📝 Please copy .env.example to .env and configure your Google Drive credentials"
    echo "   cp .env.example .env"
    exit 1
fi

# Check if required environment variables are set
echo "🔧 Checking environment variables..."
MISSING_VARS=()

if [ -z "$GOOGLE_DRIVE_CLIENT_ID" ]; then
    MISSING_VARS+=("GOOGLE_DRIVE_CLIENT_ID")
fi

if [ -z "$GOOGLE_DRIVE_CLIENT_SECRET" ]; then
    MISSING_VARS+=("GOOGLE_DRIVE_CLIENT_SECRET")
fi

if [ ${#MISSING_VARS[@]} -ne 0 ]; then
    echo "❌ Missing required environment variables:"
    for var in "${MISSING_VARS[@]}"; do
        echo "   - $var"
    done
    echo ""
    echo "📝 Please set these in your .env file:"
    echo "   GOOGLE_DRIVE_CLIENT_ID=your-client-id"
    echo "   GOOGLE_DRIVE_CLIENT_SECRET=your-client-secret"
    exit 1
fi

echo "✅ Environment variables are configured"

# Test if the backend compiles
echo ""
echo "🔨 Testing backend compilation..."
if go build .; then
    echo "✅ Backend compiles successfully"
else
    echo "❌ Backend compilation failed"
    exit 1
fi

# Start the backend
echo ""
echo "🚀 Starting backend server..."
echo "📝 The server will be available at: https://chama-backend-server.vercel.app"
echo "🔗 Test Google Drive endpoints:"
echo "   - GET  /api/v1/users/google-drive/status"
echo "   - POST /api/v1/users/google-drive/backup"
echo "   - GET  /api/v1/users/google-drive/auth-url"
echo ""
echo "⚠️  Remember to authenticate first to get a JWT token"
echo ""

go run .