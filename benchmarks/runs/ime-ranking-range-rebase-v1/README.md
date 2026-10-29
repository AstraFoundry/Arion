# Range-aware request preparation after rebase

The native driver built against main `5623c2f2` plus the retained range-observation changes completed 101 isolated real Rime processes and prepared 202 reference-free requests. The current main bilingual lexicon/admission policy is retained. Candidate text, producer type and UTF-8 start/end were validated before generating requests. No model inference or installed input-source interaction occurred.

The first preparation attempt failed before completing its warmup because the workspace still contained the old PR English data. The main engine rejected that runtime at initialization. `node scripts/stage-keyboard-runtime.mjs` regenerated the pinned main runtime/data; a newly built driver and a new output directory then succeeded. Failed output/status/logs and matching source snapshots are retained separately in `failed-old-runtime/`; they do not contribute observations to the successful preparation.

`evidence.tar.gz` preserves successful requests, native observations, preparation metadata/logs, failed-attempt evidence and exact source bytes. `archive.json` inside the bundle maps each member to its size and SHA-256; `bundle.json` pins the bundle. Executables, runtime libraries, encrypted test profiles and model weights are excluded. This is source-bound request preparation, not a hermetic build claim or current model-quality result.

```sh
node benchmarks/reproduction/llm-ime-alignment-v2/verify.mjs benchmarks/runs/ime-ranking-range-rebase-v1
```

See the [preparation and replay instructions](../../reproduction/llm-ime-alignment-v2/README.md). The historical v2 inference used a different native producer; its quality scores cannot be applied to these requests. Actual installed IMK/TSF dispatch, Windows execution, independent held-out quality and rendered typing latency remain unverified.
