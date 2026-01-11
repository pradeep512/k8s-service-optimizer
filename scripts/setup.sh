#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

echo "🔧 k8s-service-optimizer Complete Setup"
echo "========================================"
echo ""

# Step 1: Create cluster
if kind get clusters 2>/dev/null | grep -q "k8s-optimizer"; then
    echo "⚠️  Cluster 'k8s-optimizer' already exists. Delete it? (y/n)"
    read -r response
    if [[ "$response" == "y" ]]; then
        kind delete cluster --name k8s-optimizer
    else
        echo "Using existing cluster..."
    fi
fi

bash infrastructure/kind/setup.sh

# Step 2: Install metrics-server
bash infrastructure/k8s/metrics-server/deploy.sh

# Step 3: Create namespace and RBAC
echo ""
echo "🔐 Setting up namespace and permissions..."
kubectl apply -f infrastructure/k8s/namespace.yaml
kubectl apply -f infrastructure/k8s/rbac/

# Step 4: Deploy demo workloads
echo ""
echo "🚢 Deploying demo workloads..."
kubectl apply -f deployments/demo-workloads/

# Step 5: Wait for everything
echo ""
echo "⏳ Waiting for all pods to be ready..."
kubectl -n k8s-optimizer wait --for=condition=Ready pods --all --timeout=120s || echo "⚠️  Some pods may still be starting..."

echo ""
echo "✅ Setup complete!"
echo ""
echo "📊 Cluster Status:"
kubectl get nodes
echo ""
kubectl -n k8s-optimizer get pods,svc,hpa
echo ""
echo "Next steps:"
echo "  1. Test metrics: kubectl top nodes"
echo "  2. Check pods: kubectl -n k8s-optimizer get pods -o wide"
echo "  3. Build backend: cd backend && go build ./cmd/server"
echo "  4. Start dashboard: cd dashboard && npm run dev"
