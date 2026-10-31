# English candidate admission

## Problem and industrial evidence

A complete English dictionary improves coverage, but automatic recommendation needs separate evidence. The reported `xin` page contained 新/心/信 followed by XING/xinit/xinetd/xinput because the previous producer always previewed four English matches. This was a scheduling rule, not a measured estimate of input intent.

| Primary source | Supported mechanism | Transfer limit |
| --- | --- | --- |
| [Microsoft Research, ACL 2000, section 4](https://aclanthology.org/P00-1031.pdf) | Pinyin and English spelling hypotheses compete, using language evidence and English actually embedded in Chinese | Historical research; not proof of current Windows implementation or published completion thresholds |
| [QQ official help](https://skin.qq.pinyin.cn/help/pinyin05.html) | Automatic mixed-input intent and user-requested English completion are distinct | Historical official product behavior; internal scores are not disclosed |
| [Google Gboard, ACL Industry 2023](https://aclanthology.org/2023.acl-industry.60.pdf) | Candidate generation/reranking, next-word prediction and high-confidence inline completion have different roles | Production mobile system; inline confidence is not a disclosed Chinese candidate-bar rule |
| [AOSP LatinIME scoring](https://android.googlesource.com/platform/packages/inputmethods/LatinIME/+/127336e9f29d69607eab55982324b210279ae8c5/native/jni/src/suggest/policyimpl/typing/typing_scoring.h#45) | Intentional distracters do not receive ordinary case-insensitive exact-match promotion | A missing corpus count is not the same thing as an intentionally zero-probability distracter |
| [OMRON OpenWnn English prediction](https://android.googlesource.com/platform/packages/inputmethods/OpenWnn/+/5067ec8ec5a80ab12218c961f9b083fbccc71b93/src/jp/co/omronsoft/openwnn/EN/OpenWnnEngineEN.java#252) | Input specificity controls dictionary tiers; prefix results use frequency | Historical English-only engine; its thresholds are not Chinese mixed-input defaults |

These sources motivate separating coverage, admission and display. They do not supply a reusable calibrated bilingual model. Arion does not compare Rime quality directly with subtitle counts or describe the current policy as neural confidence.

## Adopted local policy

The complete ESDB70/CSpell lexicon, native case families and grouped TSV v2 remain unchanged. Membership and observed frequency are separate. The [keyboard data policy](../../apps/desktop/ime/shared/keyboard-data/README.md#english-words-in-chinese-mode) owns admission thresholds and detailed ordering.

Complete normal Pinyin protects Chinese independently of whether the leading candidate is a word, sentence or personal entry. Only a sufficiently long exact English match outside complete Pinyin may take first choice. Short exact English and admitted completions preserve Chinese first. A single English display form is advanced; other cases and completions remain pageable after a protected Chinese first-page region. Bounded lazy tail merging retains complete lookup without draining a potentially huge Chinese stream first. Sparse Chinese results can leave additional space for English. Partial English choices keep their previous engine-owned ranges and ordering.

## msime and Rime source comparison

The shared native producer now implements the alignment below. Source comparison
and isolated AppKit/engine verification do not establish installed IMK/TSF dispatch
or representative user accuracy.

The inspected msime revision is
[`1cb46c3`](https://github.com/metasequoiaime/msime/tree/1cb46c3d546126042035eea8d8fca92200380c29).
Its implementation is more conservative than Arion's previous weak-Chinese branch:

| Mechanism | Verified source behavior |
| --- | --- |
| Completion admission | Product defaults require five letters; shorter inputs of at least two letters query one exact spelling only. |
| English first choice | Requires the completion-length threshold, an eligible whole-composition query, input that cannot segment into complete Pinyin syllables, and a surviving exact English match after deduplication. A completion cannot substitute for a deduplicated exact match. |
| English exposure | The first English row normally follows the Chinese first choice. The first-page English allowance defaults to one, is configurable from one to three, and is bounded by available page room; the remaining English rows follow the Chinese list. |
| Cross-language weights | English dictionary weights order English results; they do not compete numerically with Chinese dictionary weights. |

Sources: [product defaults](https://github.com/metasequoiaime/msime/blob/1cb46c3d546126042035eea8d8fca92200380c29/crates/client-core/src/preferences.rs#L3113),
[mixed-query gates](https://github.com/metasequoiaime/msime/blob/1cb46c3d546126042035eea8d8fca92200380c29/crates/engine/src/ime/queries.rs#L400),
[candidate placement](https://github.com/metasequoiaime/msime/blob/1cb46c3d546126042035eea8d8fca92200380c29/crates/engine/src/ime/queries.rs#L614),
and [strict syllable completeness](https://github.com/metasequoiaime/msime/blob/1cb46c3d546126042035eea8d8fca92200380c29/crates/engine/src/pinyin/segment.rs#L358).
Its [design record](https://github.com/metasequoiaime/msime/blob/1cb46c3d546126042035eea8d8fca92200380c29/.agents/notes/implemented/feature/2026-10-12-mixed-english-leads-unsegmentable-input.md)
reports earlier weight-based promotion putting `dancing` ahead of the intended
Chinese word for `danci`. That is upstream's recorded failure, not an Arion run.

Rime is an engine with configurable schemas, not one universal English-mixing
policy. At Arion's pinned librime revision,
[`Candidate::compare`](https://github.com/rime/librime/blob/33e78140250125871856cdc5b42ddc6a5fcd3cd4/src/rime/candidate.cc#L35)
orders candidates by earlier start, longer consumed input, then quality.
[`TableTranslation::Peek`](https://github.com/rime/librime/blob/33e78140250125871856cdc5b42ddc6a5fcd3cd4/src/rime/gear/table_translator.cc#L73)
distinguishes `completion` from exact table matches and assigns
`exp(weight) + initial_quality - 1` to a completion, with an additional user-phrase
bonus when applicable. Translator configuration controls completion and initial
quality. These mechanisms support explicit exact/completion separation and
source bias; they do not calibrate Arion's subtitle counts against Chinese scores.
Arion's bilingual translator emits one manually scheduled stream, so lowering
an English table translator's YAML quality cannot repair its internal ordering.

### Previous Arion failure paths identified in source

Before alignment, [`english.cpp`](../../apps/desktop/ime/shared/english.cpp) required both normal spelling and a leading full-range `phrase` or `user_phrase`. A full-range `sentence`, personal candidate, or abbreviated reading failed that predicate. Consequently:

1. An input can have complete normal Pinyin while failing Chinese protection
   because its first result is a sentence rather than a dictionary phrase.
2. With protection false, every exact English display form is emitted first,
   regardless of observed frequency, followed by every admitted completion.
   There is no automatic-promotion allowance for that group.
3. After those groups, the lookup stream can still precede Chinese on one turn
   out of three. English display variants consume separate candidate positions.

This is deterministic producer scheduling. Source inspection identifies routes
to the reported behavior, but does not measure how often users encounter them.

### Implemented alignment and deliberate differences

- Normal Pinyin classification no longer depends on a dictionary-phrase first choice. Sentences and personal candidates retain Chinese protection.
- Only exact English outside complete normal Pinyin can lead at the mixed-completion length threshold, with the caret and segment at the end of the composition. A short exact form and an admitted completion follow Chinese. No completion replaces an exact word merely because it has higher corpus frequency.
- One actual display form is advanced. Exact case alternatives and remaining completions stay in their original lazy cursors. The first-page protection uses the shared menu's maximum supported size so both native page layouts use the same engine identities.
- Arion retains source-frequency and half-word admission checks in addition to msime's length gate. Familiar exact English remains after up to three full-range Chinese candidates on complete Pinyin; msime normally inserts its first English result after the first Chinese candidate.
- Unlike msime's finite mixed list, Rime can produce a long lazy stream. Arion resumes bounded two-Chinese/one-English groups after the protected region instead of hiding the complete English tail behind every Chinese candidate. Early exhaustion naturally fills remaining slots from the surviving stream. Approved partial English prefixes retain their prior full-range-Chinese-before-prefix ordering.
- Existing acceptance fixtures were updated deliberately: `hel` no longer requires first-page `hello`, lowercase `api` preserves normal Pinyin and its canonical `API` remains pageable, and `devel`/`algor` exact forms take precedence over longer completions. The inspected historical holdout is now explicitly regression material.

No word-specific blacklist, dictionary shrink, new cloud inference or direct comparison of unrelated language frequencies is introduced. Broader bilingual intent learning remains a separate evaluated change.

## Validation and limits

The current shared producer passes controlled-stream regressions for word, user-word, sentence and personal provenance; short/exact English leadership; caret-prefix suppression; case-family pagination and five-letter completion admission. Native AppKit tests cover both supported page sizes, actual selection/commit, mixed-span editing/cancellation and deferred canonical case. Before/after native panels and candidate traces are included with the implementation PR. Representative nine-slot observations:

| Input | Previous first choice / English slots | Aligned first choice / English slots |
| --- | --- | --- |
| `artifi` | `artificial` / 4 | 爱人提法 / 1 |
| `hel` | `Hel` / 8 | 合理 / 1 |
| `can` | 惨 / 2 | 惨 / 1 |
| `artificial` | `artificial` / 5 | `artificial` / 1 |
| `github` | `GitHub` / 2 | `GitHub` / 1 |
| `xin` | 新 / 0 | 新 / 0 |

These are deterministic synthetic-profile examples, not representative recall or accuracy estimates. English candidates deferred from the first page remain selectable. Windows target-OS execution and installed IMK/TSF event dispatch remain separate acceptance gates.

The [authored corpus](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/corpora/ime-english-admission-v1/README.md`) has development, regression-validation and fresh holdout inputs, each checked on five/nine-slot native menus with actual engine-index selection and exact-once commit. The validation set exposed `guan` and `devel` gaps and is now regression material; a separate holdout was frozen after those fixes. All those inputs have since been inspected and are regression material for the current alignment. Failed and successful native traces are retained with source/data identities. This small synthetic corpus does not estimate general user accuracy.

English personal learning, application context and a learned bilingual language model are not introduced. Explicit case and engine-owned mixed partial conversion reuse this lexicon; unmatched normal Pinyin and existing standalone lanes retain their ordering. Approved personal keyboard readings enter through a separate encrypted projection, not through this immutable English index. Subtitle-domain counts can omit useful technical vocabulary. Rare technical prefixes can need more input or more paging, especially under strong Chinese ambiguity. These limits require evaluation with additional independent inputs, not per-word exceptions. Native engine/text-store checks do not establish installed IMK/TSF event dispatch, and no measured typing latency improvement is claimed.
