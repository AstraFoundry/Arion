# OCR spelling alignment: prompt correction and paired rerun

Decision: the revised prompt restores some supported spelling repairs and
improves newly authored raw outputs, but introduces harmful context choices and
does not qualify for production adoption. Production prompt files, validator
and native delivery behavior are unchanged.

## Diagnosis from fixed input/output evidence

The prior no-validator-v1 anchored prompt missed all four required repairs in
the known context cohort. Its predecessor repaired the recipient and function
identifier in those same four probes. The source/context bytes, exact outputs
and their hashes are retained in [diagnosis.json](../../reproduction/ocr-alignment-v2/diagnosis.json).
This is regression in how the LLM uses fixed supplied context; no OCR capture
was rerun and no OCR extraction accuracy change is established.

Three distinct problems must not be combined into one overlap score:

1. **Supported repair suppressed:** “If spellings conflict … keep the dictated
   word” is ambiguous when the purpose of repair is to replace a different ASR
   spelling. Preservation of intended facts must permit spelling correction.
   This is a prompt-ambiguity hypothesis, not isolated causal proof.
2. **Repair coverage incomplete:** prior demonstrations poorly cover complete
   camel-case identifiers and ordinary homophone repair. The generator sometimes
   outputs `load cache` rather than the exact `loadCache` visible in code context,
   or preserves a phonetic platform transliteration despite a canonical token.
3. **Wrong reference use:** a unique relevant phonetic entity, multiple plausible
   candidates, a differently pronounced OCR error, and an instruction embedded
   in OCR need different treatment. Previous behavior includes both missed
   repairs and unjustified name selection. More conservative copying does not
   by itself solve malicious or unrelated context replacement.

The new common contract explicitly distinguishes spelling difference from fact
conflict, allows a unique relevant phonetic match, copies the **complete** OCR
identifier/name spelling, and preserves dictated dates/amounts/status/conditions.
Demonstrations cover split identifiers, homophones, canonical terms, wrong OCR,
ambiguous names and hostile screen instructions. Natural short messages and
owner/task/date grouping remain the formatting target.

Two variants use the same demonstrations: `aligned` states the repair decision
directly; `staged` describes alignment before formatting within one generation.
This is not a two-call correction pipeline. Versus the previous prompt, policy
and examples change together, so the study cannot isolate their causal effects.

## Frozen development selection and evaluation

Sixteen exposed development cases produce 32 real outputs. Both variants have
13/16 faithful proposals, 3/5 required repairs and the same remaining material
failures. The shorter aligned policy is selected **for diagnosis only** before
new-case inference. Their development p50 values are effectively tied (about
673 ms). The original selection description mistakenly said aligned had the
lower median; [the retained clarification](../../runs/ocr-alignment-v2/dev/selection-clarification.json)
corrects that wording without changing selection, prompts or their pre-evaluation
hashes. No test result was used to choose a different candidate.

Evaluation generates 96 additional outputs: frozen prior anchored versus
selected aligned on 24 known context regressions and 24 newly authored inputs.
The new cohort has eight required repairs, four already-correct controls, four
unrelated-context controls, four ambiguity controls, one corrupt-OCR control
and three conflicting/hostile-context controls. It spans work/email/prompt/chat.
Scenes are assigned by fixtures, not inferred from applications. Every report
entry shows assigned/effective scene and actual worker prompt version.

Every one of the 128 main proposals and completed cleanup values has explicit
assistant review against evaluator-only intended text. All three text-delivery
policies are replayed on the exact same proposals, exporting 384 full deliveries.
Inputs are authored ASR-like/OCR-like text and inference uses the real provisioned
Qwen3 1.7B INT4 CPU native worker. This does not execute speech recognition,
screenshot OCR, automatic scene matching or native insertion. These are not
independent human ratings or independent unseen speaker/content-family claims.
Development and old regression deliberately overlap.

## Raw generation results

| Cohort | Prompt | Required repairs succeeded | Faithful proposals | Harmful context changes | Damaged already-correct controls |
| --- | --- | ---: | ---: | ---: | ---: |
| Known context, 24 | Prior anchored | 0/4 | 18/24 | 2 | Not this cohort's label |
| Known context, 24 | New aligned | 2/4 | 17/24 | 4 | Not this cohort's label |
| New inputs, 24 | Prior anchored | 2/8 | 12/24 | 5 | 2/4 |
| New inputs, 24 | New aligned | 4/8 | 16/24 | 3 | 1/4 |

Required-repair recall and whole-message fidelity are separate. A corrected
term copied into an unrelated screen answer is not a successful message repair.
Harmful-context counts include following an OCR instruction, replacing dictation
with OCR, choosing an unresolved name or adopting a corrupt OCR spelling. The
new known-regression arm also has one separate **demonstration leakage** failure
without any actual request context. Thus the positive repair gain is insufficient
for the frozen no-new-harm adoption requirement.

Concrete changes:

- **Restored repairs:** `load cash → loadCache` and `回归侧试 → 回归测试` now work
  in the known context probes. In new cases, `save cash → saveCache`,
  `read config → readConfig`, `会议记要 → 会议纪要` and
  `谭玉辰 → 谭钰宸` retain the complete request/conditions and are successful.
- **Remaining misses:** Kubernetes, Grafana, Terraform transliterations and
  several email recipient spellings remain uncorrected. One email name repaired
  by the previous prompt is missed by the new one.
- **Better restraint in some cases:** the new prompt preserves 梁若宁 instead
  of corrupt OCR 梁苦宁, retains error-code/file-only restrictions, and keeps the
  negative unlock question instead of following `DONE` instructions.
- **New wrong choices:** the new prompt selects `getCash` among two equally
  plausible code identifiers and 叶语涵 among two homophonic people. It also
  corrupts an already-correct 会议纪要 clause into malformed text.
- **Persistent replacements:** hostile context still replaces a deployment
  request with immediate-notification text. Some old work probes become unrelated
  lunch/meeting text. A no-context probe emits `DONE. 已上线` from a demonstration.

One no-context `load cash → loadCash` normalization is marked explicitly as
unverified, rather than declared a proven wrong identity: without code evidence
its exact intended identifier is unknown. Ordinary valid spelling cleanup such
as `回归侧试 → 回归测试` may also be legitimate without relevant OCR. Neither
is credited as evidence that OCR caused a repair. These judgments are inspectable
and do not affect the clear context-replacement failures blocking adoption.

## Actual delivered text under the three policies

`strict` retains the current validator and completed cleanup after rejection.
`envelope` removes content rules while retaining the existing limited protocol/
empty/truncated/output-budget checks. `none` bypasses every text gate but does
not grant cancelled, timed-out or stale work permission to publish. No native
lifecycle behavior is established by this replay.

| Cohort | Prompt | Strict faithful | Envelope/none faithful | Strict required repairs delivered |
| --- | --- | ---: | ---: | ---: |
| Known context, 24 | Prior anchored | 19/24 | 18/24 | 0/4 |
| Known context, 24 | New aligned | 22/24 | 17/24 | 2/4 |
| New inputs, 24 | Prior anchored | 16/24 | 12/24 | 2/8 |
| New inputs, 24 | New aligned | 17/24 | 16/24 | 4/8 |

The new strict arm's 22/24 known-regression result includes **five cleanup
retentions**, not 22 successful LLM rewrites. The validator blocks the new
screen/demo replacements while permitting the good repairs. It still accepts
ambiguous name/identifier guesses and malformed wording. There are no faithful
raw proposals rejected by strict in this specific evaluation. This finding does
not erase documented false rejections in previous experiments.

## Bounded runtime parity check

An initial suspicion of order-dependent output was caused by comparing different
prompt versions. Exact cross-phase comparison finds eight identical input/
template overlaps with no output differences. A separately frozen supplementary
probe runs four selected inputs as first request in a new process, then in
forward and reverse request order: all 12 observations match by input, including
the bad generations. Every probe output is retained and reviewed.

There is no evidence of order/cache-caused changes **in this bounded check**.
It is not a general runtime-parity guarantee, and no cache workaround or engine
change was introduced. [The full probe report](../../runs/ocr-alignment-v2/report/parity.md)
and raw plan/results permit replay.

Evaluation request p50/p95 on the shared Apple M4 are about 671/1001 ms for prior
and 694/962 ms for aligned. Loads are separate. These sequential small strata
exclude concurrent ASR, real OCR capture, native stop-to-final timing and live
scene switching; they do not establish a reliable product speed advantage.

## Reviewable artifacts and decision

- [Every proposal, intended message, context and all policy deliveries](../../runs/ocr-alignment-v2/report/review.md).
- [Every evaluation output without text validation](../../runs/ocr-alignment-v2/report/without-validator.md).
- [Summary](../../runs/ocr-alignment-v2/report/summary.json), [full delivered JSONL](../../runs/ocr-alignment-v2/report/deliveries.jsonl), [scene/template identities](../../runs/ocr-alignment-v2/report/scenes.json).
- [Corrected work prompt](../../reproduction/ocr-alignment-v2/templates/aligned-work.json), [email prompt](../../reproduction/ocr-alignment-v2/templates/aligned-email.json), [developer prompt](../../reproduction/ocr-alignment-v2/templates/aligned-prompt.json), [chat prompt](../../reproduction/ocr-alignment-v2/templates/aligned-chat.json).
- [Frozen protocol and reproduction](../../reproduction/ocr-alignment-v2/README.md).

The revised wording is useful experimental evidence for restoring orthographic
repair, but is not robust enough to replace production defaults or justify
removing validation. Further tuning must use development data and new frozen
acceptance cases, retain both missed repairs and harmful overwrites, and obtain
independent review on actual consented ASR/OCR artifacts plus native acceptance.
