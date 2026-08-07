# Dog Web — Enterprise Frontend Architecture Design

**Date:** 2026-08-07
**Status:** Approved by user
**Scope:** Complete restructure of `apps/web` into a layered, domain-driven, enterprise observability frontend (Datadog/Grafana Cloud/New Relic class).

## 1. Core Principle

There is exactly **one** structure in the repository. The current structure is **updated in place** to the target structure. No parallel structures, no versioning (`v1`/`v2`), no leftover shim files. Nothing old remains after migration. This applies to files, directories, imports, DB layer and any code paths.

### Layer dependency direction

```
UI Layer
    ↓
Feature Layer
    ↓
Application Layer
    ↓
Entity Layer
    ↓
Data Access Layer (React Query)
    ↓
API Client Layer
    ↓
Backend API
```

**Rules:**
- No component calls HTTP directly. Components consume React Query hooks only.
- Entities must not depend on features.
- Features can use entities; entities cannot use features.
- Application layer combines multiple entities/APIs for complex workflows.
- All backend data flows through React Query (server state).
- Zustand holds client-only global state only (workspace, user, sidebar, theme, UI state). Never server data. Metrics are never stored in Zustand.

## 2. Routing

```
app/[locale]/
├── (marketing)/         → page, features, pricing, security, terms, docs
├── (auth)/              → login, register
└── console/w/[workspaceSlug]/
    ├── dashboard/
    ├── resources/[resourceId]/
    │   ├── page.tsx         ← Overview (no separate route)
    │   ├── monitoring/
    │   ├── metrics/
    │   ├── logs/
    │   ├── traces/
    │   └── settings/
    ├── monitors/
    ├── alerts/
    ├── nodes/
    ├── agents/
    ├── probes/
    ├── locations/
    ├── status-pages/
    └── settings/ → members | roles | api-keys | integrations | billing | system
```

- `[locale]` kept (fa/en via next-intl middleware).
- Workspace slug in route; real `workspace_id` (UUID) in API. `useWorkspace()` resolves slug → id.
- All query keys are scoped by workspace id.
- `app/` contains routing and layout only, plus auth route handlers (`/api/auth/google/*`). No business logic.
- Server Components by default; Client Components only for forms, filters, charts, interactive tables, realtime dashboards.

## 3. Entities Layer (Business Models only)

Business model entities:

```
entities/{workspace, resource, monitor, alert, probe, agent}/
├── model/
│   ├── types.ts
│   └── schemas.ts
├── api/
│   └── {entity}.api.ts        ← domain API lives in the entity, not in shared
├── hooks/
│   └── use-{entity}.ts
└── ui/
    ├── {Entity}Card.tsx
    └── {Entity}Status.tsx
```

- Entities know their own data structures.
- Metric, Log, Trace are **data streams**, not entities. They live under `features/observability/{metrics,logs,traces}`.

## 4. Features Layer

User actions and business flows:

```
features/
├── resource-management/     → ResourceTable, ResourceFilters, ResourceWizard, hooks, services
├── monitor-management/      → MonitoringSelector, DynamicMonitorForm, ExecutionSettings, HealthRuleBuilder, monitor list/table/filters/actions
├── alert-management/
├── agent-enrollment/
├── probe-management/
├── metrics-explorer/        → hooks/, adapters/, transformers/  (complex monitoring data)
├── logs-explorer/
├── dashboard/
└── observability/
    ├── metrics/
    ├── logs/
    └── traces/
```

## 5. Application Layer

Complex cross-entity workflows. Business flow lives here, not in components.

```
application/
├── resources/        → create-resource, resource-overview, resource-health
├── monitoring/       → enable-monitor, disable-monitor, configure-monitor
└── onboarding/
```

Example: Resource Detail page needs resource + monitor + alert + metric data. `getResourceOverview()` combines these; UI does not call four APIs.

## 6. Shared Layer

```
shared/
├── api/                  ← centralized HTTP client
│   ├── client.ts         ← base URL, auth headers, JWT refresh, cancellation, tracing
│   ├── endpoints.ts      ← central endpoint registry
│   └── errors.ts         ← ApiError normalization
├── data/                 ← data access layer
│   ├── query-client.ts
│   ├── query-keys.ts     ← per-domain key factories, scoped by workspace
│   ├── realtime.ts       ← SSE/WebSocket subscription helpers → cache invalidation
│   ├── pagination.ts
│   └── filters.ts
├── ui/                   ← shadcn-style primitives
├── forms/                ← RHF + zod, dynamic schema renderer
├── hooks/
├── utils/
└── types/
```

### Data flow

```
Component → React Query Hook → api-client → Backend (Go)
```

Next.js is frontend only. Route handlers (`app/api/...`) used only for special cases (auth proxy, BFF, server-only logic). No axios/fetch outside `shared/api`.

## 7. Platform Layer

```
platform/
├── auth/                 ← auth-button, use-auth
├── permissions/          ← RBAC checks, role-based UI rendering, workspace isolation
├── realtime/             ← websocket-client, sse-client, events
└── state/                ← workspace-store, user-store, console-store (zustand)
```

Realtime events invalidate React Query cache. Example: `monitor.status.changed` SSE event → invalidate monitor query → UI updates.

## 8. Plugins

```
plugins/monitoring/
├── core/                 ← types, registry, helpers
├── ping/
├── http/
├── ssl/
├── dns/
├── smtp/
├── ntp/
├── tcp/
└── domain-expiration/
```

Each plugin exports: icon, form schema, UI renderer, validation. Frontend plugins are separate from backend plugin engine. Adding a new plugin = add a folder; core app does not know every plugin.

## 9. Design System

```
design-system/
├── tokens/               ← colors, spacing, typography, radii, motion, shadows, z-index (CSS)
├── themes/               ← default.ts, brand.ts (white-labeling)
├── components/           ← StatusBadge, HealthIndicator, MetricCard, DataTable
└── patterns/             ← EmptyState, LoadingState, ErrorState
```

## 10. Widgets Layer

Composite UI built from features/entities:

```
widgets/
├── dashboard-overview/
├── alert-center/
├── resource-health-panel/
├── monitoring-map/
├── topology-view/
├── metrics-explorer/
└── console-shell/
```

## 11. State Management

- **React Query** owns all server state: resources, monitors, metrics, alerts.
- **Zustand** (to be added as dependency) owns client global state: workspace, user, sidebar, theme, UI state.
- Metrics are never placed in Zustand.

## 12. Query Keys & Cache Policy

- Query keys: per-domain factories in `shared/data/query-keys.ts`, all scoped by workspace id.
- Static data (resource types, plugin schemas): cache hours/days.
- Normal data (resources, monitors): cache 30–60s.
- Realtime data (metrics, logs, traces): SSE/WebSocket/streaming.

## 13. Error Handling

Global API error handling, error boundaries, retry strategy, empty states, loading skeletons, request tracing.

## 14. Forms

React Hook Form + Zod. `shared/forms` includes dynamic schema renderer so monitoring plugin schemas render automatically (e.g. `{ fields: [{ name: "timeout", type: "number" }] }`).

## 15. Migration / Cleanup

The following are **migrated in place** (no copies, no versioning):

| Current | Target |
|---|---|
| `app/[locale]/(console)/app/*` | `app/[locale]/console/w/[workspaceSlug]/*` |
| `shared/api-client/` | `shared/api/` (client.ts, endpoints.ts, errors.ts) |
| `entities/resource/api/resources-api.ts` | `entities/resource/api/resource.api.ts` |
| `components/` legacy | migrated into entities/features/widgets/design-system |
| `hooks/`, `types/`, `lib/` shims | deleted, imports unified |
| `widgets/` empty dirs | populated or removed |
| `features/monitors/` old | `features/monitor-management/` |

**Deleted entirely:** `hooks/`, `types/`, `lib/`, `components/ui/` (re-export layer), legacy empty dirs. All imports unified to single canonical paths.

## 16. Final Directory Tree

```
apps/web/
├── app/                  ← [locale] routing + api route handlers
├── entities/             ← workspace, resource, monitor, alert, probe, agent
├── features/             ← resource-management, monitor-management, alert-management,
│                            agent-enrollment, probe-management, metrics-explorer,
│                            logs-explorer, dashboard, observability/{metrics,logs,traces}
├── application/          ← resources, monitoring, onboarding
├── shared/               ← api, data, ui, forms, hooks, utils, types
├── platform/             ← auth, permissions, realtime, state
├── plugins/              ← monitoring/
├── design-system/        ← tokens, themes, components, patterns
├── widgets/              ← composite screens
├── components.json
├── proxy.ts
├── next.config.ts
└── tsconfig.json
```

## 17. Non-Goals

- No backend changes (Go microservices untouched).
- No new functionality beyond what exists — restructure only, preserving current behavior.
- Metric/log/trace entities are not modeled as full entities; they are data-stream features.
