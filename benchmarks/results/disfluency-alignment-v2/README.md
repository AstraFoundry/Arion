# Local disfluency cleanup and organization

Arion now has a separate local deletion stage before model organization, following the stage separation established by [Doubao artifact research](../../../docs/research/competitor-disfluency.md). The new stage improves the frozen text constraints and one default-configuration audio case. It does not establish equivalent quality to Doubao, whose annotation producer and controlled speech outputs remain unknown.

## Implementation

- `domain.CleanSpeech` (`speech-spans-v1`) emits typed deletion spans in the unchanged source. Rendering validates UTF-8 boundaries, ordering, overlap, known kinds, quoted/code protection and word-boundary safety. The producer recognizes declared hesitation/scaffolding, limited stutters and repairs with a repeated frame or matching typed value. Ambiguous expressions remain model input.
- The worker organizes cleaned text with the existing `live-v2` prompt and Qwen3 1.7B INT4. No additional model request or cloud route was added. A generative span annotator was tried and rejected for omissions and erroneous date deletion; the adopted producer is Arion's deterministic implementation.
- Organization rejection, timeout and unavailability retain completed cleanup and their explicit failure outcome. Original source remains copyable even when organization fails. Raw recognition, corrected source, cleanup annotations/text, proposal and delivered text remain distinct.
- `arion-rules-v3` preserves repeated first-person `I` for the repetition stage, instead of joining it into `II`, and protects quoted spelled letters from acronym joining. Audio review exposed this pre-existing merge and a blind spot in the first exact-string annotations.

## Paired results

Counts are delivered **exact-string constraint passes**, not semantic accuracy. Every row uses actual local inference; fixtures and raw outputs are retained in the [research archive](../../migration/README.md). Comparisons pair the same case IDs. Both workers use Go 1.27.1, macOS arm64 and `CGO_ENABLED=0`; ASR drivers require `CGO_ENABLED=1`. An initial worker build mismatch was corrected, and all 95 previously run baseline text outputs were checked against the matched build before final publication.

| Set | Cases | Baseline | Product | Improved / regressed |
| --- | ---: | ---: | ---: | ---: |
| Multilingual development probes | 21 | 11 | 21 | 10 / 0 |
| Existing oral-language challenges | 16 | 9 | 13 | 4 / 0 |
| Historical regression set | 30 | 28 | 28 | 0 / 0 |
| Originally frozen challenges, now post-review regression | 28 | 19 | 27 | 8 / 0 |
| Fresh test frozen after the audio review | 16 | 11 | 15 | 4 / 0 |
| Real ASR + synthetic audio, product default filler removal on | 8 | 5 | 6 | 1 / 0 |
| Separate diagnostic, filler removal off | 8 | 6 | 6 | 1 / 1 |

Across the 111 text cases, passes increase from 78 to 104, with 26 improvements and no baseline-to-product constraint regression. The historical set includes one verbatim bypass; its other 29 cases use generation. Historical self-correction remains 8/8. The new 16-case test is Mandarin 6/6 and English 9/10. The older 28-case set is Mandarin 7/8, English 8/8 and exploratory Spanish/French/Japanese 4/4 each. Those language counts are text constraints only, not ASR coverage or multilingual semantic certification.

The original 28 cases were frozen before the span experiments. After audio review revealed the acronym issue, they were treated as regression data for the final rule fix. The fresh 16 cases were authored and hashed before their first baseline/product model outputs; their constraints were not changed after inspection. All cases have one developer author and no independent annotations.

## Audio configuration audit

The existing audio driver hardcoded `RemoveFillers=false`, while product preferences default to true. Both configurations are retained separately. The driver now defaults to the product policy, exposes `-remove-fillers` (`FILLERS=0` through the pnpm entry point), and records the effective Boolean in every row. Scoring rejects mixed policies, and reporting verifies a declared policy. Legacy rows have null policy telemetry; their disabled configuration is documented from the retained driver and observations.

All eight raw ASR texts match between arms in each configuration. Raw synthetic-speech error is Mandarin CER 0/46 and English WER 5/31 in both configurations. This small TTS result supplies no natural-speech accuracy estimate. Both multisegment fixtures produce a validated coordinator preview before stop; rendered native preview and target insertion were not measured.

The default configuration improves the Mandarin repetition case, with no newly failing case. Its remaining failures are a retained Mandarin demonstrative-shaped filler and English recognition/spacing errors, including `back up` versus the required noun spelling. The disabled-filler diagnostic improves one repetition case but regresses one filler case; its unchanged 6/8 total must not be presented as zero regression. The original pronoun probe also allowed the corrupted `II` to pass; the fresh cases add explicit first-person constraints, and the product audio now preserves the intended pronoun.

## Timing observations

Apple M4, 32 GiB, CPU only, four intra-op threads, greedy non-thinking generation, maximum context 4096 and generation ceiling 384. Fixed prompts and model/runtime hashes are in the report. Shared weights remain resident within each arm; initial load is separate from request timing. Full live-prefix schedules are replayed for text, followed by a final request.

| Measurement | Baseline p50 / p95 | Product p50 / p95 |
| --- | ---: | ---: |
| Fresh text final request | 1385 / 2323 ms | 1025 / 2012 ms |
| Default audio stop to final | 1033 / 1550 ms | 1683 / 2091 ms |
| Filler-off diagnostic stop to final | 1731 / 2347 ms | 1255 / 2067 ms |

The candidate's default-audio latency is higher in this batch. Other desktop work, including a separate benchmark and checks, was active; these are descriptive measurements, not an isolated speed comparison or a claim that the new stage is faster. No RSS, first-token or rendered-preview performance claim is made.

## Remaining failures and acceptance boundary

The fresh test still loses a grammatically meaningful repeated auxiliary (`had had`) during model organization. This fails in both arms and worsens a source that passed its constraints. The independent cleanup preserves it, but the current organizer and fidelity proxies do not reliably preserve tense. Three older oral-language challenges, two historical cases and one originally frozen challenge also remain failing. Exact-string checks can miss changed meaning or reject equivalent wording.

These runs do not satisfy independent blind meaning/style review, natural disfluent speech, strict reviewed rewrite-corpus acceptance, native editor/IME delivery or Windows gates. No actual Doubao output comparison was run, no model default was changed, and the running installed app was not replaced or restarted.

## Validation and reproduction

Core tests with race detection, core tests/vet/format, desktop tests/vet, 58 evaluation tooling tests, quote/conditional/negation/uncertainty controls, failure/timeout/missing-worker delivery, original-copy selection and cancellation/once-only lifecycle checks pass. A five-second span-contract fuzz run executed 26,483 inputs after its retained seed corpus. Actual GenAI inference confirms that fresh execution and incremental prefill/live rewind produce identical output on the cleanup cache-parity case, and session teardown releases private KV state. This is a bounded parity test, not proof for every possible input.

[Summary](summary.json) contains 14 complete runs and identities; [paired comparisons](paired.json) distinguish improvements and regressions. The [configuration](../../configs/disfluency-alignment-report-v2.json) pins corpus, rows and manifest hashes. Reports reject incomplete coverage, changed hashes, existing output paths and inconsistent audio policies. Model/runtime/binary/source/build provenance is retained in the aggregate and archived manifests. The [earlier snapshot](../disfluency-alignment-v1/README.md) is retained as an audit trail, not the final default-configuration report.

Aggregate scoring needs only the archived evidence (`pnpm bench:archive:report --out <new-directory>`). With the separately provisioned model/runtime available, use the existing `pnpm live:text`, `pnpm live:audio` and `pnpm live:report` entry points. Always choose new output paths. For example:

```sh
CORPUS=benchmarks/corpora/disfluency-fresh-v1/corpus.json LLM_MODEL=.workspace/ablation-models/qwen3-1.7b CORRECT=1 OUT=.workspace/disfluency-text-repeat.jsonl pnpm live:text
MANIFEST=benchmarks/corpora/disfluency-audio-v1/audio-fixtures/manifest.json CORPUS=benchmarks/corpora/disfluency-audio-v1/corpus.json MODELS=.workspace/models/store BUNDLE=xasr-160-xasr-final LLM_MODEL=.workspace/ablation-models/qwen3-1.7b OUT=.workspace/disfluency-audio-repeat.jsonl pnpm live:audio
RUNS=benchmarks/configs/disfluency-alignment-report-v2.json OUT=.workspace/disfluency-summary-repeat.json pnpm live:report
```
