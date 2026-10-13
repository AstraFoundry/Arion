# Chatterfly scenario-aware speech rewriting

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

## Implications for Arion

Separate ASR text, conservative correction and an explicitly selected scene-aware rewrite candidate. Do not treat scene classification, fluent output or model self-reported reasoning as proof of correctness. A server's baseline/enhanced and pause/skill/guide events are workflow evidence, not a requirement to implement voice commands or a cloud service.

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
| What is Chatterfly's exact prompt, server LLM and active scene policy? | Unresolved | No server implementation or controlled runtime fixture capture was supplied |
| Does it rewrite each partial, a stable phrase or only a completed turn? | Partially answered | Orchestration supplies `asrStableText`; separate pause-processing methods exist. Exact trigger scheduling and runtime traces remain unresolved |
| Does a rich context field mean it reads or sends that content? | Partially answered | Concrete builder/serializer paths are established; schema, populated values and observed transmission remain separate evidence levels |
| Does the package's CoreML dependency prove local LLM inference? | Unresolved | No local generative checkpoint or exclusive remote route is established by a dependency name |
