# Official MiniCPM5-1B F16 precision/export comparison

Both F16 configurations fail the unchanged development gate. All202 actual choices and12 native responses,22 unique failed full-source/context/output tuples and47 changed Q4 pairs were personally read. This assistant review is not independent blind release acceptance.

| Configuration | Conforming /101 | Harmful | Missed repair cases | Required items repaired /47 |
| --- | ---: | ---: | ---: | ---: |
| Prior Q4 baseline |80|11|10|37|
| F16 baseline |82|15|4|43|
| Prior Q4 balanced |77|4|20|27|
| F16 balanced |82|17|2|45|

F16 improves25 paired choices and regresses18;4 further changes are both-correct normalizations. Additional required repairs do not offset increased unsupported meaning changes. Examples include ordinary unrelated OCR changing the location大理 to the behavior搭理, and nursery培植 becoming system配置. Explicit cash/not-cache meaning still fails. Neither artifact qualifies; no formatter/fresh/production expansion follows.

The official F16 file is pinned to GGUF revision `3d55fac80935ae6456986ad2384b5cbcc4d6c948`,2,166,551,936 bytes, SHA256 `68c40b08b1242754a107b9510af89aa75b10c75843ca7844643c70956b7f1e3d`. Actual219 tensors total1,080,632,832 elements; no4B or olderQwen inference. Q4 and F16 share exact tensor names/shapes, embedded template and complete binary tokenizer/architecture metadata payloads, including arrays (`sources/complete-metadata-identity.json`). Q4 tensor types are0/12/14; F16 types0/1. The initial484-file freeze precedes inference. Numerical source-conversion and quantizer lineage equivalence has not been proved, so this is a precision/export artifact comparison, not strict quantization-only causality.

All202 requests, prepared candidates and input-token counts match the prior [Q4 run](../rewrite-selector-minicpm-v1/README.md). Full OCR, original ten examples, system rules, schemas, temperature0/seed42,128-token output budget and pinned b11734 settings remain unchanged.12 forced-original ambiguity schemas per arm are not autonomous semantic choices. Both unchanged spelling and supported same-letter `readCash` normalization receive credit; unsupported `readCache` does not.

Twelve native probes use Metal99 and CPU0,4 threads,4096 context, parallel1, nonthinking, no warmup, offline, with cache-off/on/off and schema-free controls. Actual logs confirm25/25 and0/25 GPU layers, pinned template after exactly one lexer-final-LF removal and the empty thinking-block prefix. All12 have valid JSON and normal stop; cache raw parity holds. Eight supported-repair probe responses are correct, while four cash-literal responses fail, including schema-free output. These repeat two exposed sources. Runtime compatibility only permits the prefrozen202-request comparison, never release. CPU0 is diagnostic, not an automatic fallback, and may retain operation offload.

457 historical actual Go preparations are reused across914 outcomes, including712 explicit no-hypothesis skips. No new cleanup, ASR, Vision, final bodies or native target insertion. All output remains visible without semantic output rejection or source fallback. Legacy executor names remain unchanged; owned model/freeze/commands identify this F16 treatment. M4/32GB serial observations do not establish stop-to-insertion latency, ASR concurrency, whole-host peak memory or actual Apple Silicon16GB acceptance.

```sh
pnpm rewrite:selector-minicpm-f16-v1:test
pnpm rewrite:selector-minicpm-f16-v1:verify
pnpm rewrite:selector-minicpm-f16-v1:report --out .workspace/minicpm-f16-replay-new
```

Matching Make targets use hyphens and `ARGS`. Fresh inference requires private pinned weights/runtime and new exclusive output directories. Execute `preflight.py --out <new-private-directory>` with Python, then apply the exact qualification checks retained in `original-qualify.py.txt` before `pnpm rewrite:selector-minicpm-f16-v1:run --out <another-new-directory>`. The shared executor does not automatically enforce this manual gate. Do not overwrite archived outputs or regenerate controls for replay.

A passing selector component would only permit complete457-source chain, unchanged fresh/currentASR+Vision, cross-script, native cancellation/focus/offline, concurrent-ASR, independent review and actual16GB acceptance. All original release requirements remain open; production prompts, validators, provider and native integration are unchanged.
