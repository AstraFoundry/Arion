# Experiment reproduction sources

These are byte-preserving snapshots of scripts, prompt definitions, baseline Go files, generated worker configurations and evaluators used in the archived experiments. Original report configurations are under `report-configs/`; current report configurations under `../configs/` resolve archived inputs.

The exact evaluator source before adding archive support is in `evaluation-source/`. Selected product/driver source is in `selected-source/`. SHA-256 identities and original locations are in the [archive map](../migration/archive.json). A snapshot may contain old relative imports or absolute machine paths; it is retained to explain an experiment, not presented as an automatically runnable entry point.

Use current commands documented in [evaluation tooling](../../tools/evaluation/README.md). For scoring existing observations, use `pnpm bench:archive:report --out <new-directory>`; no model/runtime is needed. New inference requires separately provisioned artifacts and a new output path. Do not edit these snapshots to retrofit a new environment or silently overwrite the original execution evidence.
