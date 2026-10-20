# rewrite-local-protection-v1

84 raw generations personally inspected. Qwen direct→protected25→28/28 faithful and usable; both9/9 repairs.25 unencoded request/body pairs are identical. Residual conversational particles are a minor recorded style issue, not a perfect-fluency claim. Granite19/28 faithful,18/28 usable,7/9 repairs; actual empty EOS and hostile OCR instruction following retained.

The main known-development gate passes for Qwen protected, but the separately frozen8-case stress fails. No qualified selection, fresh24 generation or production adoption.

Complete raw native output is separate from the restored body. Missing/duplicate protected tokens produce an explicit null body; no source fallback or semantic content validator. Wrong unencoded bodies, empty output and injection markers remain visible. A valid encoding protocol does not establish semantic correctness. The original source, full OCR, encoded source, map, request, actual model/template/scene, native response and human-authored assistant judgments are retained with hashes. Scene is fixture assigned, not measured automatic detection.

Only latest Qwen3.5-2B text GGUF (1,881,825,088 tensor elements) and Granite-4.0-H-1B (1,461,538,368) Q4_K_M are eligible; stress uses Qwen only. CPU4/GPU0, reasoning off, cache off,384 output tokens. No4B or older Qwen inference. Template scope differs from earlier rounds: common prompt examples now organize multiple requirements and teach initial oral filler removal. Within the Qwen comparison only ambiguity encoding/instruction/restoration differs; unaffected requests are byte-identical. Granite necessarily changes tokenizer and native template along with weights. Its served/exported chat-template bytes match pinned upstream, and empty output is actual single-token EOS; this does not establish export precision parity.

Text-derived CMU hypotheses are not independent acoustic evidence, and unknown candidates are not exhaustively covered. A unique known dictionary match does not prove relevance or truth. The failure on misleading unique OCR is retained explicitly. Full context is not cropped. Original licensed dictionary bytes and model/license provenance are frozen.

Current host M4/32GB is development evidence. User minimum Apple Silicon16GB has not been tested. Serial arm ordering/timing drift limits causal attribution; native post-response RSS is sampled, not peak or total app memory, and excludes Python dictionary/ASR. No real ASR concurrency, natural audio/native insertion, independent blind review or release acceptance is claimed.

```sh
pnpm rewrite:local-protection:report --out .workspace/rewrite-local-protection-v1-replay
pnpm rewrite:local-protection:test
pnpm rewrite:local-protection:verify
```

Replays never launch inference, and require a new output directory. Inference commands and frozen inputs remain in `run.py`, `freeze.json` and `protocol.json`; model/runtime binaries are separately provisioned. Preserve every failed or null result.
