# 007 Configurable IME and local candidate ranking

## IME integration

IME-01: On supported systems, onboarding offers optional input-method setup after Accessibility; users may defer to Keyboard settings. Both surfaces reuse explicit install, enable/retry, switch and status controls. Installation is distinct from app startup and shows immediate progress. Continue requires actual selection and completed activation; copied, registered or enabled alone is not completion. In-flight file replacement cannot be skipped; activation may be cancelled or deferred. Unsupported systems omit setup in both directions.

An explicit installation of a missing source continues into a bounded activation attempt; updating an existing host preserves prior selection without starting an unrelated setup attempt. Activation validates installed bytes, subscribes to enabled/selected-source notifications before requesting registration/enablement, queries discovery, enables parent before child, observes actual enablement, selects the child mode and verifies actual selection plus the expected executable's host connection. The attempt has a 30-second deadline; host connection is separately bounded to 15 seconds within it. Notification wakes observation, with 250 ms polling to cover coalesced/missed notifications and non-UI clients. A notification alone does not establish that Arion changed. Enablement is requested once per attempt; selection can retry within its deadline. Background refresh never registers, enables or selects. Repeat clicks cannot create competing attempts.

The UI distinguishes copied/not discovered, registered/not enabled, enabled and selected. A successful register request whose source is not yet discovered retains validated installed bytes for observation/retry. A successful enable request with a disabled source remains pending. Real copy/register/enable API errors retain transactional rollback during replacement; retrying activation never recopies or removes the installed bundle. Stage-specific discovery, enablement, selection and connection failures retain recovery actions. Native authorization stays visible without automatic Settings navigation; modern authorization success is observed, never inferred from an API return. Explicit Keyboard settings help remains available after incomplete setup.

Cancellation, deferral, leaving setup, closing its window and quitting revoke continuation; cancellation is checked immediately before queued native selection. Cancel retains files and OS authorization. Worker completion releases its busy guard; stale completion cannot overwrite a newer attempt. Install/update/remove are excluded while activation is running. The installer verifies Developer ID signature and default Data Protection identity before replacing installed bytes; activation validates the installed payload too. Unverified development payloads fail with release-package guidance. External OS changes refresh status without restarting Arion. Removal restores a usable input source and never leaves a stale registration.

IME-02: macOS uses an InputMethodKit server bundle/client composition; Windows uses a TSF text service with edit-session locks, COM apartment/lifecycle and correct architecture registration. A MyGo window is neither an IMK server nor a TSF service. Shared contracts cross the host boundary through documented authenticated/local IPC when separate processes are required.

IME-03: Active voice composition owns its host range and revision. Partial hypotheses update marked/preedit text only inside that range. Candidate selection, Enter/confirm, Escape/cancel, deactivation and focus loss follow platform semantics. No recognizer thread writes into the editor directly.

IME-UPDATE-01: Updating an installed macOS input method MUST stage and validate new bytes before interrupting the current host, cancel owned voice composition, switch away without disabling authorization, and confirm old-process exit before replacement. Registration, prior selection and a connection from the expected executable build MUST be observed before operational update success. A pending first-install authorization remains separately pending. Failure MUST stop the failed replacement before rolling files back; an unkillable replacement MUST preserve recovery copies and report failure. Desktop restarts MUST recover a refused orphaned IPC endpoint while retaining live sockets and unrelated paths. A transport failure MUST NOT claim that the OS input source is unselected.

IME-04: A host refusing composition/read/write or a secure field produces an explicit capability result. Do not switch secretly to clipboard/keystrokes. macOS also supports the offline full-Pinyin keyboard defined below; host shortcuts and English-mode typing pass through according to the OS contract.

IME-05: Candidates must support keyboard navigation, selection, accessible names and bounded display. Selected ID/revision survives a reorder only while still present; user navigation freezes automatic selection until the next explicit composition update.

## Offline macOS keyboard typing

KBD-01: The installed IMK host provides Simplified Chinese full-Pinyin conversion without the menu-bar app or network. Pin independently licensed engine/data bytes and complete schema dependencies. Missing/corrupt runtime/data is an explicit unavailable state, never simulated conversion or silent Latin substitution.

KBD-02: Keep raw input/caret, selected segments, suffix, displayed preedit and final committed text separate. Support full Pinyin, apostrophe boundaries, abbreviation and v/ü input through the declared schema. Direct `ü`/`Ü` key characters map to the schema's `v`/`V` spelling in Chinese mode; English-mode and host-shortcut pass-through preserve the original event. Candidate choice can consume a prefix and continue conversion of the remainder. Only an actual engine commit reaches the client. Ordinary punctuation and symbols commit directly without opening symbol candidates; Chinese quotes alternate opening/closing forms. Apostrophes inside Pinyin retain syllable separation, and existing candidate paging keys keep their navigation behavior during composition.

KBD-03: Provide bounded candidate pages, stable page-relative number/mouse selection and keyboard navigation. Backspace/Delete, arrows and Home/End operate on the engine's composition; partial-selection rollback must be verified. Stale candidate UI callbacks cannot choose a newer revision. Enter commits raw input according to the configured schema; Space selects the highlighted candidate; Escape cancels.

KBD-04: Chinese/English switching uses a standalone left/right Shift press released in less than 0.3 seconds; the input menu and Shift+Space remain available. Other keys or modifiers during the press, long holds, cancellation and deactivation invalidate the pending switch. Switching shows a nonactivating, click-through language badge beside the host caret for 0.8 seconds; typing, voice takeover and focus loss dismiss it. A missing caret rectangle suppresses the badge. Switching and deactivation discard unconfirmed composition. Command-modified host shortcuts relinquish ownership before pass-through. Keyboard and voice drafts never share a mutable owner; voice start, physical typing, focus changes and late socket callbacks cannot overwrite another owner's range.

KBD-05: Learn only confirmed candidate selections through an encrypted local user database with an OS-kept key. Cancellation/stale results do not learn. Restart persistence, reset/delete and encrypted backups are explicit operations. Missing keys, authentication failure and write failure do not create a new vault or plaintext storage. No raw key history or selected text enters diagnostics or account sync.

KBD-06: Ship a universal arm64+x86_64 input source with relocatable native dependencies, all notices and prebuilt data. Schema deployment/download must not run on the typing path. Actual editor behavior, offline first activation, word/sentence quality and key-to-preedit latency require host observations; unit or engine-only results are insufficient.

## Local scoring

RNK-01: Sources may be actual recognizer alternatives, a documented phonetic/lexicon engine, or approved dictionary-derived alternatives with explicit provenance. Their absence yields a one-item set; never invent ASR n-best or neural probabilities.

RNK-02: Input carries session/generation, composition revision, exact candidate IDs/text/source, bounded approved context and language hint. Output is a permutation of the same IDs plus optional finite comparable scores. It cannot generate words or alter candidate text.

RNK-03: Scoring is cancellable, entirely local and triggered only for a supported composition revision. Stale results cannot reorder newer candidates or replace committed text. If unsupported/unavailable, preserve producer order and show capability state; this is an explicit mode outcome rather than hidden inference fallback.

RNK-04: Evaluation must compare producer order and authorized vocabulary boost against a learned ranker. Embedding similarity is not automatically a homophone-aware likelihood score. Cross-encoder query/document relevance is not automatically IME quality. Jev is TypeSafe’s decision model; Laya Multilingual is the initial local Choice evaluation candidate. Its training task, typed-head/tokenizer export parity and candidate calibration need evidence before adoption.

RNK-05: Short candidate ranking must coexist with ASR without breaking audio deadlines. Ranker maximum candidates/context length, memory and timeout are configuration with benchmarked limits. Don't deploy a large generation model solely because it can assign a number.

## Acceptance

| ID | Scenario |
| --- | --- |
| IME-A01 | Install/enable/retry/select/remove on actual OS → observed discovery, parent/child enablement, selection and expected-build connection; delayed discovery/consent, denial/timeout, cancellation/defer/window close, duplicate attempts and settings recovery; retry retains installed bytes; optional onboarding gates on completed selection |
| IME-A06 | Update selected/unselected source → old host exits before promotion, prior selection retained, expected new-build connection; failed exit/handshake rolls back or preserves recovery copies; desktop restart recovers orphaned IPC without replacing a live listener |
| IME-A02 | Streaming draft revisions → owned composition only; final confirmation commits once |
| IME-A03 | Cancel/deactivate/target change → clear owned composition; late callbacks ignored |
| IME-A04 | User types/navigates during inference → host updates win and stale range never changes |
| IME-A05 | Emoji/CJK/surrogate pairs → exact adapter range conversion; no neighboring deletion |
| KBD-A01 | Offline first activation with menu-bar app stopped → real Pinyin composition/candidates/commit |
| KBD-A02 | Apostrophes, abbreviations, v/ü, mixed punctuation and unknown input → declared schema results without fabricated alternatives |
| KBD-A03 | Page/number/mouse/navigation, partial selection and suffix continuation → selected engine identity preserved |
| KBD-A04 | Mid-buffer edit, Home/End, Backspace/Delete and selected-segment rollback → engine and marked text agree |
| KBD-A05 | Short standalone Shift release, Shift chords/holds, caret badge lifecycle, Escape/mode switch/focus change and voice/keyboard takeover → unconfirmed text discarded, late writes rejected |
| KBD-A06 | CJK/emoji neighbors and UTF-8/UTF-16 caret boundaries → exact owned-range writes in actual editors |
| KBD-A07 | Confirmed learning survives restart; cancellation does not learn; reset/delete work; wrong/missing key, tamper and I/O failure fail closed without plaintext |
| KBD-A08 | Universal bundle/dependency/data/license checks and actual offline host quality/latency observations |
| RNK-A01 | Rank result adds/drops/duplicates IDs or nonfinite scores → reject result |
| RNK-A02 | User-selected candidate moved by ranking → same ID retained, no unexpected text replacement |
| RNK-A03 | Older composition result arrives late → ignored |
| RNK-A04 | Local Chinese homophone/name fixtures → measured accuracy against both baselines |
| RNK-A05 | Missing model/unsupported set → producer order remains visibly available |

Voice IMK ownership is implemented; full-Pinyin keyboard implementation and host acceptance are in progress. Keyboard candidates have a local context ranker (ADR 017): a permutation of the engine's leading candidates by the user's own confirmed (previous word, word) pairs, never promoting a shorter prefix, with engine order when its store is unavailable. On authored context homophones it lifts later-occurrence top-1 from 32% to 89% against engine order with frequency learning ([results](../benchmarks/results/imk-context-v1/README.md)); that is RNK-A04 evidence for this producer only. A deployed learned (model) candidate ranker is not claimed. See [handoff](../docs/project/agent-handoff.md).
