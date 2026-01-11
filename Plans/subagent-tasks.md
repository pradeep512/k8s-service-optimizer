# Subagent Task Delegation Strategy

## Purpose

This document defines **when** and **how** to use Claude Code subagents to keep the main conversation focused on architecture, integration, and orchestration while delegating implementation details.

---

## Golden Rule

**Delegate to a subagent when**:
1. The task is self-contained (clear inputs/outputs)
2. Implementation requires 200+ lines of code
3. The task has specific technical requirements
4. You can verify success independently
5. The main context doesn't need to track every implementation detail

**Keep in main context when**:
1. Making architectural decisions
2. Integrating multiple components
3. Debugging cross-component issues
4. Reviewing and testing final integration

---

## Subagent Task Categories

### Category A: Infrastructure & Configuration
**Complexity**: Low  
**Code Volume**: Medium  
**Isolation**: High  

✅ **Delegate to subagent**

**Tasks**:
- Creating Kubernetes manifests (YAML files)
- Writing shell scripts for automation
- Generating configuration files
- Setting up CI/CD pipelines

**Example Delegation**:
```
Task: Create complete Kubernetes manifests for optimizer backend
Files: deployments/optimizer/backend-deployment.yaml
Requirements:
- Deployment with 2 replicas
- Service with ClusterIP and NodePort 30081
- ConfigMap for configuration
- Use service account optimizer-sa
- Resource limits: 500m CPU, 512Mi memory
- Health checks on /health and /ready endpoints
```

---

### Category B: Backend Core Components
**Complexity**: High  
**Code Volume**: High (300-800 lines per component)  
**Isolation**: High  

✅ **Delegate to subagent** (one subagent per component)

**Tasks**:

#### Task B1: Metrics Collector Service
**Delegate to**: Subagent 1  
**Files**: `backend/pkg/collector/`  
**Deliverables**:
- `collector.go` - Main collector interface
- `k8s_collector.go` - Kubernetes metrics collection
- `metrics_store.go` - In-memory time-series storage
- `types.go` - Data structures

**Requirements**:
```go
// Required interfaces
type MetricsCollector interface {
    CollectPodMetrics(namespace string) ([]PodMetrics, error)
    CollectNodeMetrics() ([]NodeMetrics, error)
    CollectHPAMetrics(namespace string) ([]HPAMetrics, error)
    GetTimeSeriesData(resource, metric string, duration time.Duration) (TimeSeriesData, error)
}

// Must collect every 15 seconds
// Store last 24 hours of data
// Provide percentile calculations (P50, P95, P99)
```

#### Task B2: Resource Optimizer Engine
**Delegate to**: Subagent 2  
**Files**: `backend/pkg/optimizer/`  
**Deliverables**:
- `optimizer.go` - Main optimization engine
- `resource_analyzer.go` - Resource usage analysis
- `recommendations.go` - Recommendation generation
- `scorer.go` - Efficiency scoring

**Requirements**:
```go
type Optimizer interface {
    AnalyzeDeployment(namespace, name string) (*Analysis, error)
    GenerateRecommendations(analysis *Analysis) ([]Recommendation, error)
    CalculateEfficiencyScore(deployment *appsv1.Deployment) (float64, error)
    EstimateCostSavings(recommendation *Recommendation) (float64, error)
}

// Algorithms needed:
// - P95 resource usage analysis
// - Right-sizing recommendations
// - HPA optimization
// - Cost calculation based on resource usage
```

#### Task B3: Traffic & Cost Analyzer
**Delegate to**: Subagent 3  
**Files**: `backend/pkg/analyzer/`  
**Deliverables**:
- `traffic_analyzer.go` - Traffic pattern analysis
- `cost_analyzer.go` - Cost calculation
- `anomaly_detector.go` - Anomaly detection
- `trends.go` - Trend analysis

**Requirements**:
```go
type Analyzer interface {
    AnalyzeTrafficPatterns(service string, duration time.Duration) (*TrafficAnalysis, error)
    CalculateServiceCost(namespace, service string) (*CostBreakdown, error)
    DetectAnomalies(metrics []Metric) ([]Anomaly, error)
    PredictResourceNeeds(service string, hours int) (*ResourcePrediction, error)
}
```

#### Task B4: API Server & WebSocket
**Delegate to**: Subagent 4  
**Files**: `backend/pkg/api/` and `backend/cmd/server/main.go`  
**Deliverables**:
- `handlers.go` - REST API handlers
- `websocket.go` - WebSocket real-time updates
- `middleware.go` - Auth, logging, CORS
- `router.go` - Route configuration
- `main.go` - Server entry point

**API Endpoints Required**:
```
GET  /api/v1/cluster/overview
GET  /api/v1/services
GET  /api/v1/services/:namespace/:name
GET  /api/v1/recommendations
POST /api/v1/recommendations/:id/apply
POST /api/v1/recommendations/:id/rollback
GET  /api/v1/metrics/:resource
WS   /ws/updates
```

---

### Category C: Frontend Components
**Complexity**: Medium-High  
**Code Volume**: High (200-400 lines per component)  
**Isolation**: High  

✅ **Delegate to subagent** (one subagent per major feature)

#### Task C1: Dashboard Foundation
**Delegate to**: Subagent 5  
**Files**: `dashboard/src/`  
**Deliverables**:
- Project initialization with Vite + React + TypeScript
- Tailwind CSS + shadcn/ui setup
- API client service with TypeScript types
- WebSocket hook for real-time updates
- Routing setup
- Base layout component

**Requirements**:
```typescript
// API Client
class OptimizerAPI {
  getClusterOverview(): Promise<ClusterOverview>
  getServices(): Promise<Service[]>
  getServiceDetails(namespace: string, name: string): Promise<ServiceDetail>
  getRecommendations(): Promise<Recommendation[]>
  applyRecommendation(id: string): Promise<void>
}

// WebSocket Hook
function useRealtimeUpdates(callback: (update: Update) => void): void
```

#### Task C2: Cluster Overview Component
**Delegate to**: Subagent 6  
**Files**: `dashboard/src/components/ClusterOverview/`  
**Deliverables**:
- Node status cards with resource usage
- Pod distribution visualization
- Real-time metrics charts
- Health status indicators

#### Task C3: Service Analyzer Component
**Delegate to**: Subagent 7  
**Files**: `dashboard/src/components/ServiceAnalyzer/`  
**Deliverables**:
- Service list with health scores
- Detailed service view
- Resource usage charts (CPU, memory over time)
- Pod instance viewer
- Traffic metrics display

#### Task C4: Optimization Panel
**Delegate to**: Subagent 8  
**Files**: `dashboard/src/components/OptimizationPanel/`  
**Deliverables**:
- Recommendations list
- Recommendation details with diff view
- Apply/Rollback buttons
- Impact preview
- Cost savings calculator

---

## Delegation Template

When delegating to a subagent, use this structure:

```markdown
## Task: [Component Name]

### Context
Brief explanation of what this component does and how it fits into the system.

### Files to Create/Modify
- path/to/file1.ext
- path/to/file2.ext

### Requirements

#### Functional Requirements
1. Must do X
2. Should handle Y
3. Must integrate with Z

#### Technical Requirements
- Language/Framework: [Go/TypeScript/etc]
- Dependencies: [list libraries]
- Interfaces to implement: [code snippet]
- Error handling: [requirements]
- Testing: [unit test requirements]

#### Data Structures
```language
// Provide key types/interfaces
```

#### Integration Points
- This component will be called by: [X]
- This component will call: [Y]
- Configuration needed: [Z]

### Success Criteria
- [ ] All interfaces implemented
- [ ] Error handling complete
- [ ] Basic unit tests pass
- [ ] Integration test scenario: [describe]

### Example Usage
```language
// Show how the component will be used
```

### Non-Requirements (Out of Scope)
- Don't implement X
- Don't worry about Y
```

---

## Main Context Responsibilities

While subagents handle implementation, you maintain in main context:

### 1. Architecture Review
After each subagent completes:
- Review interfaces and public APIs
- Verify integration points match
- Check for consistency across components

### 2. Integration Code
Write the "glue code" that connects components:
```go
// Example: Main server setup connecting all components
func main() {
    collector := collector.New(k8sClient)
    optimizer := optimizer.New(collector)
    analyzer := analyzer.New(collector)
    
    api := api.New(collector, optimizer, analyzer)
    api.Start(":8080")
}
```

### 3. End-to-End Testing
Create integration tests that verify the full flow:
```bash
# Test scenario
1. Start backend server
2. Deploy test workload
3. Wait for metrics collection
4. Request recommendations
5. Apply recommendation
6. Verify deployment updated
```

### 4. Documentation
Maintain high-level documentation:
- Architecture decisions
- Component interactions
- Deployment procedures
- Troubleshooting guides

---

## Subagent Handoff Pattern

### Before Delegation
```markdown
I'm about to delegate Task [X] to a subagent. 

Current state:
- [Component A] is complete
- [Component B] needs [X] to integrate
- Interface between them: [show code]

After subagent completes, we'll:
1. Review the implementation
2. Test integration with [Component B]
3. Move to [next task]
```

### After Subagent Returns
```markdown
Subagent completed Task [X]. 

Deliverables received:
- [file list]

Next steps:
1. Review for interface compatibility
2. Test with: [command]
3. If tests pass, integrate into main build
4. Proceed to [next component]
```

---

## Task Priority Order

### Phase 1: Foundation (Week 1)
1. Infrastructure setup (you in main context - it's coordination)
2. **Subagent B1**: Metrics Collector
3. **Subagent B2**: Optimizer Engine
4. Integration testing (you in main context)

### Phase 2: Analysis (Week 2)
5. **Subagent B3**: Analyzer
6. **Subagent B4**: API Server
7. End-to-end backend testing (you in main context)

### Phase 3: Frontend (Week 2-3)
8. **Subagent C1**: Dashboard Foundation
9. **Subagent C2**: Cluster Overview
10. **Subagent C3**: Service Analyzer
11. **Subagent C4**: Optimization Panel
12. Full integration testing (you in main context)

---

## Context Size Management

### Keep Main Context Under 30K Tokens
- Subagent results: Keep interfaces/types only, not full implementation
- Test results: Keep summary, not full output
- Documentation: Keep table of contents, link to details

### When Main Context Gets Full
Create a summary document:
```markdown
## Project State Snapshot - [Date]

### Completed Components
- [X]: Status, Key interfaces, Test status
- [Y]: Status, Key interfaces, Test status

### Active Work
- Current task: [Z]
- Blocked by: [if any]
- Next: [upcoming task]

### Integration Status
- [Component A] ↔ [Component B]: ✅ Tested
- [Component C] ↔ [Component D]: 🚧 In Progress

### Open Issues
1. [Issue description] - Priority: [High/Med/Low]
```

---

## Decision: When to Bring Back Context

Bring a subagent's implementation back into main context when:
1. Integration errors occur that need debugging
2. Interface changes are needed that affect multiple components
3. Performance optimization requires cross-component analysis
4. Final code review before release

Otherwise, keep implementation in subagent context and only track:
- Public interfaces
- Integration points  
- Test results
- Known issues

---

## Quick Reference

| Task Type | Delegate? | Reason |
|-----------|-----------|--------|
| Architecture design | ❌ Main | Needs full context |
| Single component implementation | ✅ Subagent | Self-contained |
| Integration testing | ❌ Main | Needs multiple components |
| Bug in single component | ✅ Subagent | Isolated issue |
| Cross-component bug | ❌ Main | Needs full context |
| Documentation | ✅ Subagent | Can be done independently |
| Final review | ❌ Main | Needs oversight |

---

## Estimated Subagent Count

**Total subagents needed**: 8-10
- Backend: 4 subagents
- Frontend: 4 subagents  
- Documentation: 1-2 subagents (optional)

**Main context usage**: ~40% coordination, 60% integration & testing
