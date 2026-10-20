# rewrite-pronunciation-evidence-v1

48 outputs reviewed: faithful20→21/24, scene-ready18→19/24, required repairs6→7/7. Three ambiguous matches still guessed despite explicit metadata; failed gate. Fresh24 ungenerated.

Latest Qwen3.5-2B text-only GGUF, actual1,881,825,088 tensor elements, CPU4/GPU0, non-thinking, cache off. No4B or older Qwen inference. Every generated body personally inspected; full source, complete OCR, raw native request, scene, template, output and review retained. Scene is fixture assigned, not measured automatic detection. No content validator or silent fallback. No production adoption.

Minimum target confirmed by user: Apple Silicon16GB. Current M4/32GB is development evidence, not minimum-host acceptance. Post-response native RSS samples are not peaks or total application memory. Text request timing excludes ASR and native insertion. Identical-request negative controls also slow down in the second arm; serial timing drift prevents attributing the whole difference to the treatment. Independent review, natural ASR and native concurrency remain outstanding.

```sh
pnpm rewrite:pronunciation:report --out .workspace/rewrite-pronunciation-evidence-v1-replay
pnpm rewrite:pronunciation:test
pnpm rewrite:pronunciation:verify
```

Replay output directory must be new. Model/runtime binaries are separately provisioned and hash pinned. Reports and tests do not launch inference. Frozen inputs and failed selection are preserved.

Derived CMU English pronunciation hypotheses are input metadata, not independent acoustic evidence. Original licensed dictionary bytes and revision provenance are archived. See `protocol-clarification.json`: underscore identifiers and Chinese names are unsupported; unknown dictionary candidates are skipped, so unique known matches are not exhaustive truth. Quoted content and exact identifiers have explicit controls. Serializer reconstruction is scoped and restored after checking; tampering with actual requests or derived metadata is rejected. Dictionary initialization is included in preparation time, but Python dictionary RSS is not included in native-worker samples.
