# Full historical formatter prompt revision

This experiment revises scene instructions and unrelated demonstrations for shared temporal scope, request framing, English clause organization, explicit numerical contrast and withdrawn weekdays. It runs all401 historical source cases, keeping the actual prior Go cleanup and64 spelling choices fixed by raw-row identity. Only401 formatting responses are newly generated. Synthetic281 and historical natural120 remain separate exposed strata, not held-out subjects.

The same pinned1.88B tensor Qwen3.5 Q4 model, native b11734 runtime, Metal99/CPU4/ctx4096, greedy seed42, nonthinking and256 output budget are retained. Native startup is recorded; no priming is performed. Request latency excludes reused cleanup/selection and is not full-chain or end-to-end latency. All raw bodies remain visible without validator rejection or source fallback.

```sh
pnpm rewrite:history-format-v2:report --out .workspace/rewrite-history-format-v2/replay-new
pnpm rewrite:history-format-v2:test
pnpm rewrite:history-format-v2:verify
```

[Results and every reviewed body](../../runs/rewrite-history-format-v2/report/review.md) retain both improvements and regressions. New spelling opportunities cannot be credited to this formatter-only experiment: the same missed producer hypotheses and selector decisions are fixed inputs. Source-language preservation, meaning and usable organization are separate review dimensions. Exact unchanged outputs may retain prior judgments only after actual output inspection and identity checks; no automatic overlap score establishes meaning.

No new ASR/OCR/native insertion/concurrent ASR/independent human ratings or AppleSilicon16GB acceptance. Production prompt/provider/validator remain unchanged, and no4B/olderQwen/Granite/Q8 or fresh24 inference is performed. Native replay requires provisioned pinned weights/runtime; offline evidence verification and report replay do not.
