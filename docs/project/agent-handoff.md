# Agent handoff

Read [AGENTS.md](../../AGENTS.md), [architecture](architecture.md), [repository](repository.md), and the relevant [spec](../../specs/README.md) first. Claim a task in an ignored `.workspace/todo/` file, record touched paths, then report implementation, checks and remaining gaps. No dates or duration estimates.

## Implemented

- **CORE-01:** session aggregate (monotonic revisions, exactly-once commit claim, terminal immutability), stable error codes, dictation coordinator with generations, bounded queues, cancellation, acoustic finalization, correction and single commit; mode routing for shortcut and system IME sessions. Race-tested (CTR-A01/A02/A04/A06, DSK-A02/A04/A06/A07, SPH-A01/A02/A04/A07, IME-A02/A03 at core level).
- **MAC-01:** microphone consent and capture (miniaudio/CoreAudio, 16 kHz mono, device loss as discontinuity), Accessibility consent, target capture with secure-field detection, one-route insertion ([ADR 003](../decisions/003-insertion-routes.md)), global toggle shortcut with Escape cancellation, sleep/lock cancellation.
- **ASR-01 / model provisioning:** sherpa-onnx streaming and offline adapters, pinned catalog with member digests, verified resumable downloads with atomic publish, evaluation-driven automatic bundle selection ([ADR 004](../decisions/004-automatic-model-selection.md)).
- **COR-01 (rules):** `arion-rules-v1` conservative corrections (spacing, recognizer casing, standalone fillers, approved vocabulary) keeping raw and corrected text separate. No local LLM correction.
- **UI-01:** menu bar icon and popover panel, click-through overlay pill with spring motion, settings (permissions, shortcut recorder, input mode, insertion, microphone meter, model, vocabulary, privacy, about), first-run onboarding with permission gates and model download; headless view tests and native window E2E; persisted English/Simplified Chinese interface language with live switching, typed catalogs and localized macOS permission resources ([i18n](../frontend/i18n.md), DSK-A09).
- **IMK-01:** `ArionInput.app` InputMethodKit host with marked-text composition over local IPC ([ADR 006](../decisions/006-macos-input-method-host.md)); explicit install/remove in settings.
- **MEM-01 (local):** approved vocabulary encrypted with AES-256-GCM, key in the Keychain/DPAPI ([ADR 005](../decisions/005-local-encryption.md)); no outbox or sync yet.
- **EVAL-01 / DATA-01 (partial):** production-pipeline benchmark driver, corpus builder, model-comparison experiment and catalog recording; macOS product scenario driver with an owned editor fixture.
- **WIN-01 (code):** WASAPI capture through the same miniaudio adapter, SendInput Unicode insertion with foreground/focus revalidation, guarded clipboard paste, DPAPI key storage. Compiled for windows/amd64; never run on Windows.

Verification results and their limits are recorded in [validation](../quality/validation.md).

## Open work

| Task | Priority | Depends on | Concrete completion gate |
| --- | --- | --- | --- |
| WIN-01 runtime | P0 | Windows host | Run native window E2E and product scenarios on real Windows; UI Automation value verification for insertion |
| TSF-01 | P0 | WIN-01 | TSF text service with edit sessions, registration/removal, IME-A01–A05 on real Windows |
| IMK-01 acceptance | P0 | A test profile or VM | IME-A01 install/select/remove and IME-A04/A05 host edits measured on a real input-source list |
| RNK-01 | P0 | A candidate producer exposing real alternatives | Laya Choice vs source order; the current recognizers expose one hypothesis, so no ranking is shown |
| COR-01 (model) | P1 | Correction evaluation fixtures | Optional bounded local LLM correction with harmful-edit and intent review |
| STR (streaming sweep) | P1 | Reviewed timestamp annotations | Chunk/endpoint sweep with annotated onsets (STR-A01–A04) |
| MEM-01 (sync parts) | P1 | Crypto/pairing ADR | Outbox, approval provenance UI, export/delete |
| SYNC-01 | P1 | MEM-01 | Self-hosted relay per 003 |
| Hold-to-talk | P1 | Key-up adapter (event tap) | Tested native key-down/key-up adapter (DSK-02) |
| REL-01 | P1 | Signing identities | Developer ID signing/notarization, controlled bundle ID, updater |
| MOBILE-01 | P2 | Desktop contract stabilization | Native keyboard constraints research only |

RNK-01 cannot fabricate n-best from a recognizer that exposes a single hypothesis. A dictionary/phonetic candidate source must declare its provenance and scope.

## Decisions needed at the appropriate task

- Ranking/correction checkpoints after measured gates; recognition bundles are selected by ADR 004.
- Vetted crypto provider, enrollment/recovery UX and key epochs before MEM-01/SYNC-01 persist or exchange real memory.
- Local SQLite driver and encryption boundary before persistence migration code.
- Native IMK/TSF host language and local authenticated IPC framing after minimal OS harness proves lifecycle/ABI.
- App identifier/signing identities, minimum OS/RAM and update trust before public packages. Current `local.arion.desktop` is a development identifier.

These are bounded open decisions, not permission prompts for already-authorized research or reversible scaffolding. Continue independent work, use evidence, and ask only when a choice changes user-visible scope or trust.

## Handoff format

Task ID; changed paths; requirements/acceptance IDs satisfied; exact checks and OS/hardware; model/protocol revisions; observed behavior vs unverified claims; unresolved next dependency. Update docs and task status in the same implementation change. Do not mark a task complete on Linux-only tests when its gate requires macOS/Windows host behavior.
