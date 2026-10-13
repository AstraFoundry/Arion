# Native desktop UI

Arion uses MyGo native Go packages under `apps/desktop/ui`. FSD responsibilities and downward imports remain; Go exported package symbols replace `index.ts`. Existing TypeScript/CSS examples below and in linked convention docs are illustrative, not a requirement to add a web frontend. See [repository mapping](../project/repository.md).


This section defines how the frontend is organized using **Feature-Sliced Design (FSD)**.

## Arion slices

| Layer | Slices |
| --- | --- |
| `app` | One view per window: overlay, menu bar panel, settings, onboarding, macOS consent guide |
| `pages` | `overlay`, `menubar`, `settings`, `onboarding` |
| `widgets` | `capsule` (dictation pill), `statusheader`, `modelcard`, `stepper`, `showcase` (illustrations: mock UI over a photograph) |
| `features` | `dictate`, `permissions`, `provision`, `shortcut`, `microphone`, `vocabulary`, `transcript`, `modelchoice`, `uilanguage` |
| `entities` | `dictation` (phase/notice projection), `model`, `onboarding` (step rules) |
| `shared` | `runtime` (state and actions boundary), `copy`, `theme` (Sunbeam tokens, motion presets), `kit` (coss-anatomy primitives), `icons` (Lucide, ISC), `assets` (app mark, photographs) |

Views read `runtime.State` and request `runtime.Actions`; the composition root mutates state on the main thread. Headless view tests in `ui/app` render every surface with MyGo's `ui.Tester`.

## Documents

- [`fsd-overview.md`](fsd-overview.md) — what FSD is and why it is used.
- [`layers.md`](layers.md) — responsibilities of each FSD layer.
- [`slices.md`](slices.md) — how to split code into slices.
- [`segments.md`](segments.md) — `ui`, `model`, `lib`, `api`, `config` segments.
- [`public-api.md`](public-api.md) — public API and re-export rules.
- [`import-rules.md`](import-rules.md) — cross-layer and cross-slice import rules.
- [`i18n.md`](i18n.md) — English/Simplified Chinese catalogs, preferences, formatting and native boundaries.
- [`ui-patterns.md`](ui-patterns.md) — semantic styling, no hardcoded copy, no redundant copy.
- [`design-system.md`](design-system.md) — Sunbeam tokens, coss-anatomy primitives, elevation and illustration rules.
- [`../quality/agent-skills.md`](../quality/agent-skills.md) — Agent Skills for UI craft, motion, and anti-slop.

## Quick start

1. Read [`fsd-overview.md`](fsd-overview.md) if FSD is new to you.
2. Read [`layers.md`](layers.md) to understand where a new file belongs.
3. Read [`import-rules.md`](import-rules.md) before adding any import.
4. Read [`ui-patterns.md`](ui-patterns.md) and [`design-system.md`](design-system.md) before writing UI code.
5. Apply the Agent Skills in [`../quality/agent-skills.md`](../quality/agent-skills.md) when the work is visual or motion-related.

## Core principle

Code is organized by **scope of change**, not by technical type. A feature contains everything it needs — UI, state, API, and utilities — so that changes to one feature do not leak into unrelated files.
