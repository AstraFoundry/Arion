# Agent handoff

Read [AGENTS.md](../../AGENTS.md), [architecture](architecture.md), [repository](repository.md), and the relevant [spec](../../specs/README.md) first. Claim a task in an ignored `.workspace/todo/` file, record touched paths, then report implementation, checks and remaining gaps. No dates or duration estimates.

## Implemented

- **CORE-01:** session aggregate (monotonic revisions, exactly-once commit claim, terminal immutability), stable error codes, dictation coordinator with generations, bounded queues, cancellation, acoustic finalization, correction and single commit; mode routing for shortcut and system IME sessions. Race-tested (CTR-A01/A02/A04/A06, DSK-A02/A04/A06/A07, SPH-A01/A02/A04/A07, IME-A02/A03 at core level).
- **MAC-01:** microphone consent and capture (miniaudio/CoreAudio, 16 kHz mono, persisted software input gain, device loss as discontinuity), Accessibility consent, target capture with secure-field detection, one-route insertion ([ADR 003](../decisions/003-insertion-routes.md)), global toggle shortcut with Escape cancellation, sleep/lock cancellation.
- **ASR-01 / model provisioning:** sherpa-onnx streaming and offline adapters, pinned catalog with member digests, verified resumable downloads with atomic publish, evaluation-driven automatic bundle selection ([ADR 004](../decisions/004-automatic-model-selection.md)).
- **COR-01 (rules):** `arion-rules-v3` conservative corrections (spacing, recognizer casing, standalone fillers, approved vocabulary), preserving quoted spelled letters and leaving repeated first-person I for polishing, keeping raw and corrected text separate.
- **LRW-01 (live rewriting, macOS Apple silicon):** edit-while-speaking pipeline per [ADR 011](../decisions/011-live-local-rewriting.md): segment cuts at pauses, latest-wins live previews of stable text, validated automatic final rewrite with the original kept, Qwen3 1.7B in the isolated `arion-llm` GenAI worker with KV prefix reuse, pinned model provisioning, a polish switch beside recognition in onboarding and Settings › Model, silent hf-mirror fallback and "Copy original". LRW-A01–A05 evidence in [the report](../../benchmarks/results/live-rewrite-v1/README.md); LRW-A06 (blind review) open. Independent `speech-spans-v1` cleanup now precedes organization and survives explicit organization failures; see [the alignment benchmark](../../benchmarks/results/disfluency-alignment-v2/README.md).
- **UI-01:** menu bar icon and popover panel, click-through overlay pill with spring motion, settings (permissions, shortcut recorder, input mode, insertion, microphone device, gain with suggested distances and meter, model, vocabulary, about with updates, local diagnostic ZIP export, privacy and project links), first-run onboarding with permission gates and model download; headless view tests and native window E2E; persisted English/Simplified Chinese interface language with live switching, typed catalogs and localized macOS permission resources ([i18n](../frontend/i18n.md), DSK-A09).
- **IMK-01:** `ArionInput.app` InputMethodKit host with marked-text composition over local IPC ([ADR 006](../decisions/006-macos-input-method-host.md)); explicit install/remove in settings.
- **MEM-01 (local):** approved vocabulary encrypted with AES-256-GCM, key in the Keychain/DPAPI ([ADR 005](../decisions/005-local-encryption.md)); no desktop outbox yet.
- **SYNC-01 / SYNC-08 / SYNC-09:** `services/sync` ciphertext relay with `noop` and `oidc` gates, device signatures, push/pull/snapshot/ack, revocation and `POST /v1/security-events` ([ADR 015](../decisions/015-hosted-relay-account-authorization.md)). MEM-A13–A16 are covered by unit tests against an in-memory port double and a local JWKS. `make sync-integration` is the PostgreSQL check and is not part of the default test run. Desktop enrollment remains open.
- **SYNC-10 (service):** desktop account sign-in ([account sign-in](account-signin.md)): RFC 8252 loopback, PKCE S256, refresh token in the OS credential store, waitlist join, and sign-out that keeps local keys and memory (MEM-A17). The two-path onboarding step and Settings › Account are built on it ([views](account-signin.md#views)); they have not been run in a native window on macOS or Windows. Native Keychain and Credential Manager behavior has not been executed on those operating systems.
- **MEM-02 (edit learning, macOS):** corrections of inserted text are learned on the device, shown with an undo, recalled by a one-pass index and, bounded, in the rewrite prompt ([ADR 014](../decisions/014-edit-learning.md), [edit-learning-v1](../../benchmarks/results/edit-learning-v1/README.md)). Windows does not read fields, so it does not learn yet.
- **EVAL-01 / DATA-01 (partial):** production-pipeline benchmark driver, corpus builder, model-comparison experiment and catalog recording; macOS product scenario driver with an owned editor fixture.
- **WIN-01 (code):** WASAPI capture through the same miniaudio adapter, SendInput Unicode insertion with foreground/focus revalidation, guarded clipboard paste, DPAPI key storage. Native CI has exercised window lifecycle, owned Notepad insertion and the hold-key hook; the complete packaged microphone/recognition path remains unverified ([validation](../quality/validation.md)).

Verification results and their limits are recorded in [validation](../quality/validation.md).

## Open work

| Task | Priority | Depends on | Concrete completion gate |
| --- | --- | --- | --- |
| WIN-01 runtime | P0 | Windows host | Run native window E2E and product scenarios on real Windows; UI Automation value verification for insertion |
| TSF-01 | P0 | WIN-01 | TSF text service with edit sessions, registration/removal, IME-A01–A05 on real Windows |
| IMK-01 acceptance | P0 | A test profile or VM | IME-A01 install/select/remove and IME-A04/A05 host edits measured on a real input-source list |
| RNK-01 | P0 | A candidate producer exposing real alternatives | Laya Choice vs source order; the current recognizers expose one hypothesis, so no ranking is shown |
| COR-01 (model) | P1 | Correction evaluation fixtures | Optional bounded local LLM correction with harmful-edit and intent review |
| RWT-DATA-01 | P0 | Authorized public/private sources + independent annotation | Gold-text/frozen-ASR, scene, protected intent and attack fixtures pass rewrite hash/split validation; timestamped speech cases use existing streaming annotations |
| RWT-ABI-01 | P0 | Pinned research candidates + native host | Verify reference tokenizer/logit/stop/cache parity and explicit native teardown; resolve sherpa/GenAI shared-library coexistence or specify isolated IPC |
| RWT-DRIVER-01 | P0 | RWT-DATA-01 + RWT-ABI-01 | Reviewed native benchmark driver emits real tokens/preview/lifecycle, warm/cold reset, concurrent ASR and cleanup; preserves evaluator-only labels |
| RWT-EVAL-01 | P0 | RWT-DRIVER-01 | Run gold-text and frozen-ASR strata with full ablations, concurrent-ASR evidence and blind human ratings per [009](../../specs/009-contextual-rewrite.md); select model/provider using held-out results |
| RWT-RUN-01 | P1 | RWT-EVAL-01 | macOS live pipeline integrated (LRW-01); remaining: Windows GenAI runtime pin, worker and acceptance; rendered-preview and IME-composition timing on a real host |
| LRW-REVIEW-01 | P0 | Independent raters | Blind meaning/style/hallucination review of held-out live output (LRW-A06); natural consented speech; same-fixture Typeless/Doubao comparison in an owned editor |
| STR (streaming sweep) | P1 | Reviewed timestamp annotations | Chunk/endpoint sweep with annotated onsets (STR-A01–A04) |
| MEM-01 (sync parts) | P1 | Crypto/pairing ADR | Outbox, approval provenance UI, export/delete |
| SYNC-10 UI | P1 | Desktop sync client; SYNC-07 actions | Two-path onboarding and Settings › Account are built ([views](account-signin.md#views)). A stored sign-in is detected without network when an account view shows and offers Reconnect. Remaining: device enrollment from the entitled hosted-sync row, Turn off sync and Delete server data once their actions exist, and a native run on macOS and Windows |
| SITE-01 | P1 | `@astrafoundry/design` published | `apps/site` is built ([site](../frontend/site.md), [ADR 016](../decisions/016-marketing-site.md)); remaining: replace the vendored Sunbeam/beUI copy listed in the [site README](../../apps/site/README.md#temporary-vendored-design-code) with the package |
| Hold-to-talk | P1 | Key-up adapter (event tap) | Tested native key-down/key-up adapter (DSK-02) |
| REL-01 | P1 | Signing identities | Environment-driven Developer ID signing/notarization is wired into the existing build and tagged CI; credentialed execution and Windows publisher signing remain open; controlled bundle IDs are configured; the signed updater exists ([ADR 010](../decisions/010-in-app-updates.md)) and keeps consent across updates only once Developer ID signing lands |
| MOBILE-01 | P2 | Desktop contract stabilization | Native keyboard constraints research only |

RNK-01 cannot fabricate n-best from a recognizer that exposes a single hypothesis. A dictionary/phonetic candidate source must declare its provenance and scope.

## Decisions needed at the appropriate task

- Ranking/correction checkpoints after measured gates; recognition bundles are selected by ADR 004.
- Recovery-secret enrollment stays rejected (`RECOVERY_UNSUPPORTED`) until a vetted format exists. Revocation rotates the key epoch; clients re-encrypt because the relay cannot read ciphertext.
- Local SQLite driver and encryption boundary before persistence migration code.
- Native IMK/TSF host language and local authenticated IPC framing after minimal OS harness proves lifecycle/ABI.
- App identifier/signing identities, minimum OS/RAM and update trust before public packages. The controlled identifiers are `si.astrafoundry.arion` and `si.astrafoundry.arion.inputmethod`; signing credentials and host acceptance remain open.

These are bounded open decisions, not permission prompts for already-authorized research or reversible scaffolding. Continue independent work, use evidence, and ask only when a choice changes user-visible scope or trust.

## Handoff format

Task ID; changed paths; requirements/acceptance IDs satisfied; exact checks and OS/hardware; model/protocol revisions; observed behavior vs unverified claims; unresolved next dependency. Update docs and task status in the same implementation change. Do not mark a task complete on Linux-only tests when its gate requires macOS/Windows host behavior.

## Qwen experiment handoff

Read [the paper strategy review](../research/qwen-paper-strategies.md) for PAP-00–11 dependencies and source-paper limitations. These are unexecuted research branches: establish parity and reviewed acoustic/context baselines before task-specific training; ordinary prose memory and generic writing scores do not establish faithful correction.

Read [the measured CPU report](../../benchmarks/results/qwen-pipeline-diagnostic-v1/README.md) before selecting a rewrite checkpoint. These unreviewed development results favor single-pass intent cleanup with fixed-prefix prefill during capture for further work; they do not approve a product model or establish Typeless parity. Preserve the optional native runtime decision and strict rewrite/E2E gates. Evaluate typed approved vocabulary through existing `domain.Correct` and ASR vocabulary bias separately; free-form memory/context prompting did not pass its dedicated development cases. Python workers are benchmark adapters and must not become the packaged desktop runtime by implication.
