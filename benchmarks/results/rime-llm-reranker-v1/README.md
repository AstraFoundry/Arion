# Rime LLM reranker reproduction results

**The unmodified upstream filter regresses on this authored pilot: top-1 falls from 56/100 to 45/100, with 3 rescues and 14 breakages.** Real model calls take about 289 ms median, while the upstream CPU-time measurement is below 1 ms. The investigation does not support adopting the original filter or repeating its approximately 1 ms inference claim.

This is an engine/filter-boundary reproduction on Apple M4, 32 GiB, native arm64 macOS 26.6.2. It uses actual production Rime candidate snapshots, unchanged upstream Lua, compiled Lua 5.4.8/LuaSocket and real local oMLX 0.7.2 with Qwen3.5-0.8B BF16. The author's exact runtime was not recoverable: its linked oMLX repository returned 404, so the available `jundot/omlx` was explicitly substituted and pinned. No installed Rime/Squirrel settings or Arion production behavior were changed.

The adapter supplies nine first-page candidates, preserving the exact leading five the model receives. It does not reproduce the cost of exhausting the original live translator's potentially longer lazy candidate stream; that cost and system input-method dispatch remain outside the measured boundary.

## Quality

The [100 frozen fixtures](../../corpora/rime-llm-reranker-v1/corpus.json) are original assistant-authored synthetic exploratory data. Target lists are intended readings, not an exhaustive independent linguistic annotation. This is not a held-out benchmark or a population accuracy estimate. Both conditions use identical real first-page candidates; no correct candidate is inserted.

| Group | Cases | Producer top-1 | Original filter top-1 | First-page oracle |
| --- | ---: | ---: | ---: | ---: |
| Opposing context homophones | 48 | 24 | 21 | 45 |
| Familiar words, no context | 12 | 11 | 3 | 12 |
| Abbreviations | 8 | 1 | 1 | 7 |
| Long input | 8 | 7 | 7 | 7 |
| Incomplete final syllable | 8 | 4 | 4 | 8 |
| Explicitly ambiguous readings | 4 | 4 | 4 | 4 |
| Authored names | 4 | 1 | 1 | 1 |
| Relevant clue outside context tail | 8 | 4 | 4 | 8 |
| **Total** | **100** | **56** | **45** | **92** |

MRR@9 falls from **0.7014 to 0.6439**. The correct target is in the first five candidates in 91/100 cases and the first nine in 92/100; oracle misses remain in all quality denominators. Eleven requests bypass HTTP because of the original preedit-length gate, including all eight long-input cases and three names. All 89 actual final-input HTTP requests returned 200; one reached the twenty-token limit. Immediate repeats issue no further HTTP requests.

Concrete breakages include 朋友 → 彭, 学习 → 学, 工作 → 工, 电脑 → 点, 天气 → 天 and 女孩 → 女. Actual Rime candidates mix full conversions with candidates consuming only a prefix; the original prompt/filter can promote the shorter prefix. This study preserves that behavior instead of introducing a length gate. Some no-context labels also admit other natural readings, notably 旅行/履行, so the raw target score is an authored-intent diagnostic.

The 22-fixture typing subset replays **142 actual native key snapshots**, with 124 model calls. Its final-snapshot top-1 falls from 11/22 to 8/22, with zero rescues and three breakages. These are overlapping fixtures, not additional independent linguistic samples. Input is advanced only after the previous synchronous filter call; this is an unpaced sequential replay, not a human input-rate or queue contention experiment.

## Wall latency versus the upstream timer

The model server is resident after two separately retained unrelated warmups. Lua filter state is reset per fixture; immediate snapshot repeats measure its result cache. oMLX's own prompt cache remains active. No OS file-cache eviction, CPU/power isolation, concurrent ASR or rendered-preedit timing was attempted.

| Boundary | Observations | p50 ms | p95 ms | p99 ms |
| --- | ---: | ---: | ---: | ---: |
| Final-input filter wall time, including skips | 100 | 288.84 | 333.89 | 348.23 |
| Final-input HTTP calls only | 89 | 288.85 | 336.44 | 360.76 |
| Final-input filter process CPU | 100 | 0.787 | 0.967 | 1.007 |
| Immediate identical-snapshot Lua cache | 100 | 0.211 | 0.296 | 0.457 |
| Typing HTTP calls only | 124 | 292.97 | 342.35 | 356.52 |

The first unrelated cold-load warmup took **5,038.99 ms wall time**; its server response separately reported 2.74 s model loading. The second warmup took 308.27 ms. Warmups are retained and excluded from scored resident requests. Loading and every measured request used local model bytes; no cloud inference was used.

The upstream `os.clock()` timer measures process CPU time, which mostly excludes waiting for the model server. Its near-zero logs are reproduced alongside roughly 300 ms wall waits. A Lua cache hit can be fast, but it is not an inference measurement. LuaSocket's timeout is an I/O timeout, not a demonstrated overall UI deadline.

## Reproduced implementation defects

**UTF-8 parsing.** The original parser uses multibyte Chinese punctuation inside Lua bracket character classes. Lua matches bytes, so bytes occurring inside ordinary Chinese characters are also treated as delimiters or stripped. Offline replay of the actual 89 responses through the unchanged parser yields 38 no-match results. Twelve responses start with a valid supplied candidate that the live filter did not apply. For example, context 婚礼上许下 and Pinyin `shi yan` produces model first choice 誓言 but displays 实验. This is an original-parser replay, not a corrected model-treatment score. See [parser-audit.json](parser-audit.json).

**Cache identity.** Four separate real-inference controls rotate the same first-five candidate members without changing context/Pinyin. Every second call skips HTTP and reuses a cached positional index, changing the chosen text: 声明 → 盛名, 节电 → 接地, 自愿 → 子 and 点 → 店. The cache key omits candidate identities/order. The [raw controls](../../runs/rime-llm-reranker-v1/cache-controls/summary.json) are perturbation tests, not additional quality samples.

**Partial ranking.** The model is asked for up to five ordered words, but the implementation only promotes the first successfully parsed word; other candidates retain producer order. MRR is computed from the displayed order, not from the generated list.

**README/config drift.** `min_preedit_len` is not the current code's gate. Incomplete-ending checks are used for training-log policy, not for suppressing all incomplete-input queries. The current maximum is the byte length of displayed preedit, including spaces. The boolean configuration helper also does not preserve `false` when its default is `true`; this study confines all logs to synthetic workspace data instead of relying on that helper to disable collection.

## Evidence and replay

The archive retains 100 native producer outputs, every reference-free producer request, final and typing filter requests/responses, the two warmups, four cache controls, original Lua debug logs, source/model/runtime identities and the failed initial native-link attempt. That attempt used the historical quality runner, which omitted the now-required `ranking.m`; its log is retained. The study linked the actual class and reran candidate collection without changing production source. Server startup before the model download completed is retained as a no-model startup, not an inference result.

There are **219 real model requests** in total: 89 final, 124 typing, two warmups and four cache probes. Repeated Lua cache calls do not increase that number. Compiled binaries, weights, environments, GPU caches, generated authentication secrets and synthetic encrypted profiles remain outside Git. Public server settings are an explicitly marked projection with authentication/network/integration fields omitted.

- [All 100 case outcomes](cases.csv)
- [Full final-input metrics and outputs](summary.json)
- [Typing metrics and outputs](typing-summary.json)
- [Raw run archive](../../runs/rime-llm-reranker-v1/README.md)
- [Reproduction procedure and source](../../reproduction/rime-llm-reranker-v1/README.md)

```sh
pnpm ime:rerank verify
pnpm ime:rerank test
pnpm ime:rerank report --run benchmarks/runs/rime-llm-reranker-v1/final-stock --out .workspace/rime-report-new.json
```

Verification requires neither the model nor Lua, MLX, native SDKs or the original workspace. It checks frozen file hashes, native-candidate provenance and exact regeneration of both reports. The five model-free oracle tests reject fabricated/duplicate candidate IDs, changed observations, incomplete runs, duplicate samples and evaluator-label leakage, and retain oracle misses in metric denominators.

The measured regressions justify rejecting the original implementation for direct integration. They do not establish that LLM-based candidate ranking is ineffective generally, or that a parser fix alone would improve this model. A subsequent study would need a separately declared corrected implementation, span-aware candidate policy, stronger model/selection objective and fresh independently reviewed evaluation data.
