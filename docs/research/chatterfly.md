# Chatterfly scenario-aware speech rewriting

For conventional keyboard typing, see the [keyboard investigation](keyboard-ime.md#chatterfly) and [installed-artifact follow-up](chatterfly-installed-keyboard.md), including model-loading gates and synthetic TextEdit runtime observations. This document's speech findings retain their original static evidence boundary.

## Selected artifact and evidence boundary

The supplied `Chatterfly_V1.0.6.13398.app` is an installer (`com.tencent.ChatterflyInstaller`), not the input-method executable. REA inventory and safe extraction selected its `Contents/Resources/ChatterFly.zip`, whose payload contains the actual `Chatterfly.app`, preference and task-manager apps, services/installation files and packaged resources. The main input-method plist names `com.tencent.inputmethod.chatterfly`, version `1.0.6.13398`, `YuanbaoInputController` and an InputMethodKit connection. The universal main executable contains x86_64 and arm64 slices. The arm64 analysis copy was verified byte-for-byte against the original slice.

The [sanitized ledger](evidence/chatterfly-rewrite.json) records installer/archive/binary hashes, exact REA inventory/extraction/Mach-O Evidence IDs, derived-slice identity and original byte offsets. CF identifiers denote ordinary local static observations. Ghidra `analyze_function` exceeded its 1,200,000 ms provider timeout before returning a dossier; queued native requests were cancelled. The focused function findings below instead use Apple LLVM 21.0.0 Mach-O disassembly, native symbols, selector/constant references and independently byte-verified instruction locations. They are not REA decompilation results. Full inventories and recovered instructions stay in ignored `.workspace/research/chatterfly/`. Packaged names such as Sogou/Yuanbao describe implementation heritage; they do not establish a particular LLM vendor or checkpoint.

Only supplied application-package bytes were inspected. The installer and payload were not launched; no user history database, microphone, account, credential, private endpoint or live network capture was accessed. Extracted regular files do not preserve archive symlinks. Client evidence cannot reveal the server's exact prompts, weights, routing thresholds, retention policy or observed latency.

## Static workflow surfaces

| Stage | Evidence | What is established and what remains unknown |
| --- | --- | --- |
| Native input host | Payload plist and REA Mach-O inventory | IMK controller/connection plus speech/AVFoundation and network framework dependencies exist; their availability does not prove any mode ran |
| Capture and recognition | `CF-ipc` | Start/stop/retry IPC names and separate ASR/LLM config fields exist; actual audio transport, installed models and ASR default are not established |
| Context snapshot | `CF-context-schema` | App, field and surrounding/selected-text schema, input environment, history/chat context and memory surfaces exist; field population and authorization must be checked separately |
| Scene classification | `CF-scene-and-rewrite` | A scene-result type has route/reason/path/confidence/label surfaces; thresholds and server classifier are unknown |
| Rewriting | `CF-scene-and-rewrite` | Separate baseline and enhanced outputs exist; an enhanced result cannot be assumed to preserve meaning |
| Stream handling | `CF-stream-control` | SSE-specific dispatchers distinguish scene, CMode, pause, skill, guide and performance/result handling; a delta callback is not proof of incremental ASR rewriting |
| Review/history | `CF-history-schema` | Packaged storage schema separates ASR from LLM config/results and references audio; active recording settings and actual storage/encryption behavior remain unknown |

The native seeds are `SGMacSpeechContextBuilder`, `SGMacSpeechLLMProcessor`, `SGMacSpeechService`, `SGSpeechPostProcessorFactory`, `SGSpeechPostProcessorSSE` and the independent ASR/input-delivery components. These names identify focused analysis entry points, not a proven call graph.

## Native instruction and call-path findings

Fourteen bounded function bodies are anchored by entry address, next text symbol, original instruction bytes and function-byte SHA-256 in the ledger. Calls and constants are observations; reconstructed data/control flow is a static inference. Server execution remains outside the client analysis boundary.

| Function evidence | Recovered client behavior |
| --- | --- |
| `CF-asm-fast-fields` / `CF-asm-context-builder` | The fast builder queries `IsSecureEventInputEnabled`, sets the password flag and branches out before clipboard/text acquisition on that path. Other paths call clipboard acquisition, read up to 200 native string units before the cursor, request a bounded AX snapshot for after/selected text, and distinguish confirmed no selection from unavailable AX before considering cached selection. Enhanced recognition gates recent input-history lookup for the current app. This is not a proof that all unknown/password fields are protected or that the user enabled context collection. |
| `CF-asm-context-dict` / input-content / input-env | The serializer conditionally emits app/platform, before/after/selected text, environment, clipboard, input history, scene context, screen content, memory and event/timestamp fields. The inspected builder does not establish screen scraping, chat-history acquisition or memory population merely because the serializer supports those fields. |
| `CF-asm-llm-orchestrator` | Cancels the old post-process, creates a processor for mode/session, attaches the context object and stored `llmConfigDict`, extended punctuation policy, short-utterance policy and callbacks, then passes `asrStableText` into post-processing. This is a text-conditioned stage separate from the acoustic recognizer. |
| `CF-asm-factory` | Mode value 1 selects `SGSpeechPostProcessorSSE`; another branch selects `SGSpeechPostProcessor`. Both ship; the active configuration is unknown. |
| `CF-asm-payload` | The request dictionary includes `request_id`, `asr_text`, `session_id`, `enable_pause_detect` and `vp_trigger`; optional fields include context, extended settings, guide context and user/debug fields. A short-utterance flag supplies `audit_only`. Exact server instructions are not recovered from this request. |
| `CF-asm-sse-network` | Constructs `SGEventSource` with URL, headers, timeout, method and body, registers message/error/ready-state callbacks and timeout, sets reconnect policy and calls `open`. This establishes a remote-capable post-processing path, not an observed upload or exclusive inference route. |
| `CF-asm-sse-dispatch` | Optionally decodes an encrypted frame, parses JSON and dispatches typed `cmode`, `scene`, `step_info`, `pause`, `skill`, `guide`, `perf` and error packets; reads sequence, finish and delta separately. |
| `CF-asm-scene-packet` / `CF-asm-cmode` | A finished scene packet populates route/reason/path/confidence/label. CMode accumulates deltas, calls `onDelta` for provisional output and stores completed text/tips/raw tail. Scene selection and text generation have distinct response structures; the server's classifier/prompt is unknown. |
| `CF-asm-do-post` / `CF-asm-live-result` | The service starts LLM processing from session data, stores the response, handles errors/result codes, sets `enhancedText`, calls a history helper and hands the result to `commitFinalSpeechText`. That delivery helper's exact IMK/AX/paste strategy is outside this function set. |

The resulting bounded reconstruction is **ASR stable text + context/config → post-process payload → optional SSE transport → scene/rewriting deltas and final result → enhanced-text session state → input-delivery helper**. Packet transport is streaming; this evidence does not establish that every ASR partial triggers a new LLM job. Pause processing has separate shipped methods and flags; actual stop/pause scheduling and perceived latency require controlled runtime traces.

The [disfluency follow-up](competitor-disfluency.md#chatterfly-stable-asr-text-enters-the-post-processor) traces the short-input flag, debug toggle registered false, result-selection branches and original/rewrite panel switch. It finds no lexical-retention ratio in those bounded result functions and no recovered server filler-removal prompt. The live-ASR builder explicitly selects `zh-cmn-Hans-CN`; multilingual cleanup remains unmeasured.

## Scene recognition and switching

A second pass (`CF-asm-sel-*`, `CF-scene-catalog`, `CF-scene-negative-setters` in the ledger) followed the scene path end to end in the same arm64 slice, the preference app and the full symbolized disassembly. It answers where scenes come from, which client signals feed them and what the client does with the result.

| Question | Finding | Evidence |
| --- | --- | --- |
| Which scenes does the product present? | Four usage scenes: email polish (邮件润色), work-message reply (回复工作消息, numbered points and tone), vibe-coding prompt (developer instruction with terms, constraints and acceptance criteria) and all-scene AI typing. User-enabled Skills are "chosen by the current input scene" | `CF-scene-catalog` (product copy, not server behavior) |
| Who classifies the scene? | The server. The client has no scene model, label table or app-to-scene map; the SSE stream returns `scene{route, reason, path, confidence, label}` before the rewrite, and the response times `scene_ms` and `cmode_ms` separately | `CF-asm-sel-response-parse`, `CF-asm-scene-packet` |
| What does the client send for it? | `platform`, `app{bundle_id, display_name}`, `input_env{type}`, `input_content{before_cursor ≤200, after_cursor, selected_text}`, `clipboard`, per-app `input_history` (only with enhanced recognition), `event_id`, `timestamp` | `CF-asm-sel-default-context`, `CF-asm-fast-fields`, `CF-asm-context-builder` |
| How is the app identified? | `bundle_id` of the frontmost app; `display_name` is its frontmost on-screen window title from CGWindowList (owner PID, layer 0, non-empty name), else the app name. In a browser that title is normally the page title; no URL is read | `CF-asm-sel-app-info`, `CF-asm-sel-window-title`, `CF-asm-sel-app-dict` |
| How is the field identified? | AX subrole `AXSearchField` → `search`; role `AXTextField`/`AXTextArea` → `text`; else empty. Secure input stops the builder before clipboard or text is read; the password flag is not sent | `CF-asm-sel-input-type`, `CF-asm-sel-input-env-dict` |
| Are screen text, chat history or memory sent? | Not by this build: nothing calls their setters, although the serializer supports them | `CF-scene-negative-setters` (indirect writes not excluded) |
| What does the client do with the scene? | Nothing that changes text. It attaches the scene path to the voice-edit segment so later user edits are attributed to it and reports it as `scene_path` telemetry. The delivered text is the server's enhanced output | `CF-asm-sel-edit-scene-path`, `CF-asm-sel-beacon-scene` |
| Any local page detection? | Only inherited typing-IME heuristics: browser bundle tables plus caret geometry for the address bar, and one site detected from window titles. The browser sub-scene telemetry field (`llq_*` categories) is hard-coded to 0 | `CF-asm-sel-address-bar`, `CF-asm-sel-site-title`, `CF-asm-sel-llq-scene` |

Correction: the earlier record `CF-asm-context-input-env` said `is_password` is serialized. The serializer writes only `type`, and the string does not occur in the binary.

The reconstructed flow is **frontmost app (bundle ID, window title) + field type + cursor text + clipboard + per-app history → server scene stage (label, route, confidence) → server rewrite stage for that scene, optionally a user-enabled Skill → enhanced text**. The server's classifier, its label vocabulary, thresholds and prompts are not in the client and remain unknown. No live session was run, so scene choices for particular apps were not observed.

## Implications for Arion

Separate ASR text, conservative correction and an explicitly selected scene-aware rewrite candidate. Do not treat scene classification, fluent output or model self-reported reasoning as proof of correctness. A server's baseline/enhanced and pause/skill/guide events are workflow evidence, not a requirement to implement voice commands or a cloud service.

Arion's [scene selection](../../specs/009-contextual-rewrite.md#scene-selection) keeps Chatterfly's signal order but decides locally with a deterministic table: field type, then the web page host, then the app's bundle ID. It reads the host from the page's accessibility URL instead of the window title, which carries less content (no subject lines or document names), and it never reads cursor text, clipboard or history for this purpose. The scenes mirror the product catalog (email, work, prompt, plain chat) plus search from the field type. Chatterfly's product copy advertises a vibe-coding scene with constraints and acceptance criteria; this does not establish actual server additions or formatting thresholds; Arion's prompt scene only lays out what was said.

Arion's proposed local path is bounded consent-gated context + approved memory + immutable source text + declared scene → versioned prompt and tokenizer → local prefill/decode → preservation checks → owned preview → existing final commit gate. Foreign field text remains data rather than an instruction source. Clipboard, full-screen scraping and unapproved cross-app history are not introduced by this research. See [proposed ADR 008](../decisions/008-contextual-rewrite-evaluation.md) and [009](../../specs/009-contextual-rewrite.md).

A streamed token is provisional. A newer ASR revision, cancellation, mode change, host edit or focus change invalidates pending rewrite results and target ownership. Evaluate latency to the first actual preview and to validated final text separately. Preserve an identified source/rule result when a candidate is rejected; do not silently change model, provider or network policy.

## Comparison and unresolved questions

[Typeless/Doubao focused research](competitor-rewrite.md) confirms separate refinement/organization workflows with different client surfaces and reliability mechanisms. Chatterfly exposes a richer scene/context/result schema in this selected package. This is an artifact comparison, not a ranking of product quality or measured privacy.

| Question | Status | Current evidence scope |
| --- | --- | --- |
| Which supplied binary implements the input host? | Answered | Installer → authenticated archive payload → IMK app → byte-verified arm64 slice |
| What is the client-side rewrite path? | Partially answered | Fourteen focused instruction records cover context, orchestration, payload, SSE packets and result handoff; execution and delivery-helper internals remain unresolved |
| What original/export models can Arion run locally? | Partially answered | [Pinned HF/ONNX candidates](onnx-rewrite-models.md); metadata and runtime source support only, native loading/parity pending |
| How should rewrite quality and latency be tested? | Answered as a design | [Executable evaluator and blind review chain](rewrite-evaluation.md); live LLM/native drivers and reviewed fixtures are required to obtain measurements |
| Where are scenes decided and from which signals? | Answered | Server-side; client sends bundle ID, window title, field type, cursor text, clipboard and per-app history; the scene result only feeds edit attribution and telemetry |
| What is Chatterfly's exact prompt, server LLM and active scene policy? | Unresolved | No server implementation or controlled runtime fixture capture was supplied |
| Does it rewrite each partial, a stable phrase or only a completed turn? | Partially answered | Orchestration supplies `asrStableText`; separate pause-processing methods exist. Exact trigger scheduling and runtime traces remain unresolved |
| Does a rich context field mean it reads or sends that content? | Partially answered | Concrete builder/serializer paths are established; schema, populated values and observed transmission remain separate evidence levels |
| Does the package's CoreML dependency prove local LLM inference? | Unresolved | No local generative checkpoint or exclusive remote route is established by a dependency name |

## Conservative alignment boundary

Arion's natural-layout-v1 policy retains the stage separation and original/result distinction documented for Doubao and Chatterfly. It defaults to natural prose even in work/coding scenes and adds list formatting only for explicit list intent. This corrects Arion's earlier demonstrations that split ordinary requests into bullets and inserted category labels. It is an Arion product decision, not a recovered competitor prompt or a measured claim of behavioral parity. Exact server prompts, enumeration thresholds and current-version output behavior remain unverified. See [local rewrite configuration](../operations/local-rewrite.md).
