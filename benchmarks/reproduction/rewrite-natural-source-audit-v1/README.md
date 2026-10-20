# Natural-source lineage audit

The existing120 natural read-speech test sources (60zh/60en) were audited against
original audio bytes, native acoustic extraction and the derived historical raw
view. This performs no model inference and does not reuse historical LLM outputs
as new results. [Prepared source view](../../corpora/rewrite-natural-asr-v1/README.md).

`pnpm rewrite:natural-sources:verify` checks lineage offline; the corresponding
`:test` checks coverage and rejects changed ASR text or speaker identities.
`audit.py --out <new-directory>` constructs another explicitly new source audit
and retrieves upstream source-page snapshots. Existing directories cannot be
overwritten. A new snapshot is not byte-identical historical replay; verify the
retained snapshots and original hashes instead.

This is preparation, not rewrite quality, real-time speech timing or release
acceptance. Reference labels remain evaluator-side. Existing split/speaker/session
provenance and historical exposure are preserved.
