# Ordinary-word context and pronunciation-memory probe

This is actual Qwen3-1.7B **text inference** on reported/authored ASR strings and reconstructed screen text. It does not run ASR or capture OCR. The [research](../../../docs/research/closed-loop-memory.md) explains how this complements the [audio/extraction/store/reload study](../../results/user-modify-asr-v3/README.md).

`build-fixtures.py` freezes different development/test semantic families, earlier user feedback, unrelated/injection/number/negation controls and a generic semantic prompt. The reported Codex scenario is disclosed development evidence. `run.py` starts with an empty encrypted vault, uses production extraction/storage, retrieves actual records via immutable v2 pinyin lookup and asks `arion-context-probe` to render the actual Go templates and validator decisions.

Conditions are current product application/terms, existing full-context sections, semantic context plus arrow examples, and semantic context plus retrieved pronunciation examples in one user turn. The latter contains **text-derived Mandarin pinyin**, not acoustic phoneme posteriors; Latin strings are not converted to English phonemes. It supplies no test reference. `select.py` chooses on development only; test runs product, sections and the frozen selected candidate. Raw invalid generations remain in `results.jsonl`; delivered results follow the existing validator's recorded decision. Scores distinguish raw and delivered outcomes.

## New inference

Build native research helpers with `pnpm memory:loop-build`. Separately provision both Python environments: the orchestration environment used by the ASR studies for pinyin, and the CPU ONNX GenAI environment for the existing diagnostic worker. The worker checks pinned model assets and native/reference tokenizer parity. Select the provisioned Qwen3-1.7B CPU model directory with the explicit four-thread derived config; retain its metadata and upstream asset identities. Weights/runtimes/keys stay private.

```sh
pnpm memory:context --fixtures <frozen-root> --split dev --out <new-run> \
  --model <provisioned-text-model> --model-metadata <pinned-asset-run.json> \
  --worker-python <onnx-genai-python> --memory-driver <arion-memory-loop> \
  --context-driver <arion-context-probe>
pnpm memory:context-report <frozen-root> <new-run> <new-run>/summary.json
```

Use `select.py --fixtures ... --dev-run ... --out <new-selection.json>` with the orchestration interpreter, then run once with `--split test --selection <selection.json>` and a new output path. Model metadata uses the retained diagnostic asset schema (`artifacts` with path/SHA-256/source), not an unverified fresh hash list. Its old machine paths need explicit provisioning or mapping for new inference; model-free scoring does not access them.

Make entry points are `memory-context-bench` and `memory-context-report`. Inference requires `CORPUS`, `MEMORY_SPLIT`, `MODELS`, `MODEL_METADATA`, `WORKER_PYTHON`, `MEMORY_DRIVER`, `CONTEXT_DRIVER` and `OUT`; `MEMORY_SELECTION` is required for final test, and `EVAL_PYTHON` selects the orchestrator. Reporting requires only `CORPUS`, `RESULTS` and a new `OUT` JSON file.

The Python diagnostic worker regenerates the full prompt prefix for every request with resident weights. Its compute includes repeated prefill; it does not measure the production native engine's persistent prefix cache or stop-to-insertion latency. The model is a different checkpoint/runtime from Qwen3-ASR-1.7B. There is no independent semantic review or actual user audio in this probe.

Use `pnpm memory:loop-verify` / `make memory-loop-verify` to verify the exact source/input/output bytes in both new research archives. Model-free report replay remains a separate command.
