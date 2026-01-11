# Setup Guide - k8s-service-optimizer

## Prerequisites Checklist

### Required Software

```bash
# Check versions
docker version          # Docker 20.10+
kubectl version --client # kubectl 1.24+
kind version            # kind 0.20+
go version             # Go 1.21+ (for backend)
node --version         # Node 18+ (for dashboard)
```

### Install Missing Tools

**macOS**:
```bash
brew install docker kubectl kind go node
```

**Linux (Ubuntu/Debian)**:
```bash
# Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# Go
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/bin/go/bin

# Node (via nvm recommended)
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
nvm install 18
```

**Windows**:
Use WSL2 with Ubuntu and follow Linux instructions above.

---

## Initial Project Setup

### 1. Create Project Structure

```bash
# Create project root
mkdir -p k8s-service-optimizer
cd k8s-service-optimizer

# Create directory structure
mkdir -p {docs,infrastructure/{kind,k8s/{crds,rbac},monitoring},backend/{cmd/server,pkg/{collector,optimizer,analyzer,recommender,api},internal/{k8s,models}},dashboard/src/{components,hooks,services},deployments/{demo-workloads,optimizer},scripts,tests/{integration,load}}

# Initialize Git
git init
echo "node_modules/
vendor/
*.log
.env
.DS_Store
kind-config-*.yaml" > .gitignore
```

### 2. Create Kind Cluster Configuration

**File: `infrastructure/kind/cluster-config.yaml`**

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: k8s-optimizer
nodes:
  - role: control-plane
    kubeadmConfigPatches:
    - |
      kind: InitConfiguration
      nodeRegistration:
        kubeletExtraArgs:
          node-labels: "node-role=control-plane"
    extraPortMappings:
    # Dashboard
    - containerPort: 30080
      hostPort: 3000
      protocol: TCP
    # Backend API
    - containerPort: 30081
      hostPort: 8080
      protocol: TCP
  
  - role: worker
    kubeadmConfigPatches:
    - |
      kind: JoinConfiguration
      nodeRegistration:
        kubeletExtraArgs:
          node-labels: "node-role=worker,workload-type=general"
  
  - role: worker
    kubeadmConfigPatches:
    - |
      kind: JoinConfiguration
      nodeRegistration:
        kubeletExtraArgs:
          node-labels: "node-role=worker,workload-type=general"
```

**File: `infrastructure/kind/setup.sh`**

```bash
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
```

### 3. Deploy Metrics Server

**File: `infrastructure/k8s/metrics-server/deploy.sh`**

```bash
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
```

### 4. Create Namespace and RBAC

**File: `infrastructure/k8s/namespace.yaml`**

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: k8s-optimizer
  labels:
    name: k8s-optimizer
    managed-by: k8s-service-optimizer
```

**File: `infrastructure/k8s/rbac/service-account.yaml`**

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: optimizer-sa
  namespace: k8s-optimizer
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: optimizer-role
rules:
  # Read all resources for analysis
  - apiGroups: [""]
    resources: ["pods", "services", "endpoints", "nodes", "namespaces", "events"]
    verbs: ["get", "list", "watch"]
  
  - apiGroups: ["apps"]
    resources: ["deployments", "replicasets", "statefulsets", "daemonsets"]
    verbs: ["get", "list", "watch", "update", "patch"]
  
  - apiGroups: ["autoscaling"]
    resources: ["horizontalpodautoscalers"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]
  
  - apiGroups: ["metrics.k8s.io"]
    resources: ["pods", "nodes"]
    verbs: ["get", "list"]
  
  # Custom resources for optimizer
  - apiGroups: ["optimizer.k8s.io"]
    resources: ["optimizationrecommendations", "optimizationpolicies"]
    verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: optimizer-binding
subjects:
  - kind: ServiceAccount
    name: optimizer-sa
    namespace: k8s-optimizer
roleRef:
  kind: ClusterRole
  name: optimizer-role
  apiGroup: rbac.authorization.k8s.io
```

### 5. Create Demo Workloads

**File: `deployments/demo-workloads/echo-service.yaml`**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: echo-demo
  namespace: k8s-optimizer
  labels:
    app: echo-demo
    tier: frontend
spec:
  replicas: 3
  selector:
    matchLabels:
      app: echo-demo
  template:
    metadata:
      labels:
        app: echo-demo
        tier: frontend
    spec:
      containers:
        - name: echo
          image: ealen/echo-server:0.9.2
          ports:
            - containerPort: 80
              name: http
          resources:
            requests:
              cpu: 50m
              memory: 64Mi
            limits:
              cpu: 200m
              memory: 128Mi
          readinessProbe:
            httpGet:
              path: /
              port: 80
            initialDelaySeconds: 5
            periodSeconds: 5
          livenessProbe:
            httpGet:
              path: /
              port: 80
            initialDelaySeconds: 10
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: echo-demo
  namespace: k8s-optimizer
spec:
  selector:
    app: echo-demo
  ports:
    - port: 80
      targetPort: 80
      name: http
  type: ClusterIP
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: echo-demo-hpa
  namespace: k8s-optimizer
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: echo-demo
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 50
```

### 6. Master Setup Script

**File: `scripts/setup.sh`**

```bash
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
kubectl -n k8s-optimizer wait --for=condition=Ready pods --all --timeout=120s

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
echo "  3. Initialize backend: cd backend && go mod init github.com/yourusername/k8s-service-optimizer"
echo "  4. Initialize dashboard: cd dashboard && npm create vite@latest . -- --template react-ts"
```

Make it executable:
```bash
chmod +x scripts/setup.sh
chmod +x infrastructure/kind/setup.sh
chmod +x infrastructure/k8s/metrics-server/deploy.sh
```

---

## Running the Setup

```bash
# From project root
./scripts/setup.sh
```

This will:
1. ✅ Create 3-node kind cluster
2. ✅ Install and configure metrics-server
3. ✅ Create namespace with RBAC
4. ✅ Deploy demo workloads
5. ✅ Verify everything is running

---

## Verification Commands

```bash
# Check cluster
kubectl cluster-info
kubectl get nodes -o wide

# Check metrics
kubectl top nodes
kubectl top pods -n k8s-optimizer

# Check demo workloads
kubectl -n k8s-optimizer get all
kubectl -n k8s-optimizer describe hpa

# Generate some load (optional)
kubectl run -it --rm load-gen --image=busybox -n k8s-optimizer -- sh -c "while true; do wget -q -O- http://echo-demo.k8s-optimizer.svc.cluster.local; done"
```

---

## Troubleshooting

**Metrics not available**:
```bash
kubectl -n kube-system logs -l k8s-app=metrics-server
kubectl -n kube-system describe deployment metrics-server
```

**Pods pending**:
```bash
kubectl -n k8s-optimizer describe pod <pod-name>
kubectl get events -n k8s-optimizer --sort-by='.lastTimestamp'
```

**Kind cluster issues**:
```bash
kind delete cluster --name k8s-optimizer
# Then re-run setup
```

---

## Next Steps

Once setup is complete, proceed to:
- **`02-CORE-COMPONENTS.md`** - Backend implementation
- **`03-DASHBOARD.md`** - Frontend development
- **`SUBAGENT-TASKS.md`** - Task delegation strategy

---

## Resource Requirements

- **Minimum**: 8 GB RAM, 4 CPU cores
- **Recommended**: 16 GB RAM, 6+ CPU cores
- **Disk**: ~10 GB for Docker images and data

The cluster will use approximately:
- Control plane: ~1 GB RAM
- Each worker: ~500 MB RAM
- Demo workloads: ~300 MB RAM
- Metrics server: ~100 MB RAM
