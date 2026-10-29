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

The complete ESDB70/CSpell membership and source-native case remain available. Grouped TSV v2 preserves one word rank, observed source count and ordered case family per lowercase key. Zero count explicitly means unobserved in the pinned corpus. Technical-source membership no longer boosts word rank; canonical case selection remains source based.

Automatic completions initially require at least 1,000 observed occurrences, three input letters and half the completed spelling. These named Arion development parameters are conservative evidence requirements, not vendor constants. Unknown/low-frequency words stay in the exhaustive lookup lane, and complete input can retrieve them. Minimum-prefix bucket tournaments make these lanes disjoint without repeatedly filtering/replaying a broad stream.

The producer reuses the packaged Rime Prism and `Syllabifier` to require a complete normal-spelling path. The actual leading Chinese candidate must also be a full-range dictionary word. Abbreviation-only matches such as `hel` are not strong full-Pinyin evidence.

| Evidence | Ordering |
| --- | --- |
| Normal full Pinyin and a whole Chinese dictionary word | Up to three Chinese words, familiar exact English forms, remaining Chinese, then deferred English lookup |
| No strong Pinyin evidence, with exact English or admitted completion | Exact forms and admitted completions precede abbreviated/partial Chinese; remaining English lookup and Chinese interleave one/two |
| No strong Pinyin evidence and no qualified English | Original Chinese order precedes weak English lookup |

A familiar exact word has a source lowercase spelling and meets the observed-count floor. This retains `can/an/you/bailing/jingling` while avoiding automatic promotion of Ni/Dian/XING or the low-frequency English bird name `guan` under clear Chinese input. Complete technical names remain available when their spelling does not compete with a whole normal-Pinyin word. Case alternatives remain actual engine candidates. Prefix keys appear once, and all deferred words remain pageable; no fixed four-English preview exists.

`artifi` exposed another important boundary: many partial Chinese alternatives can exist even when English evidence is clear. Admitted English must precede those weak Chinese alternatives, and its lookup tail must remain practically reachable. Counting every partial Chinese candidate as equally strong would hide useful English behind many pages.

## Validation and limits

The [authored corpus](../../benchmarks/corpora/ime-english-admission-v1/README.md) has development, regression-validation and fresh holdout inputs, each checked on five/nine-slot native menus with actual engine-index selection and exact-once commit. The validation set exposed `guan` and `devel` gaps and is now regression material; a separate holdout was frozen after those fixes. Failed and successful native traces are retained with source/data identities. This small synthetic corpus does not estimate general user accuracy.

English personal learning, application context and a learned bilingual language model are not introduced. Explicit case and engine-owned mixed partial conversion reuse this lexicon; unmatched normal Pinyin and existing standalone lanes retain their ordering. Approved personal keyboard readings enter through a separate encrypted projection, not through this immutable English index. Subtitle-domain counts can omit useful technical vocabulary. Rare technical prefixes can need more input or more paging, especially under strong Chinese ambiguity. These limits require evaluation with additional independent inputs, not per-word exceptions. Native engine/text-store checks do not establish installed IMK/TSF event dispatch, and no measured typing latency improvement is claimed.
