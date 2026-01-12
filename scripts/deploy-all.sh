#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

echo "🚀 k8s-service-optimizer Complete Deployment"
echo "=============================================="
echo ""
echo "This script will:"
echo "  1. Setup kind cluster with 3 nodes"
echo "  2. Install metrics-server"
echo "  3. Create namespace and RBAC"
echo "  4. Deploy demo workloads"
echo "  5. Build Docker images"
echo "  6. Deploy optimizer backend and dashboard"
echo ""
read -p "Continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Deployment cancelled"
    exit 1
fi
echo ""

# Step 1: Setup cluster
echo "📍 Step 1/6: Setting up kind cluster..."
if kind get clusters 2>/dev/null | grep -q "k8s-optimizer"; then
    echo "⚠️  Cluster 'k8s-optimizer' already exists."
    read -p "Delete and recreate? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        kind delete cluster --name k8s-optimizer
        bash scripts/setup.sh
    else
        echo "Using existing cluster..."
    fi
else
    bash infrastructure/kind/setup.sh
fi
echo ""

# Step 2: Install metrics-server
echo "📍 Step 2/6: Installing metrics-server..."
bash infrastructure/k8s/metrics-server/deploy.sh
echo ""

# Step 3: Create namespace and RBAC
echo "📍 Step 3/6: Setting up namespace and RBAC..."
kubectl apply -f infrastructure/k8s/namespace.yaml
kubectl apply -f infrastructure/k8s/rbac/
echo "✅ Namespace and RBAC configured"
echo ""

# Step 4: Deploy demo workloads
echo "📍 Step 4/6: Deploying demo workloads..."
kubectl apply -f deployments/demo-workloads/
echo "⏳ Waiting for demo workloads to be ready..."
kubectl -n k8s-optimizer wait --for=condition=Ready pods --all --timeout=120s
echo "✅ Demo workloads deployed"
echo ""

# Step 5: Build Docker images
echo "📍 Step 5/6: Building Docker images..."
bash scripts/build-images.sh
echo ""

# Step 6: Deploy optimizer
echo "📍 Step 6/6: Deploying optimizer..."
bash scripts/deploy.sh
echo ""

echo "🎉 Deployment complete!"
echo ""
echo "📊 Cluster Summary:"
kubectl get nodes
echo ""
kubectl -n k8s-optimizer get all
echo ""
echo "🌐 Access URLs:"
echo "  Dashboard: http://localhost:3000"
echo "  Backend API: http://localhost:8080"
echo "  API Docs: http://localhost:8080/api/v1/status"
echo ""
echo "🧪 Test the system:"
echo "  curl http://localhost:8080/health"
echo "  curl http://localhost:8080/api/v1/cluster/overview"
echo "  curl http://localhost:8080/api/v1/recommendations"
echo ""
echo "📝 Generate load:"
echo "  ./scripts/load-generator.sh"
echo ""
echo "🔍 View logs:"
echo "  kubectl -n k8s-optimizer logs -l app=optimizer-backend -f"
echo ""
echo "🧹 Cleanup:"
echo "  ./scripts/cleanup.sh"
