# Contextual rewrite evaluation

## Evidence boundary

This document defines the reviewed release protocol. Separate [Qwen development diagnostics](qwen-pipeline-diagnostics.md) report actual local CPU inference on unreviewed developer fixtures, without relaxing this protocol or asserting competitor parity. [009 Contextual rewrite](../../specs/009-contextual-rewrite.md) is normative. The Node harness, deterministic plan, validators, checked reports and blind-review packaging are implemented; a reviewed ONNX generation/native-preview driver and authorized corpora are still required. Tests use known synthetic count/serialization oracles and reject `execution:"fixture"`. They are not inference evidence.

Recognition, conservative correction, candidate ranking and contextual rewriting have separate outputs and denominators. Preserve frozen ASR inputs for rewrite comparisons; keep existing [speech evaluation](benchmark-methodology.md) for actual CER/WER/MER and [streaming evaluation](streaming-datasets.md) for endpoint/preview timing. This rewrite harness measures contention queue/drop/emission traces, not ASR reference accuracy. Run the existing ASR evaluator on the same pinned audio/model with and without the LLM load to establish recognition regressions.

## Research basis and adaptations

[GYAFC](https://aclanthology.org/N18-1012/) evaluates formality, fluency and meaning with human judgments. Arion adapts these into independent meaning/style/fluency scores, with app/task context and hallucination labels added. Formality is only one requested scenario; do not assume a formal rewrite is always preferable.

[JFLEG](https://aclanthology.org/E17-2037/) uses multiple human corrections for fluency editing and discusses GLEU/M2. It motivates retaining multiple references and separating grammatical cleanup from broader style rewriting. Arion currently computes exact-reference and nearest-reference edit proxies, **not GLEU or M2**; these proxies cannot establish semantic or stylistic superiority. If upstream GLEU is added, pin its implementation and publish separate language-specific scores.

[CoEdIT](https://aclanthology.org/2023.findings-emnlp.350/) studies instruction-driven text editing across tasks. Arion uses explicit user instructions and task strata; scenario hints, context and memory are independently removable inputs. Instruction prefixes imported from a dataset must be parsed using its reviewed schema, never guessed from arbitrary colon delimiters.

[StruQ](https://www.usenix.org/system/files/usenixsecurity25-chen-sizhe.pdf) combines structured inputs with specially trained models to address prompt injection. It motivates adversarial data boundaries and tests across transcript/context/memory surfaces. Merely surrounding data with JSON or delimiters does not establish StruQ-level protection. Arion proposals have no tools or execution privileges; controlled attack-marker tests and human intent review measure residual output contamination.

[The review of style-transfer human evaluation](https://arxiv.org/abs/2106.04747) highlights meaning, style and fluency as distinct dimensions. Keep these scores separate. Frozen automatic semantic scorers may later assist triage, but model-based similarity is not a factuality oracle and may miss negation or numerical changes. Calibrate any such scorer against the blind labels before using it for a gate.

## Hugging Face candidates and access review

Metadata revisions below were fetched from the HF repository API. They pin cards/layouts, not downloaded or imported rows. No automatic data or executable loader acquisition is implemented.

| Dataset | Pinned revision | Available use | Limitations before import |
| --- | --- | --- | --- |
| [grammarly/coedit](https://huggingface.co/datasets/grammarly/coedit/tree/e9a255c33ef910bc33a9d2b522653fa87521583e) | `e9a255c33ef910bc33a9d2b522653fa87521583e` | Author-hosted English instruction/input/target editing; card says Apache-2.0; train and validation JSONL | Primarily training material; no test split in this repository. Track component-source terms/overlap with evaluated models, task prefixes and original identities. Create a held-out test from an independent permitted source |
| [jhu-clsp/jfleg](https://huggingface.co/datasets/jhu-clsp/jfleg/tree/8b9fb0e6b3456ed57c9ba5524ef9e26b01177750) | `8b9fb0e6b3456ed57c9ba5524ef9e26b01177750` | English learner sentences, several corrections; validation/test parquet | Card says CC-BY-NC-SA-4.0. Requires access/use review; do not assume product/commercial suitability. No voice audio or app context |
| [RUCAIBox/Style-Transfer](https://huggingface.co/datasets/RUCAIBox/Style-Transfer/tree/5e1735b10088c9ef57f3c211bc1182c436a45f47) | `5e1735b10088c9ef57f3c211bc1182c436a45f47` | GYAFC EM/FR archives for English formality research | Third-party collection, absent license metadata. Recover original permissions/splits and reject until source rights are reviewed |

These datasets do not cover consented editor context, Mandarin/mixed dictation, timing, owned IME composition or streaming cancellation. Add independently reviewed private fixtures: work email, casual chat, issue/comment/code, terminology, numerical quantities/date/currency, negation/modality/uncertainty, long speech with self-correction, unchanged good text, secure-context denial, prompt injection and harmful/refusal cases. Split by author/speaker, original document/conversation, session and content family; siblings, paraphrases and frozen-ASR copies stay together. Public model training contamination can remain unknown even with evaluation-side split isolation; record that limitation.

## Experiment and commands

[desktop-rewrite.json](../../benchmarks/configs/desktop-rewrite.json) is a plan template with explicit placeholder artifact IDs. It contains 16 rule/LLM × context × memory × scenario conditions, an exact verbatim control and a full-feature FP32 reference. Three seeds/repeats × warm/cold × solo/concurrent-ASR yields 216 blocks per measured case. The configured output/token/thread values are experimental settings, not measured product defaults.

The no-thinking prompt and deterministic decoding are held fixed; throughput is measured rather than inferred from parameter count. Additional checkpoint/provider/prompt/budget comparisons use separate reviewed configs; a quantization claim requires independently verified same-base checkpoint digests and export lineage, plus identical tokenizer/template and conversion settings except precision. FP32 against INT4 is otherwise explicitly a combined model/precision comparison. Record the FP32 reference's actual peak process RSS/device footprint using the same counters and contention policy as INT4; parameter count or weight-file size cannot substitute for resident-memory measurements. The placeholder reference is a CPU FP32 export; unsupported operators/providers fail explicitly without substituting another reference. Do not mix unrelated runtime/checkpoint changes into a context ablation.

Use `node tools/evaluation/rewrite-cli.mjs` as the portable entry point (pnpm/Make wrap it):

```sh
node tools/evaluation/rewrite-cli.mjs plan --config benchmarks/configs/desktop-rewrite.json --out .workspace/benchmarks/rewrite-plan.json
node tools/evaluation/rewrite-cli.mjs validate --corpus /absolute/private/rewrite-corpus.json
node tools/evaluation/rewrite-cli.mjs run --config benchmarks/configs/desktop-rewrite.json --corpus /absolute/private/rewrite-corpus.json --driver /absolute/private/rewrite-driver.json --out .workspace/benchmarks/rewrite-run
node tools/evaluation/rewrite-cli.mjs report --results .workspace/benchmarks/rewrite-run
node tools/evaluation/rewrite-cli.mjs human-export --results .workspace/benchmarks/rewrite-run --out .workspace/benchmarks/rewrite-review --raters 3 --seed 1729
node tools/evaluation/rewrite-cli.mjs human-report --results .workspace/benchmarks/rewrite-run --review .workspace/benchmarks/rewrite-review --ratings /absolute/private/ratings.json
```

Existing plan/run/review directories cannot be overwritten. Cold means release LLM native allocations and private KV state, then reload for each measured request; the contention ASR residency policy stays fixed and is declared in runtime provenance. Warm means a persistent model after separate non-test warmups, with per-case KV reset. A model switch warms the new model before measurements. Seeds/repeated generation are not independent authors.

Concurrent ASR begins before LLM prefill and continues throughout decoding using repeated realtime replay of the pinned fixture when necessary; declare replay strategy in runtime provenance. Its events, actual processed/dropped samples and queue age are measured. A request cannot return before its declared audio duration. Exact scheduling/provider/threads/power mode and device-memory measurement availability are recorded. CPU/RSS and device memory remain separate; unsupported device counters are null.

## Private corpus schema

A manifest has `version:1`, `kind:"arion-rewrite-v1"`, `license`, `consent:true`, and `samples`. Consent asserts authorized fixture use; it cannot create rights in third-party data. Every sample requires:

| Field | Contract |
| --- | --- |
| `id`, `split` | Unique safe ID; train/dev/test |
| `clusterId`, `sessionId`, `contentId` | Source author/speaker or conservative connected content cluster; original session and content family. A source author spanning conversations stays one cluster |
| `language`, `inputKind`, `tags` | zh/en/mixed; gold_text/frozen_asr; condition tags |
| `sourceText`, `sourceSha256` | Fixed input and SHA-256 of its JSON string via evaluator `digest`; at most 65536 UTF-16 units |
| `annotation` | `{kind:"human_reviewed",path,sha256}`; reviewed intent/reference/constraints provenance, file checksum checked |
| `asrLoad` | Local `{path,sha256,durationMs}` for real contention replay; checksum checked |
| `userInstruction`, `scenario` | Trusted user task/mode and explicit scenario hint; scenario off transmits `generic`, not the gold style label |
| `context` | `{text,origin,sourceSampleIds}`; origin train/dev/user_approved; explicit empty string if absent; at most 8192 UTF-16 units |
| `approvedMemory` | `{id,text,approved:true,origin,sourceSampleIds}` records; combined text at most 8192 UTF-16 units |
| `controls` | Empty or one `{atMs,type}` relative to request start; cancel/focus_change/mode_switch/invalidate_revision |
| `expected` | Held-out labels described below; never passed to the driver |

Expected labels require `references` (nonempty normalized texts), `intent`, `protectedSpans:[{kind,text,minimumOccurrences}]`, `requiredStrings`, `forbiddenStrings`, `permittedNumbers`, `maxOutputChars` (positive Unicode-character budget or null), `verbatim`, `refusalAllowed` and `attack`. Protected kinds are name/number/url/negation/code. Controlled attack cases need at least one forbidden marker; semantic attack success additionally requires human annotation. Empty lists mean no label of that kind, not a passing semantic check. References and intent are reviewed independently from system outputs. Gold entity labels can intentionally differ from a frozen ASR input; report that stratum rather than claiming exact source preservation.

The validator rejects cross-split cluster/session/content/source-text/audio identities and test-derived context/memory sources. Train context cannot reference dev. `user_approved` records need independent provenance review; their assertion is not a proof of label independence. A manifest cannot algorithmically detect paraphrase leakage or fabricated source metadata. Reviewed corpus construction must enforce those rules.

## Rewrite driver contract

Use the existing artifact-hashed executable/args/timeout configuration and persistent protocol-1 JSONL transport from [evaluation tooling](../../tools/evaluation/README.md), with explicit `schema:"arion-rewrite-v1"`. Stdout is one response per request; stderr is discarded. Do not log plaintext context or use a shell command string.

`hello` receives `purpose:"rewrite"` and all checked artifact hashes. It echoes `execution:"live"`, schema/hashes, actual darwin/win32 hardware and runtime version/buildFlags/timer/provider/scheduler/tokenizerVersion/promptHash. Capabilities include rewrite/native-preview/cold-reset/invalidate/cleanup and concurrent-asr when requested. Model, prompt, tokenizer, ASR and optional FP comparison model IDs must have local hash/license entries; model bundles also pin external tensor data, runtime and native-host binaries. The harness cannot prove a dishonest driver's inference claims: review its actual adapter and retained OS traces.

`rewrite` receives sampleId/seed/warmup/cacheState/loadState, exact variant/effective pipeline, `identity:{session,generation,revision}`, controls and optional pinned contention audio. Its `input` contains **only** sourceText, userInstruction, enabled context, approved memory stripped to id/text/approved, and enabled scenario. The input hash covers this treatment-specific object. Reference rewrites, intent rubrics, protected spans, forbidden markers and outcome labels remain evaluator-side.

Reply fields:

| Field | Required observation |
| --- | --- |
| `protocol`, `schema`, `requestId`, `execution` | Exact request identity and live attestation |
| `effectiveFeatures`, `effectivePipeline`, `inputHash`, `cacheState`, `loadState`, `identity` | Exact applied inputs/treatments/source identity |
| `decision` | rewritten/unchanged/rejected/refused/cancelled; unchanged candidate equals source |
| `modelText`, `candidateText` | Raw generated text and validated deliverable, independently retained. Candidate is null after rejection/refusal/cancellation |
| `timings` | Nonnegative monotonic requestStartMs, loadMs, prefillStartMs, prefillEndMs, firstTokenMs or null, decodeEndMs, validationEndMs, reviewMs |
| `tokenEvents` | Actual ordered `{ms,tokenId,delta}`. Deltas concatenate to modelText; output token count includes special/EOS events, which can have empty delta |
| `usage` | Actual tokenizer inputTokens and outputTokens; both within pinned budgets. Rule/verbatim controls have zero LLM tokens |
| `previewEvents` | Actual native `{ms,text,identity}` observations, distinct from engine tokens. Final accepted candidate appears after validation |
| `lifecycleEvents`, `commitEvents` | Measured invalidation/cancel acknowledgments; no preview after injected invalidation. Candidate benchmark commits are always forbidden |
| `residency` | Every LLM fixture resets private KV state. Measured warm requests keep model resident; cold reset/reload booleans require actual native lifecycle implementation |
| `resources` | peakRssBytes, peakDeviceBytes or explicit null; rssScope (process_high_water/process_request_sampled/process_tree_request_sampled), rssSamplingIntervalMs (positive for sampled scopes, null for lifetime high-water) |
| `asr` | Null for solo. Concurrent: live/model/audio identity, start/end spanning prefill/decode and realtime replay, actual events, processedSamples/droppedSamples/maxQueueAgeMs |

Rules need to attest applied context/memory/scenario even if a declared rule policy does not use a hint. Treat that as a measured null effect, not as LLM functionality. Do not simulate token IDs for rule controls. A decoding timeout/adapter crash fails the run; refusal and constraint rejection are explicit completed outcomes and remain in acceptance denominators.

Memory scope must remain explicit in reports. A persistent process's lifetime high-water mark cannot be reset by clearing a model cache and includes earlier requests; it is not a fresh request peak. Sampled peaks record their interval and can miss shorter spikes. A worker-based runtime must include owned inference processes when comparing total memory to an in-process baseline. Review RSS method/PID coverage in retained native traces; the scope field is an attestation, not independent proof of sampling.

`cleanup` must return live schema and `cleanupComplete:true` after closing owned previews/audio/hosts and restoring owned controls. The harness attempts it after success or failure; failed/unconfirmed cleanup prevents a completed report. Drivers must also clean up on OS signal/parent exit, because dead transports cannot acknowledge cleanup.

Single-request invalidation fixtures prove discard behavior for one source, not correct behavior of a new source/revision or concurrent IME host. [Native E2E](../quality/e2e.md) must separately exercise actual old/new callbacks, host ownership and focus/mode transitions. Offline execution/no content egress likewise needs actual-host network evidence; `execution:"live"` is not that proof.

## Metrics and report integrity

- Protected exact occurrence recall sums preserved occurrences over annotated counts. Report accepted-only recall **and** delivery recall with all eligible protected cases in its denominator, plus acceptance/refusal/rejection rates. Rejecting every proposal must not look successful.
- Constraint violation rate counts reviewed required/forbidden/length/verbatim checks on accepted proposals. Exact negation presence can preserve the word while flipping scope; semantic intent is human reviewed. Unexpected numerals and controlled attack markers are narrow proxies; they do not detect arbitrary invented facts or all injection attacks. Raw model marker contamination and delivered contamination are separate.
- Exact-reference rate and minimum normalized edit distance to multiple references are descriptive overlap proxies. Their normalization is inherited from the speech evaluator; code/case/punctuation preservation uses exact strings/verbatim checks instead. Never call rewrite overlap CER/WER or treat low edit distance as natural style.
- Report actual first-token, prefill/decode/validation/ready-for-review/native-preview p50/p95/p99, output tokens, post-first-token decode throughput, preview withdrawal, load, RSS/device memory, cancellation acknowledgment and ASR queue/drop rates. Missing token/preview values retain count/nulls. Main latency/stability excludes cancellation-control cases; cancel/resources/ASR counters remain visible. User review dwell and target commit latency are outside candidate timing.
- Pair one factor at a time with source input/model/template/runtime held fixed. FP versus quantization is separately labeled. Report ASR contention deltas separately from cold/warm load. Seeded percentile bootstrap resamples source-author/content clusters, retaining every repetition. Fewer than two clusters or undefined resamples yield null CIs; no significance follows from an unadjusted comparison matrix.
- Run metadata records configuration/driver/corpus/artifact identities, actual hardware/runtime/Git state and cleanup. Private corpus, results and warmups remain under ignored workspaces with restrictive creation permissions. Report rechecks hashes, plan/coverage/unique observations, identity/timing/trace gates and recomputes metrics from checked labels/results. It never trusts a supplied score field. A failed/incomplete run cannot produce a completed aggregate. Aggregates still need privacy review before external publication.

## Blind human quality loop

Export accepted non-invalidation candidates in seeded shuffled order with fresh opaque task IDs. Raters receive source, reviewed intended meaning, explicit task/scenario/context and output; no model, precision, variant or cache labels. Keep `mapping-private.json` hidden. The declared minimum defaults to three independent raters, at least two. Rate meaning/style/fluency independently on 1–5 anchored scales; mark unsupported factual claims and unnecessary refusal separately. Rater identity is pseudonymous; collect independent votes before adjudication.

Ratings JSON is `{version:1,kind:"arion-rewrite-human-v1",ratings:[{taskId,raterId,meaning,style,fluency,hallucination,unnecessaryRefusal}]}`. Import rejects unknown/duplicate task-rater votes or incomplete coverage. Means and disagreement ranges are descriptive ordinal summaries, not interval-scale proof. Unaccepted output coverage remains in the automatic report; unmatched delivered-output coverage invalidates paired human quality comparisons. Retain reviewed adjudication notes separately; do not silently replace independent votes with consensus.

Use train/dev for prompt/constraint tuning and provisional quality/latency selection. Freeze prompts/artifacts/budgets and the rating rubric before held-out test. Repeat on actual macOS and Windows CPU baselines, then declared accelerated providers. Require hard lifecycle/privacy correctness and acceptable preservation/human quality before selecting throughput; quality gates and latency budgets remain unmeasured until the first permitted live study.
