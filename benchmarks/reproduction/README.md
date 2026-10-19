# Experiment reproduction sources

These are byte-preserving snapshots of scripts, prompt definitions, baseline Go files, generated worker configurations and evaluators used in the archived experiments. Original report configurations are under `report-configs/`; current report configurations under `../configs/` resolve archived inputs.

The exact evaluator source before adding archive support is in `evaluation-source/`. Selected product/driver source is in `selected-source/`. SHA-256 identities and original locations are in the [archive map](../migration/archive.json). A snapshot may contain old relative imports or absolute machine paths; it is retained to explain an experiment, not presented as an automatically runnable entry point.

Use current commands documented in [evaluation tooling](../../tools/evaluation/README.md). For scoring existing observations, use `pnpm bench:archive:report --out <new-directory>`; no model/runtime is needed. New inference requires separately provisioned artifacts and a new output path. Do not edit these snapshots to retrofit a new environment or silently overwrite the original execution evidence.

The new [recognition-stage memory study](user-modify-asr-v1/README.md) has its own audio builder, inference runner and model-free report command. Its identities are in the [study archive](../results/user-modify-asr-v1/archive.json); it is not part of the historical migration report command.

The [recognition memory optimization](user-modify-asr-v2/README.md) adds first-pass retrieval, original-audio re-decoding, entity-aware scoring and development-only policy selection. Use `pnpm memory:asr` for new inference and `pnpm memory:asr-report` for model-free replay; its [separate archive](../results/user-modify-asr-v2/archive.json) preserves both splits and their actual calls.

[Closed-loop memory](user-modify-asr-v3/README.md) uses actual production extraction/encrypted persistence before later audio recognition. The [ordinary-word context probe](context-memory-v1/README.md) separately evaluates text rewriting and reconstructed screen semantics. Both have model-free report commands and frozen production-source snapshots; keys, runtime binaries and model weights are separately provisioned.

[Context ranking](imk-context-v1/README.md) replays seeded homophone streams through the production typing owner and candidate bar with context ranking off and on, and times keys and candidate-bar updates.

[Offline Pinyin engine baseline](imk-quality-v1/README.md) compiles the production native engine/encrypted synthetic-key backend, freezes 100 original reference-isolated fixtures, runs paired cold/warm repetitions and provides model-free replay with corruption/leakage/incomplete-run rejection. No installed input-source or Keychain operation occurs.
