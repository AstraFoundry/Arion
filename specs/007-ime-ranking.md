# 007 Configurable IME and local candidate ranking

## IME integration

IME-01: Settings select shortcut mode or system IME mode. Installing/registering the input source is explicit and distinct from app startup; the user chooses it through OS controls. Removal restores a usable input source and never leaves a stale registration.

IME-02: macOS uses an InputMethodKit server bundle/client composition; Windows uses a TSF text service with edit-session locks, COM apartment/lifecycle and correct architecture registration. A MyGo window is neither an IMK server nor a TSF service. Shared contracts cross the host boundary through documented authenticated/local IPC when separate processes are required.

IME-03: Active voice composition owns its host range and revision. Partial hypotheses update marked/preedit text only inside that range. Candidate selection, Enter/confirm, Escape/cancel, deactivation and focus loss follow platform semantics. No recognizer thread writes into the editor directly.

IME-04: A host refusing composition/read/write or a secure field produces an explicit capability result. Do not switch secretly to clipboard/keystrokes. Normal nonvoice typing passes through according to the OS host contract; a complete Pinyin lexicon/conversion engine is not implied by this spec.

IME-05: Candidates must support keyboard navigation, selection, accessible names and bounded display. Selected ID/revision survives a reorder only while still present; user navigation freezes automatic selection until the next explicit composition update.

## Local scoring

RNK-01: Sources may be actual recognizer alternatives, a documented phonetic/lexicon engine, or approved dictionary-derived alternatives with explicit provenance. Their absence yields a one-item set; never invent ASR n-best or neural probabilities.

RNK-02: Input carries session/generation, composition revision, exact candidate IDs/text/source, bounded approved context and language hint. Output is a permutation of the same IDs plus optional finite comparable scores. It cannot generate words or alter candidate text.

RNK-03: Scoring is cancellable, entirely local and triggered only for a supported composition revision. Stale results cannot reorder newer candidates or replace committed text. If unsupported/unavailable, preserve producer order and show capability state; this is an explicit mode outcome rather than hidden inference fallback.

RNK-04: Evaluation must compare producer order and authorized vocabulary boost against a learned ranker. Embedding similarity is not automatically a homophone-aware likelihood score. Cross-encoder query/document relevance is not automatically IME quality. Jev is TypeSafe’s decision model; Laya Multilingual is the initial local Choice evaluation candidate. Its training task, typed-head/tokenizer export parity and candidate calibration need evidence before adoption.

RNK-05: Short candidate ranking must coexist with ASR without breaking audio deadlines. Ranker maximum candidates/context length, memory and timeout are configuration with benchmarked limits. Don't deploy a large generation model solely because it can assign a number.

## Acceptance

| ID | Scenario |
| --- | --- |
| IME-A01 | Install/select/remove IMK and TSF → registration and host lifecycle clean on actual OS |
| IME-A02 | Streaming draft revisions → owned composition only; final confirmation commits once |
| IME-A03 | Cancel/deactivate/target change → clear owned composition; late callbacks ignored |
| IME-A04 | User types/navigates during inference → host updates win and stale range never changes |
| IME-A05 | Emoji/CJK/surrogate pairs → exact adapter range conversion; no neighboring deletion |
| RNK-A01 | Rank result adds/drops/duplicates IDs or nonfinite scores → reject result |
| RNK-A02 | User-selected candidate moved by ranking → same ID retained, no unexpected text replacement |
| RNK-A03 | Older composition result arrives late → ignored |
| RNK-A04 | Local Chinese homophone/name fixtures → measured accuracy against both baselines |
| RNK-A05 | Missing model/unsupported set → producer order remains visibly available |

No runtime IME or ranker implementation is claimed by this foundation. See [handoff](../docs/project/agent-handoff.md).
