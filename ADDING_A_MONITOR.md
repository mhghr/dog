# Adding a monitor type

Monitor types are plug-in-like features. Type-specific UI belongs under
`apps/web/features/monitors/types/<type>/`; probe execution belongs in
`internal/probe/`. Routes, list pages, filters, and generic forms must consume
the registries and must not add type switches for presentation concerns.

## Responsibilities

### Frontend contract

- `apps/web/types/monitor.ts` owns the wire-level `MonitorType` values.
- `apps/web/features/monitors/core/definition.ts` defines the UI feature contract.
- `apps/web/features/monitors/core/registry.ts` is the single composition root.
- `apps/web/features/monitors/types/<type>/definition.ts` owns the icon, group,
  scheduling defaults, config-fields component, and API-error mapping for one type.
- `apps/web/lib/schemas.ts` owns transport serialization and form validation.
  Type-specific fields and config serialization are added here until the API
  adopts versioned per-type config schemas.
- `apps/web/lib/monitor-meta.ts` is a compatibility facade for older consumers;
  new feature code imports the registry directly.

### Backend contract

- `internal/domain/monitor.go` owns the backend monitor type constant.
- `internal/domain/validate.go` validates the persisted/API config.
- `internal/probe/<type>.go` implements `probe.Executor`.
- `internal/probe/<type>_test.go` covers success, failure, timeout, and malformed config.
- `internal/probe/probe.go` registers the executor in `DefaultRegistry`.

The scheduler and worker are type-agnostic. Do not add a type switch there.

## Checklist

For a new type called `redis`:

1. Add `redis` to `MONITOR_TYPE_VALUES` in `apps/web/types/monitor.ts`.
2. Create `apps/web/features/monitors/types/redis/definition.ts` implementing
   `MonitorTypeDefinition`.
3. Import that definition directly in `core/registry.ts` and add it to `definitions`.
4. Add the Redis form fields and validation/serialization fields in
   `apps/web/lib/schemas.ts`.
5. Add translation keys to both `apps/web/messages/en.json` and `fa.json`.
6. Add the backend `domain.MonitorType` constant and validation.
7. Implement `internal/probe/redis.go` and register it in `DefaultRegistry`.
8. Add frontend schema/registry tests and backend executor tests.
9. Run `pnpm --filter web lint`, `pnpm --filter web test`,
   `pnpm --filter web exec tsc --noEmit`, and `go test ./...`.

## Definition example

```ts
export const redisMonitorDefinition = {
  type: "redis",
  group: "network",
  icon: Database,
  defaultIntervalSeconds: 60,
  minimumIntervalSeconds: 10,
  defaultValues: { redis_port: 6379 },
  ConfigFields: RedisConfigFields,
  apiFieldMap: { "config.port": "redis_port" },
} satisfies MonitorTypeDefinition;
```

## Design rules

- Keep generic monitor pages ignorant of concrete monitor types.
- Prefer one explicit definition per type over boolean props or nested conditionals.
- Import definitions directly in the composition root; do not use a broad barrel
  export that pulls every type into unrelated bundles.
- Store stable queryable fields in the monitor/result models. Store type-specific
  configuration in `config`, numeric time-series values in `metrics`, and diagnostic
  payloads in `attributes`.
- Never silently accept an unregistered type. Registry completeness tests must fail
  when the wire contract and the UI registry diverge.
