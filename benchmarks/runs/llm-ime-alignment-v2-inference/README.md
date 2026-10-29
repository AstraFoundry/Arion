# Span-aware local LLM ranking results

The existing Qwen3.5-2B Q4_K_M local worker improves first-choice accuracy from **56/100 to 63/100**, with **21 rescues and 14 breakages**, on historical native candidates from the exposed authored corpus. Its 76 scored calls take **416.52 ms p50 / 484.36 ms p95 / 502.53 ms p99**. No automatic keyboard promotion is enabled by this result.

| Arm | Baseline correct | Ranked correct | Rescues | Breakages | Model calls | Errors |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Original context | 56/100 | 63/100 | 21 | 14 | 76 | 0 |
| Context removed | 56/100 | 56/100 | 0 | 0 | 0 | 0 |

Both all-candidate and eligible-candidate oracle recall are 94/100. Context-arm MRR improves from 0.7086 to 0.7611. Two unrelated warmups are excluded from scored latency; 78 actual model calls occurred in total. No-context abstention is policy behavior, not a model quality result. Fourteen destructive first-choice changes and roughly half-second calls block default activation despite aggregate improvement.

## Boundary and reproducibility

The [candidate archive](../llm-ime-alignment-v2/README.md) supplies exact native text/type/start/end, including different-length alternatives that consume the same input. The scorer validates stable full permutations, original identity, real range equivalence, finite scores, explicit failures and no-context abstention. It does not infer coverage from text length. Targets remain evaluator-only. Error observations remain in denominators with baseline display order.

This is not a controlled head-to-head comparison with v1's 68/100: candidate generation, coverage eligibility and request order changed. Neither study is an independent holdout. No installed IMK/TSF event loop, rendering latency, paced typing, ASR/rewrite contention, personalized history, Windows inference or cloud service was measured. The assistant wrote evaluator source during inference but ran no competing build or inference; the host was not otherwise isolated.

The production local driver/worker were rebuilt with CGO disabled. Before execution, the cached b11734 runtime archive matched the pinned SHA-256 and every selected executable/library matched its archive member; the driver verified the model hash. Environment metadata records binary hashes, exact invocation and copied Go entry/adapter/core sources. Sources remained unchanged during execution. The archive excludes weights and executables. Entry source snapshots do not claim a hermetic bit-for-bit Go dependency rebuild.

`archive.json` retains 142 original execution/source files and four separately identified post-run evaluator files. `run/report.json` is derived from original observations; replay and rejection tests passed.

```sh
node --test benchmarks/reproduction/llm-ime-alignment-v2/report.test.mjs
node benchmarks/reproduction/llm-ime-alignment-v2/verify.mjs benchmarks/runs/llm-ime-alignment-v2 benchmarks/runs/llm-ime-alignment-v2-inference
```

The runtime is reusable; the current generic selector is not yet a qualified replacement for the native ranking policy. Keep ordinary producer/personal ranking active while evaluating input-specific scoring and independent quality/latency gates.

## Compact preservation

`evidence.tar.gz` contains the original files, original archive manifest and source snapshots without changing their bytes. `bundle.json` records the originating commit, bundle SHA-256 and member count. The verifier checks this identity before temporary extraction, checks every original manifest member, and removes the temporary directory after replay. These results predate the integrated main English producer. They must not be presented as current-main quality or acceptance.
