# Separately provisioned artifacts

Model weights, runtime libraries, Python environments and compiled native drivers are not bundled with the benchmark research archive. [The model catalog](../../models/catalog.json), [research candidate catalogs](../catalogs/) and the archived run/build/acquisition manifests retain versions, sources, hashes and declared terms.

`phonetic-asr/` preserves the original acoustic, LLM and runtime acquisition manifests, including historical absolute paths. The archive manifest's [excluded entries](../migration/archive.json) pin 22 local executables/other artifacts without copying their bytes. Baseline/source snapshots and the original evaluation implementation are retained in [reproduction](../reproduction/README.md).

No external binary store was populated by this migration. Hashes identify the required artifacts; they do not make them available on another checkout. Aggregate report replay verifies archived observations and needs no weights. New inference requires explicit local provisioning and the existing artifact checks; it must never substitute mocked outputs or claim to reuse an unavailable executable.

`user-modify-asr-v1/` records pinned Qwen3-ASR-1.7B source/MLX model metadata and the resolved experimental environment. Weights and runtime remain local; measured file hashes are in the [recognition-stage run](../runs/user-modify-asr-v1/run-v1/run.json).

`user-modify-asr-v2/` records primary-paper sources and the resolved retrieval experiment environment. Both runs retain model/tokenizer and runtime hashes; the [optimization report](../results/user-modify-asr-v2/README.md) links their actual observations. Downloaded papers and provisioned runtime binaries remain local.

`user-modify-asr-v3/` and `context-memory-v1/` record paper sources, model/runtime provenance and research-driver identities for [closed-loop acquisition](../results/user-modify-asr-v3/README.md) and [text-context disambiguation](../results/context-memory-v1/README.md). AES vault keys, compiled helpers, runtime environments and weights are not archived. Recorded ciphertext and authored plaintext observations retain separate hashes.
