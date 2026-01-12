import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';

interface HPAChartProps {
  hpaHistory: Map<string, Array<{ timestamp: string; replicas: number; cpu: number; desired: number }>>;
}

export default function HPAChart({ hpaHistory }: HPAChartProps) {
  // Get the first HPA to display (most HPAs in the namespace)
  const hpaNames = Array.from(hpaHistory.keys());
  const hpaName = hpaNames[0];
  const data = hpaHistory.get(hpaName) || [];

  if (data.length === 0) {
    return (
      <div className="rounded-lg bg-white p-6 shadow">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">HPA Scaling Trend</h3>
        <div className="flex items-center justify-center h-[300px] text-gray-500">
          No HPA data available
        </div>
      </div>
    );
  }

  return (
    <div className="rounded-lg bg-white p-6 shadow">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">
        HPA Scaling Trend: {hpaName} (5min window)
      </h3>
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis
            dataKey="timestamp"
            tick={{ fontSize: 11 }}
            interval="preserveStartEnd"
          />
          <YAxis
            yAxisId="left"
            label={{ value: 'Replicas', angle: -90, position: 'insideLeft' }}
          />
          <YAxis
            yAxisId="right"
            orientation="right"
            label={{ value: 'CPU %', angle: 90, position: 'insideRight' }}
          />
          <Tooltip />
          <Legend />
          <Line
            yAxisId="left"
            type="monotone"
            dataKey="replicas"
            stroke="#10b981"
            name="Current Replicas"
            strokeWidth={2}
            dot={{ r: 2 }}
          />
          <Line
            yAxisId="left"
            type="monotone"
            dataKey="desired"
            stroke="#f59e0b"
            name="Desired Replicas"
            strokeWidth={2}
            strokeDasharray="5 5"
            dot={{ r: 2 }}
          />
          <Line
            yAxisId="right"
            type="monotone"
            dataKey="cpu"
            stroke="#ef4444"
            name="CPU %"
            strokeWidth={2}
            dot={{ r: 2 }}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
