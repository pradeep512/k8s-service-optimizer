import type {
  APIResponse,
  ClusterOverview,
  Service,
  ServiceDetail,
  Recommendation,
  NodeMetrics,
  PodMetrics,
  HPAMetrics,
  Analysis,
  CostBreakdown,
  Comparison,
  HealthCheck,
  MetricsSummary,
} from './types';

export class OptimizerAPI {
  private baseURL: string;
  private rootURL: string;

  constructor(baseURL: string = getDefaultBaseURL()) {
    this.baseURL = baseURL;
    this.rootURL = baseURL.replace(/\/api\/v1\/?$/, '');
  }

  private async fetchJSON<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const headers: Record<string, string> = {
      ...((options?.headers as Record<string, string>) || {}),
    };

    // Avoid forcing a JSON content-type on GET requests to skip preflight.
    if (options?.body && !headers['Content-Type']) {
      headers['Content-Type'] = 'application/json';
    }

    const response = await fetch(`${this.baseURL}${endpoint}`, {
      ...options,
      headers,
    });

    const data: APIResponse<T> = await response.json();

    if (!data.success) {
      throw new Error(data.error?.message || 'API request failed');
    }

    return data.data as T;
  }

  private async fetchRootJSON<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const headers: Record<string, string> = {
      ...((options?.headers as Record<string, string>) || {}),
    };

    if (options?.body && !headers['Content-Type']) {
      headers['Content-Type'] = 'application/json';
    }

    const response = await fetch(`${this.rootURL}${endpoint}`, {
      ...options,
      headers,
    });

    const data: APIResponse<T> = await response.json();

    if (!data.success) {
      throw new Error(data.error?.message || 'API request failed');
    }

    return data.data as T;
  }

  // Cluster endpoints
  async getClusterOverview(): Promise<ClusterOverview> {
    const data = await this.fetchJSON<BackendClusterOverview>('/cluster/overview');
    return mapClusterOverview(data);
  }

  async getHealth(): Promise<HealthCheck> {
    return this.fetchRootJSON<HealthCheck>('/health');
  }

  // Service endpoints
  async getServices(): Promise<Service[]> {
    const data = await this.fetchJSON<BackendServiceSummary[]>('/services');
    return data.map(mapServiceSummary);
  }

  async getServiceDetail(namespace: string, name: string): Promise<ServiceDetail> {
    return this.fetchJSON<ServiceDetail>(`/services/${namespace}/${name}`);
  }

  // Deployment endpoints (with actual metrics)
  async getDeployments(): Promise<Service[]> {
    const data = await this.fetchJSON<BackendDeployment[]>('/deployments');
    return data.map(mapDeployment);
  }

  async getDeploymentDetail(namespace: string, name: string): Promise<ServiceDetail> {
    return this.fetchJSON<ServiceDetail>(`/deployments/${namespace}/${name}`);
  }

  // Recommendation endpoints
  async getRecommendations(): Promise<Recommendation[]> {
    const data = await this.fetchJSON<BackendRecommendation[]>('/recommendations');
    return data.map(mapRecommendation);
  }

  async getRecommendationsByService(namespace: string, service: string): Promise<Recommendation[]> {
    const recommendations = await this.getRecommendations();
    return recommendations.filter(
      (rec) => rec.namespace === namespace && rec.deployment === service
    );
  }

  async applyRecommendation(id: string): Promise<void> {
    await this.fetchJSON<void>(`/recommendations/${id}/apply`, {
      method: 'POST',
    });
  }

  async dismissRecommendation(id: string): Promise<void> {
    throw new Error(`Dismiss not supported by backend for recommendation ${id}`);
  }

  // Metrics endpoints
  async getNodeMetrics(): Promise<NodeMetrics[]> {
    const data = await this.fetchJSON<BackendNodeMetrics[]>('/metrics/nodes');
    return data.map(mapNodeMetrics);
  }

  async getPodMetrics(namespace?: string): Promise<PodMetrics[]> {
    const targetNamespace = namespace || 'k8s-optimizer';
    const data = await this.fetchJSON<BackendPodMetrics[]>(`/metrics/pods/${targetNamespace}`);
    return data.map(mapPodMetrics);
  }

  async getHPAMetrics(namespace?: string): Promise<HPAMetrics[]> {
    const targetNamespace = namespace || 'k8s-optimizer';
    const data = await this.fetchJSON<BackendHPAMetrics[]>(`/hpa/${targetNamespace}`);
    return data.map(mapHPAMetrics);
  }

  async getMetricsSummary(): Promise<MetricsSummary> {
    throw new Error('Metrics summary endpoint is not available on the backend');
  }

  // Analysis endpoints
  async getAnalysis(namespace: string, service: string): Promise<Analysis> {
    return this.fetchJSON<Analysis>(`/analysis/${namespace}/${service}`);
  }

  // Cost endpoints
  async getCostBreakdown(namespace: string, service: string): Promise<CostBreakdown> {
    return this.fetchJSON<CostBreakdown>(`/cost/${namespace}/${service}`);
  }

  async getTotalCost(): Promise<number> {
    const data = await this.fetchJSON<{ totalCost: number }>('/cost/total');
    return data.totalCost;
  }

  // Comparison endpoint
  async compareServices(services: string[]): Promise<Comparison> {
    throw new Error(`Compare endpoint is not available for services: ${services.join(', ')}`);
  }

  // Optimization endpoint
  async optimizeService(namespace: string, service: string): Promise<Recommendation[]> {
    throw new Error(`Optimize endpoint is not available for ${namespace}/${service}`);
  }
}

// Backend response shapes (title-cased JSON field names from Go).
interface BackendClusterOverview {
  TotalNodes: number;
  HealthyNodes: number;
  TotalPods: number;
  HealthyPods: number;
  CPUCapacity: number;
  CPUUsage: number;
  MemoryCapacity: number;
  MemoryUsage: number;
}

interface BackendServiceSummary {
  name: string;
  namespace: string;
  type: string;
  clusterIP: string;
  ports: number;
  age: string;
}

interface BackendRecommendation {
  ID: string;
  Type: string;
  Namespace: string;
  Deployment: string;
  Priority: string;
  Description: string;
  EstimatedSavings: number;
  Impact: string;
  CreatedAt: string;
}

interface BackendNodeMetrics {
  Name: string;
  CPU: number;
  Memory: number;
  Timestamp: string;
}

interface BackendPodMetrics {
  Name: string;
  Namespace: string;
  CPU: number;
  Memory: number;
  Timestamp: string;
}

interface BackendHPAMetrics {
  Name: string;
  Namespace: string;
  CurrentReplicas: number;
  DesiredReplicas: number;
  MinReplicas: number;
  MaxReplicas: number;
  TargetCPU: number;
  CurrentCPU: number;
  Timestamp: string;
}

interface BackendDeployment {
  name: string;
  namespace: string;
  replicas: number;
  healthScore: number;
  cpuUsage: number;
  memoryUsage: number;
  status: string;
}

function getDefaultBaseURL(): string {
  if (typeof window === 'undefined') {
    return 'http://localhost:8080/api/v1';
  }

  const envBase = (import.meta as any).env?.VITE_API_BASE_URL;
  if (envBase) {
    return envBase;
  }

  // In production (built), use relative URLs so nginx can proxy
  // In development (vite dev server), use absolute URL with port 8080
  const isDev = (import.meta as any).env?.DEV;
  if (isDev) {
    // Development mode with Vite dev server - use Vite proxy
    return '/api/v1';
  } else {
    // Production mode - use relative URL for nginx proxy
    return '/api/v1';
  }
}

function mapClusterOverview(data: BackendClusterOverview): ClusterOverview {
  const nodeHealth = data.TotalNodes > 0 ? (data.HealthyNodes / data.TotalNodes) * 100 : 0;
  const podHealth = data.TotalPods > 0 ? (data.HealthyPods / data.TotalPods) * 100 : 0;

  return {
    nodeCount: data.TotalNodes,
    podCount: data.TotalPods,
    totalCPU: data.CPUCapacity / 1000,
    usedCPU: data.CPUUsage / 1000,
    totalMemory: data.MemoryCapacity,
    usedMemory: data.MemoryUsage,
    healthScore: (nodeHealth + podHealth) / 2,
  };
}

function mapServiceSummary(service: BackendServiceSummary): Service {
  return {
    name: service.name,
    namespace: service.namespace,
    replicas: 0,
    healthScore: 0,
    cpuUsage: 0,
    memoryUsage: 0,
    status: 'unknown',
  };
}

function mapRecommendation(rec: BackendRecommendation): Recommendation {
  return {
    id: rec.ID,
    type: rec.Type,
    namespace: rec.Namespace,
    deployment: rec.Deployment,
    priority: rec.Priority,
    description: rec.Description,
    estimatedSavings: rec.EstimatedSavings,
    impact: rec.Impact,
    createdAt: rec.CreatedAt,
    status: 'new',
  };
}

function mapNodeMetrics(node: BackendNodeMetrics): NodeMetrics {
  return {
    name: node.Name,
    cpuUsage: node.CPU,
    memoryUsage: node.Memory,
    cpuCapacity: 0, // Backend doesn't provide this in simple metrics
    memoryCapacity: 0, // Backend doesn't provide this in simple metrics
    podCount: 0, // Backend doesn't provide this in simple metrics
    status: 'Running',
    timestamp: node.Timestamp,
  };
}

function mapPodMetrics(pod: BackendPodMetrics): PodMetrics {
  return {
    name: pod.Name,
    namespace: pod.Namespace,
    cpuUsage: pod.CPU,
    memoryUsage: pod.Memory,
    cpuRequest: 0, // Backend doesn't provide this in simple metrics
    memoryRequest: 0, // Backend doesn't provide this in simple metrics
    cpuLimit: 0, // Backend doesn't provide this in simple metrics
    memoryLimit: 0, // Backend doesn't provide this in simple metrics
  };
}

function mapHPAMetrics(hpa: BackendHPAMetrics): HPAMetrics {
  return {
    name: hpa.Name,
    namespace: hpa.Namespace,
    currentReplicas: hpa.CurrentReplicas,
    desiredReplicas: hpa.DesiredReplicas,
    minReplicas: hpa.MinReplicas,
    maxReplicas: hpa.MaxReplicas,
    targetCPU: hpa.TargetCPU,
    currentCPU: hpa.CurrentCPU,
    timestamp: hpa.Timestamp,
  };
}

function mapDeployment(deploy: BackendDeployment): Service {
  return {
    name: deploy.name,
    namespace: deploy.namespace,
    replicas: deploy.replicas,
    healthScore: deploy.healthScore,
    cpuUsage: deploy.cpuUsage,
    memoryUsage: deploy.memoryUsage,
    status: deploy.status,
  };
}

// Export singleton instance
export const api = new OptimizerAPI();
