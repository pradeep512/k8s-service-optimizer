#!/bin/bash
set -e

echo "🔥 Starting load generator for echo-demo service..."
echo "Press Ctrl+C to stop"
echo ""

kubectl run -it --rm load-gen --image=busybox --restart=Never -n k8s-optimizer -- sh -c "
  echo 'Generating load on echo-demo service...'
  while true; do
    wget -q -O- http://echo-demo.k8s-optimizer.svc.cluster.local
    sleep 0.1
  done
"
