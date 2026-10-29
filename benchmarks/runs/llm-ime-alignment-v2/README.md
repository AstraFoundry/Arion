# Current native candidates for span-aware LLM evaluation

Status: native candidate preparation completed. Subsequent [local inference and span-aware results](../llm-ime-alignment-v2-inference/README.md) are archived separately.

This archive retains 101 actual Rime engine processes from the original PR 45 implementation and 202 reference-free model requests. All 100 contextual development fixtures are included; contexts and labels are unchanged from the exposed historical corpus. Typo correction is on, fuzzy pairs off, English enabled. There are 76 nonempty-context requests with at least two candidates covering the same input range as the first choice. No model-call count or improvement is inferred from this eligibility count.

Every candidate retains engine type and actual consumed UTF-8 start/end. The model requests contain no acceptable-answer or purpose fields. Reference answers remain in the archived corpus for evaluator use only. The native process receives no context or labels. Two model warmups reuse one separate actual native warmup snapshot.

The manifest preserves 270 original files and exact source mappings. Native candidate/request replay passed; boundary tests reject malformed snapshots and reference leakage by explicit field projection. No installed host, Windows run, independent holdout, rendered latency, ASR contention or model inference is established.

```sh
node benchmarks/reproduction/llm-ime-alignment-v2/verify.mjs benchmarks/runs/llm-ime-alignment-v2
node --test benchmarks/reproduction/llm-ime-alignment-v2/requests.test.mjs
```

The next step is actual inference using the existing completion driver plus span-aware result validation. Do not reuse v1's equal-character-count scorer or its 68/100 result for these requests.

## Compact preservation

`evidence.tar.gz` contains the original files, original archive manifest and source snapshots without changing their bytes. `bundle.json` records the originating commit, bundle SHA-256 and member count. The verifier checks this identity before temporary extraction, checks every original manifest member, and removes the temporary directory after replay. These results predate the integrated main English producer. They must not be presented as current-main quality or acceptance.
