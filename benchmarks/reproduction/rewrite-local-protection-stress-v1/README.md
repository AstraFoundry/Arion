# rewrite-local-protection-stress-v1

16 raw generations, all maps and restored/null bodies personally inspected. Direct→protected faithful1→4/8, usable0→3/8. Protected has2 protocol failures and0/2 delivered repairs; direct has2/2 spelling repairs but other meaning errors remain.

Multiple terms, literal quotation and explicitly cash-not-cache meaning still fail; repeated terms become duplicate tasks. Both protocol-rejected raw bodies are also incorrect: zero observed false rejection of a correct native body. Missing-marker counts alone never establish semantic failure.

Complete raw native output is separate from the restored body. Missing/duplicate protected tokens produce an explicit null body; no source fallback or semantic content validator. Wrong unencoded bodies, empty output and injection markers remain visible. A valid encoding protocol does not establish semantic correctness. The original source, full OCR, encoded source, map, request, actual model/template/scene, native response and human-authored assistant judgments are retained with hashes. Scene is fixture assigned, not measured automatic detection.

Only latest Qwen3.5-2B text GGUF (1,881,825,088 tensor elements) and Granite-4.0-H-1B (1,461,538,368) Q4_K_M are eligible; stress uses Qwen only. CPU4/GPU0, reasoning off, cache off,384 output tokens. No4B or older Qwen inference. Template scope differs from earlier rounds: common prompt examples now organize multiple requirements and teach initial oral filler removal. Within the Qwen comparison only ambiguity encoding/instruction/restoration differs; unaffected requests are byte-identical. Granite necessarily changes tokenizer and native template along with weights. Its served/exported chat-template bytes match pinned upstream, and empty output is actual single-token EOS; this does not establish export precision parity.

Text-derived CMU hypotheses are not independent acoustic evidence, and unknown candidates are not exhaustively covered. A unique known dictionary match does not prove relevance or truth. The failure on misleading unique OCR is retained explicitly. Full context is not cropped. Original licensed dictionary bytes and model/license provenance are frozen.

Current host M4/32GB is development evidence. User minimum Apple Silicon16GB has not been tested. Serial arm ordering/timing drift limits causal attribution; native post-response RSS is sampled, not peak or total app memory, and excludes Python dictionary/ASR. No real ASR concurrency, natural audio/native insertion, independent blind review or release acceptance is claimed.

```sh
pnpm rewrite:protection-stress:report --out .workspace/rewrite-local-protection-stress-v1-replay
pnpm rewrite:protection-stress:test
pnpm rewrite:protection-stress:verify
```

Replays never launch inference, and require a new output directory. Inference commands and frozen inputs remain in `run.py`, `freeze.json` and `protocol.json`; model/runtime binaries are separately provisioned. Preserve every failed or null result.
