# Empty-vocabulary correction learning and later recognition

**The core learning loop now has actual inference and persistence evidence.** On 72 new synthetic recurring-name utterances, no memory recognizes 3 terms and the development-selected prior retrieval policy recognizes 33. Memory starts empty and is acquired from actual earlier ASR output plus authored user corrections, through the production extractor and encrypted store. Two new retrieval/context candidates fail to improve development performance and are not promoted.

These results use different terms, voices and sentences from v2. They cannot be interpreted as a continuation of its 35% to 55% percentage comparison.

## Frozen protocol and execution

- Before any v3 inference: freeze 16 development and 24 test entities, two teaching utterances per entity, three different later utterances per entity, ordinary/unlearned/nonspeech controls and 20 historical natural-read regressions. Development/test entity sets, voice IDs, audio bytes, sessions and utterance templates are disjoint. Teaching and later sessions also have different voices/audio. Synthetic voice IDs are not independent human speakers.
- Each split starts with an empty research vault and a private random key. Every teaching audio receives actual Qwen3-ASR-1.7B inference with memory disabled. Its output becomes the inserted field text. Authored user-edited field text is passed to production `domain.ExtractCorrections`, `vocabulary.Store.Learn` and a newly opened `Store.List` in a separate Go process. No preauthored error aliases populate memory.
- Freeze the reloaded vocabulary before future recognition. Future recognition reads only audio and that snapshot. It never loads the evaluator's reference file, receives test-derived corrections or updates its counters. Adaptive outputs come from original-audio re-decoding, with no transcript substitution.

Here `test` names the held-out cohort: its `teaching.json` is a separate prior enrollment partition, while `recognition.json` is the later scored test. Enrollment uses only the earlier audio and its user correction. No later reference or output contributes to enrollment memory.
- Development compares empty context, prior `v2_hint`, the same retrieval with `Vocabulary:` formatting, and a grouped source/destination retrieval policy that abstains on close overlapping competitors. Only two new candidates are allowed; thresholds and selection rules are frozen with the corpus. Development term hits are **6/48, 23/48, 18/48 and 19/48** respectively. The [selection](../../runs/user-modify-asr-v3/selection.json) retains `v2_hint`; no final-test retuning occurs.
- Development: 32 teaching calls, 66 future inputs × four conditions = 264 future observations, **214 actual calls** including teaching, plus two warmups. Final test: 48 teaching calls, 116 future inputs × two conditions = 232 future observations, **227 actual calls** including teaching, plus two warmups. Across both: 496 paired future observations, 80 teaching observations, 441 actual model calls and four additional warmups. Shared first passes make these denominators different.

[Exact corpus and freeze](../../corpora/user-modify-asr-v3/), [raw calls, edits and storage snapshots](../../runs/user-modify-asr-v3/), [development summary](dev-summary.json), [final summary and scored outputs](test-summary.json), [source and commands](../../reproduction/user-modify-asr-v3/README.md), [archive identities](archive.json), and [paper analysis](../../../docs/research/closed-loop-memory.md) retain provenance.

## Learning and persistence

| Final-test stage | Observed outcome |
| --- | ---: |
| Initial entries | 0 |
| Earlier-session utterances / entities | 48 / 24 |
| Earlier utterances missing the target spelling | 48 |
| Those corrections producing a pair containing the full target | 43 / 48 |
| Distinct full targets present in the final snapshot | 23 / 24 |
| Stored records after updates/deduplication | 43 |
| Non-correction edits learned | 0 / 9 |
| Authored weekday-intent risk edits learned | 0 / 1 |
| Persisted/reopened teaching and control snapshots | 58 |
| Final reopened contents match the last persisted observation | Yes |
| Ciphertext unchanged throughout future recognition | Yes |
| Undo and reverse correction on a separate cloned store | Passed |

The full-target measure can miss useful partial-word pairs. It does not independently rate the precision of every extracted fragment. The single weekday edit does not establish safety on semantic rewrites generally. Random entry IDs and encryption nonces are real production behavior; new acquisition runs need not be byte-identical. Model-free replay verifies this run's exact recorded bytes.

Measured extraction/write compute for final-test events is p50 12.72 ms / p95 18.24 ms, excluding process launch. Every returned entry list is read from a newly opened encrypted file. Keys stay private; this does not measure OS credential provisioning, crash power-loss recovery or native field monitoring.

## Later audio recognition

| Condition | Term hits / 72 | Baseline term misses recovered / 69 | Previously zero-error speech regressed / 30 | Speech improved / worsened | Controls worsened / 42 |
| --- | ---: | ---: | ---: | ---: | ---: |
| Empty context | 3 | 0 | 0 | — | 0 |
| Learned-memory `v2_hint` | **33** | **30** | **0** | **37 / 1** | **0** |

The selected pipeline recognizes 45.83% versus 4.17% of this deliberately difficult synthetic recurrence set. It loses none of the three baseline term hits. Forty-eight of 69 baseline term misses retrieve the correct stored target; only 30 are subsequently recovered. Merely finding a memory entry is therefore insufficient.

Chinese CER falls from 53/466 = 11.37% to 35/466 = 7.51%; English WER from 21/366 = 5.74% to 11/366 = 3.01%; mixed MER from 175/531 = 32.96% to 120/531 = 22.60%. Twenty historical natural-read controls have no degradation. Both nonspeech inputs remain empty and no generation reaches the token ceiling. No new entity from the declared vocabulary is inserted erroneously, but that detector does not cover arbitrary new wrong names.

One primary case **does worsen**: `请转告邵栩宁，会议地点已经换了。` is initially `邢专告邵兴林，会议地点已经换了。`; an unrelated match retrieves `霁川`, and the re-decode writes `晴川告诉雪玲，会议地点已经换了。`. It is a lexical regression and an unresolved routing failure despite not inserting a known wrong memory entity.

Sixty-three of 116 inputs re-decode. On these, extra retrieval/recognition compute is p50 739 ms / p95 1,204 ms. Overall pipeline p50/p95 changes from 0.759/1.234 s to 1.241/1.976 s. These measurements exclude recording, endpointing and insertion and are not isolated-load latency guarantees.

## Limits and reproduction

This is an actual audio/extraction/encrypted-file/reload/audio loop with **simulated user edits**, not a native UI or natural-user study. Ordinary-word contextual disambiguation is examined separately in the [text-context probe](../context-memory-v1/README.md). Proper names, deliberate transliterations and system voices make the task artificial; the 20 natural reads are exposed historical regression. There is no independent semantic annotation, population confidence claim or matched competitor execution.

Weights and runtime match v2: community MLX 8-bit Qwen3-ASR-1.7B revision `a8379a2e2f9e313c9292cdf1af4055ab56d50d55`, MLX Audio commit `dad2087f790dfe96b07d5aac1405db4b578b411f`, Apple M4/macOS arm64, 32 GiB, greedy offline complete audio, automatic language detection and 192 output tokens. Source BF16 parity, real OCR/field tracking in this loop, causal microphone operation and Windows execution remain unmeasured. The research drivers do not alter product defaults.

```sh
pnpm memory:loop-report benchmarks/corpora/user-modify-asr-v3 \
  benchmarks/runs/user-modify-asr-v3/test-run \
  .workspace/user-modify-asr-v3-replayed.json
```

The corpus freeze, complete statuses, every audio/output/call identity, initial empty state, teaching provenance, persistence chain and paired final calls are checked before scoring. Superseded pre-inference shared-template design and the failed identical-silence split check are retained with their original freeze identities; neither contributes model observations. This study has a separate archive and is not included in the historical migration replay command.
