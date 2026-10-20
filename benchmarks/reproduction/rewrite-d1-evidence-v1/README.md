# Native d1 contextual spelling decisions

The official d1-omni-600M Q8 text decision component has380,732,161 parameters/179 tensors; the full multimodal family has587,161,089. This research is within the user's2B cap. Only text weights are provisioned. Custom-license research is authorized; production restrictions remain unresolved.

These are actual `/v1/systemone` classification responses with typed answers, probabilities and zero generated tokens. They are not synthesized chat completions, generated final prose or native target insertions. Six exposed development sources are reused across six packaging treatments and two actual backends:72 responses, not72 independent sources. Every response was personally reviewed with its complete source/OCR/hypotheses, all6 unique failed tuples and all24 changed successive pairs. This is not independent blind acceptance.

| Packaging | Correct per backend /6 | Harmful per backend | Missed per backend | Actual instruction clipping |
| --- | ---: | ---: | ---: | --- |
| Original22 messages in native state |3|3|0|No|
| Decoded structured examples, local options |2|2|2|Yes|
| Decoded structured examples, sentence options |3|1|2|Yes|
| No demonstrations, local options |2|2|2|Yes|
| No demonstrations, sentence options |6|0|0|Yes|
| Full policy in state, short question |3|2|1|No|

All six CPU/Metal choices agree in each treatment, while probability values differ by up to0.04234. This is not numeric or calibrated-probability equivalence. No confidence threshold filters the decisions. Request latency is an observed serial diagnostic, not a stop-to-insertion or minimum16GB SLO.

## Root cause and correction

Pinned b11734/f3fb2fcdb `server-decision.cpp::fill_task_d1omni` separately limits each two-choice question to96 tokens and each option description to45 tokens. These trained per-piece limits apply even when the whole request fits the4096 context/batch. Moving the complete old system rules into `question.instructions` therefore clipped the later meaning-conflict rules. HTTP200 and a complete JSON request did not prove those rules reached the model.

The zero-generation budget investigation renders the pinned embedded `tokenizer.chat_template.systemone`, applies exactly the pinned native piece policy and calls the actual native tokenizer/detokenizer. It retains36 full requests, every piece/token ID and all native calls. All60 earlier inference input counts and50 observed native effective prompts match. Slot-launch diagnostics show the preceding completed prompt, leaving the last prompt unobserved per worker; it is not falsely counted as observed.24 of30 earlier distinct request templates clip instructions, affecting48 responses. Full current source/OCR state remains intact.

The correction puts the complete old trusted rules into `state.policy`, keeps complete current inputs and uses a short question with complete original/replaced sentence options. All6 requests have no piece truncation, maximum514 tokens; all12 later actual usage counts match and10 actual logged prompts match. The inherited `structured` label in the original budget source freeze is preserved with an explicit metadata correction before policy inference. No request/output/score or weight is rewritten.

Fixing this serialization defect does not solve contextual correctness: corrected policy still misses loadCache and changes software configuration into nursery cultivation and explicit cash into cache. Both backends are3/6. The compact-sentence12/12 observation has clipped rules and only six exposed sources; it does not qualify that requested full prompt or a release candidate. No full-source expansion or production adoption follows.

## Evidence and limits

The four frozen source experiments are [transcript](../rewrite-selector-d1-v1/README.md), [decoded layouts](../rewrite-selector-d1-layout-v1/README.md), [compact layouts](../rewrite-selector-d1-compact-v1/README.md) and [corrected policy](../rewrite-selector-d1-policy-v1/README.md). Native19/19 Metal and0/19 CPU layer placement, complete original HTTP bytes, responses, model identity, commands, properties, logs and terminal statuses are retained. Original startup path failure occurred before server launch; its original script/freeze/log and corrected startup are preserved. Weights, runtimes and Python environments remain private. LocalJinja3.1.2 replay plus native token/prompt matching is not official Transformers/reference-logit parity.

Spelling text mechanically derived from a choice is audit illustration only; the probe did not execute a formatter or deliver that text. There is no new Go cleanup, ASR, Vision, final-body or native insertion evidence, semantic output rejection or source fallback. Model, task architecture and serialization differ from Qwen; this is not an isolated same-request model comparison. Work/email structuring, full-source/fresh/cross-script quality, native lifecycle/offline/concurrent ASR, independent review and actual Apple Silicon16GB acceptance remain required. The user has no16GB device currently and authorized32GB development-host validation only.

## Replay

`pnpm rewrite:d1-evidence-v1:report --out <new-directory>`; `:test`; `:verify`. Offline reports and verifiers read only archived files and do not require weights or original scratch data. Frozen native executors retain their original private paths and explicit archive path mappings; native reruns require provisioned pinned weights/runtime and a new output directory. They never substitute archived outputs for new inference. The policy executor additionally requires its sibling zero-generation budget qualification. Preserve the original native component's trained96/45 limits; do not increase them to mask packaging errors.
