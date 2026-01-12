# K8s Service Optimizer - API Server Quick Start

## Prerequisites

- Go 1.19 or later
- Access to a Kubernetes cluster
- kubectl configured
- Metrics Server installed in the cluster (optional but recommended)

## Quick Start

### 1. Build the Server

```bash
cd backend
go build -o server ./cmd/server/
```

### 2. Run the Server

```bash
# With defaults (port 8080, monitors 'default' namespace)
./server

# With custom configuration
PORT=9000 NAMESPACES=default,production LOG_LEVEL=debug ./server
```

### 3. Test the Server

```bash
# Run automated tests
./test-api.sh

# Or test manually
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/cluster/overview
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `LOG_LEVEL` | Logging level (info, debug, error) | `info` |
| `UPDATE_INTERVAL` | WebSocket update interval | `5s` |
| `NAMESPACES` | Comma-separated namespaces to monitor | `default` |
| `KUBECONFIG` | Path to kubeconfig file | `~/.kube/config` |

## API Endpoints

### Health & Status
- `GET /health` - Health check
- `GET /ready` - Readiness check
- `GET /api/v1/status` - System status

### Cluster Information
- `GET /api/v1/cluster/overview` - Cluster overview with node/pod stats
- `GET /api/v1/services` - List all services across namespaces
- `GET /api/v1/services/{namespace}/{name}` - Detailed service information

### Metrics
- `GET /api/v1/metrics/nodes` - Current node metrics
- `GET /api/v1/metrics/pods/{namespace}` - Pod metrics for namespace
- `GET /api/v1/metrics/timeseries?resource=X&metric=Y&duration=Z` - Time series data

### Recommendations
- `GET /api/v1/recommendations` - All optimization recommendations
- `GET /api/v1/recommendations/{id}` - Specific recommendation
- `POST /api/v1/recommendations/{id}/apply` - Apply recommendation

### Analysis
- `GET /api/v1/analysis/{namespace}/{service}` - Service analysis
- `GET /api/v1/traffic/{namespace}/{service}` - Traffic patterns
- `GET /api/v1/cost/{namespace}/{service}` - Cost breakdown
- `GET /api/v1/anomalies?resource=X&duration=Y` - Detected anomalies

### WebSocket
- `WS /ws/updates` - Real-time updates (connects with WebSocket client)

## Example Requests

### Get Cluster Overview
```bash
curl http://localhost:8080/api/v1/cluster/overview | jq
```

### Get Service Details
```bash
curl http://localhost:8080/api/v1/services/default/my-service | jq
```

### Get Recommendations
```bash
curl http://localhost:8080/api/v1/recommendations | jq
```

### Get Time Series Data
```bash
curl "http://localhost:8080/api/v1/metrics/timeseries?resource=node/worker-1&metric=cpu&duration=1h" | jq
```

### WebSocket Connection
```bash
# Using websocat (install: cargo install websocat)
websocat ws://localhost:8080/ws/updates

# Using wscat (install: npm install -g wscat)
wscat -c ws://localhost:8080/ws/updates
```

## Response Format

### Success Response
```json
{
  "success": true,
  "data": {
    // Response data here
  }
}
```

### Error Response
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message"
  }
}
```

## WebSocket Messages

The WebSocket sends periodic updates (every 5 seconds by default):

### Metrics Update
```json
{
  "type": "metrics_update",
  "timestamp": "2024-01-11T12:00:00Z",
  "data": {
    "type": "nodes",
    "metrics": [...]
  }
}
```

### Recommendation Update
```json
{
  "type": "recommendation_new",
  "timestamp": "2024-01-11T12:00:00Z",
  "data": {
    "count": 3,
    "recommendations": [...]
  }
}
```

## Troubleshooting

### Server won't start
- Check if port is already in use: `lsof -i :8080`
- Verify Kubernetes access: `kubectl cluster-info`
- Check kubeconfig: `echo $KUBECONFIG`

### Endpoints return errors
- Check server logs for details
- Verify cluster is accessible: `kubectl get nodes`
- Ensure metrics-server is installed: `kubectl top nodes`

### WebSocket won't connect
- Check CORS settings (should allow your origin)
- Verify WebSocket URL format: `ws://localhost:8080/ws/updates`
- Check browser console for errors

## Development

### Run Tests
```bash
go test ./pkg/api/... -v
```

### Build with Race Detection
```bash
go build -race -o server ./cmd/server/
```

### Run with Debug Logging
```bash
LOG_LEVEL=debug ./server
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    API Server (port 8080)                │
├─────────────────────────────────────────────────────────┤
│                                                           │
│  REST API Endpoints          WebSocket Hub               │
│  ├─ Health & Status          ├─ Client Management        │
│  ├─ Cluster Info             ├─ Broadcast Updates        │
│  ├─ Metrics                  └─ Real-time Notifications  │
│  ├─ Recommendations                                       │
│  └─ Analysis                                              │
│                                                           │
├─────────────────────────────────────────────────────────┤
│  Middleware Chain                                         │
│  └─ Recovery → Logging → CORS → Request ID               │
├─────────────────────────────────────────────────────────┤
│                                                           │
│  Backend Components Integration                          │
│  ├─ Metrics Collector (1,039 LOC)                        │
│  ├─ Optimizer Engine (2,448 LOC)                         │
│  └─ Traffic & Cost Analyzer (2,309 LOC)                  │
│                                                           │
├─────────────────────────────────────────────────────────┤
│  Kubernetes Client                                        │
│  └─ Cluster Access & Resource Queries                    │
└─────────────────────────────────────────────────────────┘
```

## Next Steps

1. **For Dashboard Integration**
   - Connect React dashboard to REST API
   - Establish WebSocket connection for real-time updates
   - Use endpoints to fetch and display data

2. **For Production Deployment**
   - Add authentication/authorization
   - Enable TLS/HTTPS
   - Configure rate limiting
   - Set up monitoring and alerting
   - Implement horizontal scaling

3. **For Advanced Features**
   - Add more granular metrics
   - Implement custom alert rules
   - Add historical data export
   - Create API documentation (Swagger/OpenAPI)

## Documentation

- **API Documentation**: `pkg/api/README.md`
- **Implementation Summary**: `API_IMPLEMENTATION_SUMMARY.md`
- **Test Script**: `test-api.sh`

## Support

For issues or questions:
1. Check the logs: Server outputs detailed logs including request IDs
2. Review error responses: All errors include codes and messages
3. Use the test script: `./test-api.sh` to verify functionality

## License

Part of the k8s-service-optimizer project.
