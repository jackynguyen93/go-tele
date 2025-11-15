#!/bin/bash
set -e

echo "============================================"
echo "Building TDLib Base Image (one-time setup)"
echo "============================================"
echo ""
echo "This builds TDLib once and caches it forever."
echo "You only need to run this script ONCE."
echo ""

# Build the base TDLib image
docker build -f Dockerfile.tdlib -t tdlib-base:1.8.19 .

echo ""
echo "✓ TDLib base image built successfully!"
echo ""
echo "Now you can use fast rebuilds with:"
echo "  docker-compose -f docker-compose.fast.yml build"
echo ""
echo "Future builds will only rebuild your Go code (~30 seconds)"
echo "instead of rebuilding TDLib (~10 minutes)"
