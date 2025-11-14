#!/bin/bash

# Test script for IP-based authorization

echo "=== Testing IP-Based Authorization ==="
echo ""

# Start docker compose
echo "1. Starting services..."
docker-compose up -d
sleep 10

echo "2. Testing with local IP (should be allowed - private IP)..."
curl -i -H "X-User-ID: user-1" \
     -H "X-User-Role: viewer" \
     http://localhost:8080/api/v1/documents
echo ""
echo ""

echo "3. Testing with simulated non-Japan IP (X-Forwarded-For header)..."
echo "   Using 8.8.8.8 (Google DNS - USA, should be DENIED)"
curl -i -H "X-User-ID: user-1" \
     -H "X-User-Role: viewer" \
     -H "X-Forwarded-For: 8.8.8.8" \
     http://localhost:8080/api/v1/documents
echo ""
echo ""

echo "4. Testing with simulated Japan IP (X-Forwarded-For header)..."
echo "   Using 1.0.16.1 (NTT Japan, should be ALLOWED)"
curl -i -H "X-User-ID: user-1" \
     -H "X-User-Role: viewer" \
     -H "X-Forwarded-For: 1.0.16.1" \
     http://localhost:8080/api/v1/documents
echo ""
echo ""

echo "5. Testing with private IP (X-Forwarded-For header)..."
echo "   Using 192.168.1.100 (Private IP, should be ALLOWED)"
curl -i -H "X-User-ID: user-1" \
     -H "X-User-Role: viewer" \
     -H "X-Forwarded-For: 192.168.1.100" \
     http://localhost:8080/api/v1/documents
echo ""
echo ""

echo "=== Test Complete ==="
echo ""
echo "Expected behavior:"
echo "- Step 2: 200 OK (localhost/private IP)"
echo "- Step 3: 403 Forbidden (non-Japan public IP)"
echo "- Step 4: 200 OK (Japan IP)"
echo "- Step 5: 200 OK (private IP)"
echo ""
echo "Stopping services..."
docker-compose down
