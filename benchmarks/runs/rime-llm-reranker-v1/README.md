# Original Rime LLM reranker run archive

Original bytes are mapped in [path-mapping.json](path-mapping.json). `candidates/` contains actual native Rime snapshots and reference-free input requests; `final-stock/` and `typing-stock/` contain the unchanged Lua filter's real local HTTP calls. Their isolated-home logs contain only authored synthetic fixture text. `warmup-stock/` retains cold loading and unrelated warmup responses; `cache-controls/` retains separate candidate-order perturbations.

`preparation/` records model acquisition, package/runtime identities, the server's complete inference log and public configuration projection, native source snapshots and the original failed link attempt. Its `native-initial-failed/status.json` deliberately remains failed. Successful candidate collection has its own completed status. Server and model-specific settings are retained without generated authentication secrets. No runtime binaries, model weights or encrypted user profiles are included.

The [result report](../../results/rime-llm-reranker-v1/README.md) describes metrics and evidence boundaries. Use `pnpm ime:rerank verify` for model-free checksum, candidate provenance and report replay.
