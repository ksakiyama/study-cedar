#!/bin/bash

# Test script for graceful shutdown

echo "=== Testing Graceful Shutdown ==="
echo ""

# Start docker compose in background
echo "1. Starting services..."
docker-compose up -d
sleep 10

# Check health endpoint
echo ""
echo "2. Checking health endpoint (should return 200 OK)..."
curl -i http://localhost:8080/health
echo ""

# Get container name
CONTAINER_NAME="cedar-app"

# Send SIGTERM to the container
echo ""
echo "3. Sending SIGTERM to container..."
docker kill --signal=SIGTERM $CONTAINER_NAME &

# Immediately check health again (should return 503)
sleep 1
echo ""
echo "4. Checking health endpoint immediately after SIGTERM (should return 503)..."
curl -i http://localhost:8080/health 2>/dev/null || echo "Server already stopped or unreachable"

echo ""
echo "5. Waiting for graceful shutdown to complete..."
docker wait $CONTAINER_NAME

echo ""
echo "6. Checking container logs for shutdown messages..."
docker logs $CONTAINER_NAME 2>&1 | tail -20

echo ""
echo "=== Test Complete ==="
echo ""
echo "Expected behavior:"
echo "- Step 2: HTTP/1.1 200 OK with status: ok"
echo "- Step 4: HTTP/1.1 503 Service Unavailable with status: shutting_down"
echo "- Step 6: Logs should show 'Health check now returning 503' and 'Server stopped gracefully'"
