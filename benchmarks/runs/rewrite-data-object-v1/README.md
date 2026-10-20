# Reference-key and uniform-data-object ablation

All72 new native/restored outputs personally inspected on both complete36-case exposed Chinese regression arms. The prior private-map Q4 baseline36 is reused by exact result/review hashes, not counted as new inference.

Lowercase reference keys alone give34/36 faithful/usable and11/11 required repairs, matching the aggregate baseline but retaining unrelated OCR function text and reversing a cash/not-cache definition that the baseline preserved. The uniform JSON-input package gives31/36 faithful/usable and9/11 repairs: it removes the unrelated function text but loses two recipient repairs and distinct multi-term/conditional investigation targets. These are definite semantic regressions, not just an unresolved naming-normalization preference. Both fail; fresh24 remain unused, production unchanged.

The lowercase arm changes only current context-object key casing. All8 no-context requests and outputs are byte-identical to the baseline. The object arm converts demonstration/current user turns together and updates the trusted system field selector; it is input JSON, not a constrained JSON output protocol. Full OCR, private maps, raw ASR-like source, actual production cleanup and unrestricted natural output remain. No output rejection or source fallback. Cosmetic changes and meaning-preserving developer-task rewording are allowed; lexical overlap is not a gate. A valid data object does not establish semantic protection.

Only pinned Qwen3.5-2B Q4_K_M/CPU4/GPU0/cacheoff/reasoningoff/384 tokens. No4B, older Qwen or Granite inference. CurrentM4/32GB is not minimum Apple Silicon16GB acceptance. Baseline is historical; serial drift and unresolved clocks prohibit causal latency claims. Q8 acquisition network/disk activity overlaps part of this run and is separately retained, further limiting performance interpretation. Sampled worker RSS is not peak/total app+ASR. No original-language, natural-ASR, native safety or independent-rater approval follows.

```sh
pnpm rewrite:data-object:report --out .workspace/rewrite-data-object-new-replay
pnpm rewrite:data-object:test
pnpm rewrite:data-object:verify
```

Replay validates the complete frozen requests, source/cleanup, model, scene, observations and manually authored reviews without launching inference; a new output directory is required. Baseline hashes are validated independently. Reused paired-report logic is also used by the separate Q8 artifact screen with explicit study-specific scope.

A report-loader path-isolation regression was reproduced and retained under `benchmarks/runs/rewrite-data-object-v1/report-loader-failure-v1`. The scoped importer is repaired; full HTML isolation checks now pass and both study reports replay byte-identically. `report-loader-fix.json` binds before/after source and report hashes. This is a postprocessing repair with zero additional model generations; inference inputs/outputs/reviews remain unchanged.
