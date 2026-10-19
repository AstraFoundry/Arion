# Original Pinyin baseline attempt: one defective authored fixture

This immutable archive preserves all 600 completed native-process observations from the first 100-case cold/warm, three-repeat engine-only diagnostic. Its 70% exact-commit aggregate includes one known input annotation error: `sentence-029` mistakenly spells 书拿来 as `...shunaliai` instead of `...shunalai`. This is not the final corrected 100-case baseline.

No original native output, corpus, source, status, request or timing byte was changed. `archive.json` maps 824 original files by original path/size/SHA-256 and records the annotation limitation separately. Native preparation and all processes completed operationally; quality misses remain ordinary observed failures. No native executable, SDK, runtime binary, user profile or key is archived.

The corrected corpus revision 2 and a full new 600-process execution with unchanged engine/driver/evaluator are preserved separately in [imk-quality-v2](../imk-quality-v2/README.md). See the [results](../../results/imk-quality-v1/README.md) for scores, process-wall outliers, limitations and exact reproduction protocol.

```sh
node benchmarks/reproduction/imk-quality-v1/verify.mjs --archive benchmarks/runs/imk-quality-v1
```

This is a model-free read-only replay of original scores, not a new inference run or a claim of actual IMK event dispatch, external-editor quality or end-to-end latency.
