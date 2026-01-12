#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

echo "🚀 Deploying k8s-service-optimizer to cluster"
echo "=============================================="
echo ""

# Deploy backend
echo "📦 Deploying backend..."
kubectl apply -f deployments/optimizer/backend-deployment.yaml
echo "✅ Backend deployed"
echo ""

# Deploy dashboard
echo "📦 Deploying dashboard..."
kubectl apply -f deployments/optimizer/dashboard-deployment.yaml
echo "✅ Dashboard deployed"
echo ""

# Wait for deployments
echo "⏳ Waiting for deployments to be ready..."
kubectl -n k8s-optimizer wait --for=condition=Available deployment/optimizer-backend --timeout=120s
kubectl -n k8s-optimizer wait --for=condition=Available deployment/optimizer-dashboard --timeout=120s
echo ""

# Show status
echo "✅ Deployment complete!"
echo ""
echo "📊 Status:"
kubectl -n k8s-optimizer get deployments,pods,svc
echo ""
echo "🌐 Access URLs:"
echo "  Dashboard: http://localhost:3000"
echo "  Backend API: http://localhost:8080"
echo "  Backend Health: http://localhost:8080/health"
echo ""
echo "📝 Useful commands:"
echo "  View logs (backend):    kubectl -n k8s-optimizer logs -l app=optimizer-backend -f"
echo "  View logs (dashboard):  kubectl -n k8s-optimizer logs -l app=optimizer-dashboard -f"
echo "  Delete deployment:      kubectl -n k8s-optimizer delete -f deployments/optimizer/"
