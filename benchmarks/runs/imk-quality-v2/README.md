# Corrected Pinyin baseline: corpus revision 2

This immutable archive preserves all 600 actual native processes for 100 original synthetic inputs, three paired repeats and separate cold/warm strata. Native operations completed; intended whole commits matched 71/100 distinct inputs, identically across repetitions and strata. All 29 target misses remain present. No acceptance threshold or competitor parity is claimed.

`archive.json` maps 824 original files, including 600 original result JSON objects, 200 reference-free requests, run plan/status/environment, preparation records and exact source snapshots. Only one authored spelling error changed from the [first attempt](../imk-quality-v1/README.md): `sentence-029` removes an extra `i`. The native engine, driver, schema and evaluator stayed unchanged, and all 600 observations were rerun. Both original and corrected records remain independently replayable.

The native driver receives only `id`, raw Pinyin and unrelated warmups; references are kept in the separate evaluator. Every fixture/repeat/stratum gets a fresh process and empty synthetic-key encrypted profile, preventing committed test material from becoming another fixture's learning. Warmups are cancelled without commit. The actual production engine/runtime/schema performs conversion; no fake outputs or user Keychain/system input-source operations are used.

```sh
node benchmarks/reproduction/imk-quality-v1/verify.mjs --negative-tests
```

The model-free verifier checks all original hashes and regenerates the complete report. Temporary-copy negatives reject output corruption, duplicate pairing, incomplete status and leaked references. It neither executes native code nor reads the original scratch workspace. Native libraries/executables/SDKs/weights/profiles remain outside the archive; their digests and pinned acquisition provenance are retained.

Read the [results](../../results/imk-quality-v1/README.md), [corpus](../../corpora/imk-quality-v1/README.md) and [reproduction protocol](../../reproduction/imk-quality-v1/README.md) for definitions. Shared-host native key-to-snapshot times are not OS event/render/end-to-end latency; fresh-process strata do not flush OS caches; the single author-target synthetic set does not establish held-out population quality, release readiness or competitor parity. The warm subprocess-wall maximum 1,003.08 ms is preserved rather than removed.
