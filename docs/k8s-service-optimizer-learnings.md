# k8s-service-optimizer: Learning Kubernetes with a Real Project


## Project Snapshot

**Project name:** k8s-service-optimizer  
**Repository:** **[[Repository Link]](https://github.com/pradeep512/k8s-service-optimizer.git)**  
**Goal:** Learn core Kubernetes concepts by deploying a multi-component app to a local multi-node cluster, observing real metrics, and validating autoscaling behavior end-to-end.

---

## What the Project Does

k8s-service-optimizer is a small platform that demonstrates how a Kubernetes-based application can be observed and tuned.

**Components**
- **Backend (Go):** Collects cluster and workload metrics, exposes REST and WebSocket APIs, and produces optimization-style insights.
- **Dashboard (React + Nginx):** Displays cluster health and recommendations. Nginx proxies API requests to the backend.
- **Demo workloads:** Purpose-built apps that generate predictable CPU/memory patterns to make metrics and scaling visible.

**Why it is useful**
- It provides a realistic, hands-on environment for practicing Kubernetes primitives (Pods, Deployments, Services, HPA, RBAC).
- It includes a repeatable deployment flow and demo workloads designed for learning.

---

## Architecture at a Glance

**Local cluster setup**
- **kind cluster (Kubernetes in Docker)** with **3 nodes**:
  - **1 control plane node**: `k8s-optimizer-control-plane`
  - **2 worker nodes**: `k8s-optimizer-worker`, `k8s-optimizer-worker2`

**Networking**
- Internal service-to-service traffic uses **ClusterIP** services.
- External access is provided using **NodePort** services mapped to localhost via kind port mappings.

**Typical runtime objects**
- `optimizer-backend` Deployment (2 replicas) + Service (NodePort)
- `optimizer-dashboard` Deployment (2 replicas) + Service (NodePort)
- `echo-demo` Deployment (HPA-enabled) + Service (ClusterIP)
- `high-cpu-app` (CPU stress) + Service (ClusterIP)
- `memory-intensive-app` (memory heavy) + Service (ClusterIP)

---

## What You Learn by Using This Project

### 1) Nodes and Scheduling (Control Plane vs Workers)
- How Kubernetes separates **control plane responsibilities** (API server, scheduler, controllers) from **workload execution** on worker nodes.
- How to confirm pod placement using:
  - `kubectl get nodes -o wide`
  - `kubectl -n <ns> get pods -o wide`

### 2) Deployments, ReplicaSets, and Pods
- How a **Deployment** manages desired state.
- How **ReplicaSets** are the mechanism used to maintain the desired number of pods.
- How updates and scaling affect the READY / UP-TO-DATE / AVAILABLE columns:
  - `kubectl -n k8s-optimizer get deploy echo-demo -w`
  - `kubectl -n k8s-optimizer get all`

### 3) Services and Service Discovery
- Difference between **ClusterIP** (internal) and **NodePort** (host-accessible).
- How a Service load-balances across pod endpoints.
- How to validate active backends behind a Service:
  - `kubectl -n k8s-optimizer get endpoints echo-demo -o wide`

### 4) Metrics Pipeline (metrics-server)
- Why `metrics-server` matters: it powers `kubectl top` and HPA decisions.
- How to inspect live resource usage:
  - `kubectl top nodes`
  - `kubectl -n k8s-optimizer top pods`
  - `kubectl -n k8s-optimizer top pods -l app=echo-demo`

### 5) HPA in Practice (Autoscaling with Real Load)
- HPA does **not** create pods directly. It updates the target Deployment’s desired replicas.
- HPA CPU metrics are based on **CPU usage as a % of the pod’s CPU request** (not node CPU %).
- Watching autoscaling live:
  - `kubectl -n k8s-optimizer get hpa -w`

**Observed scaling behavior**
- Traffic increased CPU utilization above the HPA target.
- HPA increased replicas up to `maxReplicas`.
- Scale-down occurred gradually (stabilization prevents rapid flapping).

### 6) Load Generation and Debugging “Is Traffic Still Running?”
A simple in-cluster load generator can be created with a one-off Pod (`busybox`) that repeatedly calls the `echo-demo` service.

**Key learning**
- If the load generator Pod keeps running, it continues generating traffic.
- Pressing Ctrl+C can stop the client attachment, but the Pod may still exist unless it is deleted.
- To check whether traffic is still being generated:
  - Is `load-gen` running?  
    `kubectl -n k8s-optimizer get pod load-gen`
  - Are `echo-demo` pods using CPU?  
    `kubectl -n k8s-optimizer top pods -l app=echo-demo`
  - Does the Service have endpoints?  
    `kubectl -n k8s-optimizer get endpoints echo-demo -o wide`

---

## Repro Steps Anyone Can Follow

1. Create the kind cluster (3 nodes)
2. Install metrics-server
3. Apply namespace + RBAC manifests
4. Deploy demo workloads
5. Build and load backend/dashboard images into kind
6. Deploy optimizer backend + dashboard
7. Generate load and watch HPA + Deployment scale

(These are commonly automated by a `deploy-all.sh` style script.)

---

## Practical Commands (Copy/Paste)

**Cluster and nodes**
```bash
kubectl get nodes -o wide
```

**All resources in the project namespace**
```bash
kubectl -n k8s-optimizer get all
```

**Where pods are scheduled**
```bash
kubectl -n k8s-optimizer get pods -o wide
```

**Resource usage**
```bash
kubectl -n k8s-optimizer top pods
kubectl -n k8s-optimizer top pods -l app=echo-demo
```

**Watch autoscaling**
```bash
kubectl -n k8s-optimizer get hpa -w
kubectl -n k8s-optimizer get deploy echo-demo -w
```

**Service endpoints**
```bash
kubectl -n k8s-optimizer get endpoints echo-demo -o wide
```

**Stop load generator**
```bash
kubectl -n k8s-optimizer delete pod load-gen
```

---

## Key Takeaways

- Deployed and operated a **multi-service Kubernetes app** (backend + dashboard + workloads) on a **local 3-node kind cluster**.
- Practiced Kubernetes fundamentals: **Pods, Deployments, ReplicaSets, Services, Namespaces, and RBAC**.
- Implemented and validated **observability** via **metrics-server** and `kubectl top`.
- Demonstrated **Horizontal Pod Autoscaling (HPA)** under load and learned how scaling decisions are derived from **CPU requests vs actual usage**.
- Built troubleshooting habits using **endpoints**, **pod status**, and **live watches** (`-w`) to confirm what the cluster is doing in real time.

---

## Optional: Improvements / Next Steps

- Add **Ingress** (or Gateway API) to replace NodePort access patterns.
- Add **Prometheus + Grafana** dashboards for long-term metrics and richer visualization.
- Add **resource optimization recommendations** as actionable patches (requests/limits tuning).
- Add **NetworkPolicies** to practice namespace and service-to-service isolation.

---

## Media Suggestions for Your Post

- Screenshot: `kubectl get nodes -o wide`
- Screenshot: HPA scaling output (`kubectl get hpa -w`)
- Screenshot: dashboard UI at `localhost:3000`
- Short clip: load generator running, replicas scaling up, then scaling down

