# Scene rewrite alignment experiment

The [alignment proposal](../../../docs/research/scene-rewrite-alignment.md)
and [frozen protocol](../../reproduction/scene-rewrite-alignment-v1/protocol.json)
define the product contract, treatments and acceptance gates before generation.
This actual local CPU study ran 96 generation requests on 48 synthetic texts and
113 verifier requests (24 calibration pairs plus 89 mechanically valid generated
outputs). Seven truncated generations were retained and not sent for semantic
approval. All planned cases completed; no inference output was mocked or replaced.

**Decision: do not promote this candidate.** Development selection was frozen as
no-promotion before test inference. The existing lexical gates cause false
rejections, but neither envelope-only delivery nor this local meaning verifier
provides an acceptable replacement with the tested scene prompt.

## Data and treatments

Twenty-four development and 24 frozen test texts cover Chinese/English work
messages, coding requests, email and chat. Language siblings share a content
family/split; all cases and labels are authored by one developer. These are
synthetic text diagnostics, not independent semantic accuracy, natural speech,
frozen-ASR or competitor-output measurements.

Installed production and isolated scene-template workers use the same provisioned
Qwen3 1.7B INT4 assets, greedy decoding, four CPU threads and production generation
engine/budget policy. Each raw generation is evaluated under strict production
validation, mechanical envelope-only validation, and envelope plus a local meaning
verifier. Model/context/prompt and exact overlay/binary identities are retained.
The scene arm changes both scene instructions and explicit work/prompt tags; it
is not an isolated attribution to one prompt sentence. Neither arm uses screen
context or the microphone. Background host load was not controlled.

## Results

The table reports **delivered atomic-constraint and layout proxy passes**, not
human semantic judgments. Rejections deliver completed local cleanup for scoring.

| Generation / delivery | Development | Frozen test | Test accepted outputs | Test final p95 |
| --- | ---: | ---: | ---: | ---: |
| production / strict | 12/24 | 10/24 | 22/24 | 2.18 s |
| production / envelope | 12/24 | 12/24 | 24/24 | 2.18 s |
| production / semantic | 7/24 | 10/24 | 5/24 | 2.78 s |
| scene / strict | 9/24 | 8/24 | 8/24 | 4.45 s |
| scene / envelope | 15/24 | 18/24 | 23/24 | 4.45 s |
| scene / semantic | 10/24 | 10/24 | 8/24 | 4.81 s |

Calibration accepted 7/12 faithful rewrites and rejected all 12 authored harmful
rewrites. The faithful acceptance gate was 95%, so the verifier failed before
test. It also accepted bad generated candidates outside that calibration set:
an invented review-before-delivery condition and a fabricated explanation of
TS2345. A zero-miss result on 12 mutations therefore does not establish safe
semantic verification. Same-checkpoint generation/auditing can share errors.

Envelope-only scene delivery improves structure counts but permits translation,
unsupported action/completion claims and unnecessary explanations. For example,
the source says payment development is done; the candidate adds that it is
deployed. An English pagination request acquires a review-before-delivery
condition copied from a demonstration. The v1 scene and audit templates omit
an explicit same-language instruction: this is a prompt-design gap, not proof
that the checkpoint cannot preserve language. No template was retuned on test.

The current lexical gates still reject valid structured output. The Chinese
export update correctly becomes progress/review/release/owner sections but is
rejected as `length`. An equivalent `three in the afternoon` to `3:00 PM`
conversion is rejected as `new_number`. The scene/strict arm rejects ten
test candidates that pass the frozen fact proxies; this is a triage count,
not ten proven semantic false rejections.

## Scorer limitations and inspected failures

The frozen scorer records four accepted constraint failures for scene/semantic
on test. Post-generation inspection identifies **two scorer false failures**:
`尚无法确定原因` preserves uncertainty and `线上链接保持不变` preserves the unchanged
link, but their alternate forms were absent from the frozen regexes. The other
two accepted rows translate English into Chinese. Scores were not edited after
test; the original output and the correction notes coexist in
[`analyst-observations.json`](../../runs/scene-rewrite-alignment-v1/analyst-observations.json).

The scorer can also miss harmful added content or changed request intent despite
all required words remaining. The semantic auditor accepted two such development
outputs. These counts must not be described as semantic accuracy or used alone
for production acceptance. Single-analyst notes are not independent blind ratings.

## Acceptance outcome

Coverage and paired artifact identities pass. The verifier's faithful acceptance,
the scene/semantic content-plus-layout gain and warm final p95 gates fail. The
test proxy no-additional-failure gate also fails, with the scorer caveat above.
Concrete unsafe outputs independently support the no-promotion decision.

The existing 1.5-second final latency target is unmet by the tested candidate;
its reported time includes the semantic audit. Current production also exceeds
that target in this test block. These observations are from a shared host and
do not establish a causal throughput regression or native stop-to-insertion p95.

P0 next: make language, request-versus-answer and no-invented-condition constraints
explicit in a smaller scene-specific policy; independently calibrate any semantic
checker against faithful restructuring and naturally occurring generator errors;
evaluate a new candidate on fresh test families after development selection.
Replace lexical hard gates only after a candidate meets both preservation and
usability gates. P0 release acceptance additionally requires independent blind
meaning/style/fluency review and native offline lifecycle/latency scenarios.
No production prompt, validator or model default was changed by this study.

## Reproduction and verification

[`reproduction/scene-rewrite-alignment-v1`](../../reproduction/scene-rewrite-alignment-v1)
contains corpus construction, frozen templates/protocol, isolated build overlay,
batch runner, model-free report and integrity tests. The
[raw archive](../../runs/scene-rewrite-alignment-v1) retains completed dev/test
rows, calibration, initial and checked reports, overlays, hashes and 288 blinded
delivered-output tasks with a separate mapping and **unrated** three-rater rubric.

```sh
pnpm rewrite:alignment:verify
pnpm rewrite:alignment:test
pnpm rewrite:alignment:report \
  --dev benchmarks/runs/scene-rewrite-alignment-v1/dev \
  --test benchmarks/runs/scene-rewrite-alignment-v1/test \
  --out .workspace/alignment-report-replay
```

Verification and report replay do not load a model. Integrity tests reject failed,
missing, duplicated, modified, rescored or reclassified observations. Existing
core domain/application tests passed; this verifies the unchanged foundation,
not integration of the research policy into native delivery.
