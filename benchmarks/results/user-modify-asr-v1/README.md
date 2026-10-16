# Recognition-stage correction memory, v1

**Qwen3-ASR-1.7B accepts memory together with audio, but this experiment does not establish reliable correction-pair recall.** On the 16 recurring-term cases, no memory recognized 8 terms, destination-only context recognized 9, selected correction pairs recognized 8, and a literal `USER_MODIFY` wrapper also recognized 8 while corrupting one previously correct term. The unfiltered-pair control recognized 9. These are small development diagnostics, not a measured reproduction of Doubao's model or quality.

## Execution and evidence

- Actual offline audio recognition on Apple M4, macOS arm64, 32 GiB RAM, using `mlx-community/Qwen3-ASR-1.7B-8bit` revision `a8379a2e2f9e313c9292cdf1af4055ab56d50d55`. All model/tokenizer files were checked against pinned upstream LFS or Git-blob identities before inference.
- 32 frozen inputs × 5 arms = **160 complete recognition observations**, plus five warmups in the completed run. A separate five-arm loading preflight is retained as `preflight_complete`, not counted as a full run.
- Audio is developer-authored system-voice TTS, plus generated silence/noise. Thirty speech utterances use Tingting/Samantha; a separate development warmup uses Daniel. Proper-name references encode approved spelling of deliberately transliterated pronunciation. This is not consented natural multi-session speech or independent annotation.
- All primary arms use the same audio, checkpoint, greedy decoding, automatic language detection and 192-token ceiling. Arm order is seeded and shuffled within each case. No text rewriter, deterministic mapping or output validation replaces the ASR result. No test reference enters inference or memory selection.
- [Raw run, outputs and warmups](../../runs/user-modify-asr-v1/run-v1/), [corpus and prior memory](../../corpora/user-modify-asr-v1/), [source and report replay](../../reproduction/user-modify-asr-v1/), [aggregates and every scored output](summary.json), [archive identities](archive.json).

The [feasibility research](../../../docs/research/user-modify-asr.md) documents model inputs and competitor evidence. MLX Audio source is pinned at `dad2087f790dfe96b07d5aac1405db4b578b411f`; exact runtime versions, source/native artifact hashes, hardware, power settings and Git state are in `run.json`. The official BF16 checkpoint was not provisioned: source/conversion parity remains unmeasured.

## Treatments

Prior memory comprises 13 authored approved records and synthetic counts, fixed before test utterances. Selection excludes entries older than 30 logical days and both directions of a reversed pair; it takes two entries from today, three by recall count, then fills by use count, within eight entries and 512 characters of pair text. Selected entries are Arion, MyGo, Redis, Kubernetes, 飞书, Safari, OpenAI and SQL. Counts stay frozen across paired inference.

Today's pairs, recall/use counts, expiry, reverse suppression and a bounded prompt are mechanisms observed in Doubao. **The quotas, tie-breaks, limits and serialization used here are experimental choices.** Its exact prompt, checkpoint and server-side pair extraction were not recovered. The wrapper arm tests our literal `<USER_MODIFY>` serialization, not a recovered Doubao prompt. The known slot name alone is not a model capability.

| Arm | Recognition context |
| --- | --- |
| `none` | Empty system context |
| `destinations` | Correct destination spellings from the selected entries |
| `pairs` | Same selected entries as newline-separated `source → target` |
| `tagged_pairs` | Identical pairs inside our literal `<USER_MODIFY>` wrapper |
| `all_pairs` | All 13 pairs, including reverse and expired records; intentionally unfiltered |

The last comparison changes memory contents and length as well as selection. It cannot isolate expiry, reverse suppression or each sorting feature individually. The run tests audio-stage recall of a frozen prior memory, not live mining, cloud extraction, storage durability or sequential counter updates.

## Results

Term hits are case-folded exact substring matches for Han and boundary-checked normalized matches for Latin. Error rates use the repository's frozen normalization and micro-averaged language-specific CER/WER/MER. They are lexical proxies, not semantic ratings.

| Context | Recurring terms / 16 | Improved / worsened speech vs none | Previously zero-error speech worsened / 17 | Mixed MER | Compute p50 / p95 |
| --- | ---: | ---: | ---: | ---: | ---: |
| None | 8 | — | 0 | 46/132 = 34.85% | 0.58 / 0.82 s |
| Selected destinations | 9 | 3 / 1 | 1 | 41/132 = 31.06% | 0.59 / 0.74 s |
| Selected pairs | 8 | 1 / 0 | 0 | 42/132 = 31.82% | 0.66 / 0.82 s |
| Selected pairs + wrapper | 8 | 2 / 1 | 1 | 40/132 = 30.30% | 0.74 / 0.90 s |
| Unfiltered pairs | 9 | 4 / 0 | 0 | 34/132 = 25.76% | 0.84 / 1.01 s |

All arms have zero normalized errors on seven Chinese cases (74 characters) and eight English cases (54 words). This partly reflects the easy controls and normalization: case and punctuation differences are removed, while spaces inside names such as `GitHub`/`Git Hub` still affect the error count. Those zeros are not general-language accuracy estimates. All arms emitted nothing on both nonspeech fixtures and had zero token-limit hits. All eight ordinary/homophone/reverse control sentences retained zero normalized error, although other term cases did regress.

The unfiltered context has the lowest aggregate MER here, but recurring-term recall rises by only one case. It also changes terms absent from the selected pool. These data do not establish that unfiltered memory is preferable, or that the selected Doubao-style policy is harmful in general.

## Findings visible in actual outputs

1. **The correct spelling can help without the wrong form.** For the utterance whose reference is `这个窗口是用MyGo实现的。`, the no-memory model wrote `卖狗`, destination context wrote `MyGo`, and every pair-based arm wrote `麦狗`. The pair changed one wrong Chinese rendering to another instead of applying the arrow as a rule.
2. **A wrapper can reverse a correct recognition.** `缓存服务使用Redis。` was correct with no context, destinations and plain selected pairs. The wrapper arm wrote `缓存服务使用瑞迪斯。`. Conversely, it fixed one Kubernetes utterance that plain selected pairs missed. Its 8/16 recurring-term score hides this exchange of a gain for a regression.
3. **Lower error distance can conceal a wrong entity.** For `用Codex检查这个修改。`, no memory wrote `用寇代克斯检查这个修改。` and destination context wrote `用 Kubernetes 检查这个修改。`. MER errors fell from four to one because both Latin names are single MER units. This is an erroneous entity substitution, not successful recall. It is included in the reported lexical improvement count and explicitly rejected as an entity-quality gain.
4. **An irrelevant context can alter an unlisted word.** GitHub is outside the selected eight, but one GitHub case improved with selected pairs and another split `GitHub` into `Git Hub` with destination context. The latter accounts for that arm's previously zero-error regression. A generic prompt-context effect must not be described as direct recall of a selected entry.
5. **Selection misses and expired terms stay visible.** The four deliberately unselected GitHub/Codex cases have term hits 1/4 without memory, 0/4 with destinations, and 2/4 in each pair arm. The expired Qdrant target is never exactly recovered, including in the unfiltered condition that contains it.

## Interpretation

This establishes a working recognition-stage memory input on a local 1.7B ASR checkpoint. It does **not** establish that this checkpoint understands correction pairs as Doubao's speech model does. On the primary recurring-term subset, selected pair injection produces no net recall gain; adding our slot wrapper adds a concrete regression. Merely moving the existing mapping strings into an audio model is insufficient evidence for adopting that architecture.

The useful next question is whether pronunciation-aware memory formatting or model adaptation improves recurrence without corrupting already-correct speech. It requires a fresh frozen corpus with approved prior-session corrections, independent natural recordings and entity-level review. Any prompt iteration on these disclosed cases is development tuning and must be tested elsewhere. No product model, vocabulary behavior or default was changed.

## Limits and replay

- One authored diagnostic set, two test TTS voices, no independent meaning review, no uncertainty interval or population inference. Some transliterations are artificial and not standardized pronunciations.
- The checkpoint is a community 8-bit conversion, with no source BF16 output-parity comparison. Recognition uses complete audio, not causal microphone streaming or native insertion. No Windows execution, simultaneous ASR/text rewriting or offline network audit was performed.
- Timing is model-call compute after warmup; it excludes speech capture, endpoint detection and target insertion. Background desktop activity and thermal state were not isolated. RSS/MLX peaks are cumulative process peaks, not independent per-arm allocations.
- Provisioning encountered a rate-limited metadata request, slow source/Xet transfers and three short HTTP ranges. Failed/cancelled acquisition records are retained; the successful complete weight digest was checked before any model result. None of these are inference observations or passing model tests.

Replay without a model:

```sh
node benchmarks/reproduction/user-modify-asr-v1/report.mjs \
  benchmarks/corpora/user-modify-asr-v1 \
  benchmarks/runs/user-modify-asr-v1/run-v1 \
  .workspace/user-modify-asr-replayed.json
```

The scorer rejects incomplete/duplicate/unpaired observations and changed JSON/audio/output bytes. Archived replay matched the initial summary exactly. The [source README](../../reproduction/user-modify-asr-v1/README.md) documents new inference with separately provisioned weights and runtime.
