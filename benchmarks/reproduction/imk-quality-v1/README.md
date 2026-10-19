# Engine-only Pinyin baseline protocol

This driver measures the actual pinned `ArionKeyboardEngine`, librime 1.17.0, `arion_pinyin` data and production encrypted backend with a synthetic test key. It does not invoke InputMethodKit dispatch, an editor, Keychain, system source registration, permissions or UI. It neither downloads nor stages dependencies: prepare the reviewed runtime/SDK separately.

Freeze and compile before measuring, using new ignored output directories:

```sh
node benchmarks/reproduction/imk-quality-v1/run.mjs prepare .workspace/benchmarks/imk-quality-v1/prepare-new
node benchmarks/reproduction/imk-quality-v1/run.mjs run .workspace/benchmarks/imk-quality-v1/prepare-new .workspace/benchmarks/imk-quality-v1/run-new
```

Preparation saves exact source snapshots/hashes and a native executable digest. Execution rejects changed frozen sources, data or runtime and refuses existing output directories. It executes all 100 fixtures, three paired shuffled repeats (seeds 1701–1703), and separate cold/warm strata: 600 native processes. Every process has fresh isolated encrypted learning storage. Only reference-free `id`/`input`/`warmups` JSON crosses into the native driver. Original native result JSON, requests, subprocess outcomes and source/environment identities are retained. Failed attempts keep failed status and cannot produce a completed report.

Cold means a fresh process/engine/session with no priming. Warm uses the same fresh startup followed by two unrelated cancelled raw-input warmups, keeping that engine resident for the measured input. It is not a persistent cross-fixture personalized stream. OS file caches are not evicted. Native initialization timing includes integrity verification, backend registration and session creation but excludes process/dynamic-loader startup; subprocess wall time is separate and includes process launch, driver I/O and teardown.

Per-key latency is `mach_absolute_time` immediately before production `processKey` through the synchronous immutable `snapshot` copy. Space confirmations and warmups are recorded separately. No key pacing, OS key event, native marked-text write or rendered pixels are measured. The shared host is not CPU/power-isolated; hardware, power probes, OS/toolchains, load averages, Git dirty state and exact runtime/data/source hashes accompany the run.

`evaluate.mjs` receives references separately after every native run completes. It reports normalized exact commit rate, first-candidate top-1, first-page oracle recall/MRR@9, raw exact matches and corpus-microaveraged code-point CER with the repository's normalization. Candidate metrics inspect only the actual final-input first page, never invented alternatives. Confirmation presses follow actual highlighted candidates until preedit clears or a recorded stall/16-step limit; unfinished confirmations remain failures. Three repeats are repeated measurements of the same 100 authored inputs, not independent linguistic samples.

Type-7 p50/p95/p99 and maxima retain timing counts. Exploratory quality intervals resample authored input IDs with all repeats (2,000 draws, seed 83017). They describe variation within this synthetic mixture and do not estimate a user-population error rate. No release threshold, latency budget, held-out generalization, competitor parity, disk-cold claim or OS/end-to-end latency is inferred.

Archive original completed/failed observations and matching sources under `benchmarks/runs/`; keep binaries, SDK and generated profiles in `.workspace/`. Replay evaluation without native dependencies using the archived run directory and its frozen corpus. Never overwrite recorded outputs.
