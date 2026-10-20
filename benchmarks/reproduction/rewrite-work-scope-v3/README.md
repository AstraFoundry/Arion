# Work formatting scope v3

126 actual responses:94 historical candidates and16 new controls each with baseline/candidate. Historical93/94 faithful and85 useful. API background omission is repaired but the unchanged-owner requirement is newly omitted; no net score improvement and no adoption.

All raw responses were personally reviewed against complete source/body pairs, without validator rejection or source fallback. New controls are source-only synthetic development inputs: no Go cleanup, OCR or selector execution. Historical inputs reuse actual prior cleanup and spelling choices by raw-row identity. The other307 scenes are not newly generated in this study.

Same pinned1.88B Q4 model/native b11734 runtime, Metal99/CPU4/ctx4096/greedyseed42/nonthinking/cachetrue/256token budget. Actual startup and raw timings retained; formatting request time excludes reused stages and does not establish native/concurrent/16GB latency.

```sh
pnpm rewrite:work-scope-v3:report --out .workspace/rewrite-work-scope-v3/replay-new
pnpm rewrite:work-scope-v3:test
pnpm rewrite:work-scope-v3:verify
```

[Every actual output](../../runs/rewrite-work-scope-v3/report/review.md) and per-source judgments are retained. No new ASR/OCR/full-chain/native/concurrent/independent ratings/minimum16GB/fresh24/production acceptance or4B/olderQwen/Granite/Q8 inference. Offline replay needs no weights; native reproduction needs the pinned private provisioned model/runtime.
