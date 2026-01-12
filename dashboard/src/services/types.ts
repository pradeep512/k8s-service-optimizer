// API Response wrapper
export interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
    details?: any;
  };
}

// Cluster Overview
export interface ClusterOverview {
  nodeCount: number;
  podCount: number;
  totalCPU: number;
  totalMemory: number;
  usedCPU: number;
  usedMemory: number;
  healthScore: number;
}

// Service
export interface Service {
  name: string;
  namespace: string;
  replicas: number;
  healthScore: number;
  cpuUsage: number;
  memoryUsage: number;
  status: string;
}

// Service Detail
export interface ServiceDetail {
  name: string;
  namespace: string;
  replicas: number;
  healthScore: number;
  cpuUsage: number;
  memoryUsage: number;
  status: string;
  pods: Pod[];
  metrics: ServiceMetrics;
  recommendations: Recommendation[];
}

// Pod
export interface Pod {
  name: string;
  status: string;
  cpuUsage: number;
  memoryUsage: number;
  restartCount: number;
  age: string;
}

// Service Metrics
export interface ServiceMetrics {
  avgCPU: number;
  avgMemory: number;
  maxCPU: number;
  maxMemory: number;
  p95CPU: number;
  p95Memory: number;
}

// Recommendation
export interface Recommendation {
  id: string;
  type: string;
  namespace: string;
  deployment: string;
  priority: string;
  description: string;
  estimatedSavings: number;
  impact: string;
  createdAt: string;
  status: string;
  details?: RecommendationDetails;
}

// Recommendation Details
export interface RecommendationDetails {
  currentReplicas?: number;
  recommendedReplicas?: number;
  currentCPU?: string;
  recommendedCPU?: string;
  currentMemory?: string;
  recommendedMemory?: string;
  reason?: string;
}

// Node Metrics
export interface NodeMetrics {
  name: string;
  cpuUsage: number;
  memoryUsage: number;
  cpuCapacity: number;
  memoryCapacity: number;
  podCount: number;
  status: string;
  timestamp: string;
}

// Pod Metrics
export interface PodMetrics {
  name: string;
  namespace: string;
  cpuUsage: number;
  memoryUsage: number;
  cpuRequest: number;
  memoryRequest: number;
  cpuLimit: number;
  memoryLimit: number;
}

// Analysis
export interface Analysis {
  namespace: string;
  service: string;
  healthScore: number;
  efficiency: number;
  costScore: number;
  recommendations: Recommendation[];
  insights: Insight[];
  trends: Trends;
}

// Insight
export interface Insight {
  type: string;
  severity: string;
  message: string;
  impact: string;
}

// Trends
export interface Trends {
  cpuTrend: string;
  memoryTrend: string;
  replicaTrend: string;
  costTrend: string;
}

// Cost Breakdown
export interface CostBreakdown {
  namespace: string;
  service: string;
  totalCost: number;
  cpuCost: number;
  memoryCost: number;
  storageCost: number;
  networkCost: number;
  breakdown: CostItem[];
}

// Cost Item
export interface CostItem {
  category: string;
  amount: number;
  percentage: number;
}

// Comparison
export interface Comparison {
  services: string[];
  metrics: ComparisonMetrics;
}

// Comparison Metrics
export interface ComparisonMetrics {
  cpu: number[];
  memory: number[];
  replicas: number[];
  healthScore: number[];
  cost: number[];
}

// Health Check
export interface HealthCheck {
  status: string;
  components: ComponentHealth[];
  timestamp: string;
}

// Component Health
export interface ComponentHealth {
  name: string;
  status: string;
  message?: string;
}

// Metrics Summary
export interface MetricsSummary {
  nodeMetrics: NodeMetrics[];
  podMetrics: PodMetrics[];
  timestamp: string;
}

// WebSocket Message
export interface WebSocketMessage {
  type: string;
  timestamp: string;
  data: any;
}

// HPA Metrics
export interface HPAMetrics {
  name: string;
  namespace: string;
  currentReplicas: number;
  desiredReplicas: number;
  minReplicas: number;
  maxReplicas: number;
  targetCPU: number;
  currentCPU: number;
  timestamp: string;
}

// Time Series Point
export interface TimeSeriesPoint {
  timestamp: Date;
  value: number;
}

// Resource Chart Data
export interface ResourceChartData {
  time: string;
  cpu: number;
  memory: number;
}
