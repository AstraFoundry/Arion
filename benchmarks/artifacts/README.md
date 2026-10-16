# Separately provisioned artifacts

Model weights, runtime libraries, Python environments and compiled native drivers are not bundled with the benchmark research archive. [The model catalog](../../models/catalog.json), [research candidate catalogs](../catalogs/) and the archived run/build/acquisition manifests retain versions, sources, hashes and declared terms.

`phonetic-asr/` preserves the original acoustic, LLM and runtime acquisition manifests, including historical absolute paths. The archive manifest's [excluded entries](../migration/archive.json) pin 22 local executables/other artifacts without copying their bytes. Baseline/source snapshots and the original evaluation implementation are retained in [reproduction](../reproduction/README.md).

No external binary store was populated by this migration. Hashes identify the required artifacts; they do not make them available on another checkout. Aggregate report replay verifies archived observations and needs no weights. New inference requires explicit local provisioning and the existing artifact checks; it must never substitute mocked outputs or claim to reuse an unavailable executable.
