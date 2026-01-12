# K8s Service Optimizer Dashboard

React-based web dashboard for visualizing Kubernetes cluster metrics, service health, and optimization recommendations.

## Tech Stack

- **Framework**: React 18 with TypeScript
- **Build Tool**: Vite 5
- **Styling**: Tailwind CSS 3
- **Routing**: React Router v6
- **Charts**: Recharts 2
- **State Management**: React Hooks (useState, useEffect, useContext)

## Project Structure

```
dashboard/
├── src/
│   ├── components/
│   │   └── Layout/
│   │       ├── MainLayout.tsx    # Main application layout
│   │       ├── Sidebar.tsx       # Navigation sidebar
│   │       └── Header.tsx        # Top header with WebSocket status
│   ├── hooks/
│   │   └── useWebSocket.ts       # WebSocket hook for real-time updates
│   ├── pages/
│   │   ├── Dashboard.tsx         # Cluster overview page
│   │   ├── Services.tsx          # Services list page
│   │   └── Recommendations.tsx   # Recommendations page
│   ├── services/
│   │   ├── api.ts                # REST API client
│   │   └── types.ts              # TypeScript type definitions
│   ├── App.tsx                   # Root component with routing
│   ├── main.tsx                  # Application entry point
│   └── index.css                 # Global styles with Tailwind
├── package.json
├── vite.config.ts
├── tsconfig.json
├── tailwind.config.js
└── postcss.config.js
```

## Getting Started

### Prerequisites

- Node.js 18+ and npm
- Backend API server running on `http://localhost:8080`

### Installation

```bash
cd dashboard
npm install
```

### Development

Start the development server:

```bash
npm run dev
```

The dashboard will be available at `http://localhost:3000`

### Build

Build for production:

```bash
npm run build
```

Preview production build:

```bash
npm run preview
```

## Features

### Current Implementation (C1 - Foundation)

- ✅ Vite + React 18 + TypeScript setup
- ✅ Tailwind CSS configuration
- ✅ React Router v6 navigation
- ✅ REST API client (`OptimizerAPI` class)
- ✅ WebSocket hook for real-time updates
- ✅ Main layout with sidebar navigation
- ✅ Three main pages with placeholders:
  - Dashboard: Cluster overview with metrics
  - Services: Service list with health scores
  - Recommendations: Optimization recommendations

### Upcoming Features

- **C2**: Resource utilization charts and visualizations
- **C3**: Service detail pages with pod metrics
- **C4**: Enhanced recommendations with detailed analysis

## API Integration

The dashboard connects to the backend API at `http://localhost:8080/api/v1`.

### Available API Methods

```typescript
// Cluster
api.getClusterOverview()
api.getHealth()

// Services
api.getServices()
api.getServiceDetail(namespace, name)

// Recommendations
api.getRecommendations()
api.applyRecommendation(id)
api.dismissRecommendation(id)

// Metrics
api.getNodeMetrics()
api.getPodMetrics(namespace?)

// Analysis
api.getAnalysis(namespace, service)

// Cost
api.getCostBreakdown(namespace, service)
```

### WebSocket Updates

Real-time updates are received via WebSocket at `ws://localhost:8080/ws/updates`.

```typescript
import { useWebSocket } from './hooks/useWebSocket';

const { isConnected, error } = useWebSocket(
  'ws://localhost:8080/ws/updates',
  {
    onMessage: (message) => {
      console.log('Received update:', message);
    },
  }
);
```

## Configuration

### Backend URL

Update the API base URL in `src/services/api.ts`:

```typescript
export const api = new OptimizerAPI('http://your-backend-url/api/v1');
```

### Development Port

Change the dev server port in `vite.config.ts`:

```typescript
export default defineConfig({
  server: {
    port: 3000,
  },
});
```

## Pages

### Dashboard (`/dashboard`)

- Cluster metrics overview
- Node count, pod count, CPU/memory usage
- Health score visualization
- Auto-refreshes every 30 seconds

### Services (`/services`)

- List of all services with health scores
- Real-time CPU and memory usage
- Status indicators
- Auto-refreshes every 30 seconds

### Recommendations (`/recommendations`)

- Optimization recommendations
- Priority indicators (Critical, High, Medium, Low)
- Estimated cost savings
- Apply/dismiss actions

## Development Notes

### Hot Module Replacement (HMR)

Vite provides instant HMR for fast development. Changes to React components will be reflected immediately without full page reload.

### TypeScript

All types are defined in `src/services/types.ts` and match the backend models exactly.

### Styling

Tailwind CSS is configured with a custom color palette. Use utility classes for styling:

```typescript
<div className="bg-primary-500 text-white p-4 rounded-lg">
  Content
</div>
```

### Error Handling

The API client automatically handles errors and throws exceptions with meaningful messages. Components catch and display errors gracefully.

## Troubleshooting

### WebSocket Connection Issues

If the WebSocket connection fails:
1. Ensure the backend is running on port 8080
2. Check CORS settings in the backend
3. Verify WebSocket endpoint: `ws://localhost:8080/ws/updates`

### API Connection Issues

If API calls fail:
1. Verify backend is running: `curl http://localhost:8080/api/v1/cluster/health`
2. Check browser console for CORS errors
3. Ensure API base URL is correct in `src/services/api.ts`

### Build Errors

If build fails:
1. Delete `node_modules` and `package-lock.json`
2. Run `npm install` again
3. Run `npm run build`

## Next Steps

After the foundation is complete:

1. **Component C2**: Implement resource utilization charts
2. **Component C3**: Build service detail views
3. **Component C4**: Enhance recommendations UI
4. **Testing**: Add unit and integration tests
5. **Optimization**: Add React Query for data fetching

## License

MIT
