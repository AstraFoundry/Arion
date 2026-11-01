# Documentation

This directory documents how this project is built, organized, and evolved by agents. Read the domains relevant to the task and follow their links; unrelated domains do not need to be read for every change.

## Product implementation

- [Clipboard history](project/clipboard.md) — opt-in local resource history and internal-write isolation.

- [Specifications](../specs/README.md) — normative workflows and acceptance IDs.
- [Agent handoff](project/agent-handoff.md) — implementation tasks and dependencies.
- [Repository split](project/repository-split.md) — public product ownership, private Git evidence and version-bound evaluation.
- [Central sync design proposal](project/central-sync-design.md) — recommended encrypted data scope, competitor comparison and server/client delivery design.
- [Research](research/README.md) — shipped-artifact evidence and local frameworks.

## Domain map

- [`project/`](project/README.md) — project overview, goals, architecture, and boundaries.
- [`frontend/`](frontend/README.md) — Feature-Sliced Design (FSD) conventions.
- [`backend/`](backend/README.md) — Domain-Driven Design (DDD) layered conventions.
- [`operations/`](operations/README.md) — local development, CI/CD, and deployment.
- [`quality/`](quality/README.md) — testing strategy, code-review expectations, and Agent Skills.
- [`decisions/`](decisions/README.md) — architecture decision records (ADRs).

## How to use this documentation

1. If you are new to the project, read [`project/README.md`](project/README.md) and [`project/architecture.md`](project/architecture.md) first.
2. Before writing frontend code, read [`frontend/README.md`](frontend/README.md).
3. Before writing backend code, read [`backend/README.md`](backend/README.md).
4. Before changing deployment, build, or operational behavior, read [`operations/README.md`](operations/README.md).
5. If you change behavior, architecture, or conventions, update the relevant doc in the same change set.

## Document authority and conflicts

- [Specifications](../specs/README.md) define product requirements and acceptance criteria.
- Adopted [ADRs](decisions/README.md) record technical decisions; proposals and research are not production commitments.
- Architecture and domain documentation explain implementation boundaries and workflows within those requirements and decisions.
- Code and tests show the current implementation and evidence, which may not yet satisfy the documented requirements.

When sources conflict, identify the discrepancy rather than silently choosing one or changing a requirement to match the code. Check whether an adopted decision explicitly supersedes the older guidance. If the intended product behavior, boundary or technology choice remains unresolved, ask before implementing the affected part. Local implementation details within established constraints can be decided independently.
