# Offline Pinyin engine quality and snapshot latency

The corrected 100-fixture synthetic baseline produces the intended full commit for **71/100 inputs**. The same outputs repeat in all three shuffled repetitions and both cold/warm strata; no acceptance threshold or competitor comparison was applied. This exposes quality gaps despite the separate 22-case functional regression suite passing.

This is an **engine-only development diagnostic**: actual production `ArionKeyboardEngine`, pinned librime 1.17.0/`arion_pinyin` and the encrypted backend with isolated synthetic keys. It does not exercise OS key dispatch, IMK client marked text, rendering, Keychain, an external editor or a human typing session.

## Corpus, isolation and original error

The [100 original fixtures](../../corpora/imk-quality-v1/README.md) comprise 45 common words, 35 sentences, 10 abbreviations, 5 v/ü inputs and 5 coined compound targets. No copied text, personal data or exact functional-regression inputs were used. Single author-intended targets are not exhaustive acceptable readings: an abbreviation's common alternative can count as a miss. Coined complete targets are absent from the pinned raw dictionary, but constituent characters and sentence-generation paths may exist; this is not proof of global OOV.

Native requests receive only `id`, raw `input` and fixed unrelated warmups. References are evaluated separately. Each fixture/stratum/repetition gets a fresh process and empty encrypted profile. Warmups are cancelled without commit; completed test words cannot influence another fixture. Three seeded repeats are repeated measurements of the same 100 authored inputs, not 300 independent linguistic samples.

The first 600-process attempt contained one author spelling error: `sentence-029` used `...shunaliai` for 书拿来 instead of `...shunalai`. Its original corpus, 70% score, raw outputs and sources remain intact in the [first archive](../../runs/imk-quality-v1/README.md), explicitly marked as having a defective fixture. Corpus revision 2 corrects only that extra `i`; the native driver, engine, schema and evaluator are unchanged. All 600 processes were rerun after freezing the correction. The results below come solely from the [corrected archive](../../runs/imk-quality-v2/README.md); the original unfavorable outputs were not deleted or relabeled.

## Quality

Both strata have identical quality and zero output disagreements across repetitions. Counts below use the 100 distinct fixtures; the corresponding aggregate report retains all 300 observations per stratum.

| Fixture class | Exact intended commits | First-page oracle recall |
| --- | ---: | ---: |
| Common words | 45/45 | 100% |
| Sentences | 22/35 | 62.9% |
| Abbreviations | 0/10 | 90% |
| v/ü | 4/5 | 100% |
| Coined phrases | 0/5 | 0% |
| **All fixtures** | **71/100** | **81%** |

First-candidate top-1 equals exact-commit accuracy at **71%**. First-page MRR@9 is **0.7453**; oracle recall is limited to actual returned candidates on the first page, not the whole lexicon or a partially selected reconversion search. Corpus-microaveraged CER is **13.43%** (174 edits / 1,296 reference characters across three repeats per stratum). Normalized and raw exact-commit rates agree here. Every confirmation reached empty preedit; no missing/stalled result was excluded.

Examples retain concrete misses: 请把窗户关上 → 请把窗户观赏, 我已经发给同事了 → 我已经发给同时了, 请把音量调小一点 → 请把因两条小一点. For abbreviation `gz`, the actual first page starts 工作、各种、关注、广州: the intended city is available but not ranked first. All 29 distinct misses, successes and original candidate pages are in [cases.csv](cases.csv) and the complete native observations.

Exploratory input-cluster bootstrap intervals are **62–80%** for exact commit and **8.89–18.56%** for CER (2,000 draws, fixed seed, retaining all repeats of each sampled input). These describe variation within this authored synthetic mixture. They are not population accuracy bounds, significance tests or independently reviewed linguistic quality judgments.

## Timing and resource boundary

Hardware: Apple M4, 10 CPUs, 32 GiB, macOS 26.6.2; native arm64 executable, optimized compilation targeting macOS 14, AC power and Low Power Mode off. Start/end load averages were approximately (3.33, 4.43, 5.87) / (4.78, 4.69, 5.94); the host was shared and not CPU/power-isolated. Exact OS/toolchain/build flags, thermal/power probes, Git revision/dirty state, source hashes, runtime/library/data identities and seeded order accompany the archived run.

Cold means a fresh process/engine/session without priming. Warm also begins with fresh storage/process, then keeps its engine resident after two unrelated cancelled warmups. OS file caches were not evicted. Native initialization includes artifact integrity checks/backend registration/session creation and excludes process/dynamic-loader startup. It is measured before warmups in both conditions.

All times are milliseconds; percentiles use type-7 interpolation. Key-to-snapshot spans production `processKey` through immutable native snapshot copying, with 3,936 measured input-key events per stratum. It excludes snapshot serialization, OS event dispatch, native editor writes and rendered pixels.

| Metric | Cold p50 / p95 / p99 | Warm p50 / p95 / p99 |
| --- | ---: | ---: |
| Engine input key → snapshot | 0.0467 / 0.2963 / 0.4170 | 0.0379 / 0.2922 / 0.4086 |
| First input key → snapshot | 0.0697 / 0.0886 / 0.1116 | 0.0244 / 0.0382 / 0.0518 |
| Space confirmation → snapshot | 0.0167 / 0.0226 / 0.0253 | 0.0164 / 0.0257 / 0.0301 |
| Native initialize → session | 12.33 / 15.20 / 17.25 | 12.42 / 14.96 / 18.01 |
| Complete subprocess wall time | 19.97 / 24.20 / 28.43 | 20.50 / 25.75 / 34.42 |

The maximum input-key span is 0.764 ms cold / 0.746 ms warm. **A 1,003.08 ms warm subprocess-wall outlier remains included**, while that stratum's native key maximum stays below 1 ms; process launch, dynamic loading, scheduling and driver I/O are outside the key timer, and this evidence does not attribute the outlier's cause. Peak native-process RSS maxima are approximately 22.52 / 22.53 MiB; these exclude the rest of Arion, ASR and any concurrent model. No OS/end-to-end typing latency or deadline compliance follows from these engine numbers.

## Replay and remaining gates

[summary.json](summary.json) is a projection of the complete report. The corrected archive maps 824 original files, including all 600 native result objects, 200 reference-free requests, preparation/source snapshots and unsuccessful target matches. The first archive independently retains its original 824 files and known annotation error. Runtime libraries, executables, SDKs, weights and generated encrypted profiles stay outside the archives.

```sh
node benchmarks/reproduction/imk-quality-v1/verify.mjs --negative-tests
node benchmarks/reproduction/imk-quality-v1/verify.mjs --archive benchmarks/runs/imk-quality-v1
```

Both reports reproduce without native dependencies or network access. Negative checks reject changed raw results, duplicate pairing, incomplete runs and reference-bearing driver requests. See the [reproduction protocol](../../reproduction/imk-quality-v1/README.md) for a new actual run.

The observed failures justify further independently licensed lexicon/ranking evaluation; they do not identify a proprietary competitor algorithm or authorize tuning on these disclosed fixtures as a fresh test. Real OS activation/external-editor behavior, human-reviewed vocabulary/phrase quality, unseen distribution performance, actual rendered key-to-preedit timing and concurrent-ASR contention remain open.
