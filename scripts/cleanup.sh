#!/bin/bash
set -e

echo "🧹 Cleaning up k8s-service-optimizer..."
echo ""

# Ask for confirmation
echo "⚠️  This will delete the entire cluster. Are you sure? (y/n)"
read -r response

if [[ "$response" != "y" ]]; then
    echo "Cleanup cancelled."
    exit 0
fi

# Delete kind cluster
echo "Deleting kind cluster..."
kind delete cluster --name k8s-optimizer

echo ""
echo "✅ Cleanup complete!"
