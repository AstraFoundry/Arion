# Rewrite pipeline investigation and evidence

Open the [overall interactive evidence index](index.html) for the chosen pipeline, paired reports, native reproduction, ablations, rejected experiments, source review and hashes. This study establishes a bounded improvement in name continuity; it does **not** establish a general semantic-fidelity pass.

## Reproduction and root cause

The [installed-worker native reproduction](native/README.md#installed-worker-coordinator-development-reproduction) uses actual Vision, production streaming/offline ASR, coordinator events and the immutable installed LLM worker over synthetic audio and authored screen fixtures. Case 01 publishes validated correct `Chatterfly` previews, then whole-audio finalization supplies `chatter fly`; the constructed approved mapping that matched the earlier recognized source is no longer recalled, and final delivery remains wrong. This reproduces the reported class, not the user's original recording or exact `chad fly` acoustic error. It does not prove a formatter changed an unchanged canonical input.

The chosen change carries earlier **recognized and rule-corrected** source separately, recalls relevant existing approved vocabulary from both sources, nominates supported complete names subject to current-intent qualification, handles forced ambiguous keep without model sampling, and reuses a completed preview proposal only when the complete final request fingerprint matches. Original selector/formatter prompts and formatter context isolation remain. Conflicting current phrase/literal/correction intent takes precedence. Generated preview text does not become recognition evidence or learned memory.

## Completed results and limits

| Evidence | Observed result | Interpretation |
| --- | --- | --- |
| [Chosen 24-case comparison](evidence/final-v5/reports/final-validation-report-reviewed/index.html) | Supported repairs 0/9 baseline → 9/9 chosen; preservation 15/15 in both; reviewed phases 39/48 → 48/48 | Authored exploratory contrasts; baseline is documented verbatim composition of two completed runs, not one newly generated arm |
| [Four-arm development ablation](evidence/final-v5/reports/final-development-report-reviewed/index.html) | Correct phases baseline12/16, no-prior14/16, no-screen10/16, full15/16; correct-preview entity regressions 2 → 0 | Eight exposed pairs; remaining full-arm miss is an already-wrong first ASR draft |
| [Chosen native coordinator replay](native/final-v5/summary.json) | All three Chatterfly final names corrected; two runs have validated correct previews | Screen-off duplicate uses constructed approved memory and repeats original speech; a cancelled preview is not successful preview evidence |
| [Matched cache ablation](native/final-v5/README.md) | Complete requests match except Live; raw and final text match; cache-on2 worker calls versus off3 | One final request avoided; no typical latency claim. Shorter-silence confounded pilot remains archived |
| [Rejected V1/V3 contrasts](evidence/rejected-v1/reports/fresh-contrast-report-reviewed/index.html) | V1 harms8/11 preservation finals; V3 harms5/11 and times out once | Both rejected despite repairing all five supported positive finals |
| [Native regressions](native/final-v5/summary.json) | `getUserProfile` repaired but “不要补充注释” invented; competing reviewer name unresolved; both numeric contrasts preserved | Naming or transport success does not imply semantic success |

Reviews are from a separate assistant task that inspected every actual stage, not independent human or blind review. Inputs are authored text or synthetic TTS/images with actual native OCR/ASR/LLM; approved memory was constructed after initial observations. There is no natural-speech held-out accuracy, native editor insertion, Windows acceptance or universal semantic safety claim. Nonrandom sequential arms, repeated speech and authored silence limit latency/statistical inference. The installed app was not replaced.

[Chosen stage archive](evidence/final-v5/README.md), [rejected/failed archive](evidence/rejected-v1/README.md), [final native archive](native/final-v5/README.md) and [six infrastructure checks](evidence/final-checks/results.json) retain original bytes, source identities, failures and hashes. The first baseline follow-up failed compilation with **zero model inference**; its logs and source remain. Baseline2 used a test-only overlay for a candidate-specific test helper, retaining the actual stage harness and baseline production source.

## Reusable workflow

Use the repository's pinned pnpm version and new output destinations:

```sh
pnpm rewrite:stages:run --arm candidate --cases /absolute/cases.json --out /absolute/new-run --runtime /absolute/runtime --model /absolute/model-directory
pnpm rewrite:stages:test
pnpm rewrite:stages:report --cases /absolute/cases.json --arm baseline=/absolute/baseline.jsonl --arm candidate=/absolute/candidate.jsonl --init-review /absolute/review.json
pnpm rewrite:stages:verify /absolute/report-or-archive
```

`run` accepts `baseline`, `candidate`, `no-prior`, and `no-screen`; it validates the selected model against the catalog digest/size, records observed runtime identities and effective settings, snapshots source before inference and compiles that snapshot through a Go overlay. Failed attempts remain failures. Model provisioning/download and installed-app mutation are outside this workflow. [Reverse-engineering research](../../../docs/research/rewrite-stage-preservation.md) distinguishes static Doubao/Chatterfly artifact evidence from verified runtime behavior.

## Initial discovery protocol

This corpus probes the report that correct OCR-informed preview text becomes incorrect after final rewriting. Inputs are constructed Mandarin/English transcript and screen-text strings; no microphone, recognizer, OS OCR, or external editor is exercised. These are development discovery cases, not a held-out quality set or a reproduction of the user's original recording. Names and identifiers are fictional/common technical strings authored for this investigation; no external fixture license is required.

`discovery.json` contains 36 paired requests: expanding transcripts, explicitly labeled final acoustic variants, and identical-source controls. The current GGUF engine does not branch on `Live`, so identical pairs are controls rather than an assumed reproduction. `checks` are evaluator-only substring probes and never enter the model request. They are diagnostic proxies, not independently reviewed semantic scores; for example an expanding request may retain a negation absent from the short preview.

The opt-in `TestNativeStagePreservation` test executes `gguf.Engine.Run` against the actual pinned llama.cpp runtime and local model. A pass-through HTTP transport records exact selector and formatter completion requests/responses. Each observation includes cleanup, output, native token timings, delivery verdict, errors, and phase latency; observations are fsynced individually. An exclusive output path prevents overwriting previous attempts. Stage classification follows the exact recorded requests; it does not infer a successful selector from the final text alone. Test success means the run finished, not that quality checks passed.

Run from the repository root with absolute paths:

```sh
ARION_GGUF_RUNTIME='/Applications/Arion.app/Contents/Resources/llm' \
ARION_GGUF_MODEL='/Users/steven/Library/Application Support/Arion/models/qwen3.5-2b-q4-k-m-metal' \
ARION_STAGE_CASES="$PWD/benchmarks/reproduction/rewrite-stage-preservation-v1/discovery.json" \
ARION_STAGE_OUT="$PWD/.workspace/stage-baseline.jsonl" \
go test ./apps/desktop/infrastructure/llm/gguf -run '^TestNativeStagePreservation$' -count=1 -timeout=45m
```

`ARION_STAGE_FILTER` accepts comma-separated exact case IDs. `ARION_STAGE_FORMAT_ONLY=1` is a separately labeled diagnostic that sends each source directly through the selected formatter with actual native inference; it bypasses cleanup and spelling selection and must not be reported as full-pipeline evidence. Use corrected source fixtures for isolating downstream corruption and label any such derived corpus separately. Record artifact digests, runtime/build identity, Git dirty state, hardware, power mode, effective settings, and corpus/output hashes alongside archived observations before publishing a report. Keep failed and partial attempts. Freeze discovered regressions separately before optimizing; do not relabel tuned discovery cases as held out.

## Model-free report and manual review

`report.py` compares any number of completed arms with exactly two phases per frozen case. It rejects duplicate/missing observations, partial runs, changed raw inputs, and reviews bound to other observations. Failed observations remain in the full denominators. Partial/failed attempt files remain independent archived evidence and cannot become a completed paired report.

```sh
python3 benchmarks/reproduction/rewrite-stage-preservation-v1/report.py \
  --cases /absolute/frozen-cases.json \
  --arm baseline=/absolute/baseline.jsonl \
  --arm candidate=/absolute/candidate.jsonl \
  --arm no_guard=/absolute/no-guard.jsonl \
  --init-review /absolute/manual-review.json
```

The generated review starts entirely `unreviewed`. For each phase, review source, constraints, OCR relevance, raw selector choices, formatter input/output and delivered output. Set `fidelity` to `correct`, `incorrect`, or `not-assessable` and provide a concrete rationale. A review verdict applies to the delivered result; raw formatter damage is reported separately. Retain `unreviewed` where evidence is insufficient. Reviews bind the exact corpus, every arm file, and each complete observation by SHA-256. A changed model result requires a new review rather than reusing labels.

Repeat the same corpus/arm arguments with `--review /absolute/manual-review.json --out /absolute/new-report-directory` to create the interactive standalone `index.html`, full `report.json`, input/source copies and a hash manifest. Optional `--provenance /absolute/provenance.json` copies artifact/runtime/hardware provenance unchanged; the reporter does not fabricate missing execution provenance. `--verify /absolute/new-report-directory` checks every archived file and inventory.

The first `checks` entry is the evaluator's target-name/identifier substring probe. Remaining checks are shown separately as constraint substrings; they do not affect preview correctness. The preview-to-final regression proxy denominator is the number of preview phases containing the target; total pairs and entity-evaluable pairs remain shown. Raw formatter damage requires the expected target in formatter input and a change in its exact occurrence count in the completed native formatter proposal, covering loss and duplication. Fixed-gap formatter JSON is independently stitched with its fixed parts before counting; protocol JSON itself is never treated as rendered text. These proxies neither establish acoustic correctness nor replace manual meaning review. Stage latency uses recorded completion round-trip wall time with nearest-rank p50/p95; it is not native UI first-token latency or independent speaker statistics.

Reporter tests use fabricated JSON only and are infrastructure checks:

```sh
python3 -m unittest discover -s benchmarks/reproduction/rewrite-stage-preservation-v1 -p test_report.py
```
