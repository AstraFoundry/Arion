# Phonetic ASR ablation procedure

This procedure implements the exploratory comparison proposed in [phonetic ASR research](phonetic-asr-correction.md). It measures real acoustic extraction and local CPU generation on macOS with Python 3.12. It does not train PYG-ASR or LLM-P2G, modify production correction, or establish native end-to-end latency.

The [completed Qwen3-1.7B result](../../benchmarks/results/phonetic-asr-1.7b-v2/README.md) contains all 154 cases and six conditions (924 generations). The user cancelled the remaining 4B checkpoint. Its parent execution remains cancelled; a hash-linked view verifies the complete 1.7B coverage without new inference and excludes every 4B observation. The two-model design below records the original plan, rather than a claim that both checkpoints completed.

## Frozen design

The [configuration](../../benchmarks/configs/phonetic-asr.json) fixes an X-ASR 160 ms INT8 transducer, greedy decoding, 20 ms input chunks, 0.8 s endpoint silence, 0.4 s tail padding and four CPU threads. Audio is replayed faster than realtime. Each file has fresh recognition state; endpoint-delimited results are retained separately. Text, emitted tokens, their timestamps and actual `ys_probs` log scores are saved. No ASR alternatives are fabricated.

An independently provisioned Allosaurus 1.0.2 / `uni2005` checkpoint recognizes the same original audio into one universal IPA phone sequence with `emit=1`, on CPU; its MFCC frontend internally resamples to 8 kHz. No language label, reference, expected word, lexicon or text-ASR result enters this phone recognizer. This is a generic phone sidecar, not a jointly trained Mandarin-pinyin/character system. Its approximate phone sequence is supplied as evidence, not treated as ground truth.

`pypinyin` 0.55.0 converts only the ASR text into `TONE3` pinyin with neutral tone `5`. Non-Chinese runs are preserved. Thus the English pinyin treatment repeats textual content; it does not add English grapheme-to-phoneme information. Audio IPA and text-derived pinyin have different provenance throughout the experiment.

Both Qwen3-1.7B and Qwen3-4B use the existing pinned INT4 ONNX exports, CPU ONNX Runtime GenAI diagnostic worker, four threads, greedy non-thinking generation, a 4,096-token input ceiling and a 256-token output ceiling. Tokenizer parity is checked against the reference tokenizer for each request. The models are loaded in separate processes, one checkpoint at a time. Each correction uses a fresh generator: warm weight residency with uncached prompt prefill. Development warmups are excluded; sample/treatment requests are shuffled within each checkpoint using the fixed order seed. Load cost is retained separately from generation.

| Treatment | Input to correction |
| --- | --- |
| Raw | Actual ASR text, no LLM |
| Text | ASR text |
| Text + pinyin | ASR text and pronunciation derived from that text |
| Text + scores | ASR text and emitted-token log scores, with a low-score flag |
| Text + pinyin + scores | Both auxiliary text representations |
| Text + IPA | ASR text and independently audio-derived IPA |
| IPA only | Audio-derived IPA; no original text, inferred word boundaries, language hint, context or memory |

All correction arms share the same conservative transcription instruction and three generic authored copy demonstrations (English, Mandarin and an empty utterance). These demonstrations contain no corpus reference content. They receive no cursor text, screen content, memory, current reference transcript, acceptable answer, protected-span annotation or error label. Correction is separated from style rewriting. Generated proposals are scored exactly as emitted, including incomplete generations; there is no source-text fallback or semantic repair. This does not measure the production validator or delivered text.

A required development preflight covers all languages, models and treatments with the exact prompt hash. Every text-bearing condition must return plain, complete text before test inference starts. IPA-only copying/length failures remain negative-control observations. The initial attempt lacked this gate and systematically echoed JSON/evidence despite requesting plain text; its interrupted observations are retained with failed status and cannot establish representation efficacy. Newline bytes were checked and were correct. The revised prompt was checked on development cases before the same test set was rerun. This reuse is an exploratory engineering diagnostic, not an untouched confirmatory test.

## Data and evidence

The archived speech corpus (`benchmarks/corpora/asr-v1/corpus.json`) has 274 fixtures: two training warmups, 118 development cases and 154 test cases, separated by source speaker/session/audio identity. The test set contains 60 natural AISHELL-1 Mandarin read-speech cases, 60 natural LibriSpeech English read-speech cases, 30 synthetic mixed-language technical cases and four nonspeech cases. Test speaker clusters total 44; Mandarin natural speech has only four speakers, and synthetic mixed speech has four voice clusters. Bootstrap precision is limited accordingly.

The manifest pins public-source revisions, access/license records, audio conversion and hashes. It is an existing project corpus, not a newly collected independent validation set. No spontaneous dictation, independently annotated phonetic truth or protected-span labels are added by this experiment. Model exposure to upstream public benchmarks is not ruled out.

Acoustic extraction runs train warmups and then all dev/test audio, verifying each audio digest and PCM format before inference. Development token scores determine the low-score threshold: their lower quartile, without reference-based fitting. The resulting threshold is a heuristic flag, not calibrated correctness probability. The original plan paired all 154 test cases across raw output and six conditions for each LLM, for 1,848 measured generations and 154 raw controls, with two development warmups per model/treatment. The requested completed scope contains the full 924-generation 1.7B checkpoint; the cancelled 4B remainder is not part of its result.

Archived evidence includes exact prompts, raw ASR/IPA/pinyin, decoder JSON, generation/token events, output-limit flags, completed/failure status, configuration and code hashes, input/model/runtime hashes, Git revision/dirty state, host/power settings, warmups and monotonic timings. Output directories and aggregate report files refuse overwrite. Failed/incomplete runs cannot produce a complete report. Exact research inputs and outputs are retained in the [benchmark archive](../../benchmarks/migration/README.md); personal recordings, weights and runtime environments remain outside Git.

## Metrics and interpretation

The automatic output-contract detector covers leading JSON, special chat/reasoning markers and truncation. It does not identify every explanation, quotation-only empty reply, repetition or mid-response evidence leak. Supplemental field-literal diagnostics and bounded raw-output inspection document these failures; no outputs are repaired.

The scorer reuses [the speech normalization and CER/WER/MER definitions](benchmark-methodology.md). It reports micro-averaged error counts and denominators per language; it never averages unlike Chinese character and English word denominators into one headline error rate. Natural read speech and synthetic technical speech retain separate strata.

Improved/worsened counts compare each proposal's edit distance with the same raw ASR. Corruption counts utterances whose raw recognition was correct after normalization and whose proposal introduces an error. Silence emissions are separate, including when the LLM generates text from an empty source or noisy phones. Output-limit hits remain visible and are scored as incomplete proposals. Punctuation and stylistic quality are outside normalized speech error rates.

Paired speaker-cluster bootstrap intervals compare each treatment with raw ASR and the same model's text-only arm. They are exploratory, with unadjusted multiple comparisons; repeated conditions do not create additional speakers. These automatic measures do not replace independent meaning, name, number or negation review.

The separate confidence analysis uses negative minimum emitted-token log score as utterance risk and raw reference error as the label. It reports AUC and the development-quantile threshold's precision/recall/false-positive rate. Empty-score spoken cases are counted as missing, not assigned invented confidence. This is an utterance-level detection diagnostic, not localization or probability calibration. A minimum score can also depend on utterance length and rare correct tokens.

Supplemental analysis applies a fixed utterance-level routing rule to the already observed proposals: invoke correction if any emitted-token score is below the development threshold, or if no scores exist; otherwise retain the raw recognition. The rule does not receive references, phone correctness or correction outcomes. It was declared before completed efficacy inspection. This counterfactual measures quality and invocation coverage, not actual routed execution latency or switching uncertain spans into phonetic tokens. All proposed treatments retain their individual results; the test set does not choose a winning policy.

An additional acoustic-only control runs the existing pinned offline X-ASR INT8 transducer on the same 154 test audio files, with two separate development warmups, greedy decoding and four CPU threads. Inputs contain audio only; the evaluator adds references after inference. Files are bounded below the production segment ceiling. Whole-file processing is an acoustic second-pass comparator, without realtime preview or a phonetic output mode. It runs after correction to avoid adding concurrent CPU demand.

Generation timing includes prompt prefill and token decoding on a warm model. Acoustic extraction separately measures accelerated ASR compute, pinyin conversion and phone recognition. The frozen configuration calls the intended correction timing isolated, but the observed shared desktop was not isolated from background work; that timing condition was not achieved, and no UI preview, audio pacing, host insertion, contention-controlled product latency or energy measurement is claimed. In particular, generation p95 must not be relabelled stop-to-insertion p95.

## Reproduction

Use the pinned pnpm version and the [evaluation command contracts](../../tools/evaluation/README.md#phonetic-correction-ablations). Provisioned private assets remain required; no command automatically acquires models. New inference still needs local models/runtimes. Existing results can be rescored with `pnpm bench:archive:report --out <new-directory>`. Example using the archived corpus and retained local inference assets:

```sh
make phonetic-extract CORPUS=benchmarks/corpora/asr-v1/corpus.json EVAL_PYTHON=.workspace/phonetic-venv/bin/python MODELS=.workspace/models/store PHONE_MODELS=.workspace/phonetic-models RUNTIME_ARTIFACTS=.workspace/phonetic-models/acoustic-artifacts.json OUT=.workspace/benchmarks/phonetic-repeat-acoustic
make phonetic-bench CORPUS=benchmarks/corpora/asr-v1/corpus.json EVAL_PYTHON=.workspace/ablation-venv/bin/python MODELS=.workspace/ablation-models RUNTIME_ARTIFACTS=.workspace/phonetic-models/llm-runtime-artifacts.json LLM_ARTIFACTS=.workspace/phonetic-models/llm-artifacts.json EVIDENCE=.workspace/benchmarks/phonetic-repeat-acoustic PHONETIC_PREFLIGHT=.workspace/benchmarks/phonetic-asr-v2/smoke-a OUT=.workspace/benchmarks/phonetic-repeat-correction
make phonetic-report RESULTS=.workspace/benchmarks/phonetic-repeat-correction OUT=.workspace/benchmarks/phonetic-repeat-summary.json
```

The phone checkpoint archive is pinned to SHA-256 `7dc4f20802012f64ac07deb04ade263f199c839521e58eae83c99d79028fc4de`, acquired from the author's `v1.0/uni2005.tar.gz` release. This is acquisition provenance for local research, not a product distribution/license decision. The pinned dependency lock and private artifact manifest identify the actual implementation used.
