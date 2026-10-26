# Original Rime LLM reranker reproduction

This study executes the unchanged [FuJiK0/rime-llm-reranker](https://github.com/FuJiK0/rime-llm-reranker) Lua filter at commit `700ad82ec5c5a4983652183238b77f38cfdaf85a`. Imported source, supplied configuration and MIT notice are retained in `upstream/`; its original non-English comments are preserved evidence. Arion production code and installed input-method settings are not changed.

## Boundary and treatments

The native candidate producer is the current production `ArionKeyboardEngine` with its pinned Rime runtime, an empty synthetic encrypted profile per fixture and cancelled `xuetang`/`heishan` warmups. `prepare-native.mjs` reuses the existing reference-free quality driver and encryption build helper. It links `ranking.m` because the current keyboard engine references that class, but the new profile contains no learned ranking history.

`filter-driver.lua` adapts recorded native snapshots to the original Rime filter API. It calls the original Lua function, tokenizer/model server and real LuaSocket HTTP path. It records exact request/response bodies and uses a compiled `CLOCK_MONOTONIC` clock around the filter and HTTP request. This establishes filter/engine-boundary reproduction, not installed Squirrel/librime-lua integration, OS dispatch, IMK rendering, selection acceptance or concurrent ASR.

The supplied upstream settings are preserved: five candidates, twenty output tokens, ten-byte maximum preedit, sixty-byte context tail, 0.5-second LuaSocket timeout and the supplied system prompt. Endpoint changes only to the isolated loopback server. The original implementation promotes only the first parsed candidate; it does not apply the rest of the generated ranking. It caches an index by context/preedit without including candidate identities. The upstream debug timer uses process CPU time. These properties are measured as shipped, not silently repaired.

The README's linked `oliverphilcox/oMLX` returned HTTP 404. The available `jundot/omlx` runtime is pinned at `6aba0dd3f28c690230a572a0c96cb43ee4f3d738` (0.7.2), a disclosed runtime substitution rather than a claim to recover the author's exact environment. The model is `mlx-community/Qwen3.5-0.8B-bf16` at `3067585164dbcc505eec73d554349de6a27571a4`, served under the name the Lua configuration expects. Model bytes and resolved packages remain private; their hashes and versions accompany runs.

## Commands

Use pinned pnpm 10.33.0. Provisioning requires native Apple silicon macOS, Python 3.12+, `uv`, Xcode and the already provisioned repository keyboard SDK/data. It downloads public model/runtime sources into a new private directory, never into the installed Rime directory.

```sh
EVAL_PYTHON=/absolute/python3.12 pnpm ime:rerank provision --workspace .workspace/rime-rerank-new
pnpm ime:rerank collect --workspace .workspace/rime-rerank-new --out .workspace/rime-candidates-new
```

Start oMLX with the private base path and loopback address after model acquisition completes:

```sh
HF_HUB_OFFLINE=1 TRANSFORMERS_OFFLINE=1 .workspace/rime-rerank-new/venv/bin/omlx serve \
  --model-dir "$PWD/.workspace/rime-rerank-new/models" \
  --base-path "$PWD/.workspace/rime-rerank-new/server-default" \
  --host 127.0.0.1 --port 18782 --headless
```

Retain separate unrelated warmup requests/responses and model-load timing before running measured resident-server requests. Keep model thinking configuration explicit in the server's `model_settings.json`; different thinking settings are separate treatments.

```sh
pnpm ime:rerank warmup --workspace .workspace/rime-rerank-new \
  --candidates .workspace/rime-candidates-new --out .workspace/rime-warmup-new
pnpm ime:rerank run --workspace .workspace/rime-rerank-new \
  --candidates .workspace/rime-candidates-new --out .workspace/rime-final-new
pnpm ime:rerank run --workspace .workspace/rime-rerank-new \
  --candidates .workspace/rime-candidates-new --mode typing --out .workspace/rime-typing-new
pnpm ime:rerank report --run .workspace/rime-final-new --out .workspace/rime-report-new.json
pnpm ime:rerank test
pnpm ime:rerank verify
```

The measured server used the runtime's default automatic thinking policy and emitted zero reasoning tokens; no per-request thinking flag was added to the original Lua request. The exact source revisions, resolved package metadata and `constraints.txt` are retained. `pnpm ime:rerank audit --workspace PATH --run PATH --out NEWFILE` replays actual responses through the original parser, and `controls --workspace PATH --candidates PATH --out NEW_DIRECTORY` runs the separate live cache-order probes.

`make ime-rerank ARGS='test'` dispatches the same entry point. Outputs must be new; incomplete attempts cannot produce reports. The final mode includes all 100 fixtures, an immediate identical-snapshot repeat and eight changed-candidate-order cache controls. Typing mode replays selected native key snapshots without feeding future input. Immediate repeats measure Lua cache, not inference speed. The inference engine's own prefix cache remains active and is distinct from the Lua result cache.

`archive --workspace PATH` is the one-time publisher for this study's fixed archive destination; it refuses existing archive directories. `freeze` creates the checksum inventory only after the evidence and report prose are complete and refuses to replace an existing inventory. Ordinary replications should use new workspace output paths and `report`, preserving the committed archive unchanged.

## Metrics and evidence limits

The scorer reports exact acceptable top-1, MRR@9, first-five/first-page oracle recall, rescues and breakages against the identical producer list. Oracle misses, skipped inputs, HTTP timeouts and parse failures remain in the denominator. Latency separates filter wall time, actual HTTP wall time, process CPU time and immediate Lua cache hits; percentiles use type-7 interpolation. Per-group results and every changed output remain available.

Fixtures are single-assistant-authored synthetic exploratory data. No statistical superiority, held-out generalization, human typing acceptance or release readiness follows from these measurements. The original filter accepts mixed-span candidate lists; this study does not retrofit phonetic/length policy. Cold warmups, model-load failures and development attempts are retained separately from scored resident-server runs. See the [result report](../../results/rime-llm-reranker-v1/README.md) for actual observations.
