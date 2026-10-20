# Context ranking runs

`run-5/` is the result for the current sources (with the expanded grid); `run-3/` is the same before the grid; `run-1/` is the same benchmark before the candidate bar's ordering and resizing were optimized, kept for the latency comparison. Each run holds the six request/result pairs (three seeds × ranking off/on), `run.json` with source, runtime, driver and hardware hashes, the copied sources and `report.json`. `archive.json` lists every file's SHA-256. Both runs were made with uncommitted changes in the working tree (`gitDirty: true`); the copied sources are the exact inputs.

Replay: `node benchmarks/reproduction/imk-context-v1/evaluate.mjs benchmarks/runs/imk-context-v1/run-3`. Findings are in the [results](../../results/imk-context-v1/README.md).
