# Agentic Coding Guidelines

> Arion: local-first voice input for macOS and Windows using MyGo native UI.
> Read this file first. Then follow the documentation map below before writing any code.

---

## Before you start

1. **Read the relevant docs first.** Use the task routes below, then inspect the affected code, tests and recent logs. Do not read unrelated domains by default.
2. **Resolve uncertainty before changing behavior.** Search the relevant specs and docs, read the layer/slice README, and inspect existing evidence. If product requirements, layer boundaries or technology choices remain unclear or conflict, ask a concise, specific question before implementing the affected part. Choose local implementation details independently within established constraints.
3. **Fix causes, not symptoms.** Investigate unexpected behavior before adding recovery logic. Do not hide failures behind silent defaults or bypasses.

---

## Working across repositories

- Start product development in this public Arion checkout and read its `AGENTS.md`, which mirrors this file. No parent-level instruction file is required.
- Prefer a sibling private checkout at `../Arion-evaluation`. If it lives elsewhere, resolve its actual path before running commands. Keep it outside the public repository.
- Before working in the private repository, explicitly read its `AGENTS.md` and `README.md`. Do not assume this session loaded another repository's instructions.
- Run product commands in Arion and evaluation commands in Arion-evaluation. Commit each repository's changes separately. Follow the [repository split](docs/project/repository-split.md) for ownership and setup.
- Product development belongs in this checkout. Treat the private `.workspace/product` checkout and its generated links as prepared evaluation inputs. Follow the private version-lock procedure before evaluating a new product commit.
- Private evaluations and archive checks run locally. Keep GitHub Actions disabled in Arion-evaluation and do not add workflow files there.

If an agent tool restricts access to the current project, add the private checkout as a second workspace directory. Keep Arion as the product-development entry point.

## Documentation map

Use [the documentation index](docs/README.md) for the full map and [document authority and conflicts](docs/README.md#document-authority-and-conflicts). Read the entries relevant to the task and follow their domain-specific links.

| Task | Read before implementation |
| --- | --- |
| New to the project | [Product scope](docs/project/README.md), [architecture](docs/project/architecture.md) |
| Product behavior or module boundaries | Relevant [specifications](specs/README.md), [handoff](docs/project/agent-handoff.md), [repository ownership](docs/project/repository.md) |
| Desktop UI | [Frontend conventions](docs/frontend/README.md), [UI craft skills](docs/quality/agent-skills.md) |
| Marketing site or blog | [Site](docs/frontend/site.md), [blog](docs/frontend/blog.md); apply relevant [UI craft skills](docs/quality/agent-skills.md) for UI work |
| Backend or core logic | [Backend conventions](docs/backend/README.md) |
| Build, configuration or deployment | [Operations](docs/operations/README.md) |
| Experiments, evaluation or drivers | [008 Evaluation](specs/008-evaluation.md), [repository split](docs/project/repository-split.md), [product driver](tools/driver/README.md) |
| Validation or review | [Quality](docs/quality/README.md), [testing](docs/quality/testing.md), [native E2E](docs/quality/e2e.md) |
| Creating a pull request | [Pull request conventions](docs/quality/code-review.md#pull-request-conventions) |

---

## Language and quality rules

### English only for code and comments

- All source code, comments, commit messages, and internal identifiers must be written in English.
- User-facing copy uses the approved English and Simplified Chinese catalogs, with English as the default. Additional languages require an explicit request; code, comments and identifiers remain English.

### Forbidden patterns

- **No hardcoded strings.** All user-facing text must be defined in a single place (i18n keys, constants, configuration) and referenced by identifier.
- **No redundant UI copy.** Do not repeat self-evident information. Preserve accessible names and guidance needed to understand or complete an action.
- **No duplicated implementations.** Reuse existing implementations of the same business rule or extract them to the correct layer/slice. Similar code shape alone does not justify a shared abstraction.
- **No silent failure masking.** Do not mask a root cause with a default value, a silent catch, or a conditional shortcut. Recovery or degraded behavior explicitly approved by the specs must follow its documented conditions, expose failures as required, and be tested. It must not bypass privacy, cancellation or target-identity guarantees.
- **No assumptions about hidden requirements.** If a feature is not documented and the intent is unclear, ask.

---

## Frontend: Feature-Sliced Design (FSD)

- A page is an orchestration layer. It composes widgets and features and reads routing data. It must not contain business logic or reusable UI.
- A widget is a self-contained UI block composed by pages; it may compose features. Features must not import widgets.
- A feature owns a complete user scenario (e.g., "create order"). It contains UI, model, API, and logic that are specific to that scenario.
- An entity owns domain data and rules (e.g., `User`, `Order`). It has no UI and no HTTP client logic.
- `shared` contains framework-agnostic utilities, UI primitives, and configuration that can be used by any layer.
- Imports go only downward: `app` → `pages` → `widgets` → `features` → `entities` → `shared`.
- Each Go UI slice exposes exported symbols from its package; each TypeScript slice in `apps/site` uses `index.ts`. Do not import internals of another slice.

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
5. When you rename, remove or change a command, `pnpm` script, `make` target, test, setting, user-visible control, default or version identifier, search `docs/`, `specs/`, every README (including `README.zh-CN.md` and `README.ja.md`), `tools/driver/README.md` and `apps/site/content` for the old name or value, and fix every hit in the same change. Frozen evidence under `Arion-evaluation/benchmarks/` and dated research records keep what they recorded.
6. State a volatile fact (a test or case count, a version, a size, a default) in one owning document and link to it from the others, so there is one place to update.
7. Give an ADR the next unused number and add it to [`docs/decisions/README.md`](docs/decisions/README.md); never reuse a number. Index every new document in its directory `README.md`.
8. `README.md`, `README.zh-CN.md` and `README.ja.md` carry the same facts and the same local links; change them together. `AGENTS.md` mirrors this file, so edit only `CLAUDE.md`.
9. A document written for a generic stack (TypeScript examples) must open with its Arion mapping, as [`docs/frontend/layers.md`](docs/frontend/layers.md) and [`docs/backend/README.md`](docs/backend/README.md) do.
10. Run `pnpm check` before handing off documentation changes, whether or not committing. It validates repository structure and documentation; it does not establish native functionality. Run additional checks appropriate to the change under [testing](docs/quality/testing.md), and report results and any checks not run with their reasons.

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

Follow [the PR format and screenshot workflow](docs/quality/code-review.md#pull-request-conventions) when creating or updating a PR. Describe only verified results and explicitly identify checks not run.

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

Use pnpm exactly as pinned in `package.json`; install with the frozen lockfile. Common workflows belong in Makefile and portable Node entry points. Native tests require their actual target OS. Reusable benchmark corpora, exact fixtures, per-case outputs, failed attempts and reproduction sources belong under `benchmarks/` in the private `AstraFoundry/Arion-evaluation` Git repository, with original bytes, provenance and hashes. Public product tests retain small licensed fixtures in package `testdata` and use the public `tools/driver` contract. Keep weights, runtime environments, caches, personal recordings and unfinished execution in `.workspace/`; never substitute mock output for inference evidence. Follow the [repository split](docs/project/repository-split.md) and the private repository README for archive verification and report replay. Read [008 Evaluation](specs/008-evaluation.md) before changing experiments or drivers.
