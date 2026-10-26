# General LLM keyboard-ranking alignment: measured pilot

The default experimental Go policy improves top-1 from **56/100 to 68/100**, with **20 rescues and 8 breakages** on the reused authored corpus. Its 76 actual scored model calls take **403.91 ms p50 / 514.92 ms p95**. This is development evidence, insufficient for default product enablement.

The implementation reuses the resident Qwen3.5-2B Q4_K_M through the existing `arion-llm` pipe worker and llama.cpp b11734 on Apple M4 / 32 GiB, native arm64 macOS. It uses an independent ID-only selection prompt, strict validation, coverage protection and a stable full permutation. The installed input method and production keyboard settings are unchanged. [Full architecture, integration sequence and evaluation gates](../../../docs/research/llm-ime-alignment.md).

## Quality and policy

| Treatment | Top-1 / 100 | Rescues | Breakages | Scored model calls |
| --- | ---: | ---: | ---: | ---: |
| Frozen real Rime producer | 56 | — | — | 0 |
| Initial ungated LLM, supplied context | 67 | 20 | 9 | 100 |
| Initial ungated LLM, context removed | 46 | 1 | 11 | 100 |
| Default policy, supplied context | 68 | 20 | 8 | 76 |
| Default policy, context removed | 56 | 0 | 0 | 0 |

The default policy explicitly abstains without context. Twenty-four corpus cases already have no context, so the supplied-context arm makes only 76 calls. The removed-context default row is a no-inference policy control, not an LLM capability score. The policy was chosen after the initial exploratory run; neither run is a held-out estimate. Both full runs and their raw outputs are retained.

Producer top-1 MRR is 0.7014; default supplied-context MRR is 0.7734. All nine-candidate oracle recall and eligible-set oracle recall are both 92/100. Missing targets remain in denominators. An empty personal context store preserves producer order; no separately enrolled personal-history baseline was measured and no evaluation-derived history was created.

The eight scored breakages are 课程→开创, 公式→公事 (two context variants), 治理→致力, 一直→一致, 申请→神情, 意义→一一 and 制定→制订. Some annotations admit near-synonymous or context-dependent alternatives; independent review is still needed. These errors are retained rather than relabeled after inference.

The actual candidate snapshots lack consumed-Pinyin spans. Equal Unicode scalar length is a compatibility proxy; this study does not prove that every eligible candidate consumes exactly the same input. It does prevent the original Lua study's common shorter-prefix promotion in these fixtures. No new candidate text is generated.

## Latency and real-time scheduling

The default 76 scored model calls have p50/p95/p99 **403.91 / 514.92 / 524.37 ms**. The mixed supplied-context arm, including abstentions, has 380.17 / 511.90 / 522.43 ms; those mixed numbers are not model-only latency. Removed-context abstentions have p50 0.011 ms and are not inference. The loaded worker reports approximately 1.02 s model load in the default run; its two unrelated warmups take 330.28 and 316.65 ms and are excluded from quality/latency aggregates.

A separate real monotonic replay submits **142 native prefix snapshots from 22 overlapping cases**, at 100 ms inter-key offsets with 1,000 ms inter-fixture pauses. It uses the actual local model, the 75 ms debounce and 1,500 ms deadline, and the one-running/one-pending scheduler.

| Current schedule observation | Result |
| --- | ---: |
| Input snapshots / outcome records | 142 / 142 |
| Actual model attempts | 108 |
| Cancelled/failed model attempts | 88 |
| Current proposals accepted by the Go gate | 54 |
| Stale proposals accepted | 0 |
| Sum of model-call wall intervals, including cancellations | 13,816.50 ms |
| Submission lateness p95 / maximum | 1.13 / 1.21 ms |
| Final-snapshot top-1, producer → policy | 11/22 → 12/22 |

The preliminary schedule is also retained: 108 attempts, 89 cancellations, zero stale accepts, final top-1 11/22→13/22. Schedules differ in completion timing; these are overlapping observations, not independent quality samples. The modest final-snapshot gain under paced typing must not be replaced with the larger unpaced final-input gain.

There are **496 real model attempts** across both quality runs (202 and 78, including four warmups) and both schedules (108 each). Cancelled work counts. No actual cloud/subscription/MLX requests, concurrent ASR, native IMK events or rendered candidate latency were measured. API tests use a synthetic local HTTP server and cannot establish real-service acceptance.

## Evidence and reproduction

- [Default-policy per-case report](gated-summary.json)
- [Initial ungated report](pilot-summary.json)
- [Current scheduled replay](schedule-summary.json)
- [Preliminary scheduled replay](schedule-preliminary-summary.json)
- [Source identity](source-snapshot.json)
- [Raw requests, responses, failures and event traces](../../runs/llm-ime-alignment-v1/README.md)
- [Reproduction commands and source](../../reproduction/llm-ime-alignment-v1/README.md)

```sh
pnpm ime:align verify
pnpm ime:align test
```

Verification checks frozen file hashes, the original candidate/label source hashes, exact reference-free request derivation, four report replays and historical entry-source hashes. It requires no model, runtime or original `.workspace`. Unit/race tests separately cover strict JSON, identity/span validation, no-context abstention, latest-request replacement, cancellation, deadlines, navigation freeze, provider protocol and incomplete output. Product rollout remains blocked by the observed breakages and outstanding native/concurrency/privacy gates.
