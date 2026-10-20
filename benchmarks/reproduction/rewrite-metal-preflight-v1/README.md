# Native Metal capability probe

One actual Qwen3.5-2B Q4 request uses the identical frozen formatter-v3 owner/deadline messages, budget and decoding, requesting99 GPU layers instead of0. Actual M4/32GB, nativearm64b11734. The unfiltered body was personally inspected: both owner/task/deadline groups survive; raw wording differs from CPU (交 versus交稿). No cross-provider byte parity claim.

External request≈1,542ms;586 input tokens,22 output tokens, cache0. This is one no-context formatting request, not p95, paired speed causality, two-stage total, full quality, concurrent ASR/native stop-to-insertion or minimum16GB acceptance. Startup is separate. One result cannot qualify a release.

Initial verbosity did not retain tensor-placement details. A separate same-artifact/provider/thread/context load-only process with verbose logging directly observes MTL0 placement,25/25 layers offloaded and model/KV/recurrent/compute buffers. It generates no text and is not retrospective telemetry of the original request. Both original logs, manifests and exact provenance are retained; next provider-quality work must capture placement upfront and test full regressions.

`python3 benchmarks/reproduction/rewrite-metal-preflight-v1/report.py` verifies without model loading. [Original observations](../../runs/rewrite-metal-preflight-v1/run/results.jsonl), [review](../../runs/rewrite-metal-preflight-v1/run/review.json) and [load evidence](../../runs/rewrite-metal-preflight-v1/load-observation/load.log) remain separate. `run.py`/`load_observation.py` require pinned provisioned binaries/weights and exclusive scratch paths. Production unchanged; new4B/olderQwen inference0.
