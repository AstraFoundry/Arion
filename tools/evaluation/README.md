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

The source corpus is built by [`data/build_corpus.py`](data/build_corpus.py) (run with `python3 -I`; needs pyarrow and ffmpeg, and macOS `say` for the synthetic set) from pinned downloads: AISHELL-1 test speakers (recovered by unique transcript match, partitioned into dev/test speaker groups), LibriSpeech validation/test clean, synthetic Mandarin–English technical prompts read by system voices (voices split by split), and generated silence/noise. It refuses to overwrite an existing corpus. Runs require a new output directory. Reviewed benchmark audio is retained under `benchmarks/corpora/`; production weights and runtime environments are not bundled. No command auto-downloads them. See `make help` for fixed flows. Without Make, use `pnpm bench:run --config ... --corpus ... --driver ... --out ...`, `pnpm e2e:scenarios --suite tests/e2e/desktop-scenarios.json --driver ... --out ...` and the other package scripts directly.

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

Use `REWRITE_CONFIG` and `REWRITE_PLAN` to select another experiment or a new plan path. Output directories are exclusive. Archive reusable research source/prompt/token/preview and completed historical task evidence under `benchmarks/`; keep personal recordings and active blind-review mappings private until their study is disclosed. Raters receive `tasks.json`, never `mapping-private.json`. The automatic report marks semantic quality pending until independent ratings are imported. On Windows without Make, use `pnpm rewrite:plan --config benchmarks/configs/desktop-rewrite.json --out ...` and the matching `rewrite:validate`, `rewrite:run`, `rewrite:report`, `rewrite:human-export`, `rewrite:human-report`, `rewrite:e2e` scripts with the documented arguments.

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

`run.json` retains configuration, hash identities, hardware/runtime and completed status. `results.jsonl` and `warmups.jsonl` retain actual observations, with hashes checked before report generation. Execute in scratch output paths, then archive reusable research evidence with byte identities and provenance. Failure leaves `failure.json` and incomplete status. `summary.json`/`summary.md` contain aggregates and paired comparisons; retain reusable raw research files in the versioned benchmark archive; personal data remains private. Reports reject incomplete coverage, duplicated observations and unmatched candidate sets for ranker-only claims.

E2E drivers implement `cleanup` and reply with `execution:"live", cleanupComplete:true` after restoring owned fault injections and closing owned targets/hosts. The runner attempts cleanup on both success and failure; unconfirmed cleanup prevents a passing result. Drivers also need OS signal/parent-exit cleanup, since a crashed transport cannot acknowledge cleanup.

## Phonetic correction ablations

`make phonetic-extract`, `make phonetic-bench`, `make phonetic-report` (pnpm `phonetic:extract`, `phonetic:bench`, `phonetic:report`) compare original ASR with text, text-derived pinyin, acoustic token log scores, their combination, audio-derived IPA and IPA-only correction. [The configuration](../../benchmarks/configs/phonetic-asr.json) fixes the models, decoder, evidence treatments, budgets and order before test inference. This is a separate diagnostic, not an extension of the realtime driver protocol or the production correction policy.

The acoustic worker uses sherpa's actual transducer output and Allosaurus `uni2005` with its universal `ipa` inventory. Provision a private Python environment from [`phonetic/requirements.lock`](phonetic/requirements.lock), explicitly acquire the selected phone checkpoint, and retain its source/archive/member hashes and runtime artifact manifest. The current Allosaurus loader checks its package-default model directory before respecting an alternative directory; provision the pinned checkpoint at that default path (a symlink to the private model directory is sufficient). No benchmark command installs dependencies or downloads weights. Weight acquisition for research is not approval for product redistribution.

Extraction takes `CORPUS`, `EVAL_PYTHON`, `MODELS` (ASR store), `PHONE_MODELS`, `RUNTIME_ARTIFACTS` and `OUT`. The artifact JSON is a nonempty array of `{path,sha256}` objects covering checkpoint/configuration/inventory bytes, the Python/runtime binaries and phonetic implementation. It validates the original speech corpus, isolates reference text from the worker, runs separate train warmups and extracts dev/test evidence. It records accelerated chunk replay compute time; it provides no realtime preview or insertion measurement.

Correction takes `CORPUS`, `EVAL_PYTHON` (the existing Qwen diagnostic environment), `MODELS` (LLM directory), `RUNTIME_ARTIFACTS`, `LLM_ARTIFACTS`, `EVIDENCE` (completed extraction), `PHONETIC_PREFLIGHT` and `OUT`. `LLM_ARTIFACTS` has the existing diagnostic manifest shape, with absolute paths to every model-directory file. `PHONETIC_CONFIG` optionally selects a different configuration; it must match extraction. Preflight is a completed `development-output-contract-smoke` directory containing hashed `results.jsonl` and `run.json`: one dev case in each language for every configured model/treatment, the exact prompt-controller hash and `testCases:0`. Text-bearing conditions must return plain, complete outputs; IPA-only negative-control failures remain measured outcomes. Actual IDs are checked against the corpus development split. Changed/missing/duplicate preflight evidence prevents the full run. The score threshold is a development-token quantile, not a calibrated correctness probability. All arms exclude context/memory. Development warmups are excluded, checkpoints are resident separately, and case/treatment jobs are shuffled within each checkpoint. The existing CPU Qwen worker checks tokenizer parity. Full prompts and raw proposals are retained with the research archive; provisioned environments and weights stay private.

Reporting takes `RESULTS` and a new `OUT` JSON file. It refuses incomplete runs, changed result/corpus/evidence hashes, missing or duplicate pairs and prompts that do not match their declared evidence treatment. It reuses the speech error normalization and speaker-cluster bootstrap. Archive exact research observations and publish aggregate reports; report raw CER/WER/MER, improvement/worsening, originally correct utterance corruption, silence emissions, truncation and generation timing. No N-best ranking, tone-aware acoustic pinyin, semantic-preservation gate or native lifecycle is implied. Independent spontaneous-speech and intent review remain separate work.

## Qwen development diagnostics

[Qwen pipeline diagnostics](../../docs/research/qwen-pipeline-diagnostics.md) describes actual warm CPU generation, offline Qwen-ASR and frozen-source experiments on unreviewed development fixtures. Use `make qwen-env`, explicit `make qwen-provision MODEL_IDS=...`, `make qwen-diagnostic CORPUS=... OUT=...` and `make qwen-report CORPUS=... RESULTS=...`. These do not replace reviewed `rewrite-run` or native E2E acceptance. The diagnostic worker is Python tooling, not production runtime implementation.

Use `make qwen-prefix-diagnostic CORPUS=... AUDIO_CORPUS=... DRIVER=... OUT=...` for the actual ASR replay / staged prefix comparison. Its parent-clock candidate latency and token/source parity checks do not establish native preview, insertion, cancellation or drop-free capture. The temporary idle-sleep assertion belongs only to the diagnostic process.

## Live rewrite pipeline benchmarks

`make live-fixtures`, `make live-text-bench`, `make live-audio-bench`, `make live-report` and `make scene-bench` (pnpm `live:fixtures`, `live:text`, `live:audio`, `live:report`, `scene:bench`) run the production rewrite worker and dictation coordinator; see [the methodology](../../docs/research/live-rewrite-benchmark.md). `live/score-text.mjs` and `live/score-audio.mjs` score rows with the diagnostic exact-string proxies plus acoustic error, stop-to-final latency and previews before stop; `live/report.mjs` writes a content-free summary with the hash of every input. Binaries build into `.workspace/bin`. Reusable rows and fixtures are archived under `benchmarks/`; weights, environments and unfinished executions remain in `.workspace`.

## Disfluency alignment runs

The existing `live:text`, `live:audio` and `live:report` commands also evaluate the separate speech-span cleanup stage. Text rows retain source, versioned cleanup annotations/text, model proposal, organization verdict and delivered text. `score-text.mjs` reports cleanup and delivery constraint passes independently and breaks them down by language. Languages without a declared CER/WER/MER tokenizer receive exact-string constraints only; their reference error ratio is null and excluded from the aggregate mean. A failed organization can retain successful cleanup, so an organization rejection is not itself a delivery failure.

Freeze and hash new test cases before candidate generation; previous challenge sets are regression/development data. Pin the worker, benchmark binary, cleanup source, prompt, model and runtime, and use fresh output paths for every arm. [The alignment report](../../benchmarks/results/disfluency-alignment-v2/README.md) records the paired local runs and the limits of synthetic text/TTS evaluation. This does not establish measured competitor parity or multilingual ASR coverage.

The audio driver now matches the product default (`RemoveFillers=true`). Use `FILLERS=0` only for an explicitly labelled diagnostic ablation. New audio rows record `removeFillers`; legacy rows report null, never an inferred false. Mixed policies in one run and mismatches with an explicitly declared report policy fail. The final report retains the older filler-off arm separately, including its regression.

## Recognition-stage memory research

`pnpm memory:asr` / `make memory-asr-bench` run the [retrieval and audio re-decoding study](../../benchmarks/reproduction/user-modify-asr-v2/README.md) using separately provisioned Apple Silicon MLX weights/runtime. Set `CORPUS` to the fixture directory, `MODELS` to the pinned model directory and `OUT` to a new run directory; `EVAL_PYTHON` selects the provisioned interpreter. For a final test, `MEMORY_SELECTION` supplies the development-only frozen selection JSON. pnpm also accepts the runner's explicit arguments. No command downloads weights or routes inference to a service.

`pnpm memory:asr-report` / `make memory-asr-report` need only `CORPUS`, `RESULTS` (run directory) and a new `OUT` JSON file. Explicit pnpm positional arguments follow that same order. Reports verify input/audio/call/output identities and score entity hits and wrong memory entities alongside CER/WER/MER. Adaptive rows reuse the actual first pass, so observations and actual calls have separate counts. See [the measured report](../../benchmarks/results/user-modify-asr-v2/README.md) for frozen selection, synthetic/natural-read strata, limitations and replay.

## Empty-vault and contextual memory research

`pnpm memory:loop-build` (`make memory-loop-build`) compiles the production extraction/vault and prompt/validation research helpers into `.workspace/`; set `MEMORY_BIN_DIR` for another private location. `pnpm memory:loop` (`make memory-loop-bench`) runs actual earlier-session ASR, authored user corrections, encrypted persistence/reopen and future audio using a frozen memory. See [the driver README](../../benchmarks/reproduction/user-modify-asr-v3/README.md) for required arguments and development-only selection. `pnpm memory:loop-report <fixture-root> <run> <new-json>` verifies and scores without a model. Initial empty memory, actual producing calls, write/reopen provenance and future-reference isolation are separate gates.

`pnpm memory:context` (`make memory-context-bench`) uses the same production learning code and a different 1.7B text model to compare product terms with reconstructed screen context and retrieved pronunciation examples. It does not run OCR or ASR. `pnpm memory:context-report` (`make memory-context-report`) replays its recorded generations and validator decisions. [Its README](../../benchmarks/reproduction/context-memory-v1/README.md) documents private provisioning, old asset-path mapping for new inference, frozen test selection and the distinction between raw and delivered scores. Both studies preserve original failures and have independent manifests outside the historical migration replay command.

`pnpm memory:loop-verify` / `make memory-loop-verify` verify their two archive manifests, including exact audio, observations and frozen sources. Explicit manifest paths may be supplied to the pnpm command. Verification is read-only and needs no model, private key or original workspace paths.

## Archived benchmark research

[The archive contract](../../benchmarks/migration/README.md) describes the byte-preserving research snapshot and its limitations. Corpora and recorded audio are under `benchmarks/corpora/`, raw outputs and failure metadata under `benchmarks/runs/`, and frozen scripts/evaluators under `benchmarks/reproduction/`. Original machine paths and hashes remain in historical metadata; the archive map resolves them explicitly without a local-workspace fallback.

```sh
pnpm bench:archive:verify
pnpm bench:archive:report --out .workspace/archive-report-repeat
```

Verification checks archived bytes, linked result/warmup identities, audio references, the cancelled phonetic parent and its exact completed 1.7B view, and confidence strata. Replay regenerates eight report families with the existing scorers and compares all published fields (except the live report-generation timestamp). It never runs a model or resumes cancelled checkpoints. Output directories are exclusive. Edit-learning/native, classifier, prefix/OCR and supplemental subreports are retained and hash-verified rather than regenerated by this entry point.

For a single phonetic primary report, use `node tools/evaluation/phonetic/cli.mjs report --archive benchmarks/migration/archive.json --results benchmarks/runs/phonetic-asr/phonetic-asr-v2/completed-17b-view --out <new-file>`. `--archive` is restricted to reporting; fresh inference uses its normal artifact checks. Original scripts kept as evidence can contain historical paths and imports; use current documented commands rather than editing those snapshots.

## ASR, dedicated correction and intent rewriting

`pnpm chain:env`, `chain:build`, `chain:provision`, `chain:fixtures`, `chain:run`
and `chain:report` reproduce the six-arm [correction chain diagnostic](../../benchmarks/reproduction/correction-chain-v1/README.md).
It runs actual X-ASR, local BART4CSC/FLANEC correction and the production
Qwen3-1.7B rewrite worker. BART is a spelling baseline; FLANEC top-1 input is an
explicit adaptation of its N-best task. Frozen synthetic fixtures and evaluator-only
intent labels remain separate. Per-language acoustic scores precede editing;
intent constraints and reference overlap are separate proxies. Serial engine
latency estimates do not establish native speech-to-insertion latency. Failed
provisioning and incomplete attempts are retained without successful substitutes.
