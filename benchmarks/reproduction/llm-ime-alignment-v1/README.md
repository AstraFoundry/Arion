# Reproduce the general LLM keyboard-ranking alignment

This study executes Arion's experimental Go selector over frozen real Rime candidates. It does not install/enable the input method or authorize sending personal keyboard context to a remote service. Start with the [complete alignment and evaluation plan](../../../docs/research/llm-ime-alignment.md).

## Model-free replay

```sh
pnpm ime:align verify
pnpm ime:align test
```

The verifier checks the [archive manifest](archive.json), original fixture identities, exact producer-derived requests, historical entry-source hashes and four exact report regenerations. Tests reject invented IDs/nonfinite scores, missing/duplicate observations, failed runs, label-contaminated/changed requests and stale scheduled application. Python standard library is sufficient.

## Actual local inference

Use the catalog-provisioned Qwen3.5-2B Q4_K_M and the runtime closure pinned in `scripts/llm-runtime.json`. No command downloads artifacts implicitly. Model SHA-256 must match `aaf42c8b7c3cab2bf3d69c355048d4a0ee9973d48f16c731c0520ee914699223`. Runtime executable identity in the measured run is recorded in `inference/status.json`; provisioning also requires all native dependencies and their pinned archive identity.

```sh
mkdir -p .workspace/ime-align
CGO_ENABLED=0 go build -o .workspace/ime-align/driver ./apps/desktop/cmd/arion-ime-rank-eval
CGO_ENABLED=0 go build -o .workspace/ime-align/worker ./apps/desktop/cmd/arion-llm
pnpm ime:align prepare --work .workspace/ime-align/run-new
pnpm ime:align run --work .workspace/ime-align/run-new \
  --driver .workspace/ime-align/driver \
  --worker .workspace/ime-align/worker \
  --runtime /absolute/provisioned/llama-b11734 \
  --model /absolute/provisioned/qwen3.5-2b-q4-k-m-metal \
  --model-sha256 aaf42c8b7c3cab2bf3d69c355048d4a0ee9973d48f16c731c0520ee914699223
pnpm ime:align report --work .workspace/ime-align/run-new --out .workspace/ime-align/report-new.json
```

Output directories/files must be new. `--ungated` explicitly enables no-context inference for a diagnostic repeat of the initial policy. Current strict parsing/bounds are retained; the archived initial pilot predates those parser hardening changes and its matching historical entry source is preserved. Both historical and current source enforce producer identity/coverage; no malformed outputs occurred in the measured initial pilot.

The 202 reference-free requests contain two unrelated warmups and 100 paired context/no-context fixtures shuffled with seed 42. The Go driver uses strict unknown-field rejection and never opens corpus annotations. Only Python reporting reads targets. Source fixtures remain authored, exploratory and reused from the earlier study; current policy must not be tuned on a later held-out validation set.

`source/` is the current code overlay against the recorded repository base and pinned Go modules. `historical-source/{pilot,gated}/` retains the four entry files independently hashed at each quality-run start; all hashes match their recorded originals. Historical executable bit-for-bit reproduction is not claimed. The shared worker/runtime behavior and source identity are retained; weights/binaries remain outside Git.

## Explicit generic API evaluation

The same adapter accepts the existing Republic provider transport. An API key is read only from `ARION_RANK_API_KEY`, not stored credentials. Select the endpoint/model explicitly and use synthetic requests. Local compatible servers need the `chat` format because the OpenAI provider otherwise defaults to Responses.

```sh
pnpm ime:align prepare --work .workspace/ime-align/api-new
pnpm ime:align run --work .workspace/ime-align/api-new \
  --driver .workspace/ime-align/driver \
  --provider openai --base http://127.0.0.1:8000/v1 \
  --format chat --model-id YOUR_PROVISIONED_LOCAL_MODEL
```

Set `ARION_RANK_API_KEY` in the invoking environment without putting it in logs or command arguments. This example is not evidence of MLX/API inference: only the native local worker ran real models in the archived study. Synthetic HTTP tests cover Responses/chat formats and truncation. Actual subscription authorization and product cloud-context consent are separate gates; the evaluation constructor deliberately does not sign in or read stored subscription credentials.

## Real-time scheduler/model replay

The exact reference-free schedule is [archived](../../runs/llm-ime-alignment-v1/schedule.json). It supplies 100 ms key offsets and 1,000 ms between fixtures. The opt-in Go test uses actual monotonic timers, the production application scheduler and local worker; it writes exclusively to a new observation file.

```sh
ARION_RANK_SCHEDULE=/absolute/checkout/benchmarks/runs/llm-ime-alignment-v1/schedule.json \
ARION_RANK_OBSERVATIONS=/absolute/new-schedule-observations.jsonl \
ARION_RANK_WORKER=/absolute/checkout/.workspace/ime-align/worker \
ARION_RANK_RUNTIME=/absolute/provisioned/llama-b11734 \
ARION_RANK_MODEL=/absolute/provisioned/qwen3.5-2b-q4-k-m-metal \
ARION_RANK_MODEL_SHA256=aaf42c8b7c3cab2bf3d69c355048d4a0ee9973d48f16c731c0520ee914699223 \
CGO_ENABLED=0 go test ./apps/desktop/infrastructure/llm/ranking -run TestNativeKeyboardSchedule -count=1 -v
```

This test validates model/Go scheduling, not OS event dispatch or rendered candidate UI. Native IMK integration and ASR/rewrite contention still require the gates in the alignment plan. Both archived schedule runs include cancelled inference in call/time totals.

## Archive maintenance

The actual pilot was staged in `.workspace/llm-ime-alignment-v1/`; `archive.py` records original run bytes with explicit source mappings and refuses to replace an existing run archive. The initial entry sources were restored byte-for-byte and matched to the independently recorded hashes before archiving. `freeze.py` creates the immutable manifest only after reports and documentation are complete. Never edit frozen observations to change a result; create a new versioned experiment.
