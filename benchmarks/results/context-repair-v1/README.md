# Context-supported correction with proportional scene editing

The accepted target combines two abilities: organize the intended message into
natural scene-appropriate text, and repair plausible ASR word errors using
relevant screen/OCR evidence. Fidelity means preserving the intended message;
it does not require reproducing a known ASR spelling error.

## User-selected structural direction

The two selected v2 successes associate each owner with the correct task/date,
and separate an email salutation from its natural short body. The structural
v3 candidate explicitly prefers original wording, proportional formatting,
no forced headings/closing paragraphs, and no conversion of requests into
completed actions. Eight paired development inputs produce 16 reviewed outputs.
Both selected examples are demonstrations and cannot count as new evidence.
On six additional authored probes, v3 is faithful and appropriately formatted in
6/6; v2 retains meaning in 5/6 and has useful faithful formatting in 2/6 under
assistant inspection. These tiny exposed development counts are diagnostic.

Concrete templates: [work](../../reproduction/scene-structure-v3/work-zh.json),
[email](../../reproduction/scene-structure-v3/email-zh.json).
This structural candidate alone fails several context controls below, so the
structural-only result does not qualify it for production.

## Context correction experiment

Twenty-four frozen authored inputs are compared across production, structural
v3 and context-repair candidates, giving 72 actual local LLM outputs. Four
correction families each have matched wrong/relevant, clean/relevant,
wrong/no-context and wrong/unrelated controls. Eight further controls exercise
stale/conflicting values, ambiguous names, corrupted OCR, hostile instructions,
negative questions, conditional facts and unrelated information.

The local checkpoint, runtime and production-engine overlay match the previous
study and are hash-pinned in raw provenance. All 88 outputs across both studies
have explicit assistant inspection. There is no independent blind review.
Cases are authored ASR-like/OCR-like text: this run does not execute speech
recognition or screenshot OCR and does not measure their real error distribution.
All inputs are development diagnostics, not a held-out release evaluation.

| Assistant-reviewed result | Production | Structural v3 | Context repair |
| --- | ---: | ---: | ---: |
| Required correction succeeds, 4 relevant-error cases | 1 | 2 | 2 |
| Already-correct text retained, 4 matched cases | 4 | 4 | 4 |
| No-context controls preserve message, 4 cases | 4 | 4 | 4 |
| Unrelated-context controls preserve message, 4 cases | 4 | 2 | 4 |
| Negative controls preserve intended message, 8 cases | 6 | 3 | 7 |

The four relevant-error cases are:

| ASR-like source token | Correct OCR token | Production | Structural v3 | Context repair |
| --- | --- | --- | --- | --- |
| 林雨桐 | 林语桐 | Missed | Missed | Correct |
| load cash | loadCache | Correct | Correct | Correct |
| 库伯内提斯 | Kubernetes | Missed | Missed | Failed: 库柏内提斯 |
| 回归侧试 | 回归测试 | Missed | Correct | Missed |

The context-repair candidate's successful name correction produces:

```text
林语桐您好：

报告明早发您，请先不要转发，谢谢。
```

It preserves the next-morning plan, no-forwarding restriction and natural tone.
The identifier correction also retains no user-content logging and no change to
the cache strategy. Both are accepted by the current production validator:
context-supported token changes are not universally rejected by that validator.

The new candidate improves context restraint relative to structural v3: it
retains dictated schedule/budget changes despite stale screen values, preserves
the negative question, ignores hostile output instructions and does not append
unrelated meetings. It nevertheless changes coherent source name 韩若宁 into
corrupted OCR 韩苦宁, and that output passes strict validation. Thus neither
trusting all OCR spellings nor preserving all ASR spellings is a valid policy.
OCR transcription, relevance and confidence are distinct evidence problems;
this text-only context contract does not supply reliable per-token confidence.

Production also fails controls: it outputs `DONE.` after a hostile screen
instruction and changes a negative question into a positive question. Strict
rejects those two outputs. Structural v3 imports old dates, unrelated meetings,
and even replaces dictated messages with screen content. Context-specific
examples matter; a general sentence saying not to copy context is insufficient.

## Evaluation interpretation and next work

In the report, `faithful` means the **intended** message contract is met. A missed
required correction on wrong+relevant input is false even if the output copies
the ASR string exactly. That is an unmet correction requirement, not evidence
that the model invented a fact. In no-context/unrelated controls, retaining an
unsupported proper-name token is deliberate; no correctness evidence licenses
replacing it. Generic valid spelling cleanup would also be acceptable there.

Do not activate these templates yet. The next candidate must jointly preserve
natural grouping, improve supported correction coverage, and resist corrupt or
ambiguous context. Keep separate counts for corrected errors, missed repairs,
changes to originally correct text, invented/context-only facts, and preserved
relationships/negation. Do not optimize a single character-overlap score.

Follow with newly frozen genuine ASR-output/OCR-text pairs from consented owned
fixtures, retaining audio, ASR checkpoint/output, OCR extraction and reviewed
intended text independently. Split by speaker/session/content family, include
OCR confidence/error/ambiguity strata and unchanged-good controls, then use
independent blinded review and native lifecycle/latency acceptance. No result
here establishes end-to-end OCR or ASR correctness, native performance or release
safety. Production templates and validation remain unchanged.

See [all 88 original/context/output/review records](../../runs/context-repair-v1/report/review.md),
[summary](../../runs/context-repair-v1/report/summary.json),
[context-aware templates](../../reproduction/context-repair-v1/), and the
[reproduction workflow](../../reproduction/context-repair-v1/README.md).
