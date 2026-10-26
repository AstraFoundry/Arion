# Evaluation tooling

The [general LLM keyboard alignment](../../benchmarks/reproduction/llm-ime-alignment-v1/README.md) provides `pnpm ime:align` / `make ime-align ARGS='...'` for reference-free Go model execution, frozen producer comparisons, paced scheduler/model observations and exact archive replay. See the [architecture and full evaluation plan](../../docs/research/llm-ime-alignment.md); this is an experimental adapter, not native IMK acceptance.

The isolated [Rime LLM reranker study](../../benchmarks/reproduction/rime-llm-reranker-v1/README.md) provides `pnpm ime:rerank` / `make ime-rerank ARGS='...'` for native candidate collection, original-filter local inference, report replay and archive verification. Its 100 authored exploratory fixtures and actual wall-time measurements do not change production model selection or establish native IMK/UI acceptance.

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

Arion's own drivers live in `apps/desktop/cmd`: `arion-eval-driver` runs the production dictation pipeline (coordinator, native Qwen3-ASR chunked recognizer and tail flush, correction) on real-time fixture replay; `arion-e2e-driver` runs owned product scenarios against the packaged `Arion.app` and an AppKit editor fixture. [`benchmarks/configs/desktop-models.json`](../../benchmarks/configs/desktop-models.json) compares recognition bundles: variants differ only by `pipelineOverrides.asr`, a pinned bundle ID; the other overrides remain numeric streaming controls. `node scripts/catalog-evaluation.mjs <run>` copies a completed, checksum-verified report's aggregates into `models/catalog.json` and `benchmarks/results/`.

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

## Rewrite stage preservation diagnostics

[Stage preservation fixtures and report protocol](../../benchmarks/reproduction/rewrite-stage-preservation-v1/README.md) isolate recognized sources, contextual selection, formatting and final delivery. `pnpm rewrite:stages:run --arm candidate --cases /absolute/cases.json --out /absolute/new-run --runtime /absolute/runtime --model /absolute/model` freezes source, verifies model digest and records native requests. Arms `baseline`, `no-prior` and `no-screen` provide explicit treatments. `pnpm rewrite:stages:matrix` runs the documented development/validation/native-source matrix serially; it does not run microphone or editor acceptance. Use `final-validation.json` separately for the final source-intent controls.

`pnpm rewrite:stages:test` checks the model-free reporter; `rewrite:stages:report` requires hash-bound per-output review, and `rewrite:stages:verify /absolute/report` verifies its inventory. Completed runs, original failures and reviewed reports belong in the fixture archive. Assistant reviews must remain labeled as such; command success and substring presence are not semantic acceptance.

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

## Scene rewrite alignment experiments

`pnpm rewrite:alignment:build` (`make rewrite-alignment-build`) builds an isolated
production-engine overlay for the [scene rewrite study](../../benchmarks/reproduction/scene-rewrite-alignment-v1/README.md).
Provision the pinned Apple Silicon model/runtime separately; no command downloads
weights. `rewrite:alignment:run --split dev --out <new-directory>` runs calibration
and development. A test run additionally requires `--selection <dev/selection.json>`.
Both variants use the predeclared policy, including an explicit no-promotion
selection when development fails. Existing run/build directories are not overwritten.

`rewrite:alignment:report --dev <dev-run> --test <test-run> --out <new-directory>`
replays checked rows and exports unrated blind review tasks. `rewrite:alignment:verify`
and `rewrite:alignment:test` validate the archive and report integrity without
loading a model. Make exposes `rewrite-alignment-bench` with `SPLIT`, `OUT` and
optional `SELECTION`, `rewrite-alignment-report` with `DEV`, `TEST`, `OUT`, and
the corresponding verify/test targets. [The completed results](../../benchmarks/results/scene-rewrite-alignment-v1/README.md)
retain actual failures and distinguish automatic proxies from semantic release
acceptance. Production rewriting is unchanged by this experiment.

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

## Expanded correction-chain evaluation

`pnpm chain:expanded-fixtures`, `chain:expanded-run`, `chain:expanded-report` and
`chain:expanded-test` reuse the six model treatments on a new frozen 100-case
synthetic corpus. The [study workflow](../../benchmarks/reproduction/correction-chain-expanded-v2/README.md)
separates spoken cases from silence and original regression fixtures. Semantic
ratings allow paraphrase and structure; exact substring gates are not the outcome.

## Focused scene prompt experiments

`pnpm rewrite:prompt:run --phase dev|evaluation --out <new-directory>` runs the
[frozen v2 prompt study](../../benchmarks/reproduction/scene-rewrite-prompt-v2/README.md)
using the v1 isolated worker build; evaluation requires `--selection` frozen
from complete development review. `rewrite:prompt:report --dev <run> --evaluation
<run> --out <new-directory>` checks and replays every reviewed generation.
`rewrite:prompt:verify` and `rewrite:prompt:test` need no model. Make exposes
`rewrite-prompt-bench`, `rewrite-prompt-report`, `rewrite-prompt-verify` and
`rewrite-prompt-test`. Review coverage is mandatory, and assistant judgments
must not be represented as independent human ratings. The completed v2
candidate improves structure but fails fidelity acceptance; production is unchanged.

## Context-supported ASR-text correction diagnostics

`pnpm rewrite:context:run --out <new-directory>` runs three frozen local prompt
arms on 24 authored ASR-like/OCR-like inputs. `rewrite:context:structure --out
<new-directory>` reruns the companion eight-case structural comparison.
`rewrite:context:report --context <run> --structure <run> --out <new-directory>`
checks every reviewed source/context/output pair. `rewrite:context:verify` and
`rewrite:context:test` require no model. Make exposes the corresponding
`rewrite-context-bench/report/verify/test` targets. See the
[protocol and reproduction workflow](../../benchmarks/reproduction/context-repair-v1/README.md).
Supported repair is judged against intended speech, not ASR character identity;
missed corrections and harmful context overwrites are separate errors. These
text-only development diagnostics do not claim executed ASR/OCR or held-out
quality, and do not change production defaults.

Scene review reports display both the fixture-assigned scene and the effective
model scene, plus the actual worker-reported prompt version. Explicit test
assignment must not be reported as automatic scene detection. Any reconstructed
scene value must identify the frozen runner as its evidence. `scenes.json`
contains this provenance for every case; v2 CSV exports also include model scene
and template version. Report-only revisions retain the previous rendering while
leaving raw inference and review judgments unchanged.

## Prompt and text-validator removal diagnostics

`pnpm rewrite:ungated:run --phase dev|evaluation --out <new-directory>` runs the
[frozen ablation](../../benchmarks/reproduction/rewrite-no-validator-v1/README.md);
evaluation requires `--selection` from completed development review.
`rewrite:ungated:report --dev <run> --evaluation <run> --out <new-directory>`
exports every raw proposal, cleanup judgment and strict/envelope/none delivery,
including the full no-text-validator evaluation view. `rewrite:ungated:verify`
and `rewrite:ungated:test` require no model. Make provides corresponding
`rewrite-ungated-bench/report/verify/test` targets. No-text-validation is a
policy replay on the same actual inference output, not extra generation or
removal of session/deadline/cancellation ownership. The experiment failed
adoption; application behavior is unchanged.

## OCR spelling alignment prompt diagnostics

`pnpm rewrite:ocr:run --phase dev|evaluation --out <new-directory>` runs the
[frozen prompt repair study](../../benchmarks/reproduction/ocr-alignment-v2/README.md).
Evaluation requires `--selection` from completed development review.
`rewrite:ocr:parity --out <new-directory>` runs the separately frozen 12-call
cold/forward/reverse diagnostic. `rewrite:ocr:report --dev <run> --evaluation
<run> --parity <run> --out <new-directory>` validates and exports all reviewed
raw/cleanup and strict/envelope/none deliveries, with required-repair recall,
harmful context changes and correct-text damage distinct. `rewrite:ocr:verify`
and `rewrite:ocr:test` need no model. Make provides
`rewrite-ocr-bench/report/verify/test` with the documented arguments. The study
improves some repairs but fails adoption; no production defaults are changed.

## Input packaging and iteration HTML

`pnpm rewrite:packaging:build/run/report/verify/test` reproduce the
[fixed-model serializer comparison](../../benchmarks/reproduction/rewrite-packaging-v1/README.md).
Make provides `rewrite-packaging-build/bench/report/verify/test`; the report takes
`DEV`, `EVALUATION`, `OUT`. `pnpm rewrite:release:report/test` rebuild and validate
the [offline iteration report](../../benchmarks/results/rewrite-release/README.md).
The HTML includes every reviewed output and keeps text-only assistant review
separate from independent natural-data and actual macOS acceptance requirements.

`rewrite:entity:ocr/run/report/test/verify` reproduce the
[entity evidence and actual Vision diagnostic](../../benchmarks/reproduction/rewrite-entity-evidence-v1/README.md).
Make provides `rewrite-entity-ocr/bench/report/test/verify` with `PHASE`, `OUT` and
optional `OCR`. `rewrite:retrospective:verify` and `rewrite-retrospective-verify`
check the separately archived historical manual review. Original old-run proxy
scores remain unchanged; derived views preserve original row identities.

Completed historical rewrite screens can be replayed without inference using `pnpm rewrite:model-screen:report --out <new-directory>`, `pnpm rewrite:4b-packaging:report --out <new-directory>` and `pnpm rewrite:4b-entity:report --out <new-directory>`. Matching `:test` and `:verify` commands check review coverage and archive bytes. These historical4B runs predate the user constraint; no further4B or older-Qwen inference is authorized. Current model screening prioritizes latest-generation candidates <=2B.

For the latest <=2B screen use `pnpm rewrite:small-models:report --out <new-directory>`, `pnpm rewrite:small-models:test` and `pnpm rewrite:small-models:verify`. `rewrite-small-models-run MODEL=... RUNTIME=... OUT=...` uses the pinned text GGUF/runtime and explicit CPU configuration. Reports retain unfiltered raw output, full request, model/scene identities and individual judgments. This runtime/cache policy differs from historical ONNX studies.

Latest0.8B follow-up reports use `pnpm rewrite:small-prompt:report`, `rewrite:small-cache:report`, `rewrite:cache-failed:report` and `rewrite:cache-parity:report`, each with `--out <new-directory>`. Matching `:test` and `:verify` commands check complete reviews and frozen bytes. The failed logical-batch1 experiment has zero generations and cannot become a completed inference report. Cache reports separate32 measured calls from8 warmups and disclose raw-string parity, order parity, cache token counts and sampled RSS. All output is retained without content validators. These are M4 diagnostics; no weak-host or native end-to-end acceptance follows.

Latest2B ambiguity evidence: `pnpm rewrite:2b-contrast:test` and `pnpm rewrite:pronunciation:test` validate complete raw request/review bindings. The pronunciation producer has additional stdlib tests in its reproduction directory. `:report --out <new-directory>` replays without inference; `:verify` checks frozen archive hashes. All experimental outputs remain visible without content filtering. Minimum target Apple Silicon16GB; actual minimum-device and ASR concurrency acceptance remain open.

Source-plan evidence: `pnpm rewrite:source-plan:test` checks schema/request reconstruction, derived units, constructed bodies and complete reviews; `:report --out <new-directory>` replays without inference and `:verify` checks the archive. Raw JSON and body remain separate; grammar success never establishes semantic rewrite success.

Local-protection reproduction: `pnpm rewrite:local-protection:test` checks three models/treatments; `pnpm rewrite:protection-stress:test` checks the new stress supplement. Their `:report --out <new-directory>` and `:verify` commands replay without inference. Seven codec tests cover reversible UTF8 offsets, quote/exact controls, identical unaffected requests and explicit missing/duplicate token failures; invented content is deliberately not filtered. A protocol error is not by itself an LLM semantic failure.

## Current rewrite evidence

- [rewrite-current-task-v1](../../benchmarks/reproduction/rewrite-current-task-v1/README.md)
- [rewrite-system-task-probe-v1](../../benchmarks/reproduction/rewrite-system-task-probe-v1/README.md)
- [rewrite-cleanup-preflight-v1](../../benchmarks/reproduction/rewrite-cleanup-preflight-v1/README.md)
- [rewrite-natural-source-audit-v1](../../benchmarks/reproduction/rewrite-natural-source-audit-v1/README.md)
- [rewrite-timing-audit-v1](../../benchmarks/reproduction/rewrite-timing-audit-v1/README.md)

Current-task108 and system-probe12 native outputs are fully inspected. User-tail rules can be copied as editable text; trusted system placement fixes four regressions but returns only to8/12 faithful and7/12 scene-ready on the same hard subset. No candidate qualifies, and the mandatory36-case pool and fresh24 are not reduced. Removing rejection alone does not repair wrong native content. Original request messages and restored bodies are both visible in the living HTML.

The actual production-cleanup source replay changes one36-case input; prior raw-direct experiments remain unchanged and correctly scoped. Natural-ASR preparation verifies120 historical public read-speech sources, not new candidate inference or independent held-out data. Timing audit flags9/396 historical observations without attributing a sleep/runtime cause or approving latency. Minimum target remains Apple Silicon16GB; the M4/32GB host does not substitute for it. Original-language/mixed-input and native-chain acceptance remain pending.

## Cleanup-aligned rewrite diagnostics

- [rewrite-private-spans-v1](../../benchmarks/reproduction/rewrite-private-spans-v1/README.md)
- [rewrite-natural-text-review-v1](../../benchmarks/reproduction/rewrite-natural-text-review-v1/README.md)

## Input-object and2B artifact diagnostics

- [rewrite-data-object-v1](../../benchmarks/reproduction/rewrite-data-object-v1/README.md)
- [rewrite-2b-precision-v1](../../benchmarks/reproduction/rewrite-2b-precision-v1/README.md)

Candidate-preview integrity and evidence checks use `pnpm rewrite:selector-preview-v1:payload-test`, `pnpm rewrite:selector-preview-v1:test` and `pnpm rewrite:selector-preview-v1:verify`; the payload checks add no model inference.

Selector-only diagnosis uses `pnpm rewrite:selector-screen-v1:run`, `:test`, `:report` and `:verify` under the same command prefix. It retains all source preparations/skips and generates no final bodies; passing is only a prerequisite for complete-chain evaluation.

Granite runtime diagnosis uses `pnpm rewrite:selector-model-v1:report/test/verify`; the original `:run` configuration retains its first-request failure. The explicit CPU-layer follow-up uses `pnpm rewrite:selector-model-cpu-v1:run/report/test/verify`. Runtime probe responses and HTTP errors remain separate from selector choices and delivered bodies. See the [runtime evidence](../../benchmarks/reproduction/rewrite-selector-model-v1/README.md) and [CPU comparison](../../benchmarks/reproduction/rewrite-selector-model-cpu-v1/README.md).

Public referent-description diagnosis uses `pnpm rewrite:selector-referent-v1:run/report/test/verify` and `:payload-test`. The portable evidence wrapper supports the explicit `test_payload` operation. Choice conformance and description review remain separate; all descriptions and normalized choices are retained. See the [protocol and evidence](../../benchmarks/reproduction/rewrite-selector-referent-v1/README.md).

Source-order diagnosis uses `pnpm rewrite:selector-source-last-v1:run/report/test/verify` and `:payload-test`. Complete decoded inputs and nested ordering remain unchanged; every example/current top-level source moves last. All404 actual choices,457 preparations and1,828 outcomes remain source-bound. See the [protocol](../../benchmarks/reproduction/rewrite-selector-source-last-v1/README.md).

Counterbalanced whole-sentence diagnosis uses `pnpm rewrite:selector-sentence-choice-v1:run/report/test/verify` and `:payload-test`. Preserve raw A/B labels separately from normalized operations and existing exact-span application. Twelve forced-original ambiguity cases per arm are not free semantic decisions. See the [protocol](../../benchmarks/reproduction/rewrite-selector-sentence-choice-v1/README.md).

Hybrid-model diagnosis uses `pnpm rewrite:selector-hybrid-v1:preflight/run/report/test/verify`. Native probes retain actualGPU/template/cache evidence separately from semantic task success; the full comparison reuses457 actualGo preparations and prior Qwen requests. The native adapter keeps historical frozen command sources unchanged. See [complete protocol](../../benchmarks/reproduction/rewrite-selector-hybrid-v1/README.md).

Trusted instruction-language diagnosis uses `pnpm rewrite:selector-instruction-language-v1:run --model-key qwen|hybrid --out <new-private-directory>`, plus `:payload-test`, `:test`, `:report` and `:verify`. Every non-system request field remains unchanged; comparisons preserve original Chinese rows and use explicit system-language projections. See [complete protocol](../../benchmarks/reproduction/rewrite-selector-instruction-language-v1/README.md).

MiniCPM5-1B comparison uses `pnpm rewrite:selector-minicpm-v1:run/report/test/verify`, with matching Make targets. Native preflight and its manual qualification requirement are documented in the [protocol](../../benchmarks/reproduction/rewrite-selector-minicpm-v1/README.md). Reports separate202 complete choices,12 native responses and1 interrupted log-only completion; no new final bodies or production adoption.

Official F16 comparison uses `pnpm rewrite:selector-minicpm-f16-v1:run/report/test/verify` with matching Make targets. Native manual qualification and precise Q4 comparison population are documented in the [protocol](../../benchmarks/reproduction/rewrite-selector-minicpm-f16-v1/README.md).202 complete choices and12 probes are separate; both arms fail adoption.

Youtu research replay: `pnpm rewrite:selector-youtu-v1:{report,test,verify}`; fresh execution requires private pinned weights and sequential native qualification. See [protocol](../../benchmarks/reproduction/rewrite-selector-youtu-v1/README.md).

Repository link checks accept explicit HTML id anchors and legacy a-name anchors in Markdown, including original model cards. Fenced examples do not satisfy links. Regression checks: `pnpm exec node --test scripts/lib/markdown-anchors.test.mjs`.

LFM2.5 research: `pnpm rewrite:selector-lfm25-v1:{report,test,verify}`. Native metadata logs are byte evidence because diagnostic previews may split UTF8 tokens; [protocol](../../benchmarks/reproduction/rewrite-selector-lfm25-v1/README.md) preserves raw bytes and original failures.

Existing-example ordering: `pnpm rewrite:selector-demo-order-v1:{run,payload-test,test,verify,report}`. The test alias uses the documented version2 correction; original frozen test/failure remain in the [study](../../benchmarks/reproduction/rewrite-selector-demo-order-v1/README.md).404 actual choices and zero-generation native input checks are separate.

Native d1 typed-decision evidence: `pnpm rewrite:d1-evidence-v1:{report,test,verify}`. Zero output tokens still means actual model inference. [The protocol](../../benchmarks/reproduction/rewrite-d1-evidence-v1/README.md) separates native96/45 piece clipping from total context and retains direct native answers without fabricating chat completions.

## Qwen3-ASR 1.7B and accessibility context

The [native diagnostic](../../benchmarks/reproduction/qwen17-accessibility-v1/README.md) uses `benchmarks/configs/qwen17-accessibility-asr.json` with the existing production driver and ASR harness. Individual speech files share the verified rewriter provisioning path; `arion-models ensure` explicitly provisions or verifies a catalog artifact. Absolute and driver-relative artifact paths are both supported in driver attestations. No references enter ASR requests.

`pnpm qwen17:context:report --run <window-run> [--integration <integration-run>] --out <new-summary.json>` replays the hash-bound window/output review; `pnpm qwen17:context:test` checks report integrity and `pnpm qwen17:verify` verifies the archived study without private weights. Make exposes `qwen17-context-report`, `qwen17-context-test` and `qwen17-verify`. Actual macOS collection uses owned AppKit editors; Windows remains compile-checked only.

Single-backend native acceptance declares `design: "validation"` and may contain one variant. The default design remains an ablation requiring distinct treatments. `benchmarks/configs/desktop-models.json` is the native MLX validation configuration; inference-active timing includes asynchronous previews and cancelled requests, not only time spent inside capture callbacks. Archived configurations retain their original runtime contracts.

`pnpm asr:native:verify` (or `make native-asr-verify`) checks the completed native migration evidence using the shared archive verifier, without loading models. The report and remaining native-host gates are in [native ASR acceptance](../../benchmarks/results/native-asr-v1/README.md).
