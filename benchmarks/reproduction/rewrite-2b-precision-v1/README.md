# Qwen3.5-2B Q8 artifact comparison

The Q8_0 GGUF is2,012,012,800 bytes and has1,881,825,088 actual tensor elements, below the2B limit. It comes from the same pinned export repository/revision as Q4_K_M. Actual tensor names/shapes and scalar native chat-template text match. Original API/card/acquisition/inspection bytes and hashes are retained; model weights remain private. `acquisition.json` is an acquisition-only snapshot whose `inferenceExecuted:false` does not describe the subsequent separate native screen.

All36 Q8 native/restored outputs and maps were personally inspected. All36 actual requests and encodings are byte-identical to the reused Q4 private baseline; prompt-token counts also match36/36. Q8 gives32/36 faithful/usable and9/11 required repairs versus Q4 private34/36 and11/11. It removes an unrelated OCR function addition, but loses distinct conditional targets and misses recipient/buildCache corrections; cash/not-cache meaning reverses. Higher precision is not a semantic guarantee. Both owner/date grouping and natural email paragraphs remain successful, including harmless wording changes.

No candidate qualifies; fresh24 remain unused and production unchanged. All outputs are retained without rejection or source fallback. Only this pinned Qwen3.5-2B Q8 model is loaded, CPU4/GPU0, cache off, reasoning off,384 output tokens. No4B, older Qwen or Granite inference. Maximum sampled post-response workerRSS4,418,600,960 bytes is not peak or whole-app/ASR memory. Current M4/32GB is not minimum Apple Silicon16GB acceptance.

The export family/revision is shared, but independent converter-build/base-weight numeric lineage is absent. This is therefore an artifact comparison, not fully isolated causal quantization evidence. A historical serial baseline and unresolved timer/host drift also prohibit causal speed/SLO claims. Q8 acquisition completed before Q8 native inference; its network/disk work overlaps the separate earlier Q4 input-format study, which discloses that limitation.

```sh
pnpm rewrite:2b-precision:report --out .workspace/rewrite-2b-precision-new-replay
pnpm rewrite:2b-precision:test
pnpm rewrite:2b-precision:verify
```

Replay checks full frozen requests/model/scene/raw bodies/cleanup/encodings and personally authored reviews, revalidates the reused baseline, requires a fresh output directory and never launches inference. Shared paired-report code is pinned in the archive. Source acquisition is preserved by `acquire.py`; do not rerun over an existing acquisition attempt. Weights require separately verified provisioning. No natural-data, independent-rater, original-language or native product release acceptance is claimed.

A report-loader path-isolation regression was reproduced and retained under `benchmarks/runs/rewrite-data-object-v1/report-loader-failure-v1`. The scoped importer is repaired; full HTML isolation checks now pass and both study reports replay byte-identically. `report-loader-fix.json` binds before/after source and report hashes. This is a postprocessing repair with zero additional model generations; inference inputs/outputs/reviews remain unchanged.
