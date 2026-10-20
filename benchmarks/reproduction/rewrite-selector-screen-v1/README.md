# Four-arm selector-only development screen

The full source scope remains457 tuples:401 history,24 prior controls and32 semantic-role controls. Actual Go cleanup runs457 times, reused explicitly within four arms. All1828 source-treatment outcomes are retained. Each arm makes101 actual model requests;356 sources have no hypotheses and do not call the model. The404 actual choices were personally read, with every unique failed/changed full-source/context/edited-source tuple reviewed separately.

No formatter or final-body generation occurs in this experiment. Passing only permits a new complete457-source chain, followed by the unchanged held-out/native/independent/minimum16GB acceptance requirements. No arm passes, so no further full-body or fresh evaluation is run for these candidates.

## Treatments and observed results

All arms retain the same actual1.88B Qwen3.5-2B Q4 model, tone-neutral producer, source data, examples/answers, schema, greedy seed, Metal/CPU4 and4096 context. No4B or older Qwen inference is added.

| Arm | Treatment | Conforming /101 | Harmful choice cases | Missed required-repair cases | Required candidate items repaired /47 |
| --- | --- | ---: | ---: | ---: | ---: |
| baseline | Semantic-v2 rule/data |92|3|6|41|
| after | Add after-only candidate sentence and repeated target spelling |88|12|1|46|
| balanced | Original/replacement sentence pairs without repeated target spelling |94|4|3|44|
| role | Balanced data plus explicit entity/ordinary-word/negative-target rules |83|17|1|46|

The balanced representation repairs the old `工时通记`, historical `会议记要` and customer-contact `大理` cases without losing the nine entity repairs previously recovered. It avoids most after-only regressions, but still changes three wrong-domain instruction cases and the writing source into `近口主题`; three related-context positive cases remain unresolved. Improved total conformance does not meet the no-harm gate.

The added rules retain positive repairs but introduce more wrong-object, location/action and excluded-target substitutions. Longer or more explicit rules did not improve this candidate; no universal claim that prompt optimization cannot work follows.

## Evidence and interpretation

The275-file source freeze, including evaluator labels, precedes inference. The evaluator derives from previously reviewed actual records and the human-approved same-letter policy. Its101 source/reference identities are explicit. Nine previously diagnosed errors have corrected targets;22 source cases allow either raw spelling or same-letter case/space normalization. That permission predates this experiment; historical labels and decisions are not rewritten. Required repair-item counts exclude optional canonicalization.

No evaluator labels, reference transcripts or scores enter model requests. Actual input reconstruction verifies each arm against its frozen payload builder. Removing only the declared data treatment (and replacing the declared role-system treatment) reconstructs the current actual baseline request. Shared producer output is identical across arms.

The regenerated baseline and after controls each match their corresponding old complete-chain101 actual requests, raw choices and edited-source candidates exactly:202 explicit paired records. This establishes parity for those observed controls, not universal equivalence between stage-only and full-chain execution. Stage-only cache/order and the lack of a formatter prohibit claims about final-body quality or stop-to-insertion latency.

All404 responses stop normally and satisfy the choice protocol. Semantic conformance remains separate from source preservation: a keep can preserve source fidelity while missing a required normalization under the frozen development rubric. The living report uses explicit choice conformance for intermediate results and never displays these edited-source candidates as delivered final bodies.

The457 sources and four treatment repetitions are not1828 independent subjects. This is exposed development evidence. Candidate production still misses some cross-script terminology; zero-hypothesis sources receive no spelling-recovery credit. Full context is retained; no semantic output rejection or source fallback is introduced. Production prompts/validators are unchanged.

## Next direction

Balanced data is a useful development comparison, not an adopted candidate. A compact meaning/referent justification before the decision is a possible next hypothesis; another eligible small selector model is also relevant. Neither has been tested here. Any passing selector must still execute the complete current chain and all original acceptance requirements.

## Reproduction

- `pnpm rewrite:selector-screen-v1:verify`
- `pnpm rewrite:selector-screen-v1:test`
- `pnpm rewrite:selector-screen-v1:report --out .workspace/screen-report-replay`
- `pnpm rewrite:selector-screen-v1:run --out <new-private-directory> --cleanup-worker <pinned-worker>`

The actual Go preparation, source-level skips, every model response, complete messages/roles, original worker commands/logs and manual review are archived. Weights and executable runtimes remain private.
