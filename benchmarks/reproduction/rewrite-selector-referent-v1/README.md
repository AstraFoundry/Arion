# Public referent-description selector experiment

This development study tests a concise public description of the source/context referent before each bounded choice. It preserves the same1.88B Qwen3.5-2B Q4, nativeb11734 Metal99/CPU4/4096 context/greedy42/nonthinking/cachetrue, candidate producer and457 source tuples. No4B or olderQwen inference is added. A limited official100-entry catalog observation is retained, not an exhaustive global-newest proof.

The335-file freeze precedes inference. The owned adapter injects the declared pipeline and freeze directory into the unchanged frozen screen executor; its internal executor label remains`selector-screen-v1`. Every row records the actual shared-control or new prompt file/hash and complete messages. Ten pre-existing example source tuples/decisions remain unchanged; new public example descriptions are authored from those examples, not new test answers. Some old examples overlap exposed development cases; this is not held-out evidence.

Four arms use256 tokens: baseline, public description, balanced original/replacement sentences, balanced plus description. All457 actual Go cleanup results are reused within the four arms, yielding1,828 source outcomes and1,424 explicit no-hypothesis skips. Each arm makes101 actual requests. The404 responses all finish normally with no protocol error. There are zero new final bodies, ASR or OCR generations. Natural120 sources remain within the no-hypothesis scope, not new natural rewrite inference.

| Arm | Choice conforms /101 | Harmful choice cases | Missed repair cases | Required items repaired /47 | Description review passes /101 | Joint passes /101 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| baseline |92|3|6|41|not generated|92|
| referent |92|4|5|42|78|78|
| balanced |94|4|3|44|not generated|94|
| referent-balanced |93|5|3|44|80|80|

All404 actual raw responses were personally read. All202 explanatory cases and212 candidate-description fields were read with full source/context in six explicit review ranges. All17 unique failed source/output tuples and19 changed within-run source/context/choice tuples were reviewed separately. The202 plain controls match the old actual requests after reverting only the token limit256-to128; raw choices and edited sources match202/202. Old controls are not counted again.

## Interpretation

The public descriptions do not improve choice correctness or harmful replacement. Several positive repairs improve, but others are lost and wrong-object substitutions appear. Both new arms fail the choice gate independently of how the descriptions are scored; no formatter/fresh expansion or production adoption follows.

Of212 description fields,44 fail manual source/context review across44 of202 explanatory cases. Twenty-seven of those cases have a correct choice. Failure categories include unsupported same-person identity, false claims that cash-to-cache changes only case/spacing, a wrong source role, or omitting the relevant candidate comparison. Accurate concise semantic paraphrases are accepted; literal overlap is not the rule. Incorrect includes unsupported/incomplete descriptions, not only factually false sentences. These are assistant judgments, not independent blind ratings. Per-field texts, decisions and detailed notes remain separate and source-bound.

All202 explanatory cases actually serialize the description before choice. This observes output order; it does not establish hidden reasoning or truthful self-report. Descriptions never enter the corrected source or formatter instructions. Existing candidate application checks only protocol, identities, permitted choices and exactUTF8 spans. It does not reject semantically wrong descriptions or choices, and it provides no source fallback. The HTML retains them with independent choice and description badges/filters.

Observed request medians are approximately244/599/601/929ms in arm order; medians of output tokens are7/21/7/21. These are serial selector-only observations. P95 uses nearest rank. Startup, Go cleanup, formatting, native insertion and concurrentASR are excluded. Order/cache/load effects were not independently reversed; no causal latency/SLO/min16 claim follows.

## Reproduction

- `pnpm rewrite:selector-referent-v1:payload-test` checks input projection and bounded application, without model inference.
- `pnpm rewrite:selector-referent-v1:test` validates complete source/review coverage and independent description evidence.
- `pnpm rewrite:selector-referent-v1:verify` verifies archive identities without weights.
- `pnpm rewrite:selector-referent-v1:report --out <new-directory>` replays the complete report without inference.
- `pnpm rewrite:selector-referent-v1:run --out <new-private-directory> --cleanup-worker <pinned-executable>` runs the frozen experiment.

Weights, runtime environments and the executable remain private. Reusable raw observations, source preparations/skips, public descriptions, manual annotations, failed choices and original authoring sources are archived. Full-chain/fresh/currentASR+Vision/native lifecycle/offline/concurrent/independent/actual16GB requirements remain unchanged. Cross-script coverage is still incomplete.

The next untested hypothesis is to distinguish the full sentence's task/object from the candidate's contextual role explicitly. Current explanations often interpret the dictionary meaning of a surface typo, compare the wrong pair, or supply a plausible description without constraining the decision. No success is claimed for that next hypothesis.
