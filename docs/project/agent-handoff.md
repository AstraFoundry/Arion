# Agent handoff

Read [AGENTS.md](../../AGENTS.md), [architecture](architecture.md), [repository](repository.md), and the relevant [spec](../../specs/README.md) first. Claim a task in an ignored `.workspace/todo/` file, record touched paths, then report implementation, checks and remaining gaps. No dates or duration estimates.

## Implemented foundation

- Research reports and sanitized installed-artifact identity; static coverage/limits recorded.
- MyGo v0.4.6 native module and minimal honest status window; centralized English copy.
- Portable core state transitions, candidate-set permutation validation, DTOs and adapter ports.
- Actual docs/package boundary checks, core tests, pnpm workspace/frozen lockfile and desktop build CI.
- Executed macOS production-window E2E; configured Windows native E2E.
- Evaluation tooling with 21 passing metric/driver/stream tests, factorial/streaming plans and 16 concrete product scenarios; no production inference/speech scores.

This is not a functional speech input product. The following tasks remain open.

## Dependency-ordered work

| Task | Priority | Depends on | Ownership | Concrete completion gate |
| --- | --- | --- | --- | --- |
| EVAL-01 | P0 | CORE-01 and OS/model adapters | tooling and per-OS driver | Implement audited persistent live inference/native-scenario drivers, model hashes, real preview/endpoint traces and actual target evidence; EVAL-A02/A03/A05, STR-A01–A04 |
| DATA-01 | P0 | Dataset access/annotation review | benchmarks and data conversion | Export pinned LibriSpeech/AISHELL fixtures with original splits, timestamp/turn review and consented product data; EVAL-A04, STR-A05; optional full Earnings-22 and licensed SEAME |
| CORE-01 | P0 | Foundation | core application/domain | Implement session/revision reducer, stale callback invalidation and exact-once commit coordination; CTR-A04/A06, DSK-A04 |
| MAC-01 | P0 | CORE-01 | desktop infrastructure/macOS | Mic permissions/capture, bounded PCM and toggle hotkey, AX safe context/insertion harness; DSK-A01/A03/A07, SPH-A04 |
| WIN-01 | P0 | CORE-01 | desktop infrastructure/Windows | WASAPI, native hotkey lifecycle, UIA safe target/insertion harness on real Windows; same gates as MAC-01 |
| ASR-01 | P0 | CORE-01 | desktop inference adapter, models | Pin/evaluate sherpa streaming checkpoints; local Mandarin/English fixtures and offline operation; SPH-A01/A05/A06, MOD-A01/A02 |
| RNK-01 | P0 | CORE-01, actual candidate producer | local ranker adapter, models | Evaluate Laya Multilingual Choice vs source order/vocabulary baseline and Qwen pair scorer; typed ONNX/tokenizer/reference parity; RNK-A01/A03/A04, MOD-A03/A06 |
| IMK-01 | P0 | CORE-01, MAC-01 | macOS IMK host and packaging | Voice composition/candidate harness, owned ranges, lifecycle, install/remove; IME-A01–A05 on real macOS |
| TSF-01 | P0 | CORE-01, WIN-01 | Windows native COM host and packaging | Edit-session/thread locks, architecture registration, candidates, install/remove; IME-A01–A05 on real Windows |
| COR-01 | P0 | ASR-01 | correction adapter/core policy | Conservative rule correction, optional bounded local model; compare harmful edits/names/numbers; SPH-A02/A03/A08 |
| UI-01 | P0 | CORE-01, OS capability harness | FSD UI and copy | Recording/review/error/mode selection, permission guidance, candidate selection and accessibility; DSK-A02/A06, RNK-A02 |
| MEM-01 | P1 | CORE-01, crypto ADR | local memory/storage | Approved-record review/export/delete, encrypted SQLite values, migrations and atomic outbox; MEM-A01/A04, PRV-A02/A06 |
| SYNC-01 | P1 | MEM-01, crypto/pairing ADR | self-hosted relay module | Concrete v1 wire fixtures, device enrollment, ciphertext push/pull, sibling conflict/tombstone/revocation tests; MEM-A02/A03/A05–A08 |
| REL-01 | P1 | Input/ASR/rank/correction/UI gates | deployment/CI | Signed native packages, IME registration rollback, verified model packaging and compatibility matrix on both OSes |
| MOBILE-01 | P2 | Desktop contract stabilization | future apps | Native keyboard constraints research only; no mobile app in initial delivery |

RNK-01 cannot fabricate n-best from a recognizer that exposes a single hypothesis. A dictionary/phonetic candidate source must declare its provenance and scope. IMK/TSF can be prototyped against deterministic synthetic candidates while ASR work progresses, but cannot be reported as integrated speech until wired and measured.

## Decisions needed at the appropriate task

- Production ASR/ranking/correction checkpoints and native conversion artifacts after measured hardware/language/license gates.
- Vetted crypto provider, enrollment/recovery UX and key epochs before MEM-01/SYNC-01 persist or exchange real memory.
- Local SQLite driver and encryption boundary before persistence migration code.
- Native IMK/TSF host language and local authenticated IPC framing after minimal OS harness proves lifecycle/ABI.
- App identifier/signing identities, minimum OS/RAM and update trust before public packages. Current `local.arion.desktop` is a development identifier.

These are bounded open decisions, not permission prompts for already-authorized research or reversible scaffolding. Continue independent work, use evidence, and ask only when a choice changes user-visible scope or trust.

## Handoff format

Task ID; changed paths; requirements/acceptance IDs satisfied; exact checks and OS/hardware; model/protocol revisions; observed behavior vs unverified claims; unresolved next dependency. Update docs and task status in the same implementation change. Do not mark a task complete on Linux-only tests when its gate requires macOS/Windows host behavior.
