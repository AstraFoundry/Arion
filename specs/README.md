# Arion specifications

These documents are normative for implementation. Research recommendations require evaluation before becoming dependency decisions. Requirements marked MUST are release gates; implementation status is tracked in the [handoff](../docs/project/agent-handoff.md).

| Spec | Scope |
| --- | --- |
| [001 Desktop input](001-desktop-input.md) | Shortcut and configurable IME workflows, target ownership |
| [002 Speech pipeline](002-speech-pipeline.md) | Capture, streaming revisions and correction |
| [003 Memory sync](003-memory-sync.md) | Approved memory, offline outbox, self-hosted encrypted relay |
| [004 Contracts](004-contracts.md) | Portable events, cancellation, capabilities and errors |
| [005 Privacy](005-privacy.md) | Storage, consent, keys and network boundaries |
| [006 Models and evaluation](006-models.md) | Model provenance and measurable release gates |
| [007 IME and candidate ranking](007-ime-ranking.md) | IMK/TSF composition, local candidate scoring |
| [008 Evaluation](008-evaluation.md) | pnpm/Make automation, real OS E2E, streaming benchmarks and ablations |

## Product choices

Both input modes are selectable by the user. Self-hosting is the sync priority. macOS and Windows are P0; iOS and Android are P2 future targets. A whole conventional Pinyin keyboard is outside this scope; IME work starts with voice composition and candidate selection.

Jev-style local candidate ranking is explicitly requested. [Research](../docs/research/on-device-speech.md) identifies TypeSafe Jev and independent local implementations. Evaluate Laya Multilingual Choice first; the adapter contract remains model-independent.

## Traceability

Each acceptance scenario has a stable ID. New tests and task handoffs cite these IDs. An unchecked task is not an implemented feature. Changes to privacy defaults, input ownership or wire compatibility require an ADR and updated acceptance criteria.
