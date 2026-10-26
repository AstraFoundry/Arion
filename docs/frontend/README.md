# Native desktop UI

Arion uses MyGo native Go packages under `apps/desktop/ui`. FSD responsibilities and downward imports remain; Go exported package symbols replace `index.ts`. Existing TypeScript/CSS examples below and in linked convention docs are illustrative, not a requirement to add a web frontend. See [repository mapping](../project/repository.md).

The marketing site in `apps/site` is separate from the desktop app and is the only React code in the repository ([ADR 016](../decisions/016-marketing-site.md)); its conventions are in [`site.md`](site.md).


This section defines how the frontend is organized using **Feature-Sliced Design (FSD)**.

## Arion slices

| Layer | Slices |
| --- | --- |
| `app` | One view per window: overlay, menu bar panel, settings, onboarding, macOS settings guide, update reminder |
| `pages` | `overlay`, `menubar`, `settings`, `onboarding`, `updatereminder` |
| `widgets` | `capsule` (dictation pill), `statusheader`, `modelcard`, `stepper`, `showcase` (illustrations: mock UI over a photograph), `keyfigure` (the dictation key figure, drawn live with `shared/iso`) |
| `features` | `dictate`, `permissions`, `provision`, `holdkey` (dictation key picker and consent notes), `keyboard` (input-source setup and settings guide), `microphone`, `vocabulary`, `transcript`, `modelchoice`, `uilanguage`, `update` (check, verified download, reminder presentation and protected restart), `diagnostics` (About log export), `signin` (AstraFoundry account and hosted-sync waitlist) |
| `entities` | `dictation` (phase/notice projection), `model`, `onboarding` (step rules), `holdkey` (dictation key names), `account` (account stage and copy for errors and waitlist phases) |
| `shared` | `runtime` (state and actions boundary), `copy`, `theme` (Sunbeam tokens, motion presets), `kit` (coss-anatomy primitives), `icons` (Lucide, ISC; cloud provider marks from [LobeHub Icons](https://icons.lobehub.com), MIT), `assets` (app mark, photographs), `iso` (isometric solids rasterized on the CPU, after the `anatomy` skill's kit) |

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
- [`site.md`](site.md) — the `apps/site` marketing site: routing, copy catalogs, links, static-only rules.
- [`../quality/agent-skills.md`](../quality/agent-skills.md) — Agent Skills for UI craft, motion, and anti-slop.

## Quick start

1. Read [`fsd-overview.md`](fsd-overview.md) if FSD is new to you.
2. Read [`layers.md`](layers.md) to understand where a new file belongs.
3. Read [`import-rules.md`](import-rules.md) before adding any import.
4. Read [`ui-patterns.md`](ui-patterns.md) and [`design-system.md`](design-system.md) before writing UI code.
5. Apply the Agent Skills in [`../quality/agent-skills.md`](../quality/agent-skills.md) when the work is visual or motion-related.

## Core principle

Code is organized by **scope of change**, not by technical type. A feature contains everything it needs — UI, state, API, and utilities — so that changes to one feature do not leak into unrelated files.

## Update surfaces

Settings → About keeps automatic checking and automatic verified downloading as separate preferences. Automatic downloads require checks to be enabled; enabling them never authorizes an automatic restart. The download choice is retained when checks are turned off. System notification settings control reminders; About exposes no reminder switch or permission retry action.

Update progress distinguishes downloading, verified staging (installation pending restart), protected applying, and installed/ready to relaunch. A queued manual restart can be canceled before applying begins. Failures identify the check, preparation, apply or restart step and retry its action. An unrecoverable installation instead offers Quit and reinstall instructions; it never offers another swap. Failed reminder-state persistence is visible and pauses automatic reminders.

The separate `updatereminder` page composes `features/update.Reminder` in a bootstrap-owned nonactivating window after input ends. Policy owns its visibility. It exposes View update, Later and Dismiss; it never downloads, installs or restarts directly. The tray/menu entry stays available independently of notification permission and transient reminder suppression.

## Notification permissions

General and About have no notification switches. Signed-in standard-center accounts receive product notices automatically when OS notifications are allowed, without requiring hosted-sync entitlement. Bootstrap requests unknown OS permission once at a natural idle point after sign-in or an eligible update; denial is not automatically requested again. OS permission is refreshed on activation. See [product notifications](../project/product-notifications.md).
