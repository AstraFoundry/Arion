# Context ranking (上下文调频) results

Apple M4 (Mac16,12), arm64, three seeds × 50 pairs × 4 occurrences per condition. Both conditions keep Rime's own frequency learning; the ranking condition adds Arion's context ranker ([protocol](../../reproduction/imk-context-v1/README.md), [corpus](../../corpora/imk-context-v1/README.md), [raw runs](../../runs/imk-context-v1/README.md)).

## Quality

| Target words | Ranking off | Ranking on |
| --- | ---: | ---: |
| First occurrence top-1 | 21 / 150 (14%) | 21 / 150 (14%) |
| Later occurrences top-1 | 145 / 450 (32.2%) | **399 / 450 (88.7%)** |
| Later occurrences MRR | 0.612 | **0.933** |
| Committed as intended | 588 / 600 | 588 / 600 |

First occurrences are identical: with nothing learned, the ranker returns the engine's order. On later occurrences 47 pairs improve and none regress. Rime's frequency learning alone follows the most recent choice for a Pinyin, so homophones used after different words keep displacing each other; the context ranker keeps them apart.

The remaining later-occurrence misses are words the engine ranks third or lower that had been confirmed only once in that context: one confirmation lifts a word two places (a deliberate limit against promoting a one-off choice). 元音 (after 英语) was never offered within six pages, so `yuanyin-yingyu` misses in both conditions. The context word was intended top-1 in 97.5% of commits in both conditions.

## Latency

Milliseconds, p50 / p95 / p99 (max). `run-1` is before the candidate bar stopped ordering its window out between words and resizing on every key; `run-3` is after.

| Span | run-1, ranking on | run-3, ranking off | run-3, ranking on |
| --- | ---: | ---: | ---: |
| Letter key (engine, menu, text view, candidate bar) | 0.275 / 2.92 / 4.52 (28.5) | 0.173 / 0.753 / 0.855 (29.1) | 0.192 / 0.809 / 0.906 (29.4) |
| Selection key (Space or number) | 1.66 / 4.70 / 5.37 (7.1) | 0.049 / 0.076 / 0.094 (7.2) | 0.057 / 0.084 / 0.103 (5.6) |
| Candidate bar update alone | 0.143 / 3.23 / 4.52 (14.8) | 0.056 / 0.154 / 0.175 (15.5) | 0.057 / 0.156 / 0.181 (17.0) |

A sampling profile showed `orderFront` (re-ordering the window into the window server after every commit) dominating the old letter-key cost. The bar now hides by becoming transparent and click-through and leaves the window list after 2 s idle; its window only grows while shown. Context ranking costs about 0.02–0.06 ms per key. The 28–29 ms maxima are each process's first key, which builds the window and loads the engine's first candidates.

## Limits

One authored, exposed corpus of ambiguous homophones on one machine; it measures adaptation to repeated context, not general typing quality, and it is not a held-out test. Context is only Arion's own previous commit in the same client (within 120 s); text already in the field is not read. Timings exclude OS event dispatch and compositor frames, and the host was not CPU or power isolated.
