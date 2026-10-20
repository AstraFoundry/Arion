# Youtu-LLM-2B Q8 development comparison

Both configurations are rejected. All202 actual outputs and12 native responses,29 unique failed full-source/context/output tuples and58 changed canonical-control pairs were personally reviewed. Assistant review is not independent blind release acceptance.

| Configuration | Conforming /101 | Harmful | Missed repair cases | Required items repaired /47 |
| --- | ---: | ---: | ---: | ---: |
| Prior Qwen baseline |92|3|6|41|
| Youtu baseline |75|3|23|24|
| Prior Qwen balanced |94|4|3|44|
| Youtu balanced |84|4|13|34|

Five paired choices improve,32 regress and21 changes are both-correct normalizations. Preserving explicit cash meaning improves over MiniCPM, but sufficient context still fails to repair load cash, meeting-minute spellings and acceptance-report spellings. Incorrect OCR still replaces server configuration with plant cultivation. No formatter/fresh expansion follows.

The official Q8 artifact is pinned to Tencent GGUF revision `698ad804316c9b815f9881a709d583c6214e0927`,2,089,462,528bytes, SHA256 `621a3484fa5da1615dd0c65d4b091b32db09eb69ca94c3b61da1b38a894a76a0`. Actual418 tensors total1,961,560,064 elements, architecture deepseek2. Upstream revision `8b0e73594661a945a7f5aafc737b63e3d8ad3f75` and embedded/native template exactly match. No4B or olderQwen inference. Different architecture/tokenizer/template/quantization prevents isolated parameter-count or quantization causality.

The custom Youtu license is authorized for local research only. Its Section0 includes an EU territorial clause; production usage and redistribution restrictions remain unresolved. Original upstream/GGUF license bytes and receipts are retained; their text matches after CRLF/LF normalization, not original-byte identity.

A570-file freeze precedes12 probes. Actual Metal99/CPU0 logs confirm33/33 and0/33 GPU layers, nonthinking empty prefix and exact template identity. Twelve responses normally stop with valid JSON and cache-off/on/off parity; all keep. Four negative controls are correct and eight positive repairs are missed, including free-output controls. These repeat two exposed sources and qualify compatibility only. CPU0 is diagnostic, not an automatic fallback, and may retain operation offload.

A supplemental source freeze precedes202 comparison requests; the original native freeze stays unchanged. Immutable executors are reused with owned model identity; legacy labels remain explicit. Full messages, ten original examples, complete OCR, candidates, schemas, temperature0/seed42 and128-token budget match prior actual controls. Twelve forced-original ambiguity schemas per arm are not autonomous semantic decisions. Unchanged spelling and supported same-letter readCash normalization both receive credit; unsupported readCache does not.

457 actual historical Go preparations are reused across914 outcomes, including712 explicit no-hypothesis skips. No new cleanup, ASR, Vision, final bodies or native insertion. All raw outputs remain visible with no semantic rejection or source fallback. M4/32GB development observations do not establish stop-to-insertion latency, concurrent ASR, whole-host peak memory or actual Apple Silicon16GB acceptance. Production prompts, validators, provider and integration are unchanged.

```sh
pnpm rewrite:selector-youtu-v1:test
pnpm rewrite:selector-youtu-v1:verify
pnpm rewrite:selector-youtu-v1:report --out .workspace/youtu-replay-new
```

Matching Make targets use hyphens and ARGS. Fresh inference requires the pinned private weights/runtime and new exclusive directories. Run preflight.py, personally inspect all native responses and apply the exact qualification in original-qualify.py.txt before run.py. The shared executor does not itself enforce this gate; original-launch.py.txt records the sequential check. No archive or control regeneration is needed for report replay.

A passing selector would only permit complete457-source final-body chain and unchanged fresh/currentASR+Vision/cross-script/native focus/cancellation/offline/concurrent-ASR/independent/actual16GB acceptance. Original release requirements remain open.
