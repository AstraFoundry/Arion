# Edit learning and vocabulary recall (v1)

This report measures the [ADR 014](../../../docs/decisions/014-edit-learning.md) pipeline: how well corrections are extracted, the effect of learned memory on recognition, and recall latency, scaling, concurrency and fault behaviour. Aggregates are in [summary.json](summary.json); per-sample text stays in `.workspace/benchmarks/edit-learning-v1/private.jsonl`.

- **Host:** Apple M4 (10 cores), macOS arm64, Go 1.27.1.
- **Code:** git `28296c9` plus the uncommitted ADR 014 change set.
- **Rerun:**

  ```sh
  make memory-bench OUT=… CORPUS=… DEV=… TEST=…
  ```

  The same runs as `pnpm memory:bench`; without `CORPUS`, it skips the replay.

## Summary

- **Extraction** on 47 labelled edits: pair precision 0.95, pair recall 0.91. Nothing was learned from 23 non-corrections, and 1 of 4 known-risk edits was learned.
- **No harm on held-out data.** Memory learned from 114 dev-split corrections changed none of 150 test-split dictations. The test sentences do not repeat the dev errors, so this corpus shows neither a benefit nor a harm.
- **When an error recurs, recall fixes it.** Re-dictating the same dev sentences after learning cut errors from 8.5 % to 3.6 % overall, and mixed Mandarin–English technical speech from 23.9 % to 7.3 % MER. 35 dictations improved and 1 worsened. This is an in-sample ceiling, not a held-out estimate.
- **Recall latency barely moves with memory size.** With 2,000 entries (the cap), the median replacement takes 0.5 µs and the p99 18 µs. The previous per-entry loop took 321 µs median and 687 µs p99, so the index is about 640× faster at the median and 38× at p99. The rewrite prompt received at most one recalled term per test sample, against 39 if every entry were sent.
- **Learning never touches the dictation path.**
  - `Track` returns in ≤ 13 µs p99 and `Settle` in ≤ 12 µs p99.
  - A host that stops answering is abandoned after 1.45 s and 3 reads, in the background.
  - A learned write takes 10–13 ms median (encrypt, then fsync and atomic rename), off the main thread.
- **Every injected storage fault is safe.** In all 5 faults, dictation kept working. A corrupt or unwritable file stayed unchanged, and learning failed explicitly.

## 1. Extraction on labelled edits

[edit-learning-cases.json](../../configs/edit-learning-cases.json) has 47 developer-authored edits:

- 20 corrections, which should be learned;
- 23 non-corrections: continued typing, typing elsewhere, punctuation, deletions, insertions, rewrites, clearing, reordering, emoji, a pronoun, a lowered name, a single letter;
- 4 risk cases: content changes that look like misrecognitions.

**This is a development set.** It and the dev replay guided five rule changes:

- the "two thirds removed" size limit;
- trimming punctuation off a change;
- exempting sentence-start and lowered case;
- exempting single Latin letters;
- exempting grammar-dependent homophones.

The held-out evidence is section 2.

| Metric | Value |
| --- | --- |
| Pair precision / recall | 20/21 = 0.95 / 20/22 = 0.91 |
| Non-corrections learned | 0 / 23 |
| Risk cases learned | 1 / 4 (`monday` → `tuesday`) |
| Extract latency per edit | p50 2.8 µs, p99 21 µs |

There are two misses:

- `飞输群里` → `飞书群里` was learned as `输群 → 书群`. A one-character CJK change is widened with its right-hand neighbour, and here the word sits on the left.
- `python three twelve` → `Python 3.12` is longer than the 16-character pair limit.

A changed weekday cannot be told apart from a misheard word by text alone. The undo handles that case.

## 2. Effect on recognition (real ASR output, replayed)

**Recognition.** The default bundle, `xasr-160-xasr-final`, recognized the private corpus through the production driver in real time:

- dev split: run `models-dev-2`;
- test split: run [edit-learning-asr](../../configs/edit-learning-asr.json), with different speakers and voices.

**Learning.** For each dev dictation in corpus order, the inserted text (recognition plus `arion-rules-v1`) was "corrected" by the user to the reference. `ExtractCorrections` and `Learn` built the memory. Test references are never read for learning (EVAL-A04).

**Applying.** `domain.Correct` ran with and without that memory on each test recognition. The two outputs were scored with the evaluator's normalization. The Go scorer reproduces the harness exactly on raw output (zh 47/808, en 39/983, mixed 82/371). The memory-off baseline includes the product's rule correction, which lowers mixed to 75/371. Offline rule correction matched the driver's local-correction text on 141 of 150 samples.

114 dev corrections gave these verdicts:

| Verdict | Count |
| --- | --- |
| learned | 36 |
| no pair | 43 |
| too many changes | 19 |
| unchanged | 15 |
| too large | 1 |

After expiry and deduplication, 39 entries remained.

| Split | Unit | Memory off | Memory on | Improved / worsened |
| --- | --- | --- | --- | --- |
| Test (held out), zh | CER | 5.82 % | 5.82 % | 0 / 0 |
| Test (held out), en | WER | 3.97 % | 3.97 % | 0 / 0 |
| Test (held out), mixed | MER | 20.2 % | 20.2 % | 0 / 0 |
| Dev again (in-sample ceiling), zh | CER | 4.22 % | 1.79 % | 11 / 0 |
| Dev again, en | WER | 3.71 % | 3.23 % | 3 / 1 |
| Dev again, mixed | MER | 23.9 % | 7.3 % | 21 / 0 |
| Dev again, overall | | 8.54 % | 3.62 % | 35 / 1 |

How to read the table:

- **Held out.** Different sentences, speakers and voices produce different errors, so none of the 39 entries fired. The useful result is that learned memory did no harm on new speech.
- **In-sample ceiling.** This is the product's intended case, where the same name or term is misheard the same way again. It is an upper bound: re-dictation by the same voice does not reproduce errors exactly.
- **A realistic held-out estimate** needs a corpus where a person dictates recurring names and terms across sessions. Public read speech has none, and a consented personal corpus is open work.

Examples of entries learned from dev corrections:

- `送火` → `纵火`
- `沙发` → `Safari`
- `插队了` → `changelog`
- `Uber let us` → `Kubernetes`
- `somit` → `commit`

LibriSpeech references are lowercase, so a few English entries also lower case inside a spelling fix (`Cozette` → `cosette`). A real user would not type that, but the corpus references do.

## 3. Recall latency and scaling

`domain.Vocabulary` is measured on the 150 test recognitions × 20 repetitions. The entries combine the 39 learned ones with generated Latin and Han terms. The Han terms are drawn from a 39-character alphabet, which crowds the index buckets: a worst case for p99. "Legacy" is the per-entry loop the index replaced.

| Entries | Build | Apply p50 / p99 | Prompt terms p99 | Full `Correct` p50 / p99 | Legacy apply p50 / p99 |
| --- | --- | --- | --- | --- | --- |
| 10 | 0.01 ms | 0.4 / 1.8 µs | 2.4 µs | 1.0 / 4.7 µs | 2.0 / 4.0 µs |
| 100 | 0.04 ms | 0.4 / 2.9 µs | 4.1 µs | 0.9 / 5.2 µs | 17.6 / 29.2 µs |
| 1,000 | 0.38 ms | 0.5 / 8.4 µs | 23.8 µs | 1.0 / 11.0 µs | 155 / 273 µs |
| 2,000 (cap) | 0.82 ms | 0.5 / 18.2 µs | 47.7 µs | 1.0 / 21.5 µs | 321 / 687 µs |
| 10,000 | 5.1 ms | 0.7 / 94.3 µs | 215 µs | 1.2 / 99.3 µs | 1,607 / 2,965 µs |

The index is rebuilt once per vocabulary change, not per dictation. The prompt receives at most eight relevant entries (CTX-06), and in this replay it never received more than one.

## 4. Store latency (AES-256-GCM, fsync, atomic rename)

These are 20 probes per size. Each probe learns one entry, counts a recall, lists the store and undoes the entry.

| Entries | File | Learn p50 / p99 | Recall p50 / p99 | List p50 / p99 |
| --- | --- | --- | --- | --- |
| 0 | 32 B | 10.0 / 16.6 ms | 10.1 / 12.3 ms | 0.23 / 0.53 ms |
| 100 | 14 KB | 11.0 / 12.9 ms | 10.9 / 13.8 ms | 0.63 / 1.10 ms |
| 1,000 | 143 KB | 12.0 / 13.1 ms | 12.0 / 14.2 ms | 1.41 / 2.03 ms |
| 1,999 | 285 KB | 13.2 / 19.3 ms | 12.9 / 15.0 ms | 1.93 / 2.15 ms |

macOS `fsync` dominates every write, at about 10 ms. Writes run on the tracker's or the recall callback's goroutine, never on the main thread or in the dictation critical section.

## 5. Edit tracker, concurrency and faults

**Tracker** at production timings (0.4 s poll, 250 ms read budget, 15 s idle):

- `Track` p99 13 µs, `Settle` p99 12 µs, settle to report p99 0.14 ms;
- a hung host is abandoned after 1,451 ms and 3 reads;
- an edit that settles by idle costs about 38 reads.

**Extraction** grows with the insertion, because it compares word lists:

| Inserted length | p50 | p99 |
| --- | --- | --- |
| 50 runes | 4.5 µs | 10.6 µs |
| 200 runes | 30 µs | 101 µs |
| 1,000 runes | 369 µs | 533 µs |
| 2,000 runes (limit) | 1.27 ms | 1.40 ms |

All of it runs once per edit, in the background.

**Concurrency.** 8 readers applied the index for 3 s while a writer learned one entry at a time, re-listed the store and swapped in a rebuilt index:

- 6.7 M applies and 196 writes, with 0 write errors;
- the final store held exactly the expected 1,190 entries;
- apply latency was p50 0.8 µs and p99 27 µs under writes, against p50 0.8 µs and p99 14 µs idle;
- the maximum was one 40 ms outlier among 6.7 M applies, consistent with a scheduler or GC pause on a fully loaded machine.

**Faults** (all pass):

| Fault | Learning | Memory file | Dictation |
| --- | --- | --- | --- |
| Bit flip | Fails explicitly | Unchanged | Works without memory |
| Truncated file | Fails explicitly | Unchanged | Works without memory |
| Wrong key | Fails explicitly | Unchanged | Works without memory |
| Crash leftover `.tmp` | Succeeds; leftover replaced | Updated | Works with memory |
| Read-only directory | Fails explicitly | Unchanged and readable | Works with memory |

The unit tests run with `-race` and cover the same behaviour:

- `TestEditTracker*` and `TestCoordinator*Follow*`;
- `TestCrashLeftoverIsIgnored`, `TestCorruptFileFailsClosed` and `TestConcurrentWritersKeepEveryEntry`;
- `TestLearnedNoticeUndo`.

## Not covered

- The native macOS path, `TestLearnFromEditInOwnedEditor`, needs Accessibility granted to the test runner and has not run on this host.
- Windows does not follow fields, so it learns nothing.
- No consented personal corpus measures held-out recurrence.
- The rewrite model's use of recalled terms was not re-measured. Terms reach the prompt as before, only fewer of them.
