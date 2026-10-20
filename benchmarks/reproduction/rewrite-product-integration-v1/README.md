# Selected rewrite product integration

This directory checks the port of the user-adopted experimental baseline into the macOS product. It is not another model-selection round or held-out quality benchmark.

- `producer-parity.json`: 457 exact source/context/scene preparations exported from `rewrite-selector-screen-v1/run/preparations.jsonl`. Go pronunciation hypotheses, byte offsets, ordering, kinds and unknown characters must match.
- `payload-fixtures.py`: exports 101 actual baseline selector requests and 16 work-scope-v4 formatter payloads spanning four scenes and four script profiles. It performs no inference. `payload-parity.json` preserves those expected payloads.
- Production templates are byte-identical copies in `apps/desktop/infrastructure/llm/prompt/{selection,format}.json`. The per-request `enable_thinking:false` explicitly matches the preceding runtime-level thinking-off setting; it is the only declared added request field.
- `llm/spelling` and `llm/prompt` package tests verify producer/payload parity. `llm/gguf` tests verify preparation emits no generations, oversized input fails before generation and unexpected reasoning is a protocol error.
- `llm/gguf/native_test.go` is the opt-in actual packaged-worker integration test. Set `ARION_GGUF_RUNTIME`, `ARION_GGUF_MODEL` and a new `ARION_GGUF_OBSERVATIONS` path. It checks native output, session-boundary parity, generation-free preparation, prompt cancellation and continued service after cancellation. It does not capture audio, run OS OCR or insert into another application.

Completed observations and reviewed outputs are stored in [the run directory](../../runs/rewrite-product-integration-v1/README.md). Weights/runtime environments and unfinished execution remain in `.workspace`. The selected local product model is `qwen3.5-2b-q4-k-m-metal`; retired model inference requires downloading the original pinned weights again. See [product configuration](../../../docs/operations/local-rewrite.md).

Run `python3 benchmarks/reproduction/rewrite-product-integration-v1/verify.py` for offline byte/review verification without model weights. This does not rerun inference.
