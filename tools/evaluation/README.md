# Evaluation tooling

Dependency-free Node workspace managed by pnpm. Tests exercise metric oracles, isolation guards and driver transport; they do not execute speech models. Start with [008 Evaluation](../../specs/008-evaluation.md), [methodology](../../docs/research/benchmark-methodology.md), [streaming datasets](../../docs/research/streaming-datasets.md) and [E2E](../../docs/quality/e2e.md).

## Commands

```sh
make install
make check
make desktop-check
make e2e
make ablation-plan
make stream-plan
make datasets-check
make bench-validate CORPUS=/absolute/private/corpus.json
make bench-run CORPUS=/absolute/private/corpus.json DRIVER=/absolute/private/driver.json OUT=.workspace/benchmarks/run-a
make stream-bench CORPUS=/absolute/private/stream-corpus.json DRIVER=/absolute/private/driver.json OUT=.workspace/benchmarks/stream-a
make bench-report RESULTS=.workspace/benchmarks/run-a
make e2e-scenarios DRIVER=/absolute/private/os-driver.json OUT=.workspace/e2e/product-a
```

Plans write with exclusive creation; choose `PLAN` explicitly for another plan.

Arion's own drivers live in `apps/desktop/cmd`: `arion-eval-driver` runs the production dictation pipeline (coordinator, sherpa-onnx recognizer and finalizer, correction) on real-time fixture replay; `arion-e2e-driver` runs owned product scenarios against the packaged `Arion.app` and an AppKit editor fixture. [`benchmarks/configs/desktop-models.json`](../../benchmarks/configs/desktop-models.json) compares recognition bundles: variants differ only by `pipelineOverrides.asr`, a pinned bundle ID; the other overrides remain numeric streaming controls. `node scripts/catalog-evaluation.mjs <run>` copies a completed, checksum-verified report's aggregates into `models/catalog.json` and `benchmarks/results/`.

The private corpus is built by [`data/build_corpus.py`](data/build_corpus.py) (run with `python3 -I`; needs pyarrow and ffmpeg, and macOS `say` for the synthetic set) from pinned downloads: AISHELL-1 test speakers (recovered by unique transcript match, partitioned into dev/test speaker groups), LibriSpeech validation/test clean, synthetic Mandarin–English technical prompts read by system voices (voices split by split), and generated silence/noise. It refuses to overwrite an existing corpus. Runs require a new output directory. No inference driver, dataset audio or production weights are bundled. No command auto-downloads them. See `make help` for fixed flows. Without Make, use `pnpm bench:run --config ... --corpus ... --driver ... --out ...`, `pnpm e2e:scenarios --suite tests/e2e/desktop-scenarios.json --driver ... --out ...` and the other package scripts directly.

## Contextual rewrite commands

The separate [rewrite contract and complete fixture/driver schemas](../../docs/research/rewrite-evaluation.md) extend the JSONL transport without changing the ASR protocol. Plans and tests do not run an LLM. Corpus annotations, prompt versions, native drivers and model bytes must be provisioned and independently reviewed before a live run.

```sh
make rewrite-plan
make rewrite-validate CORPUS=/absolute/private/rewrite-corpus.json
make rewrite-run CORPUS=/absolute/private/rewrite-corpus.json DRIVER=/absolute/private/rewrite-driver.json OUT=.workspace/benchmarks/rewrite-a
make rewrite-report RESULTS=.workspace/benchmarks/rewrite-a
make rewrite-human-export RESULTS=.workspace/benchmarks/rewrite-a OUT=.workspace/benchmarks/rewrite-review-a RATERS=3
make rewrite-human-report RESULTS=.workspace/benchmarks/rewrite-a REVIEW=.workspace/benchmarks/rewrite-review-a RATINGS=/absolute/private/ratings.json
make rewrite-e2e DRIVER=/absolute/private/rewrite-os-driver.json OUT=.workspace/e2e/rewrite-a
```

Use `REWRITE_CONFIG` and `REWRITE_PLAN` to select another experiment or a new plan path. Output directories are exclusive; keep raw source/prompt/token/preview and human task files private. Raters receive `tasks.json`, never `mapping-private.json`. The automatic report marks semantic quality pending until independent ratings are imported. On Windows without Make, use `pnpm rewrite:plan --config benchmarks/configs/desktop-rewrite.json --out ...` and the matching `rewrite:validate`, `rewrite:run`, `rewrite:report`, `rewrite:human-export`, `rewrite:human-report`, `rewrite:e2e` scripts with the documented arguments.

## ASR corpus protocol v1

A JSON manifest contains `version: 1`, a nonempty `license`, explicit `consent: true` for the evaluator's authorized fixture use, and `samples`. Consent is a provenance assertion, not a mechanism to obtain third-party rights. Include source card/revision/access-review records alongside private manifests.

Each sample supplies:

| Field | Meaning |
| --- | --- |
| `id` | Unique letters/digits/underscore/hyphen identifier |
| `split` | `train`, `dev` or `test`; original splits, never random clips across speakers |
| `speakerId`, `sessionId` | Namespaced source identities; speaker is the bootstrap cluster |
| `language`, `unit` | `zh/cer`, `en/wer`, `mixed/mer` |
| `tags` | Condition labels such as `noise`, `internal-pause`, `long-form`, `technical-name` |
| `audio` | Local `path` relative to manifest, SHA-256 `sha256`, positive `durationMs` after conversion |
| `reference` | Verbatim gold transcript; empty only for nonspeech fixtures |
| `context` | Explicit string, empty if unavailable; never derive from the current test reference |
| `approvedMemory` | Records `{id,text,approved:true,origin,sourceSampleIds}`; origin `train`, `dev` or `user_approved`; test sources forbidden |
| `acceptableCandidates` | Optional nonempty list of gold acceptable texts; labels withheld from driver |
| `protectedSpans` | Optional `{kind,text}` entries; kinds `name/number/url/negation`; exact substrings in reference |
| `speechIntervals` | Optional acoustic speech `[startMs,endMs]` intervals relative to audio; ordered and nonoverlapping |
| `referenceTurns` | Intended user turns `{startMs,endMs}`, including internal pauses; required for spoken streaming fixtures |
| `timingSource` | Required with intervals: `{kind:"human_verified" or "forced_alignment_reviewed",sha256}` annotation artifact identity |
| `referenceTimeline` | Optional one `{text,endMs}` per normalized reference unit; same tokenization/order; enables conditional stable-unit latency |

Development runs use train warmups; test runs use dev warmups. At least `warmupCount` separate fixtures are required. Audio hashes are validated before inference. A public corpus without original speaker identities cannot support speaker-independent claims: recover metadata before import. Multi-speaker long-form recordings require a conservative, documented cluster/split mapping; do not invent independent speaker IDs per clip.

## Driver configuration

The private JSON configuration supplies `executable`, an `args` string array, `timeoutMs` (1–43200000; explicit long-form timeout) and nonempty `artifacts` entries `{id,path,sha256,license}` for the actual engine, checkpoints, tokenizer and converted binaries. Paths are relative to the configuration. Pin every artifact the driver uses. No shell command string is accepted. The same config shape is used by product E2E drivers, whose artifacts include application/host binaries.

The process is persistent and sequential. One JSON request per stdin line; exactly one JSON response per stdout line. Each includes `protocol:1` and an echoed `requestId`. Stdout is reserved for protocol. Stderr is discarded: write content-free diagnostic codes and controlled trace files instead. Requests are bounded by explicit time/output limits. The runner rejects a realtime reply that finishes before 98% of the audio duration; failures abort, preserve incomplete status and cannot become a report.

`hello` requests identify `purpose` (`benchmark` or `e2e`) and verified artifact hashes. Benchmark replies require `execution:"live"`, matching `artifacts`, `hardware:{os,cpu,powerMode,ramBytes}`, `runtime:{version,buildFlags,timer}`. Add GPU/OS build/thread/runtime/provider details to provenance. Native preview measurements require `hardware.os` `darwin` or `win32`. E2E replies additionally require `platform` and `capabilities` listed in the scenario suite.

An `infer` request carries `sampleId`, `seed`, `warmup`, `cacheState`, `variant:{id,features,...}`, effective `pipeline`, and `input:{audio,context,approvedMemory}`. Variant streaming overrides are merged into the request. Seed controls declared stochastic operations, not independent sample identity. `cacheState:"cold"` requires model-residency reset **per request**; `warm` uses resident models after warmups. Reset each sample's inference/session state in both modes. Load time is recorded separately and included in compute time for cold runs. All warmup outputs are retained, excluded from metrics.

The reply contains:

```json
{
  "protocol": 1,
  "requestId": "2",
  "execution": "live",
  "effectiveFeatures": { "memory": false, "ranker": "source", "correction": "none" },
  "effectivePipeline": { "asr": "sherpa-streaming-pinned", "sampleRate": 16000, "chunkMs": 40, "endpoint": "pinned-v1", "endpointSilenceMs": 500, "pace": "realtime", "threads": 2 },
  "cacheState": "warm",
  "rawText": "hello",
  "selectedText": "hello",
  "finalText": "hello",
  "candidates": [{ "id": "producer-1", "text": "hello", "source": "recognizer" }],
  "rankedIds": ["producer-1"],
  "events": [{ "ms": 200, "text": "hel" }, { "ms": 1050, "text": "hello", "final": true }],
  "timings": { "speechStartMs": 0, "speechEndMs": 1000, "firstPartialMs": 200, "finalMs": 1050, "reviewMs": 1060, "rankerMs": null, "loadMs": 0, "computeMs": 300 },
  "resources": { "peakRssBytes": 104857600 }
}
```

This is a protocol illustration, not a measured run. `events` are actual engine emissions, including final raw ASR at `finalMs`. All times use one monotonic clock origin. `firstPartialMs` is the first nonempty nonfinal engine event or null. Driver-reported speech onset/end are distinguished from independent annotation-based streaming metrics. `reviewMs` is when the final candidate/correction is ready for review; user dwell/commit time is excluded. `computeMs` is union elapsed inference-active wall time, excluding replay sleep, never double-counting overlapping stages. RSS is peak process RSS, not GPU memory. Source ranking supplies null ranker time; local ranking supplies measured time.

`selectedText` is the actual top ranked candidate, or raw ASR if the producer has no candidates. This automated top-1 policy is an evaluation condition, not a simulated user's choice. With correction off, `finalText` equals `selectedText`. Report raw, selected and corrected error rates separately; correction harm compares final against selected. A producer with one hypothesis remains a one-candidate producer; the driver cannot fabricate n-best.

Streaming replies additionally require `timings.audioStartMs`, actual `previewEvents:[{ms,text,final?}]`, and `streaming:{endpointEventsMs,processedSamples,droppedSamples,maxQueueAgeMs}`. The native preview trace records production view publication/render observation, not a copy of engine events. Include final raw ASR preview between engine final and review ready. Long recordings keep one continuous session with cumulative engine/preview transcript; intermediate endpoints are listed without resetting the benchmark's accumulated transcript. Include all endpoints and sample counters. Audio decoding/resampling settings and replay scheduling must be recorded in runtime provenance.

`execution:"live"` and hashes are driver attestations. The harness cannot independently prove that a dishonest driver ran a model; review the adapter and preserve OS/runtime traces. Unit tests include synthetic protocol data only; they are explicitly not benchmark/E2E inference evidence.

## Outputs

`run.json` retains configuration, hash identities, hardware/runtime and completed status. `results.jsonl` and `warmups.jsonl` are private content-bearing traces, with hashes checked before report generation. Failure leaves `failure.json` and incomplete status. `summary.json`/`summary.md` contain aggregates and paired comparisons; do not publish raw files by default. Reports reject incomplete coverage, duplicated observations and unmatched candidate sets for ranker-only claims.

E2E drivers implement `cleanup` and reply with `execution:"live", cleanupComplete:true` after restoring owned fault injections and closing owned targets/hosts. The runner attempts cleanup on both success and failure; unconfirmed cleanup prevents a passing result. Drivers also need OS signal/parent-exit cleanup, since a crashed transport cannot acknowledge cleanup.

## Qwen development diagnostics

[Qwen pipeline diagnostics](../../docs/research/qwen-pipeline-diagnostics.md) describes actual warm CPU generation, offline Qwen-ASR and frozen-source experiments on unreviewed development fixtures. Use `make qwen-env`, explicit `make qwen-provision MODEL_IDS=...`, `make qwen-diagnostic CORPUS=... OUT=...` and `make qwen-report CORPUS=... RESULTS=...`. These do not replace reviewed `rewrite-run` or native E2E acceptance. The diagnostic worker is Python tooling, not production runtime implementation.

Use `make qwen-prefix-diagnostic CORPUS=... AUDIO_CORPUS=... DRIVER=... OUT=...` for the actual ASR replay / staged prefix comparison. Its parent-clock candidate latency and token/source parity checks do not establish native preview, insertion, cancellation or drop-free capture. The temporary idle-sleep assertion belongs only to the diagnostic process.

## Live rewrite pipeline benchmarks

`make live-fixtures`, `make live-text-bench`, `make live-audio-bench`, `make live-report` and `make scene-bench` (pnpm `live:fixtures`, `live:text`, `live:audio`, `live:report`, `scene:bench`) run the production rewrite worker and dictation coordinator; see [the methodology](../../docs/research/live-rewrite-benchmark.md). `live/score-text.mjs` and `live/score-audio.mjs` score rows with the diagnostic exact-string proxies plus acoustic error, stop-to-final latency and previews before stop; `live/report.mjs` writes a content-free summary with the hash of every input. Binaries build into `.workspace/bin`; rows, fixtures and weights never leave `.workspace`.
