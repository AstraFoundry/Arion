# Source-indexed editable-plan package

All56 raw native generations and constructed bodies personally inspected. Latest Qwen3.5-2B text-only GGUF (1,881,825,088 tensor elements), CPU4/GPU0, non-thinking, cache off. No4B or older Qwen inference.

| Package | Faithful | Faithful and scene-ready | Required repairs | Request p95 |
| --- | --- | --- | --- | --- |
| Direct body |24/28|20/28|8/9|6992ms|
| Editable source plan |19/28|6/28|0/9|13924ms|

All plan outputs are valid JSON and untruncated. That is protocol success, not rewrite success. Three ambiguous English spans stay unresolved, but necessary English/Chinese corrections are missed, source clauses lose punctuation, free parts repeat anchors/restrictions, owner groups fail and emails collapse to one paragraph. Oral fillers and correction discourse remain copied. Do not adopt this package; the pre-frozen fresh24 are ungenerated.

This is a package comparison: source segmentation, new instructions without demonstrations, grammar and rendering change together. It does not prove all structured decoding is inferior. Source-unit keys do not prove semantic coverage; editable parts can omit or duplicate meaning. Raw JSON, unit/anchor inputs and constructed bodies are separately retained. No semantic output validator or silent fallback. A protocol failure would have null body and remain in the denominator; no failures occurred here.

Identical384-token generation caps expose plan overhead rather than equal useful-text budgets. CPU serial order and host timing drift prevent causal timing attribution to grammar alone. Native post-response RSS samples are not peaks or total application memory. The Python dictionary is excluded. No concurrent ASR or native insertion was measured. Minimum target Apple Silicon16GB; current M4/32GB is development evidence. Independent review and natural ASR remain required.

```sh
pnpm rewrite:source-plan:report --out .workspace/rewrite-source-plan-v1-replay
pnpm rewrite:source-plan:test
pnpm rewrite:source-plan:verify
```

Report replay does not launch inference; output directory must be new. Frozen inputs and all failures remain immutable. Producer tests are `python3 benchmarks/reproduction/rewrite-source-plan-v1/test_plan.py`.
