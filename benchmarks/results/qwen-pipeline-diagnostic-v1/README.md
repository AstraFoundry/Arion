# Qwen pipeline development results

These are real local CPU inference results, not Typeless parity or release acceptance. Hardware: Apple M4, 32 GiB RAM, macOS arm64, low-power mode disabled. GenAI 0.17.1 / ORT 1.30.0; four intra-op threads, one inter-op thread, greedy decoding, resident weights with fresh per-request KV state. [The reviewed content-free aggregates](summary.json) retain corpus/config/prompt/worker/result hashes and runtime identities. Raw audio, transcripts, token traces and disclosed historical review tasks are retained in the [research archive](../../migration/README.md). Weights and runtime environments remain separately provisioned.

The 24 text cases and 14 synthetic speech cases are developer-authored and unreviewed. There is one content author cluster; no held-out test, independent human ratings or executed Typeless reference. All pass counts below are exact-string/format/truncation proxies, **not semantic accuracy**. English spoken/written numeral forms and synthetic technical-name pronunciation affect audio and frozen-source scores. Latency excludes native preview/insertion; thermals and unrelated host tasks were not isolated.

## Text and model comparisons

| Treatment | Constraint passes / 24 | Warm generation p95 | Interpretation |
| --- | --- | --- | --- |
| Original text control | 8 | No inference | Leaves repairs/fillers and has no memory/style changes |
| Qwen3 0.6B basic | 12 | 0.554 s | Fast, with substantial failure/harm risk |
| Qwen3 0.6B full | 9 | 0.708 s | Longer instructions did not improve this checkpoint |
| Qwen3 1.7B basic | 8 | 1.100 s | Original dates/amounts and formatting often remain |
| Qwen3 1.7B full | 11 | 2.039 s | Insufficient evidence for useful default rewriting |
| Qwen3 4B full | 16 | 5.879 s | Improves surface outcomes, remains slow and imperfect |
| Qwen3 4B intent-only | 17 | 6.633 s | All six gold-text repair cases pass; style still limited |
| Qwen3 4B intent then format | 19 | 15.220 s | All six repairs pass; three of four style cases pass |

The last two treatments share an intent-cleanup prompt, with a second format-only generation in the latter. Its trusted verbatim mode bypasses inference. Clean-mode passes are 15/22 versus 17/22; both treatments pass the two verbatim cases. Exact reference proximity cannot establish that fluent text or correct grammar was preserved. Observed development failures include extraneous action wording, omitted content, literal input structures, unchanged terminology and insufficient context resolution.

The same 4B full prompt was 16/24 in a second run with p95 10.208 s. Treat these timings as screen measurements with host variability, not a reproducible release SLO or a causal checkpoint-speed estimate. Worker lifetime high-water RSS was about 1.54 GiB for 0.6B, 2.65 GiB for 1.7B, and 3.64–3.80 GiB for 4B in these run families; it is not per-request memory and excludes other processes/device memory.

## Feature and reasoning ablations

For 1.7B, full versus no context/memory/scene yielded 11/12/10/9 passes. The context-off gain comes from the controlled attack stratum; neither context resolution nor terminology passed either of their two dedicated cases. 4B also failed these dedicated memory/context proxies. More prompt context is not demonstrated to help this implementation. Typed approved replacements and ASR vocabulary bias should be evaluated explicitly, using existing domain correction rather than expecting free-form memory prose to work.

On identical unaffected inputs, 21 context pairs and 21 memory pairs produced identical text. Case inputs and held-out outcome labels stayed separate. This checks visible repeatability/isolation, not privacy leakage or universal determinism.

With a fixed 256-token output ceiling, 1.7B full non-thinking/thinking scored 11/6; reasoning had eight failed/truncated deliveries and p95 10.806 s. Compact non-thinking/thinking scored 9/0, with 18 failed/truncated reasoning deliveries and p95 13.272 s. Compact instructions alone scored 8/24 for 0.6B and 9/24 for 1.7B; they did not improve the desired behavior. Reasoning is stripped only after a complete reasoning block; incomplete blocks never become input text.

The intent-v2 follow-up changes the examples and explicitly preserves original subjects/actions. It still passes all six gold-text repairs and removes one observed extraneous-action failure, but loses the English repetition case: 16/24 versus 17/24. This is a development tradeoff, not a demonstrated semantic improvement. The follow-up timing was 3.353 s p95 versus 3.422 s for the earlier intent prompt in that run; host-dependent timing variation remains material. Independent review must select between these alternatives.

## Speech and frozen-source pipelines

| Actual acoustic pipeline | Mandarin CER | English WER | Mixed MER | Explicit stop to result p95 |
| --- | --- | --- | --- | --- |
| Production X-ASR 160 | 8.19% | 34.69% | 45.45% | 62 ms |
| X-ASR 160 + current rules | 8.19% | 34.69% | 45.45% | 50 ms |
| X-ASR 160 + X-ASR acoustic finalizer | 2.34% | 28.57% | 36.36% | 171 ms |
| Qwen3-ASR 0.6B INT8 batch | 1.17% | 32.65% | 33.33% | 758 ms |

Mandarin/English/mixed denominators are 171 characters / 49 words / 33 mixed units, from eight/four/two synthetic clips. These small, synthetic strata are not broad language benchmarks. No reviewed speech-onset/word timing or native rendered preview was measured. Qwen-ASR produces one final result after a realtime fixture wait in this adapter; it does not establish native streaming capability. Its output preserves spoken repair processes, so an ASR decoder alone is not the demonstrated contextual-cleanup stage.

Four source pipelines supply 56 frozen inputs to 4B intent-only and intent-then-format. Both yielded 15/56 constraint passes; generation p95 was 4.515 s versus 9.377 s. By acoustic source, both had 2/14 for streaming-only, 3/14 for current rules, 5/14 for X-ASR finalization and 5/14 for Qwen-ASR. This measures rewriting of actual, fixed recognition output; it is not a live composed native dictation run. Exact numeral spelling/protected-code constraints can fail semantically equivalent or acoustically unrecoverable input, requiring independent review.

## Prefix prefill during real ASR replay

[The prefix comparison](prefix-summary.json) contains six declared Mandarin/English/mixed synthetic pairs with actual streaming recognition and acoustic finalization. The 4B intent-v2 system-message prefix is computed during speaking; final ASR text is appended after recognition. Both conditions use the identical 2176-token search allocation and 128-token generation ceiling. All six ASR texts, complete prompt identities and final LLM outputs match across conditions; both have two strict-constraint passes. Prefix computation finished before recognition in all six cases.

| Parent-clock observation | Sequential generation | Prefix computed during replay |
| --- | --- | --- |
| ASR result to candidate p50 | 3.053 s | 1.334 s |
| ASR result to candidate p95 | 3.797 s | 2.190 s |
| First ASR partial p95 | 1.221 s | 1.218 s |
| Estimated explicit stop to candidate p95 | 3.947 s | 2.363 s |

The last row is a component estimate and excludes acoustic-response serialization. This is no native insertion/preview measurement, no queue/drop proof and no general latency guarantee. Six synthetic pairs cannot support population confidence intervals. Total compute is moved into speaking time, not eliminated.

One earlier full run had a parent/child timing gap of about 358 seconds in one case, with identical text outputs. Its raw measurements are retained and excluded as a whole from the uncontaminated comparison; no outlier was silently deleted. A failed cleanup attempt is also preserved as failed. The successful replay owns a temporary idle-sleep assertion, checks parent/engine timing gaps and confirms owned ASR subprocess termination. Precise physical cause of the earlier interruption was not established.

## Engineering decision and handoff

Prioritize the existing streaming recognizer plus acoustic finalization, followed by one optional local intent-cleanup proposal, with fixed-prefix computation overlapped with capture. Qwen-ASR is a precision challenger for finalization, with greater post-stop cost in this sample. A second formatting LLM pass does not improve these frozen-input constraints and approximately doubles engine latency; do not make it the default. These are development priorities, not accepted production model choices.

Keep code/verbatim on a deterministic source path, original text available and generated proposals under explicit review. Do not enable automatic insertion from these results. No LLM product runtime, Windows native test, IME lifecycle test, cold run, ASR contention run, source-FP/export parity, model distillation or semantic/Typeless parity gate was completed.

P0 work is source/export parity, independent intent and fact annotations, consented natural speech and a same-fixture Typeless comparison in an owned editor. Train/distill a compact editing model only from reviewed data; freeze prompt/checkpoint before held-out evaluation. Evaluate approved aliases through existing `domain.Correct` and actual ASR bias separately from LLM memory. Then implement the native sidecar/host adapter and run the [strict rewrite protocol](../../../docs/research/rewrite-evaluation.md), including cancellation/focus/revision ownership, cold/warm, concurrent ASR and blind meaning/style/fluency/hallucination ratings.

[Commands and evidence boundaries](../../../docs/research/qwen-pipeline-diagnostics.md) describe explicit provisioning and repetition. Private unscored tasks are available in `.workspace/benchmarks/qwen-text-blind-review` and `.workspace/benchmarks/qwen-frozen-blind-review`. Keep their mapping files private from raters.
