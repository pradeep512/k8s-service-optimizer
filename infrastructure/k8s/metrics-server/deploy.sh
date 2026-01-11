#!/bin/bash
set -e

echo "📊 Installing metrics-server..."

# Download latest metrics-server
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# Patch for kind (insecure TLS)
kubectl -n kube-system patch deployment metrics-server --type='json' -p='[
  {"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"},
  {"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname"},
  {"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--metric-resolution=15s"}
]'

echo "⏳ Waiting for metrics-server to be ready..."
kubectl -n kube-system rollout status deployment metrics-server --timeout=120s

echo "✅ Metrics-server ready!"
echo ""
echo "Testing metrics collection:"
sleep 10
kubectl top nodes || echo "⚠️  Metrics not ready yet, wait 30s and try: kubectl top nodes"
