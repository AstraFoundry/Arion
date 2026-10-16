# 009 Contextual local rewriting

## Status and scope

This spec defines an optional local LLM stage and its evaluation boundary. The production desktop implements the [live rewriting pipeline](#live-rewriting-pipeline) of [ADR 011](../docs/decisions/011-live-local-rewriting.md) on macOS (Apple silicon): previews while speaking and one validated, automatically inserted rewrite after stop, in an isolated ONNX Runtime GenAI worker. It also implements [scene selection](#scene-selection) from the focused field, web site and app. Context text capture, Windows and the strict [rewrite evaluator](../docs/research/rewrite-evaluation.md) protocol remain unimplemented. Competitor artifact findings do not establish that Arion or a competitor executes this pipeline.

Rewriting is separate from acoustic recognition, conservative correction and Laya candidate ranking. A ranker permutes existing producer candidates; a rewriter can generate a new text proposal. Baseline dictation remains available with rewriting disabled. macOS shortcut, Windows shortcut and optional IMK/TSF hosts share the portable request/result contract; host integration must be verified on the actual OS.

## Product pipeline

1. Obtain revisioned ASR text and conservative correction. Preserve their original text independently of any rewrite.
2. Capture a bounded context snapshot and approved memory under [rewrite context](#rewrite-context). Select a scenario from explicit user choice and permitted application hints. Unknown/secure targets deny context acquisition before reading their values. Clipboard remains unread; other sources need their own switch.
3. Build a versioned prompt with trusted rewrite instructions separated from untrusted transcript/context/memory. Pin template, tokenizer, checkpoint, ONNX graph, conversion, quantization and execution provider. User text may describe commands; it never grants tool execution, networking or target writes.
4. Run bounded non-thinking generation locally. Keep token streams and provisional rewrite previews separate from recognized drafts and owned IME composition. Partial-source speculation is optional: every new source revision cancels it, and it cannot become the final candidate by mistake.
5. Validate the final proposal against the selected mode, output budget and protected content. Verbatim mode forbids changes to code, punctuation, capitalization and spacing. Names, numbers, URLs, negation, modality and speaker intent need preservation; string checks alone cannot prove semantic preservation.
6. Deliver the validated proposal automatically ([ADR 011](../docs/decisions/011-live-local-rewriting.md)) and keep the recognized original available to copy. A generated instruction, reasoning block, skill invocation or model refusal is never inserted as ordinary dictated text. Rejection, timeout, missing model and unsupported runtime are explicit organization outcomes recorded with the result. They retain the completed local speech-cleanup result; raw recognition and rule-corrected source remain separate.
7. Commit only through the existing once-only target capability after source identity, focus and mode checks. Source revision, cancellation, focus change or mode switch invalidates the proposal and owned composition. A late token, preview or callback cannot write to a target.

## Live rewriting pipeline

- LRW-01: Previews rewrite only text of closed recognizer segments (stable text). The open segment is displayed as recognized, after the preview.
- LRW-02: A segment's audio is cut inside the pause that closed it, not at the moment the endpoint was reported. Per-segment finalization is a bundle choice; without it the finalizer decodes the whole utterance once after stop.
- LRW-03: Latest wins. A newly closed segment cancels the running preview; a preview computed for older stable text is never displayed. Previews never commit and failures of previews never end the session.
- LRW-04: Stop cancels previews before the final rewrite. Cancellation, mode change or a new session discards the final rewrite; the worker releases the session's KV state when the session ends.
- LRW-05: Every proposal passes `domain.ValidateRewrite` before display or insertion: not truncated or empty, no chat markup or echo, bounded length change, no new numbers or unsupported content, no dropped negation or recognized uncertainty, the final value of a self-correction kept, and no large deletion without a repair cue. Deletion checks use the independently cleaned source, preserving quoted spans and the original checks for all-padding input. A correction explicitly replacing uncertainty with certainty releases only the immediately superseded clause's hedge. These are fidelity proxies, not semantic proof ([diagnostic](../docs/research/rewrite-reliability.md)). Live previews keep the open source's trailing punctuation.
- LRW-06: The model runs in a separate worker process that never loads the speech runtime. Prompt instructions and demonstrations are a fixed, versioned prefix; dictated text and approved terms are data in the last user turn, and chat control markers in dictated text are rejected. Reusing KV state must not change greedy output.
- LRW-07: Model weights download only on the user's choice, as SHA-256-pinned files published atomically; the runtime ships signed inside the app. Platforms without a pinned runtime report the capability as unsupported.
- LRW-08: Polishing enabled by the user stays enabled while its worker is loading or unavailable. If no worker can serve the final request, record `unavailable` and deliver the completed local cleanup; never label that session as polishing disabled.

- LRW-09: With polishing enabled, `domain.CleanSpeech` produces versioned deletion spans in the unchanged rule-corrected source before organization. Spans have UTF-8 byte boundaries, known kinds, increasing nonoverlapping ranges and cannot intersect quoted/code text. The local producer handles declared hesitation/scaffolding, limited stutters, and repairs with a repeated frame or matching typed value. Ambiguous expressions remain model input; a cue alone is insufficient. Cleanup adds no content, preserves all-padding input, and is retained if organization fails. Disabled polishing and verbatim mode do not run this stage. Span production is Arion's implementation, not a recovered competitor model.

## Scene selection

The scene sets the layout of a rewrite, never its content. Scenes: `chat` (one plain paragraph, the default), `email` (greeting, body, closing), `work` (concise message, enumerated points as numbered lines), `prompt` (instruction for a coding agent: code names, paths, commands and error text exact, enumerated requirements as numbered lines), `search` (query without sentence-final punctuation) and `list` (numbered lines). They mirror the scenes Chatterfly presents ([research](../docs/research/chatterfly.md#scene-recognition-and-switching)); search follows its field-type signal.

- SCN-01: With **Match the app** on (default; under Settings › Model while polishing is on), the scene is decided once per session after target capture by `domain.DecideScene`: a scene the user chose for the place wins, then a search field, then the host of the web page owning the focus, then the app's bundle ID, else `chat`. A listed site matches its exact host, ignoring `www.`; workspace domains such as `feishu.cn` also match every subdomain. Off, every session uses `chat`.
- SCN-02: Inputs are metadata only: bundle ID, AX role/subrole and the http(s) host of the outermost `AXWebArea`'s `AXURL`. The host is read only while polishing and Match the app are both on. Field text, page titles, paths, queries, clipboard and history are not read for scene selection. Hosts and bundle IDs stay in session memory and are never logged; the review event reports only the scene.
- SCN-03: Every preview and the final rewrite of a session use the same scene. Secure fields still fail before capture (DSK-08); no context is read from them.
- SCN-04: Scene tags are prompt structure: a proposal containing any scene tag is rejected as markup. Layout added by a scene passes the same validation as any other proposal.
- SCN-05: Opening the menu bar panel reads the frontmost app's focused field (app name, field type and, under SCN-02's gate, site) and shows its scene with whether it is automatic or the user's choice. Choosing a scene there stores it for that place: the site without `www.` when there is one, else the app, and a search field of that place separately (`domain.SceneKey`). **Automatic** removes the choice. Choices are kept with the preferences on this device; secure fields, Arion itself and unreadable targets show no card.

## Rewrite context

[ADR 013](../docs/decisions/013-rewrite-context-on-device.md) lets polishing read three sources, each with its own switch under Settings › Model, all on by default while polishing is on.

- CTX-01: **Text around the cursor**: up to 200 characters before and after the selection of the captured field, read through Accessibility (macOS). Windows does not read it.
- CTX-02: **Screen content**: OS text recognition of the focused window only (macOS: ScreenCaptureKit + Vision accurate, Screen Recording access, macOS 14 or later; Windows: `PrintWindow` + `Windows.Media.Ocr`, compile-checked only). The image is released inside the call; all recognized text is retained after prompt-marker sanitization. OCR is not shortened to extracted terms or a character count. A request exceeding the configured model token budget fails explicitly instead of silently dropping context.
- CTX-03: **Recent dictations**: the last five texts inserted in the same place (`domain.SceneKey`), kept 30 minutes in memory; turning the switch off clears them.
- CTX-04: Secure fields and the password-manager denylist (`domain.ContextDenied`) read nothing. Context is read once per session after target capture, in parallel with speaking; previews use it once read and the final rewrite waits at most 1.5 s. It stays in session memory and is never logged, synced or sent anywhere.
- CTX-05: Context is data. It is sanitized of prompt markers and section tags. Both previews and final rewrites pass the complete sanitized OCR text in `<screen>` and cursor/history in their own sections, using `context-v2+live-v2` whenever context is present. Discussion context may resolve plausible ordinary-word misrecognitions as well as names and identifiers; preserve intent, questions, uncertainty, negation and facts. Context is never an instruction or permission to answer the dictation. Validation treats context as evidence for spellings and rejects `context_copied` proposals and context standing in for most of the dictation (capped coverage credit). Speech repeating the end of the text before the cursor may be dropped.
- CTX-06: **Approved memory** reaches the prompt's terms only for entries whose heard or written form occurs in the text being rewritten, at most eight (`domain.Vocabulary.Terms`), so the prompt does not grow with the memory ([ADR 014](../docs/decisions/014-edit-learning.md)).

## Runtime requirements

- Work offline after explicit model provisioning; no cloud inference route or silent model substitution.
- Model residency and per-session KV state are different lifecycles. Warm sessions may reuse immutable weights, but private context/KV state cannot leak between independent fixtures or users. Cold evaluation releases native model allocations and cache state, then includes reload cost.
- Capture callbacks never execute LLM inference. ASR and LLM workers have bounded queues, cancellation and recorded scheduling/thread policies. Concurrent ASR quality and queue pressure must be evaluated before enabling speculative generation.
- Cancellation tokens alone do not prove safety. Check generation/session/source revision at native preview publication, IME mutation and final target commit.
- No checkpoint, latency target or acceleration provider becomes an accepted product default without actual-host measurements and reviewed model/export provenance. CPU is the comparison baseline; provider acceleration is a separate experiment.

## Evaluation requirements

Use [008 Evaluation](008-evaluation.md) for ASR and native product safety, plus the [rewrite methodology and protocol](../docs/research/rewrite-evaluation.md). The rewrite harness uses the persistent JSONL transport with explicit schema `arion-rewrite-v1`; existing ASR drivers must fail on unsupported rewrite operations.

Evaluate both reviewed gold-text inputs and frozen real-ASR inputs. Hold each source text constant across rewriting variants. Report these strata separately so ASR mistakes do not become alleged rewriting failures or improvements. End-to-end speech evaluation still includes actual audio, endpointing and source-revision changes in the native driver.

Disfluency cases must distinguish hesitation, repetition and self-correction from meaningful uses of the same expressions (demonstratives, comparisons, literal quotes, negation and uncertainty). Report each evaluated recognition language separately. Exploratory text-only probes in another language do not establish ASR coverage or multilingual fidelity; see the [disfluency diagnostic](../docs/research/rewrite-reliability.md#multilingual-disfluency-follow-up).

Factorial comparisons cover rule versus LLM, context, approved memory and scenario hint. A verbatim control exposes unsolicited changes. Separate FP and quantized artifacts with the same reviewed source checkpoint expose conversion/quantization loss; if lineage differs, label the comparison as model/precision rather than causal quantization. Fix prompt/tokenizer/output budgets/no-thinking decoding/provider/threads within a comparison. Report warm/cold residency and solo/concurrent-ASR strata independently.

Automated checks cover protected exact occurrences, declared constraints, unsupported numeral and attack-marker proxies, refusals, delivery coverage, native timing/resource traces and lifecycle invalidation. Meaning, style, fluency and unsupported factual claims require blind independent human review. A release cannot claim semantic success from reference overlap, embedding similarity or an LLM judge alone. Any native stale publication or target commit in the candidate benchmark invalidates the run. Existing output directories, incomplete coverage and changed traces cannot produce a completed report.

## Acceptance

| ID | Required evidence |
| --- | --- |
| RWT-A01 | Provisioned local rewriting works offline; neither audio nor context goes to an inference service |
| RWT-A02 | Session/generation/source revision matches; cancellation and revision invalidation discard tokens/proposals and prevent stale preview/write |
| RWT-A03 | Focus/mode changes invalidate shortcut and owned IME composition; final commit rechecks ownership exactly once |
| RWT-A04 | Secure/unknown target context is denied before reads; clipboard/screen/history use explicit bounded capabilities; prompt data cannot authorize commands |
| RWT-A05 | Verbatim/code and protected meaning constraints hold; original and rewrite remain distinct, and the original stays available after automatic insertion (ADR 011) |
| RWT-A06 | Missing artifact, unsupported provider, rejection and timeout are explicit; no fabricated live result or silent cloud routing |
| RWT-A07 | Reviewed hashes, isolated splits, held-out labels, matched factorial controls, warm/cold/load strata and real token/native traces produce a reproducible checked report |
| RWT-A08 | Blind meaning/style/fluency/hallucination review meets declared independent-rater coverage; output coverage differences invalidate paired quality claims |

### Live pipeline acceptance

| ID | Required evidence |
| --- | --- |
| LRW-A01 | Previews appear while capture continues; stale previews are never displayed (`TestLivePreviewThenFinalRewrite`, `TestStalePreviewNotDisplayed`) |
| LRW-A02 | Stop/cancel discard previews and the final proposal; a dead worker is an explicit `unavailable` outcome (`TestCancelDuringFinalRewrite`, client tests) |
| LRW-A03 | Rejected, timed-out and unavailable rewrites insert the rule-corrected text with a recorded outcome (`TestRejectedRewriteKeepsSource`, `TestRewriteFailureAndTimeoutAreExplicit`, `TestValidateRewrite`) |
| LRW-A04 | Worker has no speech-runtime dependency; KV reuse equals a cold run (`TestWorkerDependencies`, `TestReuseParity` with a provisioned model) |
| LRW-A05 | Held-out development gate on Apple M4: text stage delivers at least ten more proxy passes than the source with zero worsened cases; the product audio variant has zero worsened cases, previews before stop in at least 80% of multi-segment fixtures and stop-to-final p95 under 1.5 s ([report](../benchmarks/results/live-rewrite-v1/README.md)) |
| LRW-A06 | Blind independent meaning/style/hallucination review of live output (open; same coverage rules as RWT-A08) |

### Scene selection acceptance

| ID | Required evidence |
| --- | --- |
| SCN-A01 | Field > site > app > default precedence, subdomain matching and fixed scene without Match the app (`TestDecideScene`, `TestFollowAppDecidesScene`) |
| SCN-A02 | Scene tags are rejected as markup (`TestSceneTagsAreMarkup`) |
| SCN-A03 | Held-out scene corpus on Apple M4: auto scenes deliver more layout-proxy passes than fixed chat with zero worsened cases; the live held-out set keeps its LRW-A05 result ([report](../benchmarks/results/scene-switching-v1/README.md)) |
| SCN-A04 | The site host is read from Safari and Chromium web content on a trusted build, and not read with the preference off (open: native verification) |
| CTX-A01 | Context reaches previews and the final rewrite as terms; history is per place and expires; denied apps are never read (`TestRewriteContextAndHistory`, `TestHistoryExpiresAndClears`) |
| CTX-A02 | Bounds, sanitizing, term extraction and the context validation rules (`TestRewriteContextBounded`, `TestValidateRewriteWithContext`, `TestContextTerms`) |
| CTX-A03 | Held-out context ablation on Apple M4 with production OCR and worker ([report](../benchmarks/results/context-ablation-v1/README.md)); a gain requires zero worsened cases in its run |
| CTX-A04 | Real window capture, recognition and cursor reads on a trusted build (`TestReadContextFromWindow`, open), and Windows OCR on Windows (open) |
| SCN-A05 | A user's choice beats the built-in rules for exactly its place and Automatic restores them; the panel card shows, switches and retranslates the scene (`TestSceneOverrides`, `TestSceneOverrideWins`, `TestSceneCard`, `TestPanelWithSceneCard`) |

## Implementation order

Implemented: source-bound cancellation and live preview, deterministic validation, isolated worker, KV-reuse parity check and the live development benchmark (LRW-A01–A05). P0: reviewed corpus and independent blind review (LRW-A06, RWT-A08); source-checkpoint/export parity; native shortcut/IME acceptance with rendered preview timing. Scene selection from field, site and app (SCN-A01–A03). P1: Windows runtime and acceptance; native site-host verification (SCN-A04); consent-gated context text; ASR-contention and power measurements. P2: broader personalization after harm measurements justify it.
