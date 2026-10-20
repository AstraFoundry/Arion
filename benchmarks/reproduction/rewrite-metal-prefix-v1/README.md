# Native prefix reuse on the complete text chain

This experiment changes only cache_prompt false to true in the frozen full-chain payloads. Same156 raw inputs, current actual Go cleanup,18 conditional actual spelling selections,156 source-only formatting calls, original full OCR, four scene workers, model/runtime/Metal99/CPU4/context4096/one slot/greedy seed42/budgets and forward order. No extra warmups or microbatch override. The earlier continuous runner is reused through a payload adapter; its pipeline/variant metadata deliberately identifies that base implementation. SourceControl continues to reference byte-identical original chain inputs. Cache treatment and freeze identity are recorded separately.

All174 new raw outputs were personally reread against complete sources and the historical uncached controls. Exact parity is a separate condition from meaning/style/repair. Nonzero native cache_n verifies reuse; it does not establish privacy or a latency SLO. No validator rejection, source fallback or hidden-reference repair is used. Meaning uncertainty and copied-source usability limits remain unchanged.

```sh
pnpm rewrite:metal-prefix-v1:report --out .workspace/rewrite-metal-prefix-v1/replay-new
pnpm rewrite:metal-prefix-v1:test
pnpm rewrite:metal-prefix-v1:verify
```

Run requires the same provisioned local model/runtime and pinned actual Go worker: `run.py --out <new-path> --cleanup-worker <pinned-worker>`. Report replay needs neither binary nor weights. Original source, request/response, logs, actual cache counters, host metadata, full review and selection decisions are archived. The exact b11734 longest-common-prefix and truncation implementation is retained in [the existing source evidence](../rewrite-timing-audit-v1/source-provenance.json).

This is one forward order on exposed synthetic36 and historical read-speech120, not independent new subjects. Timing controls are historical serial runs, not randomized or interleaved measurements; no isolated causal speedup claim. There are no dedicated cross-session canaries, reverse order, explicit cold/warm strata, real ASR/OCR capture, native insertion, concurrency, independent ratings or actual AppleSilicon16GB acceptance. Initial uncached requests remain in aggregates. Production model/prompt/provider/validator remains unchanged; fresh24 stay ungenerated. No4B, olderQwen, Granite or Q8 inference.

The pairing audit records exact baseline archive/results/row identities after inference and verifies that the pre-existing baseline archive is intact. That audit is retained as paired-controls.json; it is not retroactively described as a new pre-run control freeze.

[The zero-generation miss audit](../../runs/rewrite-metal-prefix-v1/run/first-miss-audit.json) finds10 zero-cache responses, each the first occurrence of that worker/prompt prefix. Seven development cases include such a miss. The other29 have descriptivep95≈829ms; that subset does not replace full36p95≈2420ms or prove a priming policy. Reproduce the audit with `python3 benchmarks/reproduction/rewrite-metal-prefix-v1/audit.py`.
