# Eligible hybrid-model native and selector comparison

This study tests another actual sub2B open checkpoint: official Granite4.0-h-1b Q4_K_M,1,461,538,368 tensor elements and901,162,208 bytes. It is Apache2, lists Chinese support, and has40 blocks (4 attention,36 Mamba,zero MoE experts). The GGUF hash, upstream/export revisions, actual tensor header and exact upstream/embedded template equality are retained. This is distinct from the previously rejected1.63B dense Granite4.0-1b. No4B or olderQwen is tested; model naming or quantized file size does not stand in for full parameter count.

The345-file freeze precedes native inference and covers a12-probe compatibility phase followed conditionally by202 selector requests. The adapter reuses the unchanged frozen model-comparison executor, whose internal pipeline/worker label remains `selector-model-v1`; per-row prompts deliberately retain the shared screen template identity, while the owned freeze and actual native command bind this hybrid model. Both phases use nativeb11734, CPU4,4096 context, parallel1, greedy42, reasoningoff,128tokens and complete existing prompt/source/context. Source preparations are replayed from457 original actual Go observations; no newGo,ASR,Vision,formatter or final-body inference occurs. All914 source-treatment outcomes retain712 no-hypothesis skips, including natural120 sources per arm. Cross-script candidate misses remain explicit.

## Native compatibility, not semantic acceptance

Metal99 and CPU-layer0 each execute six probes over two exposed sources: positive cache-off, a different negative source, positive cache-on return, positive cache-off return, and unconstrained-schema positive/negative requests. Actual placements are41/41 GPU layers and0/41 respectively; CPU-layer0 may still use default operation offload. Both actual templates match pinned upstream/embedded text.

All12 return valid JSON and normal stop; the three schema-positive cache-off/on/off raw outputs match exactly in each placement. Eight choices match their limited task. Four negative choices wrongly change explicit cash/not-cache wording into readCache, including both no-schema probes. Grammar forcing and cache difference do not explain away those semantic failures. This is a narrow compatibility/cache observation, not12 independent quality samples, all-cache-behavior proof or release acceptance. No CPU fallback is introduced. Qualification permits only the preregistered Metal selector comparison.

The HTML uses a derived runtime view bound to all12 original rows. It adds no inference, final-body or semantic-fidelity credit. Original requests, responses, native commands, logs, template props and sampled worker RSS remain separate from the task labels.

## Complete paired selector result

| Model / arm | Conforming /101 | Harmful cases | Missed cases | Required repairs /47 |
| --- | ---: | ---: | ---: | ---: |
| prior Qwen / baseline |92|3|6|41|
| hybrid / baseline |81|20|0|47|
| prior Qwen / balanced |94|4|3|44|
| hybrid / balanced |82|19|0|47|

Every202 hybrid request and prepared input matches its prior actual Qwen counterpart exactly. Qwen is reused comparison evidence, not regenerated/newly counted responses. Forty-one choices change:9 improve,32 regress. Both hybrid arms are rejected. Repairing all required positive items is insufficient when unrelated/hostile OCR overrides source meaning. Examples include a destination becoming an interaction verb, nursery cultivation becoming system configuration, legality becoming a drinking method, and cash becoming cache despite an explicit definition. Allowed same-letter canonicalization is accepted; source-word overlap is not the criterion.

All202 raw choices/edited sources and all12 native outputs were personally read. All19 unique failed source/context/output tuples and41 changed paired source/context/choices were separately reviewed. These are assistant development judgments, not independent blind acceptance. Each arm inherits12 forced-original ambiguity schemas; they are protocol observations, not free semantic choices. No semantic output rejection/source fallback is added; existing identity/enum/exact-span application remains. Production prompts/provider/validator/native integration are unchanged.

This changes checkpoint, tokenizer and architecture, so neither parameter-count nor precision-only causal conclusions follow. Serial request timings and sampled post-response RSS exclude startup/native insertion/concurrentASR and are not peak/whole-host/min16 acceptance. Repeated exposed sources are not held-out generalization proof. Full457-source chain/fresh/currentASR+Vision/native lifecycle/offline/concurrent/independent/actual Apple Silicon16GB requirements remain open on the M4/32GB host.

## Reproduction

- `pnpm rewrite:selector-hybrid-v1:preflight --out <new-private-directory>` runs the frozen12 probes.
- `pnpm rewrite:selector-hybrid-v1:run --out <new-private-directory>` runs202 selector requests after the protocol's compatibility gate.
- `pnpm rewrite:selector-hybrid-v1:test` reuses the original model-screen corruption tests and adds native-template/cache/task-scope checks.
- `pnpm rewrite:selector-hybrid-v1:verify` verifies archived hashes without private weights or inference.
- `pnpm rewrite:selector-hybrid-v1:report --out <new-directory>` replays complete evidence without inference.

The new policy-driven model-screen verifier preserves the immutable historical CPU checker while parameterizing actual template/layer placement for new experiments. A missing-json-import failure in this newly factored verifier was caught after inference; its original source/error record is retained, then fixed and retested. It generated zero model responses and changed no frozen input or raw output. Historical replay sources remain unchanged.

The next untested prompt hypothesis is trusted instruction language. The pinned official card describes predominantly English instruction tuning with multilingual data. Translating the existing system rules into English while preserving every Chinese example/source/context/schema would test that factor separately. No improvement is claimed; neither current arm is adopted.
