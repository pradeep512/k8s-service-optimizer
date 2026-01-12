import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';

interface ResourceChartProps {
  title: string;
  data: Array<{ time: string; cpu: number; memory: number }>;
  unit: string;
}

export default function ResourceChart({ title, data, unit }: ResourceChartProps) {
  return (
    <div className="rounded-lg bg-white p-6 shadow">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">{title}</h3>
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="time" />
          <YAxis label={{ value: unit, angle: -90, position: 'insideLeft' }} />
          <Tooltip />
          <Legend />
          <Line type="monotone" dataKey="cpu" stroke="#8b5cf6" name="CPU" strokeWidth={2} />
          <Line type="monotone" dataKey="memory" stroke="#3b82f6" name="Memory" strokeWidth={2} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
