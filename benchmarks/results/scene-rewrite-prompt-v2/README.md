# Focused scene prompts v2: complete generation review

Decision: retain the new prompts as experimental candidates; do not activate them
in production. They improve organization but fail the frozen requirement of no
additional material errors on matched fresh cases.

## Change and experiment

The previous scene study mixed languages and scenes in its demonstrations and
omitted an explicit original-language rule. This revision uses a common fidelity
contract, an explicit output language, exactly one scene policy and, in the
selected candidate, two demonstrations from that same scene and language.
Work dictation may become bullets without spoken numbering; developer dictation
becomes a request with its stated constraints. Neither may answer the request,
invent a diagnosis, claim completion or copy reference context.

[Work prompt](../../reproduction/scene-rewrite-prompt-v2/templates/examples-work-zh.json)
and [developer prompt](../../reproduction/scene-rewrite-prompt-v2/templates/examples-prompt-zh.json)
are concrete versioned templates, not recovered Chatterfly server instructions.
Production `live.json`, `context.json`, routing and validators remain unchanged.

Two candidates generated 32 development outputs from 16 previously exposed cases.
The same-scene/language demonstration variant was selected after inspecting all
outputs: 16/16 faithful and 15/16 useful for their scenes, versus 13/16 and 4/16
for the zero-demonstration variant. Selection and review hashes were frozen
before evaluation. Evaluation then generated 112 outputs: production versus the
selected candidate on 24 old test regressions and 32 newly authored cases from
16 new content families. Four fresh cases include authored hostile screen text.
All 144 source/output pairs have explicit assistant review records.

The model is the provisioned Qwen3 1.7B INT4 CPU model, with four threads and
greedy decoding on Apple M4. Weight/tokenizer/runtime/binary/build/template hashes,
loads, exact requests, cleanup values, outputs and failures are archived. The
experimental engine is the isolated v1 production-engine overlay; scenes and
languages come from fixture metadata. No microphone, private screen capture,
cloud inference or native editor was used.

## Results

These are single-assistant judgments, not independent human semantic ratings.
“Scene ready” requires faithful content, a valid output envelope and useful
scene organization. All counts score raw proposals, including rejected ones;
they are not delivered-product success rates. Timeout/no output counts as failed
fidelity coverage; it does not imply a generated false fact.

| Cohort | Prompt | Faithful | Scene ready | Strict accepted | Faithful rejected by strict | Unfaithful accepted by strict |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| Fresh, 32 | Production | 28 | 9 | 30 | 0 | 2 |
| Fresh, 32 | Selected v2 | 26 | 22 | 24 | 3 | 1 |
| Old test, 24 | Production | 24 | 12 | 22 | 2 | 0 |
| Old test, 24 | Selected v2 | 22 | 20 | 20 | 3 | 1 |

Fresh paired errors: v2 introduces five errors on cases with faithful production
outputs, fixes three production failures (one was a timeout), and both variants
fail the English schedule case differently. This fails the frozen adoption gate.
There is one invalid envelope per arm on fresh data, and one additional truncated
v2 output on old test data. Existing strict validation rejects six faithful v2
outputs across the 56 evaluation cases, yet accepts two semantically bad ones.

| Fresh timing | Production | Selected v2 |
| --- | ---: | ---: |
| Request p50 | 1.009 s | 1.048 s |
| Request p95 | 1.665 s | 2.300 s |
| Without-context p95, 28 cases | 1.304 s | 1.495 s |
| With-context p95, 4 cases | 11.234 s | 2.300 s |

These are request wall times on a shared host, not native stop-to-final latency.
The production context arm contains a real timeout/cancellation; its approximately
11.2-second wall time includes cancellation completion after the 10-second
deadline. Model loads are recorded separately. The run groups candidates by
scene/language and does not establish live switching or prefix-rebuild latency.
Small strata and non-interleaved arms do not establish a reliable speed advantage.

## Output review findings

- **Useful work organization:** v2 separates completed export, pending security
  review, release-after-approval and the owner into bullets without changing the
  facts (`test-export-zh`). Similar improvements retain roles and conditions.
- **False rejection:** “three in the afternoon” becomes “3:00 PM” in
  `test-reschedule-en`, rejected as `new_number`. “不要改变” becomes “不改变”
  in `fresh-batch-zh`, rejected as `negation_dropped` despite retaining negation.
- **Lost restriction:** v2 `fresh-cause-zh` retains investigation-only/no-fix but
  drops the separate “do not guess configuration values” requirement.
- **Invented actions/repetition:** v2 `test-retry-zh` adds database/no-code
  restrictions and truncates; `fresh-units-zh` repeats the full request.
- **Reference contamination:** v2 `fresh-archive-en` appends a next-Monday meeting
  from unrelated context; `fresh-headers-zh` echoes screen/dictation markup.
- **Meaning errors passing strict:** v2 turns “Friday's review” into “I might
  miss Friday. Please review.” and adds a good-traffic attendance condition in
  another case. The source's `Friday.s` typo is a disclosed fixture limitation;
  production repaired the event relationship correctly.
- **Old prompt also fails:** production `fresh-schedule-en` changes all three
  weekday relations and still passes strict. In `fresh-literal-zh`, it deletes
  the quoted no-restart instruction even though recorded cleanup retained it.

The [complete review](../../runs/scene-rewrite-prompt-v2/report/review.md) contains
every source, output and judgment; [CSV](../../runs/scene-rewrite-prompt-v2/report/cases.csv)
and [summary](../../runs/scene-rewrite-prompt-v2/report/summary.json) are replayable.
[Raw evidence](../../runs/scene-rewrite-prompt-v2/) retains errors and selection.

## Acceptance and next experiment

Keep the explicit language and focused scene/demonstration design as the next
candidate's starting point. Before another test, develop better request-versus-
completion, full-constraint retention and untrusted-context isolation on new
**development** families. Do not tune against this now-exposed fresh test cohort.
Repeated full requests and protocol echoes must fail presentation acceptance.
Investigate larger local model capacity only as a separately measured candidate;
this study does not prove a prompt-only or model-size-only explanation.

Do not relax the current lexical gate wholesale: it causes false rejection but
also fails to detect important relational errors. A replacement requires
separately calibrated fidelity evidence, equivalent-quantity normalization and
retained lifecycle/envelope checks. Independent blinded review on natural
voice/frozen-ASR data and actual offline native cancellation/focus/once-only
commit and load-stratified latency remain required before release. See the
[alignment acceptance plan](../../../docs/research/scene-rewrite-alignment.md).
