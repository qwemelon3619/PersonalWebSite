#!/bin/bash

# Build script for Personal Blog microservices

set -e

echo "Building Personal Blog microservices..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Build services
echo "Building Auth Service..."
cd services/auth-service && go build -o ../../bin/auth-service ./cmd/main.go && cd ../..

echo "Building Post Service..."
cd services/post-service && go build -o ../../bin/post-service ./cmd/main.go && cd ../..

echo "Building API Gateway..."
cd services/api-gateway && go build -o ../../bin/api-gateway ./cmd/main.go && cd ../..

echo "All services built successfully!"
echo "Binaries are available in the 'bin' directory."