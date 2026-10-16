# 008 Evaluation and automation

## Scope

The repository MUST use the exact pnpm version in `package.json` and a frozen workspace lockfile. Go remains the native runtime. [Makefile](../Makefile) owns recurring developer workflows; pnpm commands also work on Windows without Make.

Evaluation infrastructure is implemented; microphone, inference, IMK/TSF and relay drivers remain implementation tasks. No missing driver/model/capability may be reported as a passing feature test. Harness conformance fixtures are not model results.

## Required evidence

1. Native shell E2E runs the production window with the actual OS event loop, checks visible native content, resizing and close lifecycle. Content captures are not OS screenshots/GPU readbacks.
2. Product scenarios use isolated owned editors, devices and relay fixtures. Actual host observations and hashed traces are required. Assertions remain outside the driver. Missing capabilities fail explicitly.
3. Benchmark inference receives audio, context and approved memory only. Reference text, acceptable candidates, protected spans and timing annotations remain in the evaluator.
4. Local model/runtime and audio files MUST match pinned SHA-256 digests. Record hardware, power mode, runtime version/build flags/timer, Git revision/dirty state, effective treatments and pipeline settings.
5. Training, development and test audio/speakers/sessions MUST be separated. Test-derived memory is forbidden. Timestamp labels require independent review. Access/license evidence belongs to the corpus provenance.
6. Runs retain failed status and completed observations; incomplete runs cannot produce a completed report. Existing output directories cannot be overwritten. Reusable research corpora, recorded fixtures, raw benchmark observations, warmups, failed/cancelled attempts and reproduction sources MUST be archived under `benchmarks/` with original bytes, explicit path mappings, provenance and SHA-256 identities. Personal recordings, weights, runtime environments and unfinished execution remain private in ignored workspaces. CI run uploads remain aggregate-only unless a separate fixture archive is explicitly selected.

## Experiments

[Factorial configuration](../benchmarks/configs/desktop-factorial.json) enumerates memory × local Laya ranking × local correction, with the ASR checkpoint, chunk size and endpoint policy held fixed. Source order is the ranking baseline. Three seeds and separate cold/warm strata are paired by sample. A persistent driver retains warm models; cold requests explicitly reset model residency and include load cost. Dev warmups are excluded from measured test observations and retained separately.

[Streaming configuration](../benchmarks/configs/desktop-streaming.json) fixes all three features off and sweeps chunk sizes 20/40/80/160/320 ms against endpoint silence 200/500/800 ms. These are experimental settings, not accepted product defaults. Realtime replay must use actual monotonic scheduling; offline accelerated replay cannot supply product latency evidence. Rendered native preview events and engine events are recorded separately.

Report corpus CER/WER/MER, candidate oracle recall/top-1/MRR, erasure, harmful correction and exact protected-span proxies, first partial/stop-to-review/ranker/load p50/p95/p99, RTF and process RSS. Streaming adds reviewed-onset and preview latency, premature/missed intended-turn endpoints, endpoint lag, stable-unit latency with correctness coverage, queue age, dropped samples and a final-three-unit exact-match proxy. The tail proxy is not a phonetic diagnosis. [Methodology](../docs/research/benchmark-methodology.md) defines units, formulas and paper adaptations.

Paired speaker-cluster bootstrap CIs are exploratory. Seeds/repeats are not independent speakers. Candidate sets must match for ranker-only comparisons; mismatches invalidate that comparison. No significance claim follows from an unadjusted matrix of CIs. Release quality thresholds require a measured baseline and held-out decision; current latency budgets remain provisional.

## Acceptance

Scenario-aware generative rewriting uses the separate [009 contract](009-contextual-rewrite.md) and [rewrite methodology](../docs/research/rewrite-evaluation.md). Its plans and reports do not change ASR accuracy, Laya candidate producer gates or existing streaming metrics. Gold-text and frozen-ASR quality strata, native token/preview latency, concurrent ASR, prompt/model provenance and blind meaning/style review are distinct required evidence.

| ID | Scenario |
| --- | --- |
| EVAL-A01 | Frozen pnpm workspace and portable automation tests pass on CI |
| EVAL-A02 | Production window lifecycle passes on actual macOS and Windows separately |
| EVAL-A03 | Missing driver/OS capability fails; synthetic metric fixtures do not claim inference |
| EVAL-A04 | Cross-split speaker/audio/session or test-derived memory is rejected |
| EVAL-A05 | Stale response, changed treatment/pipeline, fabricated candidate IDs or invalid trace is rejected |
| EVAL-A06 | Incomplete/duplicate/unpaired observations or changed result checksums cannot produce a completed report |
| EVAL-A07 | Seeded plans/bootstrap reproduce; corpus rates are micro-averaged and silence has its own denominator |
| EVAL-A08 | Archived fixture/result hashes and supported reports verify from a checkout without the original scratch workspace; failures/cancellations keep their status |
| STR-A01 | Fixed model, realtime chunks and independent timestamp labels yield latency/error/stability tradeoffs |
| STR-A02 | Internal thinking pause is not labeled a valid end merely because VAD detects silence |
| STR-A03 | Engine and rendered-preview traces are distinct; absent first partials remain visible |
| STR-A04 | Long continuous input retains session state; bounded queues/drops/resource growth are measured |
| STR-A05 | Imported HF data pins revision, original split and access terms; no test labels enter memory/context |
