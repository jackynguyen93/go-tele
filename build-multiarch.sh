#!/bin/bash

# Build multi-architecture Docker image for both ARM64 and AMD64
# This allows the image to run on Apple Silicon Macs AND Linux servers

set -e

# Configuration
REGISTRY="ghcr.io/jackynguyen93"
IMAGE_NAME="tdclient"
VERSION="v0.7.6"

echo "========================================="
echo "Building Multi-Architecture Docker Image"
echo "========================================="
echo "Registry: ${REGISTRY}"
echo "Image: ${IMAGE_NAME}"
echo "Version: ${VERSION}"
echo "Platforms: linux/amd64, linux/arm64"
echo ""

# Check if buildx is available
if ! docker buildx version &> /dev/null; then
    echo "Error: docker buildx is not available"
    echo "Please enable Docker BuildKit"
    exit 1
fi

# Create and use a new builder instance
echo "Setting up buildx builder..."
docker buildx create --name multiarch-builder --use --bootstrap 2>/dev/null || docker buildx use multiarch-builder

# Login to GitHub Container Registry
echo ""
echo "Please make sure you're logged in to GitHub Container Registry:"
echo "  echo YOUR_TOKEN | docker login ghcr.io -u jackynguyen93 --password-stdin"
echo ""
read -p "Press Enter to continue..."

# Build and push multi-architecture image
echo ""
echo "Building and pushing multi-architecture image..."
echo "This will take several minutes..."
echo ""

docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag ${REGISTRY}/${IMAGE_NAME}:${VERSION} \
  --tag ${REGISTRY}/${IMAGE_NAME}:latest \
  --file Dockerfile.cached \
  --push \
  .

echo ""
echo "========================================="
echo "Build Complete!"
echo "========================================="
echo ""
echo "Image pushed to:"
echo "  ${REGISTRY}/${IMAGE_NAME}:${VERSION}"
echo "  ${REGISTRY}/${IMAGE_NAME}:latest"
echo ""
echo "The image now supports both:"
echo "  - linux/amd64 (Intel/AMD servers)"
echo "  - linux/arm64 (Apple Silicon Macs)"
echo ""
echo "On your server, pull the new image:"
echo "  docker pull ${REGISTRY}/${IMAGE_NAME}:latest"
echo "  docker compose up -d"
echo ""
