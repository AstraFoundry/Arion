# Installed-worker rewrite rejection diagnostic

This is an authored development diagnostic using the installed Qwen3 1.7B
INT4 worker, not a replay of private historical sessions. Support logs retain
verdicts and durations but omit transcripts and generated candidates, so the
exact markers behind earlier `markup` events cannot be recovered.

`inputs.json` freezes four simple utterances across empty, history and screen
contexts. `followup-inputs.json` adds six request, enumeration and preservation
cases across empty and history contexts after the first diagnostic passed.
The screen and history strings are authored text; no screen capture, microphone,
foreign field read or target insertion occurs. Cases are development probes,
not held-out quality evidence. Results live in
[`benchmarks/runs/rewrite-rejection-v1`](../../runs/rewrite-rejection-v1).

## Observed failures

All 12 simple cases passed. Of the 12 follow-up cases, eight passed, two reached
their output budgets and two lost negation. A request to inspect logs and explain
slow rewriting elicited an invented diagnostic answer beginning with possible
causes instead of edited dictation. It consumed all 128 allowed output tokens
without context and all 278 with history. History enlarged the output allowance
because `Engine.budget` counts the complete variable user turn, including context,
rather than transcript tokens alone. The history request took about 18.4 seconds
in this diagnostic without the application's timeout.

A two-requirement enumeration was reduced to a sentence about checking logs,
dropping the requirements and their negation in both context conditions. The
production validator returned `negation_dropped`; the later content checks do
not run once an earlier check rejects a candidate.

No current-worker `markup` or `length` failure was reproduced in these 24 cases.
The prompt opens `<dictation>` without a closing tag, but this observation alone
does not prove the cause of historical marker emissions. Older context-memory
experiments contain outputs copying context plus `<dictation>` tags under
different experimental prompts; they are not evidence of the exact candidates
in the installed app's support logs. Likewise, the historical `length` verdict
does not reveal whether that output expanded excessively or deleted too much.

The runner checks all model files against `models/catalog.json`, verifies the
pinned runtime archives, and records both upstream and installed dylib hashes
(packaging/signing can change bytes). Manifests retain worker/source/input/output
identities, dirty checkout state, platform and power observations. The packaged
worker's embedded `live.json` and `context.json` were byte-matched to the source.
Its build metadata reports revision `4356914796f484064ffcd562846b7dcd91a1350b`
with local modifications. Model hashes do not establish source/export parity.

The archived runner was parameterized after the original runs; generation and
validation operations are unchanged. Build the production validator from the
repository root and separately provision the explicitly named local model,
installed worker/runtime and pinned archives. No weights are downloaded:

```sh
mkdir -p .workspace/rewrite-rejection-v1
cd apps/desktop
CGO_ENABLED=0 go build -o ../../.workspace/rewrite-rejection-v1/validate ./cmd/arion-context-probe
cd ../..
python3 benchmarks/reproduction/rewrite-rejection-v1/run.py \
  --inputs benchmarks/reproduction/rewrite-rejection-v1/followup-inputs.json \
  --out .workspace/rewrite-rejection-repeat
```

Use a new output directory. Four CPU threads, greedy decoding, a 4096-token
total bound and 384-token maximum output match the production worker policy.
The diagnostic waits for complete worker results without the application's
10-second timeout, exposing output that the app would cancel earlier.
Requests alternate context templates, so repeated full-prefix computation is
included; background apps and ASR contention are not controlled. Timings do not
establish normal session latency, native acceptance or release performance.
