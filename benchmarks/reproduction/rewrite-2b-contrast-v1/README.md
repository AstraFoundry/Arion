# rewrite-2b-contrast-v1

32 outputs reviewed: both15/16 faithful,13/16 scene-ready,3/3 required repairs. Same unsupported readCash guess; failed gate. Fresh24 ungenerated.

Latest Qwen3.5-2B text-only GGUF, actual1,881,825,088 tensor elements, CPU4/GPU0, non-thinking, cache off. No4B or older Qwen inference. Every generated body personally inspected; full source, complete OCR, raw native request, scene, template, output and review retained. Scene is fixture assigned, not measured automatic detection. No content validator or silent fallback. No production adoption.

Minimum target confirmed by user: Apple Silicon16GB. Current M4/32GB is development evidence, not minimum-host acceptance. Post-response native RSS samples are not peaks or total application memory. Text request timing excludes ASR and native insertion. Identical-request negative controls also slow down in the second arm; serial timing drift prevents attributing the whole difference to the treatment. Independent review, natural ASR and native concurrency remain outstanding.

```sh
pnpm rewrite:2b-contrast:report --out .workspace/rewrite-2b-contrast-v1-replay
pnpm rewrite:2b-contrast:test
pnpm rewrite:2b-contrast:verify
```

Replay output directory must be new. Model/runtime binaries are separately provisioned and hash pinned. Reports and tests do not launch inference. Frozen inputs and failed selection are preserved.
