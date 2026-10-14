# Rewrite context ablation (v1)

Real local CPU runs on Apple M4 (32 GiB, macOS arm64):
- Rewriting: the production `arion-llm` worker (Qwen3 1.7B and 4B INT4, greedy, 4 threads), the production validator and term extraction.
- Screen text: the production Vision recognizer (`arion-ocr`).

[summary.json](summary.json) holds every number below with the hashes of corpora and private result files. [ocr.json](ocr.json) holds recognition quality and time. The methodology is in [the live rewrite benchmark](../../../docs/research/live-rewrite-benchmark.md#rewrite-context); the decision is [ADR 013](../../../docs/decisions/013-rewrite-context-on-device.md).

**Limits.**
- One developer wrote every window, error and proxy.
- Windows are HTML renderings, not captures of real apps. Sources are text with hand-written ASR-like errors, not speech.
- Scores are exact-string proxies with no human ratings. The host was not isolated, so latencies vary between batches.
- Not measured: a local VLM (no weights downloaded), Windows OCR (compile-only), and real window capture (SCN/CTX-A04 open).

## Screen recognition (30 windows, 2×)

| Vision level | Character error vs visible text | Time p50 / p95 |
| --- | --- | --- |
| Accurate | 5.8% | 107 / 221 ms |
| Fast | 28.2% (no Chinese) | 30 / 42 ms |

## Held-out context set (30 cases)

The set has 11 screen cases, 6 cursor cases, 6 history cases and 7 negatives. Every source scores 8 of 30 without rewriting. Each row's worsened count is 0 unless marked.

**First pass, context sections (`context-v1` prompt):**

| Context | 1.7B | 4B |
| --- | --- | --- |
| None | 8 | 9 |
| Cursor | 8 | – |
| History | 9 | – |
| Screen, true text | 9 | – |
| Screen, Vision accurate | 9 | – |
| Screen, Vision fast | 8 | – |
| All, Vision accurate | 9 | 15 (1 worsened¹) |

¹ The 4B model replaced a dictation about 张伟 with the on-screen message from 张巍, and validation accepted it. The coverage credit for context spellings is now capped, so context can no longer stand in for most of a dictation. This fix was informed by the held-out set.

**Final pass**, after choosing the delivery mode on a separate 16-case development set and capping the credit:

| Context (4B) | Passes | Screen | History | Cursor | Negatives | p50 / p95 |
| --- | --- | --- | --- | --- | --- | --- |
| None | 9 | 0/11 | 1/6 | 1/6 | 7/7 | 1.01 / 1.96 s |
| Cursor, terms² | 9 | 0 | 1 | 1 | 7 | 1.26 / 2.07 s |
| History, terms² | 10 | 0 | 2 | 1 | 7 | 1.50 / 2.74 s |
| Screen true text, terms² | 15 | 5 | 2 | 1 | 7 | 1.26 / 2.15 s |
| Screen Vision accurate, terms² | 14 | 5 | 1 | 1 | 7 | 1.30 / 2.16 s |
| Screen Vision fast, terms² | 13 | 3 | 2 | 1 | 7 | 1.29 / 2.16 s |
| All sources, context sections | 16 | 5 | 3 | 1 | 7 | 1.84 / 2.83 s |
| **All sources, terms (product)** | **16** | **6** | **2** | **1** | **7** | **0.86 / 1.49 s**³ |
| All sources, terms, 1.7B (product) | 9 | 0 | 2 | 0 | 7 | 0.44 / 0.74 s³ |

² These rows validated without the context, which is stricter than the product; the validator rejected 2–4 correct proposals.

³ This batch ran on a quieter host. Compare latency within one batch only.

The development set agreed: 4B went from 4 to 8 passes with context sections and to 9 with terms, while 1.7B stayed at 4 in every mode. On both sets, terms matched context sections at lower latency, and they keep the shared prompt prefix.

## Findings

- **The screen is the useful source.** It gives +5–6 passes on 4B, mostly names and terms that appear in the window. Vision accurate loses one case against the true text; fast loses two more because it cannot read Chinese.
- **Cursor text did not help here,** and history helped by +1–2. With a 4B model, the remaining cursor failures are spellings the model did not apply, and continuations it did not shorten.
- **Model size decides.** Qwen3 1.7B, the shipped default, ignores context in every delivery mode. Qwen3 4B uses it, at about 2× the latency and memory.
- **No negative case worsened in the final pass.** That covers distracting windows, injected instructions, similar names and unrelated history. The first pass shows that validation must stop context from replacing the dictation.

## Commands

```sh
make context-ocr CORPUS=.workspace/context/context-heldout-corpus.json OUT=.workspace/context/ocr.jsonl
make live-text-bench CORPUS=.workspace/context/context-heldout-corpus.json LLM_MODEL=.workspace/ablation-models/qwen3-4b SCENE=corpus CORRECT=1 CONTEXT=cursor,history,screen SCREEN=accurate OCR=.workspace/context/ocr.jsonl CONTEXT_AS=terms OUT=.workspace/benchmarks/context/final/4b-product-all-accurate.jsonl
make live-report RUNS=benchmarks/configs/context-ablation-report.json OUT=benchmarks/results/context-ablation-v1/summary.json
```
