import { useEffect, useState } from 'react';
import { api } from '../services/api';
import type { Service, ServiceDetail as ServiceDetailType, Recommendation } from '../services/types';

interface ServiceDetailProps {
  service: Service;
  onClose: () => void;
}

export default function ServiceDetail({ service, onClose }: ServiceDetailProps) {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [details, setDetails] = useState<ServiceDetailType | null>(null);
  const [recommendations, setRecommendations] = useState<Recommendation[]>([]);

  useEffect(() => {
    const fetchDetails = async () => {
      try {
        setLoading(true);
        const [detailData, recsData] = await Promise.all([
          api.getDeploymentDetail(service.namespace, service.name), // Changed to use deployment detail endpoint
          api.getRecommendationsByService(service.namespace, service.name)
        ]);
        setDetails(detailData);
        setRecommendations(recsData);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch service details');
        console.error('Error fetching service details:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchDetails();
  }, [service.namespace, service.name]);

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      <div className="flex min-h-screen items-center justify-center px-4">
        {/* Backdrop */}
        <div
          className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"
          onClick={onClose}
        />

        {/* Modal */}
        <div className="relative z-50 w-full max-w-4xl rounded-lg bg-white shadow-xl">
          {/* Header */}
          <div className="border-b border-gray-200 px-6 py-4">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-xl font-semibold text-gray-900">
                  {service.name}
                </h2>
                <p className="text-sm text-gray-500">
                  Namespace: {service.namespace}
                </p>
              </div>
              <button
                onClick={onClose}
                className="rounded-md text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                <span className="sr-only">Close</span>
                <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>

          {/* Content */}
          <div className="px-6 py-4">
            {loading ? (
              <div className="flex items-center justify-center py-12">
                <div className="text-gray-600">Loading service details...</div>
              </div>
            ) : error ? (
              <div className="rounded-lg bg-red-50 p-4 text-red-800">
                <h3 className="font-semibold">Error</h3>
                <p className="mt-1 text-sm">{error}</p>
              </div>
            ) : details ? (
              <div className="space-y-6">
                {/* Overview Stats */}
                <div className="grid grid-cols-4 gap-4">
                  <div className="rounded-lg bg-gray-50 p-4">
                    <div className="text-xs text-gray-600">Replicas</div>
                    <div className="mt-1 text-2xl font-semibold text-gray-900">
                      {details.replicas}
                    </div>
                  </div>
                  <div className="rounded-lg bg-gray-50 p-4">
                    <div className="text-xs text-gray-600">Health Score</div>
                    <div className="mt-1 text-2xl font-semibold text-gray-900">
                      {details.healthScore.toFixed(0)}
                    </div>
                  </div>
                  <div className="rounded-lg bg-gray-50 p-4">
                    <div className="text-xs text-gray-600">CPU Usage</div>
                    <div className="mt-1 text-2xl font-semibold text-gray-900">
                      {details.cpuUsage.toFixed(2)}
                    </div>
                  </div>
                  <div className="rounded-lg bg-gray-50 p-4">
                    <div className="text-xs text-gray-600">Memory Usage</div>
                    <div className="mt-1 text-2xl font-semibold text-gray-900">
                      {(details.memoryUsage / 1024 / 1024).toFixed(0)} MB
                    </div>
                  </div>
                </div>

                {/* Pods */}
                {details.pods && details.pods.length > 0 && (
                  <div>
                    <h3 className="mb-3 text-lg font-semibold text-gray-900">Pods</h3>
                    <div className="overflow-hidden rounded-lg border border-gray-200">
                      <table className="min-w-full divide-y divide-gray-200">
                        <thead className="bg-gray-50">
                          <tr>
                            <th className="px-4 py-2 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                              Name
                            </th>
                            <th className="px-4 py-2 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                              Status
                            </th>
                            <th className="px-4 py-2 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                              CPU
                            </th>
                            <th className="px-4 py-2 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                              Memory
                            </th>
                            <th className="px-4 py-2 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                              Restarts
                            </th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-gray-200 bg-white">
                          {details.pods.map((pod) => (
                            <tr key={pod.name}>
                              <td className="whitespace-nowrap px-4 py-2 text-sm text-gray-900">
                                {pod.name}
                              </td>
                              <td className="whitespace-nowrap px-4 py-2 text-sm">
                                <span className={`inline-flex rounded-full px-2 py-1 text-xs font-semibold ${
                                  pod.status === 'Running' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                                }`}>
                                  {pod.status}
                                </span>
                              </td>
                              <td className="whitespace-nowrap px-4 py-2 text-sm text-gray-600">
                                {pod.cpuUsage.toFixed(2)}
                              </td>
                              <td className="whitespace-nowrap px-4 py-2 text-sm text-gray-600">
                                {(pod.memoryUsage / 1024 / 1024).toFixed(0)} MB
                              </td>
                              <td className="whitespace-nowrap px-4 py-2 text-sm text-gray-600">
                                {pod.restartCount}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>
                )}

                {/* Recommendations */}
                {recommendations.length > 0 && (
                  <div>
                    <h3 className="mb-3 text-lg font-semibold text-gray-900">Recommendations</h3>
                    <div className="space-y-2">
                      {recommendations.map((rec) => (
                        <div
                          key={rec.id}
                          className="rounded-lg border border-gray-200 p-4"
                        >
                          <div className="flex items-start justify-between">
                            <div className="flex-1">
                              <div className="flex items-center space-x-2">
                                <span className={`inline-flex rounded-full px-2 py-1 text-xs font-semibold ${
                                  rec.priority === 'high'
                                    ? 'bg-red-100 text-red-800'
                                    : rec.priority === 'medium'
                                    ? 'bg-yellow-100 text-yellow-800'
                                    : 'bg-blue-100 text-blue-800'
                                }`}>
                                  {rec.priority}
                                </span>
                                <span className="text-sm font-medium text-gray-900">
                                  {rec.type}
                                </span>
                              </div>
                              <p className="mt-2 text-sm text-gray-600">{rec.description}</p>
                              {rec.estimatedSavings > 0 && (
                                <p className="mt-1 text-xs text-green-600">
                                  Estimated savings: ${rec.estimatedSavings.toFixed(2)}/month
                                </p>
                              )}
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* Metrics Summary */}
                {details.metrics && (
                  <div>
                    <h3 className="mb-3 text-lg font-semibold text-gray-900">Metrics Summary</h3>
                    <div className="grid grid-cols-2 gap-4">
                      <div className="rounded-lg border border-gray-200 p-4">
                        <div className="text-sm font-medium text-gray-900">CPU</div>
                        <div className="mt-2 space-y-1 text-sm text-gray-600">
                          <div>Average: {details.metrics.avgCPU.toFixed(2)}</div>
                          <div>Max: {details.metrics.maxCPU.toFixed(2)}</div>
                          <div>P95: {details.metrics.p95CPU.toFixed(2)}</div>
                        </div>
                      </div>
                      <div className="rounded-lg border border-gray-200 p-4">
                        <div className="text-sm font-medium text-gray-900">Memory</div>
                        <div className="mt-2 space-y-1 text-sm text-gray-600">
                          <div>Average: {(details.metrics.avgMemory / 1024 / 1024).toFixed(0)} MB</div>
                          <div>Max: {(details.metrics.maxMemory / 1024 / 1024).toFixed(0)} MB</div>
                          <div>P95: {(details.metrics.p95Memory / 1024 / 1024).toFixed(0)} MB</div>
                        </div>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <div className="py-12 text-center text-gray-600">
                No details available for this service
              </div>
            )}
          </div>

          {/* Footer */}
          <div className="border-t border-gray-200 px-6 py-4">
            <div className="flex justify-end">
              <button
                onClick={onClose}
                className="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
