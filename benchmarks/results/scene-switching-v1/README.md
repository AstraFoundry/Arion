# Scene switching results (v1)

These are real local CPU runs on Apple M4 (32 GiB, macOS arm64): the production `arion-llm` worker with Qwen3 1.7B INT4, greedy decoding, 4 threads, and the production scene selection (`domain.DecideScene`, `ModelScene`, `FinishScene`). [summary.json](summary.json) holds every number below with the hash of each corpus and private result file; [classifier.json](classifier.json) holds the classifier runs. The methodology is in [the live rewrite benchmark](../../../docs/research/live-rewrite-benchmark.md#scene-selection). The decision is in [ADR 012](../../../docs/decisions/012-context-scene-selection.md).

**Limits.** One developer wrote every case and label. Inputs are text, not speech. Scores are exact-string proxies for content and layout, not semantic quality, and there are no human ratings. Targets come from fixture metadata, not from capturing real apps (SCN-A04 is open). The final batch ran while unrelated processes loaded the host heavily (load average above 100, memory compressed), so its latencies are noisy and not compared.

## Scene classifier

118 labelled contexts: common mail, messaging, coding, AI-assistant and search apps and sites, plus documents and unlisted targets ([contexts](../../configs/scene-contexts.json)).

| Run | Correct | Notes |
| --- | --- | --- |
| First run | 107 / 118 | Found a real bug: any `google.com` subdomain inherited search, so `docs.google.com` was search |
| After fix | 113 / 118 | Exact-host matching plus tenant domains, and four added bundle IDs; same contexts, so not held-out |

The remaining misses need a URL path (iCloud Mail, Feishu documents on the messenger domain) or are unlisted (Superhuman, Stack Overflow, Taobao search). One decision takes about 50 ns. Of the 173 apps installed on the development Mac, 20 are recognized; the rest are mostly utilities that never receive dictation.

## Rewriting by scene

The final pipeline keeps prompt `live-v2` unchanged and applies the product's rule correction first. "Fixed chat" is the product before scene selection.

| Corpus | Fixed chat | Automatic scene | Worsened (auto) | Scene agreement |
| --- | --- | --- | --- | --- |
| **Held-out scene set** (30, written after diagnosis) | 14 | **21** | 1¹ | 30 / 30 |
| Development scene set (33, used for diagnosis) | 19 | 28 | 0 | 33 / 33 |

| Held-out passes by scene | chat | email | work | prompt | search |
| --- | --- | --- | --- | --- | --- |
| Fixed chat | 4 / 6 | 2 / 6 | 4 / 6 | 4 / 6 | 0 / 6 |
| Automatic | 4 / 6 | 4 / 6 | 4 / 6 | 4 / 6 | 5 / 6 |

Almost all of the gain comes from email layout and from search finishing. Work and prompt numbered lines remain unreliable on new cases: the 1.7B model sometimes keeps spoken ordinals inline, and the validator rejected one proposal that translated terms and one that converted a number.

¹ A WeChat message with two spoken items became a numbered list in the chat scene. The fixed-chat run produced the same output, so this is existing chat behavior, not caused by scene selection.

## Live rewrite regression

| Set | Before (live-v2) | After (scene pipeline) | Worsened |
| --- | --- | --- | --- |
| Held-out live set (30) | 27 | 27 | 0 |
| Development live set (24) | 17 | 17 | 0 |

LRW-A05 still holds: the prompt and its KV prefix are unchanged, and the ordinal-aware validator changed no verdict on these sets.

## Rejected: teaching scenes in the prompt

`live-v3` added work, prompt and search instructions plus six demonstrations. Measured with the original validator and raw text:

| Set | live-v2 | live-v3 |
| --- | --- | --- |
| Held-out live set (30) | 27 | **22** |
| Development live set (24) | 17 | 17 |
| Development scene set, automatic scene (33) | 19 | 17 |

`live-v3` also raised held-out final latency (p50 0.65 → 0.82 s on the same host before the load spike). In one coding case it answered the dictated request instead of transcribing it, which the length check rejected. Scene layout therefore moved to `ModelScene` and `FinishScene`.

## Commands

```sh
make scene-bench
make live-text-bench CORPUS=.workspace/scene/scene-heldout-b-corpus.json LLM_MODEL=.workspace/ablation-models/qwen3-1.7b SCENE=auto CORRECT=1 OUT=.workspace/benchmarks/scene/final-heldoutb-auto.jsonl
make live-report RUNS=benchmarks/configs/scene-switching-report.json OUT=benchmarks/results/scene-switching-v1/summary.json
```
