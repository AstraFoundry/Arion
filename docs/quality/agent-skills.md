# Agent skills

Project-level Agent Skills live in [`.claude/skills/`](../../.claude/skills/). The `.agents` symlink points to `.claude`; Windows checkouts without Git symlink support can read the canonical `.claude/skills/` directory directly.

Installed skills encode UI craft, motion, and anti-slop conventions. When writing or reviewing frontend UI, apply the relevant skill instead of inventing visual defaults.

## Local skills

These skills are checked into [`.claude/skills/`](../../.claude/skills/) directly. They are not listed in [`skills-lock.json`](../../skills-lock.json) and are not installed with `pnpm dlx skills add`.

### `find-simplifications`

| Skill | Use when |
|-------|----------|
| `find-simplifications` | Seeking evidence-backed simplifications: dead, duplicated, speculative, or unnecessarily maintained APIs, config, tests, docs, or infrastructure (removable maintenance obligations). Survey and propose. A survey is not permission to implement. |

Canonical path: [`.claude/skills/find-simplifications/`](../../.claude/skills/find-simplifications/) (`SKILL.md`, plus [`references/historical-patterns.md`](../../.claude/skills/find-simplifications/references/historical-patterns.md)).

## Installed skills

Sources are pinned in [`skills-lock.json`](../../skills-lock.json).

### `emilkowalski/skills`

| Skill | Use when |
|-------|----------|
| `emil-design-eng` | General UI polish, component design, and animation taste. |
| `animate` | Building a web animation from scratch. |
| `animate-expo` | Building React Native / Expo motion. |
| `animation-vocabulary` | Naming a motion effect from a vague description. |
| `apple-design` | Gesture-driven UI, springs, sheets, and Apple-style motion on the web. |
| `ask-sonner` | Installing or debugging Sonner toasts. |
| `find-animation-opportunities` | Finding places that should (or should not) animate. Read-only. |
| `improve-animations` | Auditing existing motion and producing implementation plans. |
| `pick-ui-library` | Choosing a library instead of hand-rolling a common UI primitive. Explicit invoke only. |
| `prototype` | Building multiple UI variants behind a picker. Explicit invoke only. |
| `review-animations` | Reviewing animation code against a high craft bar. |
| `write-swift` | Writing or reviewing Swift. |

### `jakubkrehel/make-interfaces-feel-better`

| Skill | Use when |
|-------|----------|
| `make-interfaces-feel-better` | Polishing UI details: radius, type, shadows, icons, micro-interactions. |

### `wheresryan22/anatomy`

| Skill | Use when |
|-------|----------|
| `anatomy` | Drawing isometric technical figures. `ui/shared/iso` ports its kit's camera and shading to MyGo for the onboarding dictation key figure and the model step's typewriter figure. The examples' generated pages and PDFs are not committed; `node .agents/skills/anatomy/examples/<name>/build.mjs` rebuilds a page. |

### `yetone/kill-ai-slop`

| Skill | Use when |
|-------|----------|
| `kill-ai-slop` | Removing generic AI visual and copy tics from UI, landing pages, or docs. |

## Add or update skills

Install into this repository (not globally):

```sh
pnpm dlx skills@latest add <owner/repo> --skill '*' -a cursor -y --copy
```

Refresh installed skills:

```sh
pnpm dlx skills update
```

List what is installed:

```sh
pnpm dlx skills list
```
