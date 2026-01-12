import { useEffect, useState } from 'react';
import { api } from '../services/api';
import type { Recommendation } from '../services/types';

export default function Recommendations() {
  const [recommendations, setRecommendations] = useState<Recommendation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [applying, setApplying] = useState<string | null>(null);

  useEffect(() => {
    fetchRecommendations();
  }, []);

  const fetchRecommendations = async () => {
    try {
      setLoading(true);
      const data = await api.getRecommendations();
      setRecommendations(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch recommendations');
      console.error('Error fetching recommendations:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleApply = async (id: string) => {
    try {
      setApplying(id);
      await api.applyRecommendation(id);
      // Refresh recommendations after applying
      await fetchRecommendations();
    } catch (err) {
      console.error('Error applying recommendation:', err);
      alert('Failed to apply recommendation: ' + (err instanceof Error ? err.message : 'Unknown error'));
    } finally {
      setApplying(null);
    }
  };

  const handleDismiss = async (id: string) => {
    try {
      await api.dismissRecommendation(id);
      // Refresh recommendations after dismissing
      await fetchRecommendations();
    } catch (err) {
      console.error('Error dismissing recommendation:', err);
      alert('Failed to dismiss recommendation: ' + (err instanceof Error ? err.message : 'Unknown error'));
    }
  };

  const getPriorityColor = (priority: string) => {
    switch (priority.toLowerCase()) {
      case 'critical':
        return 'bg-red-100 text-red-800';
      case 'high':
        return 'bg-orange-100 text-orange-800';
      case 'medium':
        return 'bg-yellow-100 text-yellow-800';
      case 'low':
        return 'bg-green-100 text-green-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-lg text-gray-600">Loading recommendations...</div>
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
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Recommendations</h1>
        <div className="text-sm text-gray-600">
          {recommendations.length} recommendations
        </div>
      </div>

      {recommendations.length === 0 ? (
        <div className="rounded-lg bg-white p-12 text-center shadow">
          <p className="text-gray-600">No recommendations available</p>
          <p className="mt-2 text-sm text-gray-500">
            Your cluster is optimized!
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {recommendations.map((rec) => (
            <div
              key={rec.id}
              className="rounded-lg bg-white p-6 shadow transition-shadow hover:shadow-md"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center space-x-3">
                    <span
                      className={`inline-flex rounded-full px-3 py-1 text-xs font-semibold ${getPriorityColor(
                        rec.priority
                      )}`}
                    >
                      {rec.priority}
                    </span>
                    <span className="text-sm font-medium text-gray-600">
                      {rec.type}
                    </span>
                  </div>

                  <h3 className="mt-2 text-lg font-semibold text-gray-900">
                    {rec.deployment} ({rec.namespace})
                  </h3>

                  <p className="mt-2 text-sm text-gray-600">
                    {rec.description}
                  </p>

                  <div className="mt-4 flex items-center space-x-6 text-sm">
                    <div>
                      <span className="text-gray-600">Estimated Savings:</span>
                      <span className="ml-2 font-semibold text-green-600">
                        ${rec.estimatedSavings.toFixed(2)}/month
                      </span>
                    </div>
                    <div>
                      <span className="text-gray-600">Impact:</span>
                      <span className="ml-2 font-medium text-gray-900">
                        {rec.impact}
                      </span>
                    </div>
                  </div>

                  <div className="mt-2 text-xs text-gray-500">
                    Created: {new Date(rec.createdAt).toLocaleString()}
                  </div>
                </div>

                <div className="ml-4 flex flex-col space-y-2">
                  <button
                    onClick={() => handleApply(rec.id)}
                    disabled={applying === rec.id}
                    className={`rounded-md px-4 py-2 text-sm font-medium text-white transition-colors ${
                      applying === rec.id
                        ? 'bg-gray-400'
                        : 'bg-primary-600 hover:bg-primary-700'
                    }`}
                  >
                    {applying === rec.id ? 'Applying...' : 'Apply'}
                  </button>
                  <button
                    onClick={() => handleDismiss(rec.id)}
                    disabled={applying === rec.id}
                    className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50"
                  >
                    Dismiss
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="rounded-lg bg-white p-6 shadow">
        <h2 className="text-lg font-semibold text-gray-900">
          Recommendation Details
        </h2>
        <p className="mt-2 text-sm text-gray-600">
          Detailed analysis and impact assessment will be displayed here (Component C4)
        </p>
      </div>
    </div>
  );
}
