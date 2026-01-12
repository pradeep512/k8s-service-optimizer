You are Claude Code. Create a detailed, implementation-ready plan to finish the missing dashboard components (C2: Resource Utilization charts, C3: Service Details view). The plan must be grounded in this repo and reference exact files and API endpoints where relevant.

Scope and goals
1) Implement Resource Utilization charts on the Dashboard page (Component C2).
2) Implement Service Details panel on the Services page (Component C3) with clickable rows.
3) Ensure the plan respects current backend API capabilities in `backend/pkg/api/router.go` and response shapes in `backend/pkg/api/handlers.go`.
4) Avoid inventing backend endpoints. If data is missing, list options:
   - derive from existing endpoints
   - adjust UI to show partial data
   - add backend endpoint (optional, explicit)

Repo constraints and references
- Dashboard pages:
  - `dashboard/src/pages/Dashboard.tsx`
  - `dashboard/src/pages/Services.tsx`
- API client and types:
  - `dashboard/src/services/api.ts`
  - `dashboard/src/services/types.ts`
- Backend API routes:
  - `backend/pkg/api/router.go`
  - `backend/pkg/api/handlers.go`
- Backend models:
  - `backend/internal/models/types.go`
- Optional: dashboard layout and header:
  - `dashboard/src/components/Layout/`

What to include in the plan
- A short summary of current gaps (why the UI shows placeholders).
- Proposed data flows for each component using existing endpoints:
  - `/api/v1/cluster/overview`
  - `/api/v1/metrics/nodes`
  - `/api/v1/services`
  - `/api/v1/services/:namespace/:name`
  - `/api/v1/analysis/:namespace/:service`
  - `/api/v1/traffic/:namespace/:service`
  - `/api/v1/cost/:namespace/:service`
- Specific UI changes for:
  - C2: charts to show CPU/Memory usage over time (or current usage with time series if available).
  - C3: service detail panel populated when a row is clicked.
- Changes needed to `dashboard/src/services/api.ts` and `dashboard/src/services/types.ts` to reconcile backend response shape differences (Go JSON field casing vs TS expectations).
- A phased implementation plan:
  1) data/model fixes
  2) UI wiring
  3) UX polish (loading, errors, empty states)
  4) optional enhancements
- Call out any backend work needed if real time-series or per‑service metrics are missing, with exact file references for where to add it.

Formatting requirements
- Output a single Markdown plan.
- Use headings, numbered steps, and short code snippets when needed.
- Keep instructions explicit enough that a developer can follow without guessing.
- Reference file paths in backticks.
