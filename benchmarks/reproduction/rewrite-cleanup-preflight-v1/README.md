# Production cleanup preflight

The actual domain.CleanSpeech implementation was executed on all36 synthetic
regression sources, and its deletion spans were replayed with ApplySpeechSpans.
Only lp-stress-correction changes: the superseded request is removed before the
remaining load cash request. The repeated phrase and numeric self-correction
fixtures remain unchanged. Original inputs, outputs and frozen source bytes are
retained. No ASR/LLM inference or product modification occurs in this preflight.

The archived Engine.Run source explicitly passes cleanup.Text to the model.
The direct text diagnostics bypassed that step, so their raw-source loop is not
by itself a production reproduction. All other35 inputs remain unchanged by the
current producer. Future pipeline runs must retain raw ASR, cleanup/spans and
actual model input separately, rather than silently replacing the corpus.

`pnpm rewrite:cleanup-preflight:verify` checks snapshots and span replay.
`:report --out <new-directory>` executes the archived source in an isolated Go
module and requires byte-identical observations. Go1.25.7 or a compatible local
toolchain is required; absent capability fails, never substitutes mock output.
