# Context ranking benchmark protocol

`run.mjs` compiles the production keyboard engine, context ranker, typing owner, candidate bar and encrypted learning backend (synthetic test key) with `driver.mm`, then replays each seeded stream twice in fresh profiles: context ranking off (baseline: engine order plus Rime's own frequency learning) and on. Both conditions learn every confirmed choice; only `contextRanking` differs.

```sh
node benchmarks/reproduction/imk-context-v1/run.mjs .workspace/benchmarks/imk-context-v1/NEW_DIRECTORY
node benchmarks/reproduction/imk-context-v1/evaluate.mjs benchmarks/runs/imk-context-v1/run-3
```

The driver simulates a user who knows the word they want. It types the Pinyin through `ArionTyping.handleEvent` into a real `NSTextView` client, finds the word in Arion's menu (turning pages with Page Down if needed) and confirms it with Space or its number key. A word not offered within six pages, or a choice that leaves input composing, is a miss. Rank is the word's position in the menu when typing finished; top-1 means Space alone confirmed it.

Timings use `mach_absolute_time` around each `handleEvent`, which includes the engine, ranking, menu, `NSTextView` marked/inserted text and the synchronous candidate bar update. The candidate bar is also timed alone, from layout to `displayIfNeeded`. Neither includes OS event dispatch or compositor frames. The host is not CPU or power isolated.

`evaluate.mjs` reads only the archived raw results. It reports first-occurrence and later-occurrence top-1, MRR, selection keys per word, context-word scores and timing percentiles, plus the pairs whose later-occurrence top-1 differs between conditions.
