import { useState, useEffect } from 'react';
import { useWebSocket } from '../../hooks/useWebSocket';

export default function Header() {
  const [lastUpdate, setLastUpdate] = useState<string>('');

  const { isConnected, error } = useWebSocket(
    getWebSocketURL(),
    {
      onMessage: (message) => {
        setLastUpdate(new Date(message.timestamp).toLocaleTimeString());
      },
    }
  );

  useEffect(() => {
    // Set initial time
    setLastUpdate(new Date().toLocaleTimeString());
  }, []);

  return (
    <header className="flex h-16 items-center justify-between border-b border-gray-200 bg-white px-6">
      <div className="flex items-center space-x-4">
        <h2 className="text-lg font-semibold text-gray-900">
          Cluster Overview
        </h2>
      </div>

      <div className="flex items-center space-x-4">
        {/* WebSocket Status */}
        <div className="flex items-center space-x-2">
          <div
            className={`h-2 w-2 rounded-full ${
              isConnected ? 'bg-green-500' : 'bg-red-500'
            }`}
          />
          <span className="text-sm text-gray-600">
            {isConnected ? 'Connected' : error || 'Disconnected'}
          </span>
        </div>

        {/* Last Update */}
        {lastUpdate && (
          <div className="text-sm text-gray-500">
            Last update: {lastUpdate}
          </div>
        )}

        {/* User Info */}
        <div className="flex items-center space-x-2">
          <div className="h-8 w-8 rounded-full bg-primary-500 flex items-center justify-center">
            <span className="text-sm font-medium text-white">A</span>
          </div>
          <span className="text-sm font-medium text-gray-700">Admin</span>
        </div>
      </div>
    </header>
  );
}

function getWebSocketURL(): string {
  if (typeof window === 'undefined') {
    return 'ws://localhost:8080/ws/updates';
  }

  const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws';
  return `${scheme}://${window.location.hostname}:8080/ws/updates`;
}
