# Closed-loop recognition memory research

The [measured report](../../results/user-modify-asr-v3/README.md) covers actual earlier audio recognition, authored user corrections, production extraction, encrypted writes, reload and different later audio. [Research](../../../docs/research/closed-loop-memory.md) separates paper mechanisms and unimplemented acoustic training from tested lexical approximations.

## Sources and provisioning

- `build-fixtures.py` authors entity/voice/session/template-disjoint splits, preserves original AIFF/WAV bytes and freezes all feedback, labels and selection rules before inference. Natural-read regression retains source IDs/hashes. `superseded-build-shared-templates.py` is pre-inference design evidence, not the supported builder.
- `run.py` recognizes actual teaching audio, sends edited field snapshots to `apps/desktop/cmd/arion-memory-loop`, then uses only the reloaded frozen memory for future recognition. It never opens `evaluation.json`. `common.py` reuses immutable v2 artifact checks and exposes an independent split validator. `retrieval.py` contains both predeclared candidates and the prior comparator.
- `report.mjs` verifies complete paired calls, frozen inputs, actual producing output, prior feedback, initial empty state and the persistence chain, then reuses repository speech metrics and adds entity/routing outcomes.
- `select.py` uses development metrics only and retains the prior policy when candidates do not outrank it. The selected policy's source hashes must match before test teaching starts.

Build the two small Go research drivers with `pnpm memory:loop-build` or `make memory-loop-build`; `MEMORY_BIN_DIR` optionally sets the output directory. The command builds real production extraction/vocabulary and prompt/validation imports with `CGO_ENABLED=0`. Exact source snapshots are retained in `production-source/`; replay checks their hashes and does not build or run them. New inference builds the current checkout and records its identities.

Separately provision the pinned Qwen3-ASR MLX model and the v1/v2 private Python environment, including `pypinyin==0.55.0` and `editdistance==0.8.1`. No command downloads a model or routes content online. Weights, runtime, compiled drivers and **vault keys** stay in `.workspace/`. The corpus and runs contain only authored/reusable data, not personal recordings.

## New inference

```sh
pnpm memory:loop --fixtures <frozen-fixture-root> --split dev \
  --model <pinned-mlx-model-directory> --driver <arion-memory-loop> \
  --out <new-development-directory>
pnpm memory:loop-report <frozen-fixture-root> <development-directory> <new-summary.json>
```

Place the development report at `<development-directory>/summary.json`, then run:

```sh
.workspace/user-modify-asr-venv/bin/python \
  benchmarks/reproduction/user-modify-asr-v3/select.py \
  --fixtures <frozen-fixture-root> --dev-run <development-directory> \
  --out <new-selection.json>
```

Only then run `pnpm memory:loop` with `--split test --selection <selection.json>` and a different output directory. Never tune to its results. Validate split separation with `common.verify_separation` before inference. Corpus freeze checks cover both splits; the runner's inference inputs exclude future references.

Make uses `CORPUS`, `MEMORY_SPLIT`, `MODELS`, `MEMORY_DRIVER`, `OUT`, optional `MEMORY_SELECTION` and `EVAL_PYTHON`. It maps to `memory-loop-bench`. `EVAL_PYTHON` defaults to the separately provisioned `.workspace/user-modify-asr-venv/bin/python`.

## Model-free replay

```sh
pnpm memory:loop-report benchmarks/corpora/user-modify-asr-v3 \
  benchmarks/runs/user-modify-asr-v3/test-run \
  .workspace/closed-loop-test-replayed.json
```

Make uses `memory-loop-report CORPUS=... RESULTS=... OUT=...`. Output files/directories must be new. Exact observation replay needs no weights, runtime, private key or original machine paths. Fresh acquisition has random production entry IDs and encryption nonces and is not promised to reproduce bytes. Historical hashes identify the archived observations; missing private artifacts cannot be substituted by mocks.

`pnpm memory:loop-verify` / `make memory-loop-verify` verify both new study manifests, including frozen source snapshots. Explicit manifest paths may be supplied as pnpm positional arguments. The historical migration's archive command remains separate.

Native field reading, actual human edit behavior, actual OCR, Keychain provisioning and microphone/insertion timing are outside this driver. The separate context experiment measures text-model behavior; combining those two studies is not a measured native product pipeline.
