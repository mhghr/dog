# Dog Repository AI Agent Instructions

## Mandatory Repository Intelligence Workflow

Before investigating, modifying, or explaining any code, you MUST understand the repository context first.

For every coding task, bug fix, refactor, feature implementation, architecture change, or debugging session:

You MUST start with Repowise MCP.

Required first steps:

1. Use `repowise_search_codebase` to locate relevant code, symbols, modules, and patterns.
2. Use `repowise_get_context` to understand architecture, dependencies, and relationships.
3. Use `repowise_get_symbol` when a specific function, type, class, handler, service, or interface is involved.
4. Use `repowise_get_risk` before significant changes or refactoring.
5. Use `repowise_get_health` when investigating hotspots, complexity, maintainability, or suspicious areas.

Do not begin with:
- grep
- ripgrep
- find
- opening random files
- guessing file locations
- scanning the repository broadly

Only read source files directly after Repowise identifies the relevant areas.

---

# Repository Analysis Rules

The Dog repository is a large enterprise monitoring platform.

Before changing code, always identify:

- affected service
- affected package
- existing architecture pattern
- existing implementation examples
- dependencies
- callers and consumers
- database impact
- API impact
- frontend impact
- backward compatibility concerns

Prefer understanding the existing system over introducing new patterns.

---

# Required Workflow For Every Task

Follow this sequence:

## Phase 1 — Understand

Before writing code:

- Understand the user request.
- Use Repowise to analyze the repository.
- Identify affected components.
- Identify existing similar implementations.

## Phase 2 — Investigate

Use Repowise to determine:

- relevant files
- symbols
- dependencies
- callers
- data flow
- architecture relationships

Avoid unnecessary file reads.

## Phase 3 — Plan

Before implementation:

Explain:

- root cause
- proposed solution
- affected areas
- possible risks

For large changes, validate the design against existing architecture.

## Phase 4 — Implement

When modifying code:

- Follow existing project conventions.
- Reuse existing abstractions.
- Avoid duplicate implementations.
- Keep changes minimal.
- Do not introduce new patterns without checking existing ones.

## Phase 5 — Validate

After changes:

Run appropriate validation:

Backend:
- go test
- go vet
- compile checks

Frontend:
- type check
- lint
- build validation

Database:
- migration validation
- schema compatibility check

---

# Architecture Awareness

Dog is an enterprise monitoring platform.

Respect these architectural boundaries:

## Backend

Services:
- API
- Worker
- Scheduler
- Agent Gateway
- Probe Agent
- Monitoring Agent
- OTEL ingestion
- Metric processing

Do not move logic between services without understanding boundaries.

---

## Messaging

The platform uses event-driven architecture.

Before changing asynchronous flows, inspect:

- NATS subjects
- consumers
- publishers
- event schemas
- retry behavior
- idempotency handling

---

## Metrics

Time-series data belongs to VictoriaMetrics.

Before modifying metrics:

Check:

- metric naming
- labels
- aggregation strategy
- retention expectations
- query patterns

---

## Database

Before changing database code:

Check:

- migrations
- models
- repositories
- nullable fields
- tenant isolation
- indexes

Never assume schema structure.

---

## Frontend

Before changing UI/data flow:

Understand:

- Server Components
- Client Components
- React Query usage
- API clients
- caching strategy
- loading states
- error handling

Avoid creating independent data fetching for every component.

---

# Debugging Rules

When debugging:

Do not immediately patch symptoms.

First determine:

1. Where the error originates.
2. Which layer owns the problem.
3. Existing patterns for solving similar problems.
4. Side effects of the fix.

Use Repowise to analyze:

- callers
- dependencies
- related implementations
- risk

---

# Refactoring Rules

Before refactoring:

Required:

- Analyze current implementation with Repowise.
- Check change impact.
- Identify duplicated patterns.
- Check affected modules.

Avoid:

- large uncontrolled rewrites
- unnecessary abstractions
- changing public APIs without analysis

---

# Context Efficiency Rules

Optimize context usage:

- Prefer Repowise context over reading many files.
- Read only required source files.
- Avoid loading entire directories.
- Avoid repeated searches.
- Use skeleton/context information when available.

The goal is maximum accuracy with minimum context consumption.

---

# Communication Rules

Before making significant changes:

Provide:

1. Current understanding.
2. Root cause.
3. Implementation plan.
4. Expected impact.

After changes:

Provide:

1. Files changed.
2. Reason for changes.
3. Validation performed.
4. Remaining risks.

---

# Final Rule

Repowise is the primary repository intelligence layer.

The default workflow is:

User Request
      |
      v
Repowise Analysis
      |
      v
Relevant Source Inspection
      |
      v
Implementation
      |
      v
Validation

Do not skip repository analysis.