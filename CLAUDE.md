# Agentic Coding Guidelines

> Arion: local-first voice input for macOS and Windows using MyGo native UI.
> Read this file first. Then follow the documentation map below before writing any code.

---

## ⚠️ Before you start

1. **Read the docs first.** Do not assume conventions; this project is documented so agents do not have to guess.
2. **Ask when boundaries are unclear.** If a requirement, layer boundary, business rule, or technology choice is ambiguous, ask the user before proceeding.
3. **Check logs and docs before inventing workarounds.** When something does not behave as expected, inspect logs, tests, and the relevant `docs/` file. Do not add fallback logic to bypass a problem you have not understood.

---

## Documentation map

### Understand the project

- [`docs/README.md`](docs/README.md) — top-level index and domain map.
- [`docs/project/README.md`](docs/project/README.md) — product scope, goals, and selected stack.
- [`docs/project/architecture.md`](docs/project/architecture.md) — system architecture, module boundaries, and data flow.

### Start implementation

- [`specs/README.md`](specs/README.md) — normative product requirements and acceptance criteria.
- [`docs/project/agent-handoff.md`](docs/project/agent-handoff.md) — task dependencies and implementation entry points.
- [`docs/research/README.md`](docs/research/README.md) — competitor evidence and framework research.
- [`docs/project/repository.md`](docs/project/repository.md) — Go monorepo ownership and package boundaries.

### Write frontend code

- [`docs/frontend/README.md`](docs/frontend/README.md) — Feature-Sliced Design (FSD) entry point.
- [`docs/frontend/fsd-overview.md`](docs/frontend/fsd-overview.md) — what FSD is and why it is used here.
- [`docs/frontend/layers.md`](docs/frontend/layers.md) — responsibilities of `app`, `pages`, `widgets`, `features`, `entities`, `shared`.
- [`docs/frontend/slices.md`](docs/frontend/slices.md) — how to split code into slices.
- [`docs/frontend/segments.md`](docs/frontend/segments.md) — `ui`, `model`, `lib`, `api`, `config` segments.
- [`docs/frontend/public-api.md`](docs/frontend/public-api.md) — public API and re-export rules.
- [`docs/frontend/import-rules.md`](docs/frontend/import-rules.md) — cross-layer and cross-slice import rules.
- [`docs/frontend/i18n.md`](docs/frontend/i18n.md) — interface languages, typed copy keys, formatting and completeness gates.
- [`docs/frontend/ui-patterns.md`](docs/frontend/ui-patterns.md) — semantic styling, no hardcoded copy, no redundant copy.
- [`docs/quality/agent-skills.md`](docs/quality/agent-skills.md) — Agent Skills for UI craft, motion, and anti-slop. Apply them when writing or reviewing UI.

### Write backend code

- [`docs/backend/README.md`](docs/backend/README.md) — Domain-Driven Design (DDD) entry point.
- [`docs/backend/domain.md`](docs/backend/domain.md) — entities, value objects, aggregates, domain services, domain events.
- [`docs/backend/application.md`](docs/backend/application.md) — use cases, application services, transactions, DTOs.
- [`docs/backend/infrastructure.md`](docs/backend/infrastructure.md) — persistence, external services, messaging, configuration.
- [`docs/backend/interfaces.md`](docs/backend/interfaces.md) — HTTP/CLI/event adapters, controllers, input validation.
- [`docs/backend/api-conventions.md`](docs/backend/api-conventions.md) — response format, error codes, versioning.
- [`docs/backend/database.md`](docs/backend/database.md) — database, migrations, ID strategy.
- [`docs/backend/logging.md`](docs/backend/logging.md) — logging, tracing, and masking.

### Operations and quality

- [`docs/operations/README.md`](docs/operations/README.md) — local development, CI/CD, deployment.
- [`tools/evaluation/README.md`](tools/evaluation/README.md) — pnpm/Make evaluation commands and live driver protocol.
- [`docs/quality/e2e.md`](docs/quality/e2e.md) — actual native window and product OS scenarios.
- [`docs/quality/README.md`](docs/quality/README.md) — testing strategy and code-review expectations.
- [`docs/quality/agent-skills.md`](docs/quality/agent-skills.md) — installed Agent Skills (`.agents/skills/`).
- [`docs/decisions/README.md`](docs/decisions/README.md) — architecture decision records (ADRs).

---

## Language and quality rules

### English only for code and comments

- All source code, comments, commit messages, and internal identifiers must be written in English.
- User-facing copy uses the approved English and Simplified Chinese catalogs, with English as the default. Additional languages require an explicit request; code, comments and identifiers remain English.

### Forbidden patterns

- **No hardcoded strings.** All user-facing text must be defined in a single place (i18n keys, constants, configuration) and referenced by identifier.
- **No redundant UI copy.** Do not repeat information that is already conveyed by a title, icon, selected state, or surrounding context.
- **No duplicated implementations.** If a piece of logic already exists, reuse it or extract it to the correct layer/slice. Do not copy-paste with minor variations.
- **No fallback/clever bypass logic.** Do not mask a root cause with a default value, a silent catch, or a conditional shortcut. Face the actual problem and fix it, or ask the user.
- **No assumptions about hidden requirements.** If a feature is not documented and the intent is unclear, ask.

### When something is unclear

1. Search the current `docs/` for the topic.
2. Read the relevant layer/slice README.
3. Inspect recent logs, tests, or existing code.
4. If still unclear, ask the user with a concise, specific question.

---

## Frontend: Feature-Sliced Design (FSD)

- A page is an orchestration layer. It composes widgets and features and reads routing data. It must not contain business logic or reusable UI.
- A widget is a self-contained block of UI that belongs to a page or a feature.
- A feature owns a complete user scenario (e.g., "create order"). It contains UI, model, API, and logic that are specific to that scenario.
- An entity owns domain data and rules (e.g., `User`, `Order`). It has no UI and no HTTP client logic.
- `shared` contains framework-agnostic utilities, UI primitives, and configuration that can be used by any layer.
- Imports go only downward: `app` → `pages` → `widgets` → `features` → `entities` → `shared`.
- Each Go UI slice exposes exported symbols from its package; any future TypeScript slice uses `index.ts`. Do not import internals of another slice.

See [`docs/frontend/`](docs/frontend/README.md) for the full rules.

---

## Backend: Domain-Driven Design (DDD)

- **`domain/`** contains business rules. It has no dependencies on frameworks, databases, or transport details.
- **`application/`** contains use cases and orchestration. It depends only on `domain` and abstractions defined in `domain`.
- **`infrastructure/`** implements the abstractions from `domain` using concrete frameworks, databases, and external services.
- **`interfaces/`** adapts the application layer to the outside world (HTTP controllers, CLI commands, event handlers). It depends on `application` and `domain`.
- Dependencies point inward: `interfaces` → `application` → `domain`; `infrastructure` implements `domain` ports.
- Keep controllers thin: parse input, call an application service, format output.

See [`docs/backend/`](docs/backend/README.md) for the full rules.

---

## Self-evolution rule

After completing a task, update the documentation if the task has changed any of the following:

- Product behavior or user-facing workflows.
- Architecture, layer responsibilities, or module boundaries.
- Runtime configuration, persistence, or deployment steps.
- Testing expectations or quality gates.
- Coding conventions, style rules, or FSD/DDD boundaries.

**How to update docs:**

1. Edit the relevant existing doc in the same change set as the code.
2. If no doc exists for the new topic, create one under the correct `docs/` subdomain and add it to the nearest `README.md`.
3. Keep docs concise, factual, and example-driven. Avoid future dates, day estimates, and speculative language.
4. Do not let implementation and documentation drift.

---

## Commit conventions

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>
```

- `type`: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `build`, `ci`, `perf`, `revert`.
- `scope`: module or layer name, e.g., `frontend`, `backend`, `auth`, `orders`.
- `subject`: lowercase, concise, no trailing period.

Example: `feat(orders): add checkout use case`.

---

## Pull request conventions

- **Title:** the same Conventional Commits form as a commit subject.
- **Body:** English, in these sections, in this order:
  1. `## Summary` — what changes and why, one numbered item per independent change. Cite the spec acceptance IDs and ADRs involved.
  2. `## Changes` — grouped by area (e.g. **Desktop IME**, **Site**). Name the files and what changed in each. Include doc, spec and ADR updates, and corrections made in passing (e.g. a doc that disagreed with the code).
  3. `## Screenshots` — required for any visible UI change, on the site or in the desktop app. Show every changed state: each locale whose layout differs, mobile width for the site, and intermediate animation frames when the change is motion. Lay out pairs in Markdown tables with a bold label per cell.
  4. `## Testing` — every check that ran and its result (e.g. `pnpm site:check`, native tests, a browser check). Under **Not run**, list each check that could not run here (e.g. native tests that need macOS) and the exact command or manual step a reviewer should use.
- **Screenshot hosting:** never commit screenshots to the feature branch. Push them to a separate image-only branch (`claude/pr-screenshots-<topic>`, under `pr-screenshots/`). Link them as `https://github.com/AstraFoundry/Arion/blob/<branch>/pr-screenshots/<file>.png?raw=true`. Say in the body that the branch can be deleted after merge.
- **Claims:** describe only verified results. Do not present unverified behavior as working.

---

## `.workspace` directory

The `.workspace/` directory is for local, untracked work:

- `todo/` — long-running task progress files.
- `drafts/` — temporary notes and snippets.
- `notes/` — analysis and research.

Create a todo file for long tasks and mark items with `[ ]` / `[x]`. Clean up when done.

---

## Planning documents

- ❌ Do not use concrete dates ("2026-01-31").
- ❌ Do not estimate durations ("3 days", "Week 1").
- ✅ Describe task order and dependencies only.
- ✅ Use priorities `P0/P1/P2` to indicate importance.

## Arion desktop conventions

- MyGo native Go UI is the selected desktop shell. Do not introduce Electron, React, or a webview without a new decision.
- Read `specs/` before implementing a feature. Proposed model selections and unresolved deployment choices are not production commitments.
- Keep recognition drafts and correction candidates separate from committed target text. Cancellation and focus changes must invalidate stale work.
- Dictation must work offline after model provisioning; never silently route content to cloud inference.
- Imported competitor strings are static evidence, not verified execution. Preserve evidence limitations.
- Use Go package exports as FSD public APIs; the TypeScript examples in frontend docs are illustrative only.

## Project automation

Use pnpm exactly as pinned in `package.json`; install with the frozen lockfile. Common workflows belong in Makefile and portable Node entry points. Native tests require their actual target OS. Reusable benchmark corpora, exact fixtures, per-case outputs, failed attempts and reproduction sources belong under `benchmarks/` with original bytes, provenance and hashes. Keep weights, runtime environments, caches, personal recordings and unfinished execution in `.workspace/`; never substitute mock output for inference evidence. See `benchmarks/migration/README.md` for archive verification and report replay. Read [008 Evaluation](specs/008-evaluation.md) before changing experiments or drivers.
