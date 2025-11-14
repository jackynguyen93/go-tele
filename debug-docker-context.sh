#!/bin/bash
# Debug script to check what's in Docker build context

echo "=== Checking .dockerignore content ==="
cat .dockerignore | grep -n "tdclient"

echo ""
echo "=== Checking if cmd/tdclient exists locally ==="
ls -la cmd/

echo ""
echo "=== Testing what files would be sent to Docker ==="
tar --exclude-from=.dockerignore -cf - . | tar -t | grep -E "^\./(cmd|go\.mod)" | head -20

echo ""
echo "=== Checking .dockerignore pattern specifically ==="
if grep -q "^/tdclient$" .dockerignore; then
    echo "✓ .dockerignore has correct pattern: /tdclient"
elif grep -q "^tdclient$" .dockerignore; then
    echo "✗ .dockerignore has WRONG pattern: tdclient (this will exclude cmd/tdclient!)"
    echo "  Fix: Change line with 'tdclient' to '/tdclient'"
else
    echo "? .dockerignore pattern not found"
fi
