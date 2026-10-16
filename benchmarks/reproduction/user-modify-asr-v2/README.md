# Recognition memory optimization, v2

The code implements a first-pass retrieval / contextual audio re-decoding experiment. It does not edit the transcript into the desired spelling. The [research](../../../docs/research/user-modify-asr-optimization.md) distinguishes paper methods from local adaptations; the [measured report](../../results/user-modify-asr-v2/README.md) supplies results and limitations.

## Files

- `policy.py`: phonetic span retrieval, expiry/reverse filtering, destination-only contexts and already-correct entity preservation. It imports v1's immutable selector and SHA helper for the legacy comparison.
- `build-fixtures.py`: copies exposed v1 audio into development, then creates different-voice test TTS and imports natural-read controls deterministically by ID. Exact WAV/AIFF bytes are retained; voice regeneration is not guaranteed byte-identical.
- `run.py`: verifies model, tokenizer, corpus and warmup hashes; runs actual Qwen3-ASR-1.7B audio inference; retains every call, context, route and timing. References and entity labels are excluded from the inference-loop input object.
- `report.mjs`: reuses repository CER/WER/MER and percentile metrics, verifies audio/observation hashes and actual-call provenance, then scores lexical errors, spacing-tolerant entity hits and wrong memory-entity insertion.
- `select.py`: applies the frozen development eligibility/ranking rule and records the selected policy before test execution.

The same MLX Audio commit, model revision and 8-bit weights as v1 are used. Local dependencies add `pypinyin==0.55.0` and `editdistance==0.8.1`; resolved versions and actual runtime/source artifact hashes accompany each run. Weights/environments are separately provisioned under `.workspace/`. The v1 source checkpoint parity limitation remains.

## New inference

From the repository root, with the pinned model and runtime already provisioned:

The common entry point is `pnpm memory:asr` (or `make memory-asr-bench`) with `CORPUS`, `MODELS` and `OUT`; add `MEMORY_SELECTION` for the frozen test policy. `EVAL_PYTHON` selects the experimental interpreter. pnpm accepts explicit runner arguments too. The direct commands below expose the same arguments.

```sh
HF_HUB_OFFLINE=1 .workspace/user-modify-asr-venv/bin/python \
  benchmarks/reproduction/user-modify-asr-v2/run.py \
  --fixtures benchmarks/corpora/user-modify-asr-v2/dev \
  --model .workspace/ablation-models/qwen3-asr-1.7b-mlx8 \
  --out .workspace/user-modify-v2-dev-new

node benchmarks/reproduction/user-modify-asr-v2/report.mjs \
  benchmarks/corpora/user-modify-asr-v2/dev \
  .workspace/user-modify-v2-dev-new .workspace/user-modify-v2-dev-new/summary.json
```

The archived `selection.json` freezes this study's development choice. A new study must apply the declared criteria to its own development result before reading test outputs. Use `--selection <frozen-selection.json>` with the test fixtures to run only empty context, legacy pairs, all eligible destinations and the chosen adaptive arm. New output directories and report files must not exist.

The baseline runs first because retrieval depends on it; other arms are shuffled. Adaptive rows explicitly reuse that measured baseline and invoke a second audio call only when selected terms exist. `calls.jsonl` contains actual calls; `results.jsonl` contains paired condition observations. These counts differ. Logical adaptive compute includes the first pass, retrieval and re-decode; single-pass baselines report their own context-call compute. No native stop-to-insertion or concurrency measurement is implied.

## Replay

```sh
node benchmarks/reproduction/user-modify-asr-v2/report.mjs \
  benchmarks/corpora/user-modify-asr-v2/test \
  benchmarks/runs/user-modify-asr-v2/test-run \
  .workspace/user-modify-v2-test-replay.json
```

Scoring needs no model. Case, punctuation and the declared whitespace normalization affect lexical scores. Entity hits tolerate Latin spacing/case and count only the known approved vocabulary; they are not independent semantic review. The test includes new synthetic recurrence and exposed natural general-speech regression, not independently collected natural recurrence.

The common model-free entry point is `pnpm memory:asr-report <fixtures> <run-directory> <new-output.json>` or `make memory-asr-report CORPUS=... RESULTS=... OUT=...`. The study has its own archive manifest and is separate from the historical migration replay command.
