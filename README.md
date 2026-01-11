# k8s-service-optimizer

An intelligent Kubernetes service optimization platform that provides real-time service health analysis, automatic resource optimization recommendations, cost efficiency scoring, and traffic pattern analysis.

## Features

- **Real-time Service Health Analysis** with predictive insights
- **Automatic Resource Optimization** recommendations
- **Cost Efficiency Scoring** and optimization paths
- **Traffic Pattern Analysis** with anomaly detection
- **Interactive Dashboard** for cluster visualization
- **Automated Optimization Execution** with rollback safety

## Architecture

The platform consists of:
- **Backend API Server** (Go) - REST + WebSocket API
- **Metrics Collector** - Real-time Kubernetes metrics collection
- **Optimization Engine** - Resource analysis and recommendations
- **Cost & Traffic Analyzer** - Cost calculation and traffic pattern detection
- **Web Dashboard** (React + TypeScript) - Interactive visualization
- **Kubernetes Cluster** (kind) - 3-node local cluster with metrics-server

## Quick Start

### Prerequisites

- Docker 20.10+
- kubectl 1.24+
- kind 0.20+
- Go 1.21+
- Node 18+

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd k8s-service-optimizer
```

2. Run the complete setup:
```bash
./scripts/setup.sh
```

This will:
- Create a 3-node kind cluster
- Install and configure metrics-server
- Create namespace with RBAC
- Deploy demo workloads
- Verify everything is running

3. Build and run the backend:
```bash
cd backend
go build ./cmd/server
./server
```

4. Start the dashboard:
```bash
cd dashboard
npm install
npm run dev
```

5. Access the dashboard at `http://localhost:3000`

## Usage

### Generate Load
```bash
./scripts/load-generator.sh
```

### Check Metrics
```bash
kubectl top nodes
kubectl top pods -n k8s-optimizer
```

### View Cluster Status
```bash
kubectl -n k8s-optimizer get all
kubectl -n k8s-optimizer describe hpa
```

### Cleanup
```bash
./scripts/cleanup.sh
```

## Project Structure

```
k8s-service-optimizer/
├── backend/              # Go backend API server
├── dashboard/            # React frontend
├── infrastructure/       # Kubernetes manifests
├── deployments/          # Demo workloads
├── scripts/             # Helper scripts
└── tests/               # Integration tests
```

## Documentation

See the `Plans/` directory for detailed documentation:
- `k8s-optimizer-master.md` - Master blueprint
- `setup-guide.md` - Detailed setup guide
- `subagent-tasks.md` - Development task breakdown

## Development

The project follows a modular architecture with clear separation of concerns:

- **Metrics Collector** - Collects pod, node, and HPA metrics every 15s
- **Optimizer Engine** - Analyzes resource usage and generates recommendations
- **Cost Analyzer** - Calculates cost savings and efficiency scores
- **Traffic Analyzer** - Detects patterns and anomalies
- **API Server** - Provides REST and WebSocket endpoints

## License

MIT
