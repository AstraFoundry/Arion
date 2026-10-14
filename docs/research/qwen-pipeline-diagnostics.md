# Qwen pipeline diagnostics

## Scope

Development experiments execute real local inference on Apple M4 / macOS arm64, with native CPU ONNX Runtime GenAI through an isolated Python worker. They complement the [reviewed native rewrite protocol](rewrite-evaluation.md); they do not relax that protocol or satisfy release acceptance.

The private developer-authored corpus has 24 gold-text challenges: Mandarin, English and mixed language; spoken repairs, fillers/repetition, negation/uncertainty, numbers, names/URLs/code, context, terminology, email/list formatting and quoted prompt injection. Fourteen are synthesized by macOS Tingting/Samantha for speech diagnostics. There are no independent annotations, human ratings, held-out test results or executed Typeless references. All content is one developer author cluster. Do not compute population confidence intervals or describe reference/constraint proxies as semantic accuracy.

`arion-qwen-diagnostic-v1` is intentionally separate from `arion-rewrite-v1`. The latter still requires reviewed annotations, actual native preview, lifecycle controls, cold residency and concurrent ASR evidence. This diagnostic only implements warm, solo CPU engine generation on macOS. Windows, native preview, IME insertion, cancellation, secure-context collection, power use and air-gapped execution remain unmeasured. Model load time is separate from generation. Two-stage generation sums the two observed engine durations; it excludes inter-stage IPC and native rendering. The decoder removes terminal EOS and outer whitespace; raw tokens/text remain private. Verbatim mode must bypass generation in the product. RSS is the worker's cumulative process high-water mark, not per-request allocation or device memory.

The [measured result report](../../benchmarks/results/qwen-pipeline-diagnostic-v1/README.md) contains reviewed aggregates and implementation priorities.

## Executed design

- Hold the source text fixed across Qwen3 0.6B and 1.7B INT4, basic/full prompts, and full-prompt removal of context, memory or scene hints.
- Compare intent cleanup and a second format-only pass. The second pass consumes the first actual result; a truncated/empty first result fails explicitly. Verbatim bypass uses the trusted mode, without label-based repair. Compare clean-mode cases separately to isolate formatting from bypass.
- Test a shorter Chinese instruction prompt separately. Its examples use values outside the challenge cases. Prompt iteration uses development labels; it is not a held-out improvement.
- Compare reasoning enabled/disabled with the same 256-token generation ceiling. Incomplete reasoning has no candidate and fails delivery; reasoning is never ordinary output text.
- Replay the speech subset through production X-ASR 160 ms, deterministic correction and X-ASR acoustic finalization. Independently decode the same audio with Qwen3-ASR 0.6B INT8 after an explicit-stop fixture wait. Qwen3-ASR emits one final result in this adapter; it is not a streaming recognizer.
- Freeze these four real source pipelines for matched downstream LLM comparisons. Original developer references and outcome labels never enter the inference prompt. Frozen-text results isolate the rewrite stage; adding separately measured ASR and generation durations estimates serial engine latency and does not measure native end-to-end insertion.

The CPU checkpoint packages are pinned in [the diagnostic catalog](../../benchmarks/catalogs/qwen-diagnostic-models.json), derived from [earlier model research](onnx-rewrite-models.md). Provisioning checks sizes, LFS SHA-256 and small-file Git blob identities, and records actual native libraries. The measured runtime tuple is GenAI 0.17.1 / ONNX Runtime 1.30.0. ORT 1.31.0 produced a duplicate Objective-C class warning during the initial probe; it was removed before measured runs. CPU threads are fixed at four intra-op and one inter-op; decoding is greedy, every request creates fresh generation/KV state, and checkpoint changes create a new worker. Exports load successfully, but source-checkpoint FP parity, quantization provenance and desktop packaging are not established.

## Repeatable commands

Explicit provisioning uses `uv` and a private Python 3.12 environment. Project automation remains pnpm; Python is an experiment dependency, not a desktop runtime decision. Ordinary `make check` never downloads or runs a model.

```sh
make qwen-env
make qwen-provision MODEL_IDS=qwen3-0.6b,qwen3-1.7b
make qwen-diagnostic CORPUS=.workspace/ablation-corpus.json OUT=.workspace/benchmarks/qwen-text-new
make qwen-report CORPUS=.workspace/ablation-corpus.json RESULTS=.workspace/benchmarks/qwen-text-new
make qwen-diagnostic QWEN_CONFIG=benchmarks/configs/qwen-compact-diagnostic.json QWEN_PROMPTS=tools/evaluation/diagnostic/prompts-compact.json CORPUS=.workspace/ablation-corpus.json OUT=.workspace/benchmarks/qwen-compact-new
make qwen-diagnostic QWEN_CONFIG=benchmarks/configs/qwen-thinking-diagnostic.json QWEN_PROMPTS=tools/evaluation/diagnostic/prompts-compact.json CORPUS=.workspace/ablation-corpus.json OUT=.workspace/benchmarks/qwen-thinking-new
make qwen-asr-diagnostic CORPUS=.workspace/ablation-audio/corpus.json DRIVER=.workspace/qwen-asr-driver.json OUT=.workspace/benchmarks/qwen-asr-new
```

`EVAL_PYTHON`, `MODELS`, `QWEN_CONFIG` and `QWEN_PROMPTS` select explicit paths; existing output directories cannot be overwritten. `QWEN_ASR=1` adds the verified ASR archive during provisioning. The ASR driver configuration must pin its script, actual converter artifacts/tokenizer and native runtime binaries. Audio fixtures and curated references stay private; reproducing the identical run requires their manifest digest, not newly improvised text.

On systems without Make, use root pnpm scripts `qwen:env`, `qwen:provision`, `qwen:diagnostic`, `qwen:report` and `qwen:asr-diagnostic`, with the same CLI arguments/environment. macOS execution is required by this diagnostic version. `freeze-asr.mjs` verifies completed ASR traces and corpus hashes before building matched source cases:

```sh
node tools/evaluation/diagnostic/freeze-asr.mjs --corpus .workspace/ablation-corpus.json --audio-corpus .workspace/ablation-audio/corpus.json --native .workspace/benchmarks/qwen-native-v1 --qwen-asr .workspace/benchmarks/qwen-asr-v2 --out .workspace/ablation-frozen-new.json
make qwen-diagnostic QWEN_CONFIG=benchmarks/configs/qwen-frozen-diagnostic.json QWEN_PROMPTS=tools/evaluation/diagnostic/prompts-two-pass.json CORPUS=.workspace/ablation-frozen-new.json OUT=.workspace/benchmarks/qwen-frozen-new
node tools/evaluation/diagnostic/review-export.mjs --results .workspace/benchmarks/qwen-frozen-new --corpus .workspace/ablation-frozen-new.json --out .workspace/benchmarks/qwen-frozen-review-new
```

Blind development tasks carry no model identities or fabricated ratings. Keep `mapping-private.json` away from independent raters. They remain separate from the strict release review protocol; their author labels first need independent annotation review.

## Interpretation and remaining gates

Constraint pass means required exact strings occur, forbidden exact strings do not, protected exact strings occur, verbatim bytes match and output is not truncated. It can miss hallucinated prose and fail semantically equivalent numeral spellings or capitalization; report these surface proxies separately from human meaning/style judgments. Reference edit ratios mix language-specific normalization only as a per-case diagnostic mean, not a pooled CER/WER/MER. Speech error counts retain separate language denominators; English written/spoken numeral conventions and synthetic technical pronunciation limit interpretation.

Full/no-context identical prompts produced identical output in all 21 unaffected cases; full/no-memory did so in all 21 unaffected cases. This checks treatment isolation and visible repeatability, not general determinism or absence of privacy leakage.

P0: source-checkpoint FP/reference parity and export-lineage review; independent annotations and consented natural speech; compare Typeless on the identical synthetic fixtures in an owned editor; freeze model/prompt before held-out evaluation. P1: run the strict native driver with actual preview, cold/warm, concurrent ASR, protected-content validation, explicit user review and focus/cancellation/IME ownership. Host thermal state and other desktop tasks were not isolated; diagnostic latency is not a release SLO. Choose speculative prefill or automatic insertion only after those measurements and an accepted product decision.

## Prefix prefill experiment

A further adapter compares ordinary post-ASR generation with system-prefix prefill during the actual production ASR fixture replay. It uses six declared synthetic cases, the same 4B intent-v2 prompt, fixed 2176-token search allocation and 128 generated tokens. Each request has a fresh generator; no personal KV state is reused. The worker verifies that the independently tokenized system prefix is exactly the beginning of the complete prompt before appending its remaining tokens. Prefix prefill generates no candidate tokens.

This is actual concurrent ASR inference and staged generation, but remains outside the strict native rewrite driver: no native rendered preview, endpoint annotations, queue/drop counters or focus/cancellation lifecycle measurements. ASR-result-to-candidate is measured on the parent clock. A stop estimate adds the recognizer's internal finalization duration and excludes its response serialization; it must not be labeled native stop-to-insertion latency.

The initial completed comparison contained a host scheduling/clock gap: parent-observed delay greatly exceeded child inference time in one case. Retain that whole run and its outlier privately; do not remove the outlier or treat that run as an uncontaminated latency comparison. The adapter now owns a temporary macOS idle-sleep inhibition process, checks response wall deadlines and rejects parent/engine gaps above one second. This bounds measurement integrity, not an allowance to hide ordinary slow model inference. An earlier attempt failed on an unsupported ASR cleanup request; it remains failed. The ASR protocol only owns a subprocess, whose observed termination is now recorded instead of inventing a cleanup acknowledgment.

```sh
make qwen-prefix-diagnostic CORPUS=.workspace/ablation-corpus.json AUDIO_CORPUS=.workspace/ablation-audio/corpus.json DRIVER=.workspace/benchmarks/driver.json OUT=.workspace/benchmarks/qwen-prefix-new
```

Prefix staging can hide fixed prompt computation during speech; it does not reduce total compute or establish improved semantic quality. Short utterances may finish before prefill, and final context/memory tokens still need computation after stop. Actual target OS runs, source-bound invalidation and independent quality review remain required.
