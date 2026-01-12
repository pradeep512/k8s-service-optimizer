import { useEffect, useState } from 'react';
import { api } from '../services/api';
import type { ClusterOverview } from '../services/types';
import ResourceChart from '../components/Charts/ResourceChart';
import HPAChart from '../components/Charts/HPAChart';

interface MetricsHistory {
  timestamps: string[];
  cpuData: number[];
  memoryData: number[];
  hpaHistory: Map<string, Array<{ timestamp: string; replicas: number; cpu: number; desired: number }>>;
}

const MAX_DATA_POINTS = 60; // Keep last 60 data points (5 minutes at 5s intervals)

export default function Dashboard() {
  const [overview, setOverview] = useState<ClusterOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [metricsHistory, setMetricsHistory] = useState<MetricsHistory>({
    timestamps: [],
    cpuData: [],
    memoryData: [],
    hpaHistory: new Map()
  });

  // Fetch cluster overview every 10 seconds
  useEffect(() => {
    const fetchOverview = async () => {
      try {
        setLoading(true);
        const data = await api.getClusterOverview();
        setOverview(data);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch cluster overview');
        console.error('Error fetching cluster overview:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchOverview();
    const interval = setInterval(fetchOverview, 10000); // 10s refresh
    return () => clearInterval(interval);
  }, []);

  // Fetch metrics and build time-series history every 5 seconds
  useEffect(() => {
    const fetchMetrics = async () => {
      try {
        const [hpa, nodes] = await Promise.all([
          api.getHPAMetrics(),
          api.getNodeMetrics()
        ]);

        const now = new Date().toLocaleTimeString();

        // Calculate total cluster CPU and Memory
        const totalCPU = nodes.reduce((sum, n) => sum + n.cpuUsage, 0) / 1000; // to cores
        const totalMemory = nodes.reduce((sum, n) => sum + n.memoryUsage, 0) / 1024 / 1024 / 1024; // to GB

        setMetricsHistory(prev => {
          // Add new data point to time-series
          const newTimestamps = [...prev.timestamps, now].slice(-MAX_DATA_POINTS);
          const newCpuData = [...prev.cpuData, totalCPU].slice(-MAX_DATA_POINTS);
          const newMemoryData = [...prev.memoryData, totalMemory].slice(-MAX_DATA_POINTS);

          // Update HPA history for each HPA
          const newHpaHistory = new Map(prev.hpaHistory);
          hpa.forEach(h => {
            const history = newHpaHistory.get(h.name) || [];
            history.push({
              timestamp: now,
              replicas: h.currentReplicas,
              cpu: h.currentCPU,
              desired: h.desiredReplicas
            });
            newHpaHistory.set(h.name, history.slice(-MAX_DATA_POINTS));
          });

          return {
            timestamps: newTimestamps,
            cpuData: newCpuData,
            memoryData: newMemoryData,
            hpaHistory: newHpaHistory
          };
        });
      } catch (err) {
        console.error('Error fetching metrics:', err);
      }
    };

    fetchMetrics();
    const interval = setInterval(fetchMetrics, 5000); // 5s refresh for near real-time
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-lg text-gray-600">Loading cluster overview...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="rounded-lg bg-red-50 p-6 text-red-800">
          <h3 className="font-semibold">Error</h3>
          <p className="mt-2">{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>

      {overview && (
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
          {/* Node Count */}
          <div className="rounded-lg bg-white p-6 shadow">
            <div className="text-sm font-medium text-gray-600">Nodes</div>
            <div className="mt-2 text-3xl font-bold text-gray-900">
              {overview.nodeCount}
            </div>
          </div>

          {/* Pod Count */}
          <div className="rounded-lg bg-white p-6 shadow">
            <div className="text-sm font-medium text-gray-600">Pods</div>
            <div className="mt-2 text-3xl font-bold text-gray-900">
              {overview.podCount}
            </div>
          </div>

          {/* CPU Usage */}
          <div className="rounded-lg bg-white p-6 shadow">
            <div className="text-sm font-medium text-gray-600">CPU Usage</div>
            <div className="mt-2 text-3xl font-bold text-gray-900">
              {((overview.usedCPU / overview.totalCPU) * 100).toFixed(1)}%
            </div>
            <div className="mt-1 text-xs text-gray-500">
              {overview.usedCPU.toFixed(2)} / {overview.totalCPU.toFixed(2)} cores
            </div>
          </div>

          {/* Memory Usage */}
          <div className="rounded-lg bg-white p-6 shadow">
            <div className="text-sm font-medium text-gray-600">Memory Usage</div>
            <div className="mt-2 text-3xl font-bold text-gray-900">
              {((overview.usedMemory / overview.totalMemory) * 100).toFixed(1)}%
            </div>
            <div className="mt-1 text-xs text-gray-500">
              {(overview.usedMemory / 1024 / 1024 / 1024).toFixed(2)} /{' '}
              {(overview.totalMemory / 1024 / 1024 / 1024).toFixed(2)} GB
            </div>
          </div>

          {/* Health Score */}
          <div className="rounded-lg bg-white p-6 shadow sm:col-span-2 lg:col-span-4">
            <div className="text-sm font-medium text-gray-600">Cluster Health Score</div>
            <div className="mt-2 text-3xl font-bold text-gray-900">
              {overview.healthScore.toFixed(1)}
            </div>
            <div className="mt-2 h-2 w-full rounded-full bg-gray-200">
              <div
                className={`h-2 rounded-full ${
                  overview.healthScore >= 80
                    ? 'bg-green-500'
                    : overview.healthScore >= 60
                    ? 'bg-yellow-500'
                    : 'bg-red-500'
                }`}
                style={{ width: `${overview.healthScore}%` }}
              />
            </div>
          </div>
        </div>
      )}

      {/* Resource Charts */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <ResourceChart
          title="Cluster Resource Usage (5min window)"
          data={metricsHistory.timestamps.map((time, i) => ({
            time,
            cpu: metricsHistory.cpuData[i],
            memory: metricsHistory.memoryData[i]
          }))}
          unit="Cores / GB"
        />
        <HPAChart hpaHistory={metricsHistory.hpaHistory} />
      </div>
    </div>
  );
}
