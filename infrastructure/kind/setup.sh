#!/bin/bash
set -e

echo "🚀 Setting up k8s-service-optimizer cluster..."

# Create cluster
kind create cluster --config infrastructure/kind/cluster-config.yaml --name k8s-optimizer

# Wait for cluster to be ready
echo "⏳ Waiting for cluster to be ready..."
kubectl wait --for=condition=Ready nodes --all --timeout=120s

# Show cluster info
echo "✅ Cluster created successfully!"
kubectl cluster-info --context kind-k8s-optimizer
kubectl get nodes -o wide

echo ""
echo "📊 Cluster Summary:"
kubectl get nodes --no-headers | wc -l | xargs -I {} echo "  Nodes: {}"
echo "  Context: kind-k8s-optimizer"
