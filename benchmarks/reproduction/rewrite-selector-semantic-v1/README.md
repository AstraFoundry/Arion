# Semantic selector policy experiment

This development experiment changes selector instructions and examples only. Both arms use the same tone-insensitive hypotheses and latest work-v4/other-v2 formatter. Inputs, evaluator, protocol and241 source hashes were frozen before inference. The actual1.88B Qwen3.5-2B Q4 model runs on M4/32GB with Metal, four CPU threads and no thinking. No4B or older Qwen inference was added.

The609 actual responses comprise120 conditional choices and489 final bodies: candidate401 historical sources,24 prior tone controls and32 new role controls; baseline32 role controls. Every response was personally reviewed. No semantic output rejection or source fallback is used. Sources, effective scenario, complete actual requests, responses and judgments remain visible.

| Paired development observation | Prior selector | New selector |
| --- | ---: | ---: |
| Harmful substitutions /32 role controls |6|2|
| Faithful role bodies /32 |26|30|
| Useful role bodies /32 |14|16|
| Relevant-context role repairs /4 |0|0|
| Repairs /8 prior typo opportunities |3|1|
| Useful bodies /24 prior controls |16|15|
| Useful historical bodies /401 |344|338|

The new policy is rejected. Three historical question/term repairs improve, while nine previously corrected person names remain unresolved. Historical fidelity is400 confirmed/one source uncertainty; preserving an unresolved name is not successful correction. Natural120 outputs remain119 confirmed/one uncertainty/75 useful. English same-letter `read cash` → `readCash` is allowed; unsupported `readCache` is not.

## OCR failure analysis

Related pronunciation is candidate availability, not evidence of the same referent. Deployment configuration and nursery cultivation sound alike but describe different things. With an instruction in the screen, the old selector replaces five wrong-domain terms; the new selector still replaces two. Wrong-domain context without instruction causes one wrong replacement in the old selector and none in the new selector. These eight fixed-source groups are development ablations, not32 independent utterances or a general causal estimate.

Hypotheses are identical between arms. Across context conditions they need not be identical: exact source wording in the instruction suppresses selector calls for the import/writing groups. Those comparisons mix producer availability and selection; do not attribute their outcome to instruction resistance alone. The nursery related context also produces a candidate on the later forbidden `配置` span, rather than on cultivation. Review spans and grammatical role, not just candidate text.

The new global preference to keep coherent original words suppresses valid entity spelling repairs. The next policy should preserve supported entity normalization and separately explain referent/action/quoted-word boundaries through unrelated contrasting examples. More keep decisions alone are not a quality improvement. New wording must pass positive repairs and negative controls together.

## Metadata limitation

The frozen v2 executor hardcodes the old selector template label/hash. All101 candidate selector rows therefore retain incorrect diagnostic metadata, even though their complete actual requests match the frozen new policy and examples. `selector-template-provenance.json` binds each raw row and request to the effective prompt file. Raw evidence is unchanged. The HTML shows effective identity with an explicit caveat. This is development diagnostic evidence, not clean release provenance.

Future execution must use the versioned `gguf_cleanup_chain_v3.py` and explicit `pipeline.template_identity(arm, stage, scene)` ownership. Its shared identity function rejects a prompt absent from the pre-inference freeze and records the actual message hash. The v2 executor and executed experiment files remain immutable. The v3 identity contract has unit coverage but has not been used for new inference in this round.

Initial derived recovery bookkeeping carried stale flags for12 changed historical bodies. The initial review/report and an explicit amendment are retained; fidelity/usefulness judgments and all raw inference bytes are unchanged.

## Reproduce

- `pnpm rewrite:selector-semantic-v1:verify` verifies archived byte identities.
- `pnpm rewrite:selector-semantic-v1:test` checks actual evidence corruption and future identity ownership.
- `pnpm rewrite:selector-semantic-v1:report --out .workspace/semantic-report-replay` replays the complete report without weights.
- `pnpm rewrite:selector-semantic-v1:run --out <new-private-directory> --cleanup-worker <pinned-worker>` reproduces the original frozen v2 run, including its documented metadata defect. Do not reuse it for new policies.

Native integration, current ASR plus Vision, concurrent timing, cancellation/focus/revision/once-only delivery, independent review and actual Apple Silicon/16GB acceptance remain open. Production prompts and validators are unchanged.
