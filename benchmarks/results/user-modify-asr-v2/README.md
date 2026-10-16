# Recognition-stage memory optimization, v2

**Compact, retrieved spelling context improves new synthetic term recall from 7/20 to 11/20 on the same local Qwen3-ASR-1.7B checkpoint.** It adds no measured wrong memory entities or lexical regressions in this test. Full-list destination context also reaches 11/20, but substitutes Kubernetes for Codex once. These are diagnostic results, not natural-user quality estimates or a reproduction of Doubao's model.

## Research and intervention

The [paper analysis](../../../docs/research/user-modify-asr-optimization.md) distinguishes contextual recognition from instruction-based replacement, trained bias tags from literal wrappers, and trained acoustic retrieval from our text-derived approximation. The [previous experiment](../user-modify-asr-v1/README.md) remains unchanged.

Prior memory is byte-identical to v1: 13 authored approved records with synthetic counts. Expiry and reverse suppression leave ten destinations. The optimized policy uses the recognized first-pass text to retrieve at most two records by pinyin/lexical edit distance, with recall/use counts breaking ties. Already-present target spellings are skipped. Wrong forms are retrieval keys only; the model receives `The relevant terms are <destinations>.` together with the **original audio**. With no retrieval, the delivered output is the actual first-pass result. No text rewriting, spelling substitution, reference text or acceptable-answer list enters inference.

The old eight-entry correction-pair policy and the all-ten destination list are separate single-pass controls. The latter changes both pool coverage and formatting relative to the old pairs, so that contrast cannot isolate either factor alone. The adaptive treatment changes retrieval, formatting and call count together; this study does not attribute their gains independently.

## Development and frozen test

- The exposed 32 v1 inputs are development/regression data, never new held-out evidence. Six conditions produce 192 paired observations from 140 actual model calls, plus six warmups.
- Before development inference, [the test inputs were frozen](../../corpora/user-modify-asr-v2/freeze.json): 38 new authored inputs, including 20 recurring-term cases, other speech controls and two nonspeech fixtures. Their Eddy/Karen TTS voices differ from development's Tingting/Samantha. Twenty natural-read controls are imported deterministically from the historical `asr-v1` test corpus (ten Chinese, ten English); these are exposed general-speech regression, not new natural recurrence.
- Memory, new utterances and audio stayed fixed. [Predeclared criteria](../../runs/user-modify-asr-v2/selection-criteria.json) reject candidates with new wrong entities, previously zero-error regressions or nonspeech emissions, then rank term hits, worsened cases and pipeline p95. Development exact retrieval reaches 12/20, fuzzy retrieval 13/20 and fuzzy retrieval with the short hint 14/20; all three pass the guard criteria. [The selected policy](../../runs/user-modify-asr-v2/selection.json) and source hashes were frozen before test inference. There was no test-driven retuning.
- The final 58-input test has four conditions: 232 paired observations from 191 actual model calls, plus four warmups. Adaptive observations share the measured first-pass call and only add a call on retrieval. Across development and test: **424 observations, 331 actual model calls, ten additional warmups**.

The [development scores](dev-summary.json), [test scores and every output](test-summary.json), [raw calls and runs](../../runs/user-modify-asr-v2/), [exact inputs and provenance](../../corpora/user-modify-asr-v2/), [source/replay instructions](../../reproduction/user-modify-asr-v2/README.md) and [archive identities](archive.json) retain the evidence.

## Final test results

| Condition | Term hits / 20 | New wrong memory entities | Previously zero-error speech regressed / 34 | Improved / worsened speech | Mixed MER | Pipeline p50 / p95 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Empty context | 7 | 0 | 0 | — | 58/164 = 35.37% | 0.458 / 1.260 s |
| Old selected pairs | 7 | 0 | 2 | 3 / 3 | 50/164 = 30.49% | 0.528 / 1.549 s |
| All eligible destinations | 11 | 1 | 0 | 6 / 0 | 44/164 = 26.83% | 0.491 / 1.349 s |
| Retrieved short hint | **11** | **0** | **0** | **4 / 0** | **44/164 = 26.83%** | 0.627 / 1.264 s |

The selected policy improves term hits by four cases, or 20 percentage points; it leaves 9/20 terms unresolved. It changes none of the 20 natural-read outputs and worsens none of the 34 ordinary/homophone/reverse/natural speech controls. All conditions emit nothing on both nonspeech inputs and have zero token-limit hits. Chinese CER is 8/290 = 2.76% and English WER is 3/259 = 1.16% for empty context, full destinations and the selected policy; old pairs reach 9/290 and 4/259 respectively.

Seventeen of 58 inputs trigger a second audio recognition. On those routed inputs, median additional measured compute relative to their own first pass is **464 ms**, including retrieval. Overall p95 is dominated by longer natural-read inputs that do not route; its near equality to baseline does not make the second pass free. Timings exclude recording, endpointing and insertion, and were not obtained under isolated thermals or desktop load.

## Actual outputs and unresolved cases

| Case | Empty context | Retrieved short hint |
| --- | --- | --- |
| `fresh-redis-1` | 瑞吉斯里的缓存暂时不要删除。 | Redis里的缓存暂时不要删除。 |
| `fresh-redis-2` | 请确认瑞迪斯是否已经连接成功。 | 请确认 Redis 是否已经连接成功。 |
| `fresh-codex-1` | 让寇代克斯整理一下测试结果。 | 让Codex整理一下测试结果。 |
| `fresh-codex-2` | 后，在克斯没有修改任何配置。 | Codex没有修改任何配置。 |

The full destination list writes `让 Kubernetes 整理一下测试结果。` on `fresh-codex-1`. Its smaller lexical edit distance counts as an improvement, yet its entity is wrong. Entity review must therefore accompany CER/WER/MER; neither lexical score nor hit rate alone establishes preservation of intent.

Among 13 baseline term misses, retrieval finds the intended target in nine. Four misses (Arion twice, MyGo twice) have first-pass spellings such as `阿宇阳`, `阿里`, `麦秆` and `麦克` outside the frozen aliases/threshold. Five retrieved targets remain wrong after re-decoding (Kubernetes twice, GitHub twice, Safari once). This separates a text-retrieval bottleneck from failures after context delivery; it does not identify their acoustic or internal-model causes.

## Provenance, limits and replay

The model is the same community 8-bit conversion as v1: `mlx-community/Qwen3-ASR-1.7B-8bit`, revision `a8379a2e2f9e313c9292cdf1af4055ab56d50d55`, weight SHA-256 `bf304b009cc7eca79283056f787b44c952d24ac22cec787b39732bba3c23c13c`. MLX Audio source is pinned to `dad2087f790dfe96b07d5aac1405db4b578b411f`. Actual model/tokenizer bytes, inference sources, environment, hardware, power settings and Git state are recorded in each `run.json`. Execution used Apple M4/macOS arm64 with 32 GiB RAM, greedy complete-audio recognition, automatic language detection and a 192-token ceiling. Local pinyin/edit-distance dependencies are separately provisioned; weights/environments remain in `.workspace/`.

This is one small, authored synthetic recurrence test with no independent semantic annotation, significance claim or population confidence interval. The natural reads only check historical general-speech regression. Latin entity matching tolerates spacing/case and wrong-entity detection covers the ten eligible memory targets, not arbitrary semantic substitutions. The v2 primary set includes all ten eligible targets (20 cases), unlike v1's selected recurring subset (16 cases); tables are not directly interchangeable. Source BF16 parity, actual cross-session user recurrence, trained acoustic retrieval/adapters, causal streaming, native insertion and Windows inference remain unmeasured. Production vocabulary behavior and default models are unchanged.

Model-free replay, using a new output file:

```sh
pnpm memory:asr-report benchmarks/corpora/user-modify-asr-v2/test \
  benchmarks/runs/user-modify-asr-v2/test-run \
  .workspace/user-modify-asr-v2-test-replay.json
```

Both archived reports and the development selection replay identically. The study has its own archive; it is not part of the historical migration's eight-family report command.
