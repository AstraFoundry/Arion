# Rejected and exploratory rewrite experiments

This immutable evidence snapshot retains completed matrix-1 development, validation, and native text-stage arms; the completed fresh prior contrast matrix; cancelled discovery; fixed-gap formatter failures; prompt-only V2/V3 exploratory trials; source snapshots; exact native completion requests/responses; logs; and hash-bound assistant reviews. V1 and V3 are rejected. Aggregate repair gains do not override newly harmful current-intent changes. V3 ambiguous-prior timeout remains visible as a failed, unassessable phase.

Reports are self-contained under `reports/`. Reviewer judgments came from a separate assistant task that read every recorded stage, not independent human or blind review. Automated substring/count checks are proxies. All inputs are authored development or synthetic TTS/Vision fixtures, not the user's original failure recording. Native input pairs begin from the last recorded ASR draft, not an actual LLM preview event. Separate real Coordinator preview/stop evidence and installed-worker evidence are already archived in this study's `native/` and `installed-v1/` directories and are not duplicated here.

`manifest.json` records the SHA-256 and original-to-archive mapping for every copied file. Original overlay files and execution metadata intentionally retain their original absolute paths; use `pathMappings` to rebase those references for replay. Weights, native runtime environments, packaged worker executables, caches and personal recordings are omitted; per-run identity records retain observed runtime/model hashes. Executable replay requires separately provisioned matching native artifacts. Matrix arms include exact source snapshots and effective configuration. The first exploratory fixed-layout trial predates the available attempt-2 source snapshot; its raw requests/responses are retained, but no claim of exact compiled first-trial source recovery is made. Review serialization scripts record the manually inspected judgments, not automated model quality tests.

Verify this archive from the repository root:

```sh
python3 benchmarks/reproduction/rewrite-stage-preservation-v1/report.py --verify benchmarks/reproduction/rewrite-stage-preservation-v1/evidence/rejected-v1
```

No report here promotes the rejected experiments or replaces the later chosen-source tests.
