# Phonetic evidence ablation: complete Qwen3-1.7B checkpoint

Status: completed exploratory engineering benchmark. All 154 test cases have raw streaming ASR, an offline acoustic control and all six correction treatments for Qwen3-1.7B: 924 measured generations, 154 raw controls, 12 development generation warmups and two separate offline-control warmups. The user cancelled Qwen3-4B; its two completed test generations and warmups are retained in the archive and excluded from this report. The original two-model execution remains marked cancelled_by_user. This report uses a hash-linked, coverage-verified view of the complete 1.7B checkpoint, without new inference. No product correction or desktop insertion behavior changed.

## Findings

The acoustic-only second pass lowers all three error rates, improving 60 utterances and worsening seven without corrupting a previously correct utterance in this corpus. The text-only editor changes little, and adding text-derived pinyin provides no measured gain. Text plus universal audio IPA worsens all three rates; the IPA-only condition largely fails to recover orthographic speech.

Passing raw token scores into this unadapted editor causes repetition, evidence-field emission and demonstration copying. The two score-bearing conditions and IPA-only also hit the output limit. These are measured system failures, not clean estimates of the intrinsic value of confidence or phonetic information. The fixed confidence route still invokes 136 of 154 cases and does not rescue the IPA condition. A dedicated pronunciation-aware acoustic/P2G model remains a different experiment.

For nonspeech, text-only and text-pinyin return a pair of quotation marks rather than an empty string in all four cases. These count as strict nonempty emissions even though speech normalization removes the punctuation. The scores condition copies a demonstration, and IPA conditions can emit phone symbols. Therefore silence-emission counts and normalized corruption counts have different meanings.

## Quality

Lower error rates are better. CER uses 60 natural Mandarin read-speech cases, WER 60 natural English read-speech cases, and MER 30 synthetic technical mixed-language cases. Four nonspeech cases are counted separately. Improved/worsened compare utterance edit counts with the same streaming ASR source. The originally-correct denominator includes correctly silent cases; these counts are normalized transcript proxies, without independent semantic review.

| Condition | Mandarin CER | English WER | Mixed MER | Improved / worsened vs raw | Originally correct corrupted | Silence emissions | Output-limit hits |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Raw streaming ASR | 8.17% | 7.63% | 36.39% | 0 / 0 | 0 / 60 | 0 / 4 | 0 |
| Offline acoustic control | 5.82% | 3.97% | 22.10% | 60 / 7 | 0 / 60 | 0 / 4 | 0 |
| qwen3-1.7b / Text | 8.42% | 7.53% | 36.39% | 1 / 2 | 1 / 60 | 4 / 4 | 0 |
| qwen3-1.7b / Text + text-derived pinyin | 8.42% | 7.63% | 36.39% | 0 / 2 | 1 / 60 | 4 / 4 | 0 |
| qwen3-1.7b / Text + token scores | 85.77% | 39.27% | 231.27% | 1 / 46 | 26 / 60 | 4 / 4 | 7 |
| qwen3-1.7b / Text + pinyin + scores | 72.65% | 13.12% | 82.48% | 0 / 48 | 23 / 60 | 4 / 4 | 3 |
| qwen3-1.7b / Text + audio IPA | 19.06% | 10.38% | 73.32% | 1 / 14 | 5 / 60 | 4 / 4 | 0 |
| qwen3-1.7b / Audio IPA only | 314.73% | 334.69% | 275.20% | 0 / 153 | 59 / 60 | 4 / 4 | 2 |

Raw denominators are 808 Chinese characters, 983 English words and 371 mixed units. Insertions can make error rates exceed 100%; results are not clamped. No failed or truncated proposal is replaced with its source. The automatic output-contract count detects leading JSON, chat/reasoning markers and truncation; it is not a complete semantic validator. Supplemental literal-field diagnostics also count evidence markers emitted inside a response. Exact errors, denominators, strata, format failures and paired speaker-cluster bootstrap intervals are in [summary.json](summary.json). Offline-control paired comparisons are in [supplemental.json](supplemental.json).

## Confidence and selective correction

The fixed threshold is -0.7410774230957031, the development emitted-token lower quartile; no reference was used to fit it. Scores are conditional log-softmax values on the selected transducer path, not calibrated word-correctness probabilities. Risk is the negative minimum emitted-token score. The table evaluates error detection for spoken cases; missing scores are excluded from AUC and counted explicitly.

| Stratum | Scored / missing spoken cases | AUC | Precision | Recall | False-positive rate |
| --- | --- | --- | --- | --- | --- | --- |
| all | 149 / 1 | 0.840 | 68.70% | 96.77% | 73.21% |
| zh | 60 / 0 | 0.830 | 54.55% | 92.31% | 58.82% |
| en | 60 / 0 | 0.799 | 66.10% | 100.00% | 95.24% |
| mixed | 29 / 1 | 0.857 | 96.43% | 96.43% | 100.00% |

An utterance-level counterfactual invokes correction for any below-threshold emitted token or missing scores, and otherwise retains raw recognition. The policy invokes 136/154 cases; 5 have no scores. It does not inspect reference text or choose between treatments based on their outputs. This is an exploratory routing calculation over existing observations, not actual selective execution or uncertain-span phonetic decoding.

| Routed treatment | Mandarin CER | English WER | Mixed MER | Originally correct corrupted |
| --- | --- | --- | --- | --- |
| qwen3-1.7b/text | 8.42% | 7.53% | 36.39% | 1 / 60 |
| qwen3-1.7b/text_pinyin | 8.42% | 7.63% | 36.39% | 1 / 60 |
| qwen3-1.7b/text_scores | 57.05% | 39.27% | 229.92% | 24 / 60 |
| qwen3-1.7b/text_pinyin_scores | 52.35% | 13.12% | 79.78% | 19 / 60 |
| qwen3-1.7b/text_ipa | 19.06% | 10.38% | 73.32% | 5 / 60 |
| qwen3-1.7b/ipa_only | 242.70% | 328.79% | 267.12% | 44 / 60 |

The detector is utterance-level. These results do not establish accurate word localization, calibrated confidence or a safe automatic replacement threshold. Language, length and rare correct tokens can affect its minimum-score heuristic. Complete detector counts are in [confidence.json](confidence.json).

## Timing

Hardware: Apple M4, 32 GiB RAM, darwin/arm64; four CPU threads, warm model residency, fresh uncached prefill per request, greedy non-thinking INT4 ONNX generation with input/output ceilings of 4,096/256 tokens. The checkpoint has its own process; conditions are shuffled within it.

| Correction condition | Generation p50 (s) | Generation p95 (s) | Output-contract failures |
| --- | --- | --- | --- |
| qwen3-1.7b / Text | 2.97 | 4.88 | 0 |
| qwen3-1.7b / Text + text-derived pinyin | 3.21 | 5.60 | 0 |
| qwen3-1.7b / Text + token scores | 5.47 | 15.62 | 7 |
| qwen3-1.7b / Text + pinyin + scores | 5.42 | 14.27 | 3 |
| qwen3-1.7b / Text + audio IPA | 3.69 | 6.33 | 0 |
| qwen3-1.7b / Audio IPA only | 5.96 | 12.83 | 2 |

Generation is engine prefill plus decode, without model load, real audio pacing, native UI, target insertion or cross-request prefix reuse. The frozen configuration described intended timing as isolated, but observed background workloads mean that condition was not achieved. Timings do not establish a controlled latency ranking. Acoustic component distributions in the primary aggregate cover all 272 development/test files, not only the 154 test files. The offline acoustic control has compute p50/p95 0.13/0.34 s over test cases; it is whole-file audio processing. Separate component percentiles must not be summed into an end-to-end percentile.

## Method and limits

- X-ASR streaming INT8, greedy decoding, 20 ms accelerated chunks, 0.8 s endpoint silence and 0.4 s tail padding provide the fixed raw recognition and real token log scores. A separately pinned offline X-ASR INT8 checkpoint supplies the acoustic-only second-pass comparator.
- Text-derived pinyin uses pypinyin 0.55.0 TONE3, neutral tone 5; English runs remain text. It provides a pronunciation constraint, without a new audio observation.
- Allosaurus 1.0.2 uni2005 recognizes a single universal IPA path directly from original audio, without ASR text, reference, language hint or vocabulary context. Its frontend internally uses 8 kHz MFCCs. This is not a Mandarin-tone-aware multitask model or verified phonetic ground truth.
- The six conditions share the conservative editor prompt and three generic text-copy demonstrations. IPA-only removes the current ASR transcript from that editor; its result does not evaluate a separately trained phoneme-to-grapheme decoder. No context/memory or real N-best lattice is supplied.
- The initial prompt systematically echoed evidence/JSON. That attempt was stopped and retained as failed, and its numbers are excluded from representation-efficacy conclusions. Actual newline bytes were correct. After a development-only 36-observation output-contract preflight, the revised frozen prompt reran the same test set. This is reused-test exploratory evidence, not an untouched confirmatory test.
- The corpus has 44 test speaker/voice clusters, including only four natural Mandarin speakers and four synthetic mixed-language voices. Read speech and synthetic technical fixtures do not establish spontaneous-dictation performance. Model exposure to upstream public benchmarks is not ruled out.
- Paired cluster bootstrap intervals are exploratory with unadjusted multiple comparisons. Automatic error rates and correct-source corruption do not establish intent, protected-name, number or negation preservation. No release-quality acceptance is claimed.

See the [full procedure](../../../docs/research/phonetic-asr-benchmark.md), [paper and competitor investigation](../../../docs/research/phonetic-asr-correction.md), [frozen configuration](../../configs/phonetic-asr.json) and [008 evaluation contract](../../../specs/008-evaluation.md).

## Evidence

Audio, exact prompts, recognition/phone outputs, generated proposals, token traces, warmups, artifact manifests and host-contention samples are retained in the [research archive](../../migration/README.md). Use `pnpm bench:archive:report --out <new-directory>` to regenerate the complete primary report without weights. The cancelled parent and failed initial attempt retain their original states. [supplemental.json](supplemental.json) records completed run/result hashes, model and code hashes, runtime versions, preflight results and invalid-attempt provenance. The exact original complete primary report SHA-256 is 209dc76abe141ccfa0cb057987b271944b1aba861745deb3b00661b560660a54.
