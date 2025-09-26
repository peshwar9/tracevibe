#!/bin/bash

# Simple script to run TraceVibe Docker container

set -e

# Configuration
DOCKER_HUB_USER="your-dockerhub-username"  # Replace with your Docker Hub username
IMAGE_NAME="tracevibe"
VERSION="${1:-latest}"
PORT="${2:-8080}"

echo "🚀 Starting TraceVibe Docker container..."
echo "📦 Image: ${DOCKER_HUB_USER}/${IMAGE_NAME}:${VERSION}"
echo "🌐 Port: ${PORT}"

# Stop any existing container
docker stop tracevibe 2>/dev/null || true
docker rm tracevibe 2>/dev/null || true

# Run the container
docker run -d \
    --name tracevibe \
    -p ${PORT}:8080 \
    -v tracevibe-data:/app/data \
    --restart unless-stopped \
    ${DOCKER_HUB_USER}/${IMAGE_NAME}:${VERSION}

echo "✅ TraceVibe is now running!"
echo "🌐 Access it at: http://localhost:${PORT}"
echo ""
echo "To view logs: docker logs tracevibe"
echo "To stop: docker stop tracevibe"