#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

echo "🔨 Building Docker images for k8s-service-optimizer"
echo "=================================================="
echo ""

# Build backend image
echo "📦 Building backend image..."
cd backend
docker build -t k8s-optimizer-backend:latest .
echo "✅ Backend image built successfully"
echo ""

# Build dashboard image
echo "📦 Building dashboard image..."
cd ../dashboard
docker build -t k8s-optimizer-dashboard:latest .
echo "✅ Dashboard image built successfully"
echo ""

# Load images into kind cluster
echo "📥 Loading images into kind cluster..."
kind load docker-image k8s-optimizer-backend:latest --name k8s-optimizer
kind load docker-image k8s-optimizer-dashboard:latest --name k8s-optimizer
echo "✅ Images loaded into kind cluster"
echo ""

echo "✅ All images built and loaded successfully!"
echo ""
echo "Images created:"
echo "  - k8s-optimizer-backend:latest"
echo "  - k8s-optimizer-dashboard:latest"
echo ""
echo "Next: Run ./scripts/deploy.sh to deploy to cluster"
