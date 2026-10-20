# Scene rewrite alignment and acceptance

## Product target

Work dictation should become an organized work message; a coding request should
become an organized instruction to the receiving developer or agent. Reordering,
headings, bullet points, concise paraphrase and equivalent numeric formatting
are legitimate transformations. Actors and their responsibilities, amounts,
deadlines, conditions, negation, uncertainty and every requirement must survive.
The editor must not answer the dictated question or implement a dictated task.

[ADR 018](../decisions/018-staged-gguf-rewriting.md) adopted the staged local
templates and delivery-only validation. This document remains the study record,
not a release acceptance. It proposed replacing layout-only scene mapping and
scene-independent lexical delivery gates.
Verbatim mode retains its exact-content contract. Original recognition,
local speech cleanup, model proposal and actual delivered text stay distinct.

## Bounded competitor alignment

[Byte-verified client evidence](evidence/rewrite-validator-comparison.json)
shows stable ASR/context post-processing and original/enhanced results in
Chatterfly, and explicit completion/no-rewrite/error lifecycle in Doubao.
Their inspected client completion paths do not enforce Arion's character
coverage or novelty ratios. Their server-side prompts, fidelity checks and
measured semantic quality remain unknown. Align the workflow and transformation
contract; do not invent their server validator or add a cloud inference route.

## Proposed responsibilities

1. **Generation policy:** pass the actual work/prompt/email/chat scene to a
   versioned local template. Work and prompt instructions explicitly allow
   grouping and rewording without requiring spoken ordinals. Preserve the
   dictated language; a scene is not permission to translate. Demonstrate role,
   condition and uncertainty preservation and the difference between editing a
   request and answering it. Keep context as reference data and close its
   sections consistently.
2. **Delivery validity:** retain session/revision/generation identity,
   cancellation, focus/mode invalidation, once-only commit, empty/truncated
   output, protocol/reasoning/scene-marker checks and an absolute output budget.
   No semantic approval can override a failed delivery-validity check.
3. **Content fidelity:** replace lexical-coverage/novelty/negation-count gates
   as unconditional acceptance rules for scene rewriting. Preserve explicitly
   technical identifiers, paths and URLs; normalize equivalent quantities
   before comparing them. These checks protect declared facts rather than all
   words. They still cannot prove role, condition or negation scope, which need
   separate calibrated semantic evidence. A local semantic model is an
   experimental candidate, not a correctness oracle.
4. **Diagnostics:** retain lexical metrics in private evaluation for triage and
   record only bounded verdict/timing/count fields in product support logs.
   Raw prompts, dictated text and context remain excluded from product logs.
5. **Failure handling:** retain completed local cleanup and an explicit
   organization outcome under the current product contract. Do not strip
   internal markers, insert truncated answers or silently widen gates to hide
   a generation or verifier failure. Keep the recognized original available.

The proposed local semantic audit is evaluated only on final fixed text in
this study. It does not establish preview scheduling or native commit safety.
Its latency must be counted in final delivery before adoption; model load and
cold-context costs remain separate observations. Additional verifier calls per
preview are not adopted here.

Implementation order after a candidate passes development selection:

- P0: update `packages/core/domain/scene.go` and the versioned templates in
  `apps/desktop/infrastructure/llm/prompt` so work/prompt are explicit model
  policies, rather than aliases chosen only by spoken ordinals. Add language,
  role/condition and request-versus-answer regressions before changing routing.
- P0: make the rewrite validation contract in `packages/core/domain/rewrite.go`
  explicit about scene/mode, delivery validity and a separately identified
  fidelity report. Technical-token/quantity checks operate on declared source
  spans and equivalent values, never a blanket digit or word substitution.
  A semantic verifier, if qualified, is a domain-owned port implemented by the
  isolated local worker; framework/model calls do not enter the domain.
- P0: wire final verification in `packages/core/application/live.go` with the
  same immutable source identity, deadline and cancellation as generation.
  Keep current preview behavior during the initial final-only pilot until
  preview acceptance and timing are measured. Recheck focus and once-only
  commit after verification, and preserve separate raw/cleanup/proposal values.
- P0: update bounded diagnostics, regression cases and `specs/009` with the
  accepted behavior in that same change set. The current normative production
  gates are not rewritten merely because this experimental proposal exists.
- P1: package and roll out only after independent content review and actual
  native offline/lifecycle tests pass. A failed generation or verifier gate
  blocks this sequence; it does not authorize relaxing another gate silently.

## Frozen experiment

[`protocol.json`](../../benchmarks/reproduction/scene-rewrite-alignment-v1/protocol.json)
is the pre-generation treatment and gate declaration. The experiment uses the
same provisioned Qwen3 1.7B INT4 weights, tokenizer, CPU provider, four threads,
greedy generation and production engine. An isolated Go build overlay supplies
the experimental templates, permits actual scene tags and closes dictation
sections. Production source and the installed app are unchanged.

Forty-eight authored English/Chinese texts span work messages, coding prompts,
email and chat. Their 24 content families are divided into development and
test families; language siblings remain together. Technical strings provide
mixed-vocabulary probes but do not establish mixed-speech recognition coverage.
Twenty-four separate authored source/candidate pairs calibrate equivalent
paraphrases and known harmful role/time/number/condition/intent changes.

The two generation treatments are current production and scene-aware prompts.
Each actual generation is scored under three delivery policies on exactly the
same raw output:

| Policy | Mechanical gate | Content gate |
| --- | --- | --- |
| strict | Current production validator and timeout/cancellation | Current lexical rules |
| envelope | Empty/truncated/cancelled/marker checks and absolute size bound | None; diagnostic risk-control arm, never an adoption recommendation |
| semantic | Same envelope | Local source/candidate meaning verifier; malformed or uncertain verdict rejects |

The meaning verifier receives only source, candidate and scene. Corpus labels,
reference rewrites and evaluator constraints are never transmitted to either
generation or verification. The verifier uses the same checkpoint but a
separate process/template; correlated model failures remain possible.
Its structured pair is excluded from speech cleanup to avoid altering the
evidence it is supposed to judge. This isolation is a research overlay only.

Both raw and delivered outputs are retained, including rejections and partial
outputs. Rejection delivers the separately retained cleanup for scoring.
Generation and verifier times are recorded independently and added for final
latency. A fixed 10-second request deadline sends real cancellation to the
worker. Timeouts count as failures, not missing observations. No request uses
a microphone, target editor, foreign screen or network inference service.

## Scoring and selection

Scoring reports complete case coverage, raw and delivered atomic-constraint
passes, structural-layout passes, rejected constraint-passing candidates,
accepted constraint failures, calibration false acceptance/rejection and
p50/p95 final time. A heading or line break alone is not a content pass.
Regular expressions permit declared alternate expressions but remain proxies:
they are not independent human semantic ratings and may miss a relation error
or flag an otherwise valid paraphrase. Preserve all failing rows for review.

Calibrate and score development data first. Freeze the development-only
selection, including an explicit no-promotion decision when gates fail, before
opening test generation. Run the predeclared factorial on test for diagnosis;
do not tune templates, thresholds or labels from test outputs. Preserve the
initial failed results if a later experiment uses a changed policy.

The predeclared diagnostic gates require zero harmful calibration acceptance,
at least 95% faithful calibration acceptance, at least four additional delivered
content-and-layout passes on each 24-case split, no additional delivered
constraint failures, full coverage and warm final p95 below 1.5 seconds.
All conditions are required for a candidate recommendation; their numerical
thresholds are experiment decisions, not competitor measurements.

## Batch verification and release acceptance

P0: replay the frozen verifier calibration and paired text generation; verify
asset/source/fixture/row hashes, coverage and every retained failure. Run the
existing core lifecycle tests to establish the unchanged foundation's behavior,
not to claim the research policy is integrated. Commands and raw observations
are linked from the [reproduction workflow](../../benchmarks/reproduction/scene-rewrite-alignment-v1/README.md).

P0: before production adoption, build independently reviewed natural-dictation
and frozen-ASR corpora, split by speaker/session/content family, with unchanged
good text, verbatim, role swaps, dates/amounts, repaired values, uncertainty,
conditional clauses, technical identifiers, unrelated context and injection.
Use the [existing rewrite evaluator](rewrite-evaluation.md) with at least three
independent blinded raters and complete matched meaning/style/fluency and
hallucination coverage. Automatic proxies or a same-model judge cannot replace
that gate. Require no new harmful delivery and improved scene appropriateness.

P0: after integration, verify actual offline microphone-to-editor behavior,
single commit, original availability and no stale publication under cancellation,
focus/mode changes and new source revisions on each supported target OS.
Run solo/concurrent-ASR and cold/warm strata with actual native timing/resource
traces. Stop-to-final p95 includes all final generation, queueing and validation
and must meet the declared 1.5-second target on the supported device/load stratum.

P1: broader models, prompt variants or semantic verifiers need their own frozen
development selection and fresh test families. Reject a candidate that merely
trades false rejection for harmful delivery, or trades structure for answering
the user's dictated request. Keep production defaults until all applicable
adoption gates have evidence.

## Completed experiment

[The completed study](../../benchmarks/results/scene-rewrite-alignment-v1/README.md)
retains 96 actual generation and 113 verifier calls, full planned coverage and
the development-frozen no-promotion decision. Scene/envelope raises the frozen
test content-and-layout proxy from 10/24 to 18/24 but permits concrete unsafe
generation. The meaning verifier accepts only 7/12 faithful calibration pairs
and also misses bad natural generator outputs. Scene/semantic achieves 10/24
on test with final request p95 around 4.81 seconds on this shared host.

The v1 templates omit an explicit same-language instruction, and two test
constraint failures are scorer synonym misses. These limitations are disclosed
without retuning the frozen experiment. No production policy was changed.
The proposal remains gated on a new development candidate and fresh test data,
independent meaning review, and native lifecycle/latency acceptance.

## Focused prompt v2 follow-up

[The completed v2 study](../../benchmarks/results/scene-rewrite-prompt-v2/README.md)
adds an explicit output language and isolates demonstrations by actual scene and
language. Development selected the demonstration variant before 32 fresh cases
and 24 previous-test regressions were evaluated against production. Every one
of the 144 generated outputs has a source-linked assistant review; this is not
independent human review.

On fresh data, faithful-and-scene-ready raw proposals increase from 9/32 to
22/32, but faithful outputs decrease from 28/32 to 26/32, including one failed
envelope in each arm. Five matched cases acquire new material errors. Across
all 56 evaluation cases, strict validation rejects six faithful v2 proposals
and accepts two unfaithful ones. No production template, routing or gate is
changed. The candidate fails adoption; the next experiment must use new
families for request/completion, full-constraint retention and context isolation,
followed by fresh test data and the independent/native acceptance above.

## Supported recognition repair and proportional editing

The user-selected examples establish a concrete style preference: group an
owner with the matching task/date, separate an email salutation from its body,
and preserve concise natural wording instead of forced headings or formal
phrases. [The structural/context follow-up](../../benchmarks/results/context-repair-v1/README.md)
retains 16 structural and 72 context-conditioned outputs with complete assistant
review. The two user-selected pairs are exposed demonstrations, not test data.

Content fidelity must be evaluated against reviewed intended speech, allowing
plausible ASR homophones and identifier spellings to be repaired from relevant
OCR discussion. A supported replacement is not automatically unsupported new
content. In four authored relevant-error probes, the contextual candidate
repairs a recipient name and function identifier, but misses two other terms.
It preserves seven of eight negative controls but adopts a corrupted OCR name
in the remaining case. Production accepts both good repairs and that bad
replacement. Neither blanket ASR copying nor blanket OCR trust meets the target.

All inputs in this diagnostic are authored ASR-like/OCR-like text, not actual
acoustic or screenshot processing. Future acceptance needs separately retained
true ASR/OCR artifacts and reviewed intended text, unchanged-good controls,
context relevance/error/ambiguity strata, supported-correction recall and
harmful-overwrite rates. Grouping and repair must pass together. Production
behavior remains unchanged; independent and native release gates still apply.

## Complete text-validator removal experiment

[The validator-ablation study](../../benchmarks/results/rewrite-no-validator-v1/README.md)
compares two prompt candidates in development, freezes the selected candidate
for diagnosis, and compares it with the previous candidate composite on 24
known context regressions and 24 newly authored inputs. All 128 raw proposals
and completed cleanup values have assistant review, and all 384 policy deliveries
are exported under strict, inherited envelope and no-text-validation policies.
These are text-only, explicitly assigned-scene diagnostics, not native behavior
or independent human semantic ratings.

On the new cases, selected-prompt strict delivery reaches 22/24 faithful and
20/24 scene-ready outputs. Complete text-gate removal reaches 21/24 and 19/24,
releasing one good structured request alongside an omitted restriction and an
unusable protocol echo. OCR regression also blocks adoption. The no-text-gate
arm bypasses every text check, but preserves runtime deadline/cancellation and
source ownership; no test grants stale work the right to publish. The current
production validator is unchanged. Specific false rejects still warrant a
calibrated replacement, while wholesale removal fails this experiment.

## Orthographic alignment prompt follow-up

[The OCR-alignment rerun](../../benchmarks/results/ocr-alignment-v2/README.md)
separates different ASR/OCR spelling from conflicting facts, teaches complete
identifier/name copying for a unique relevant phonetic match, and explicitly
preserves source dates/amounts/status and all restrictions. Two wording variants
share demonstrations; the shorter direct policy is frozen from development
for diagnosis, not promotion. All 128 proposals and cleanup values are inspected,
and 384 strict/envelope/none deliveries are exported with per-case scene/template
provenance. Inputs remain authored text rather than executed ASR/OCR.

Required repair improves from 0/4 to 2/4 on known context probes and from 2/8
to 4/8 on new probes. New raw faithful proposals increase from 12/24 to 16/24,
but known-regression raw fidelity decreases from 18/24 to 17/24, with more
context replacements and one demonstration leak. Strict cleanup retention
raises selected delivered fidelity to 22/24 on old and 17/24 on new cases;
these are not all successful LLM outputs. Ambiguous identity guesses and one
correct-text corruption still pass strict. The candidate fails adoption.

An initial order/cache suspicion was a cross-version comparison error. Eight
identical-input/template phase overlaps and a separate 12-call cold/forward/
reverse probe show no output mismatch in their bounded cases. No runtime cache
fix or production behavior change follows. The report retains both the
source-based diagnosis and this negative parity evidence.

## Fixed-model input packaging follow-up

The [packaging comparison](../../benchmarks/results/rewrite-packaging-v1/README.md)
compares XML/JSON/labeled sections with identical models, prompts, demonstrations
and absolute decoding budget. All 150 outputs were inspected. Labeled sections
improve raw fidelity from 17/24 to 22/24 on old regressions and 17/24 to 20/24 on
new probes, but required fresh repairs only improve from 3/7 to 4/7. Ambiguous
spelling guesses and missing task grouping remain failures. It is a development
baseline, not a release decision. No production integration is claimed. The
[offline HTML](../../benchmarks/results/rewrite-release/index.html) consolidates
reviewed iterations and explicitly tracks remaining macOS release evidence.

The clean-demonstration / instruction-language development follow-up reviewed
54 outputs: both alternatives fell from 16/18 faithful to 14/18, so evaluation
was not run. The [scene-task wrapper](../../benchmarks/results/rewrite-task-wrapper-v1/README.md)
then reviewed 132 outputs with matched models/templates: new scene-ready outputs
improved 14→16/24, but old fidelity declined 22→21/24 and request p95 increased
1150→1590 ms. A question became an assertion and passed strict validation. Neither
follow-up replaces the labeled development baseline or qualifies for production.

## Entity evidence and actual Vision follow-up

The [entity-policy diagnostic](../../benchmarks/results/rewrite-entity-evidence-v1/README.md)
reviewed 100 native LLM outputs and 24 actual macOS Vision observations on 12
authored images. An explicit exact-match-first rule did not improve development:
both policies preserve 5/8 correct-source controls and repair 3/5 required terms.
Accurate Vision retains the fixture names/code, while fast loses much Chinese.
The product already uses accurate. A demonstration alias is emitted even with
empty OCR; coincidentally correct output must not be reported as OCR benefit.
Identical source/OCR strings can represent wrong-ASR or wrong-OCR worlds, so a
unique phonetic match alone cannot establish truth. OCR correction remains in
scope; stronger source/context evidence and natural-data validation are needed.

The original alignment-v1 96 generations and 113 audit outputs now also have
[complete retrospective review](../../benchmarks/reproduction/rewrite-retrospective-v1/README.md).
This adds current assistant judgments without altering original observations,
selection or proxy scores. The HTML distinguishes the evidence and includes
all original audit outputs separately from rewriting totals.

## Completed model and packaging diagnostics

The [checkpoint screen](../../benchmarks/runs/rewrite-model-screen-v1/report/review.md), [serializer screen](../../benchmarks/runs/rewrite-4b-packaging-v1/report/review.md) and [entity follow-up](../../benchmarks/runs/rewrite-4b-entity-v1/report/review.md) retain 56 personally inspected generations completed before the user excluded further 4B testing. On eight exposed work/email probes, larger-checkpoint fidelity stayed6/8; JSON reached7/8 but missed the recipient repair; exact-match priority fell7→6/8. No candidate qualified, and the16 frozen follow-up probes were not generated. These checkpoint/export comparisons do not isolate parameter count or quantization.

Subsequent inference must prioritize the latest model generations at <=2B; do not rerun4B or older Qwen generations. Historical1.7B observations remain reference evidence. Candidate acquisition must pin revisions, licenses, actual tensor/model counts and runtime bytes. Cross-runtime timings are not a model-only causal comparison.

## Latest <=2B generation screen

The [latest small-model screen](../../benchmarks/runs/rewrite-small-models-v1/README.md) reviewed48 unfiltered outputs from Qwen3.5-0.8B/2B text-only Q4_K_M bundles (actual tensors0.752B/1.882B). Both use the same pinned llama.cpp CPU4/GPU0/non-thinking configuration and24 fixed cases. Meaning is retained22/24 vs23/24; useful scene output20/24 vs23/24. Both match four known-answer repairs but still guess an ambiguous function;0.8B also replaces one dictated message with an OCR list.2B groups both owner/deadline probes. Cosmetic single bullets are reported separately from semantic failures. No model is promoted.

Without prompt caching, request p95 is2029/4466ms, maximum sampled RSS1.65/3.16GB. These are M4 diagnostic observations, not weak-machine acceptance, peak RSS or stop-to-insertion latency. Historical ONNX calls used prefix caching and are not a causal speed comparison. Continue0.8B prompt/cache work for constrained devices;2B remains a quality comparator. No older-Qwen or4B inference was added.

## Latest0.8B prompt and cache follow-ups

The [four-scene prompt package](../../benchmarks/runs/rewrite-small-prompt-v1/README.md) reviewed32 development outputs. Both packages retain meaning14/16; useful scene organization improves10→11/16, but the candidate loses getProfile repair, still guesses readCash and splits owners from deadlines. The predeclared development gate fails; frozen fresh24 are not generated. Numeric paraphrases are judged by meaning, not literal overlap:4 seconds preserves the correction away from40. No candidate is promoted.

The [default-batch cache screen](../../benchmarks/runs/rewrite-small-cache-v1/README.md) reviewed40 outputs:32 measured requests plus8 warmups over8 exposed probes, two cache policies and two orders. Actual prefix hits551/572 tokens reduce request p95≈2644→701ms, but raw cache parity is14/16. Both alternatives preserve meaning; this still fails the existing exact-greedy-output requirement. Order parity is16/16; synthetic warmup canaries do not appear in later bodies, which is not a general privacy proof.

A [logical-batch1 attempt](../../benchmarks/runs/rewrite-small-cache-parity-v1/README.md) fails before generation: the pinned native capability probe needs2 tokens. Failed status, original log, environment and corresponding source are retained. A separate [physical-microbatch1 screen](../../benchmarks/runs/rewrite-small-cache-parity-v2/README.md) retains normal logical capacity and reviews another40 outputs. Cache parity and order parity are16/16 each on this subset, but cached request p95≈1614ms; forward/reverse cached p95≈921/1614ms show substantial timing drift. Uncached p95≈9845ms and cold prefix warmups remain explicit. The drift cause is unproven. This is not stop-to-insertion latency or low-end/concurrent-ASR acceptance, and neither cache policy is activated in production.
