# Prompt optimization and complete text-validator removal

Decision: the new prompt improves organization on newly authored cases, but is
not a qualified replacement. Removing the whole text validator reduces delivered
quality and exposes protocol output. Retain production behavior while replacing
specific false-reject rules only after calibrated evidence; neither the new
prompt nor validator removal was activated in the application.

## Controlled experiment

Two new candidates share an explicit clause/owner/condition/uncertainty/request
preservation policy. `minimal` has one same-scene demonstration; `anchored` has
four including developer restrictions, negative questions and context correction
or restraint. Sixteen exposed development cases produce 32 raw outputs. After
inspecting every output and cleanup, the anchored candidate was frozen for
**diagnosis only**: 14/16 intended-message passes versus minimal 12/16, with
remaining hostile-context and recipient-repair errors. Both fail promotion.

Evaluation compares the selected prompt against a frozen **prior-candidate
composite**: context-repair-v1 for work/email and v2 examples for prompt/chat.
This baseline is not the installed production prompt. Both arms generate all
24 known context regressions and 24 newly authored Chinese cases, yielding 96
additional raw outputs. The new cases contain known semantic challenge types;
they are not independent unseen speakers/content families. All scenes are
fixture-assigned, not automatically detected. Per-case effective scene and
worker-reported template version are displayed throughout the report.

All 128 raw outputs and their completed cleanup values were individually
reviewed by the assistant against evaluator-only intended messages. This is
not independent human semantic evaluation. Inputs are authored ASR-like/OCR-like
text, not recorded recognition or executed screenshot OCR. The same local
Qwen3 1.7B INT4 model, four-thread native worker and hash-pinned v1 isolated
production-engine overlay serve every arm. No target editor is written.

## What removal means

Each exact raw proposal is replayed under three policies; generation is not
rerun or silently repaired when policy changes:

| Policy | Text checks | Rejection behavior |
| --- | --- | --- |
| strict | Current domain validator, including lexical/number/negation and known protocol/empty/truncation checks | Retain completed local cleanup with explicit verdict |
| envelope | Remove content gates; retain the existing diagnostic envelope for empty, truncation, known protocol markers and absolute output budget | Retain completed cleanup |
| none | Remove every text gate, including markup, relative/absolute text length, truncation and empty checks | Deliver worker text exactly on runtime success |

Runtime deadline/cancellation/source-ownership protection is not a text
validator. `none` cannot turn cancelled/stale work into a valid session result
or fabricate a result after timeout. Those controls are retained in the replay;
actual native lifecycle behavior was not tested. In this run every request
completed, so `none` delivers all raw proposal strings unchanged. Empty and
truncated bypass semantics are separately exercised by tests, not fabricated
as inference observations.

The inherited envelope recognizes a declared marker set, not all possible
markup. In particular, an unknown `<done>` tag is not caught mechanically.
Envelope acceptance is not a claim of semantic or presentation correctness.

## Newly authored cases: delivered result

“Faithful” includes required supported ASR repair. “Scene ready” additionally
requires useful natural organization and a valid mechanical envelope. A missed
repair is recorded separately from a model-invented fact. Counts below are
**delivered text**, including cleanup after rejection, rather than raw-pass rates.

| Prompt | Policy | Faithful / 24 | Scene ready / 24 | Raw proposals delivered | Envelope-invalid proposals delivered |
| --- | --- | ---: | ---: | ---: | ---: |
| Prior composite | strict | 22 | 16 | 17 | 0 |
| Prior composite | envelope | 19 | 14 | 24 | 0 |
| Prior composite | none | 19 | 14 | 24 | 0 |
| New anchored | strict | 22 | 20 | 21 | 0 |
| New anchored | envelope | 21 | 20 | 23 | 0 |
| New anchored | none | 21 | 19 | 24 | 1 |

The new prompt improves developer action/constraint lists and retains a
first-person “I will confirm and reply” email that the prior composite changes
into an unrelated not-finalized-draft claim. It also preserves a negative file
question and seat/no-seat travel branches that the prior prompts replace with
screen/demo content. It still omits a no-settings-change restriction in another
chat request and misses supported recipient/platform repairs.

Strict-to-none changes are retained case by case in `summary.json`:

- **Useful rewrite released:** new `f-prompt-all` cleanly organizes the Store.go
  investigation and all three restrictions, but strict rejects it as
  `unsupported_content`. None releases it, improving this case's organization.
- **Lost constraint delivered:** new `f-chat-question` drops “先不要帮我修改设置”.
  Strict delivers the intact cleanup; envelope/none deliver the incomplete
  request. This is not a false rejection.
- **Protocol output delivered:** new `f-email-no-greeting` repeats the source
  under `[email]`, `[dictation]`, `[screen]`, `[response]` and commentary. Its
  underlying facts remain present, but the complete text is unusable as email.
  Strict/envelope retain the clean source; none publishes all of it unchanged.
- **Other good proposals released:** the prior composite has three faithful
  proposals rejected on relative length/novel wording: retry detail with
  repetition, a natural timeout/file-scope paraphrase, and a context-supported
  `getProfile` request. These have presentation limitations; getting them through
  a gate is not automatically a scene-quality gain.
- **Other errors released:** prior `f-email-actor` changes the uncertainty's
  referent; `f-prompt-context` imports a database-migration instruction and
  `<done>` from screen; two chat cases become screen/demo content. Strict blocks
  these four proposals, while none and the inherited envelope publish them.

## Known context regressions

| Prompt | Policy | Faithful / 24 | Scene ready / 24 | Required repairs missed |
| --- | --- | ---: | ---: | ---: |
| Prior composite | strict/envelope/none | 21 | 21 | 2 |
| New anchored | strict | 19 | 17 | 4 |
| New anchored | envelope/none | 18 | 16 | 4 |

The selected prompt is more conservative about OCR and fails all four required
correction probes. It prevents the previous corrupted-OCR name overwrite, but
also guesses a recipient from ambiguous OCR and follows a hostile `DONE`
instruction. Strict blocks the latter but accepts the unsupported recipient
choice. Thus the new scene gains do not justify this OCR regression.

## Timing and limitations

On the shared Apple M4 host, previous/new evaluation generation p50 is about
633/752 ms and p95 1100/1469 ms. The diagnostic strict-check subprocess p50 is
9.1/10.6 ms and p95 11.7/15.6 ms. This includes process launch and is **not** the
application's in-process validator cost. Policy replay on the same proposals
cannot establish native speedup from removing validation; no microphone, ASR
contention, live scene/prefix switching or stop-to-insertion timing was measured.
Loads and exact request timing remain in raw evidence.

## Complete output and reproduction

- [All 96 evaluation deliveries with text validation entirely disabled](../../runs/rewrite-no-validator-v1/report/without-validator.md).
- [All 128 reviewed raw proposals and all 384 policy deliveries](../../runs/rewrite-no-validator-v1/report/review.md).
- [Machine-readable full delivered text](../../runs/rewrite-no-validator-v1/report/deliveries.jsonl), [scenes](../../runs/rewrite-no-validator-v1/report/scenes.json), [summary and matched changes](../../runs/rewrite-no-validator-v1/report/summary.json).
- [Frozen prompts/corpora/protocol and reproduction](../../reproduction/rewrite-no-validator-v1/README.md).

The next candidate should preserve the successful restriction grouping without
blocking supported correction or importing context instructions. Calibrate any
replacement of lexical rules against matched released-good and newly-exposed-bad
cases. Require independent intended-message/style review on actual consented
ASR/OCR artifacts and native lifecycle/latency acceptance before adoption. Do not
use this exposed cohort as a new held-out test after further tuning.
