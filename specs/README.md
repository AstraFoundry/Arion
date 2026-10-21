# Arion specifications

These documents are normative for implementation. Research recommendations require evaluation before becoming dependency decisions. Requirements marked MUST are release gates; implementation status is tracked in the [handoff](../docs/project/agent-handoff.md).

| Spec | Scope |
| --- | --- |
| [001 Desktop input](001-desktop-input.md) | Shortcut and configurable IME workflows, target ownership |
| [002 Speech pipeline](002-speech-pipeline.md) | Capture, streaming revisions and correction |
| [003 Memory sync](003-memory-sync.md) | Approved memory, offline outbox, self-hosted encrypted relay, hosted account authorization |
| [004 Contracts](004-contracts.md) | Portable events, cancellation, capabilities and errors |
| [005 Privacy](005-privacy.md) | Storage, consent, keys and network boundaries |
| [006 Models and evaluation](006-models.md) | Model provenance and measurable release gates |
| [007 IME and candidate ranking](007-ime-ranking.md) | IMK/TSF composition, local candidate scoring |
| [008 Evaluation](008-evaluation.md) | pnpm/Make automation, real OS E2E, streaming benchmarks and ablations |
| [009 Contextual rewriting](009-contextual-rewrite.md) | Optional local scene-aware rewrite candidates, preservation and evaluation gates |

## Product choices

Both input modes are selectable by the user. Self-hosting is the sync priority. macOS on Apple silicon and Windows x64 are P0; Intel macOS is unsupported for future releases ([ADR 021](../docs/decisions/021-apple-silicon-only-macos.md)); iOS and Android are P2 future targets. macOS IME work includes offline Simplified Chinese full-Pinyin typing, candidate selection/editing and encrypted learning alongside voice composition ([ADR 017](../docs/decisions/017-offline-pinyin-keyboard.md)); implementation and native acceptance remain required.

Jev-style local candidate ranking is explicitly requested. [Research](../docs/research/on-device-speech.md) identifies TypeSafe Jev and independent local implementations. Evaluate Laya Multilingual Choice first; the adapter contract remains model-independent.

## Traceability

Each acceptance scenario has a stable ID. New tests and task handoffs cite these IDs. An unchecked task is not an implemented feature. Changes to privacy defaults, input ownership or wire compatibility require an ADR and updated acceptance criteria.
