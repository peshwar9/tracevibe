#!/bin/bash

# Multi-platform Docker build script for TraceVibe

set -e

# Configuration
DOCKER_HUB_USER="your-dockerhub-username"  # Replace with your Docker Hub username
IMAGE_NAME="tracevibe"
VERSION="${1:-latest}"

echo "🚀 Building TraceVibe Docker image for multiple platforms..."
echo "📦 Image: ${DOCKER_HUB_USER}/${IMAGE_NAME}:${VERSION}"

# Check if Docker buildx is available
if ! docker buildx version > /dev/null 2>&1; then
    echo "❌ Docker buildx is not available. Please install Docker Desktop or enable buildx."
    exit 1
fi

# Create and use a new builder instance (supports multi-platform)
echo "🔧 Setting up Docker buildx..."
docker buildx create --name tracevibe-builder --use --bootstrap 2>/dev/null || docker buildx use tracevibe-builder

# Build and push multi-platform image
echo "🏗️  Building for linux/amd64, linux/arm64, linux/arm/v7..."
docker buildx build \
    --platform linux/amd64,linux/arm64,linux/arm/v7 \
    --tag ${DOCKER_HUB_USER}/${IMAGE_NAME}:${VERSION} \
    --tag ${DOCKER_HUB_USER}/${IMAGE_NAME}:latest \
    --push \
    .

echo "✅ Multi-platform build complete!"
echo "🐳 Image available at: ${DOCKER_HUB_USER}/${IMAGE_NAME}:${VERSION}"
echo ""
echo "To test the image:"
echo "  docker run -p 8080:8080 ${DOCKER_HUB_USER}/${IMAGE_NAME}:${VERSION}"
echo ""
echo "To test with persistent data:"
echo "  docker run -p 8080:8080 -v tracevibe-data:/app/data ${DOCKER_HUB_USER}/${IMAGE_NAME}:${VERSION}"