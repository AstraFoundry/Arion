# Windows local polishing validation

This is a native runtime and lifecycle validation on the existing Windows development host. Four authored text cases exercise Chinese, English, explicit lists and context-backed spelling selection followed by formatting. No audio, real editor insertion, held-out model quality, broad hardware qualification or GPU acceleration is measured here.

A separate [installed-runtime repeat](../windows-installed-local-rewrite-v1/README.md) validates the final cancellation-aware worker and the application-downloaded model. It preserves a new set of original observations and actual installed artifact hashes without replacing any earlier data here; its uncontrolled background activity prevents treating it as an isolated latency benchmark.

Host: Windows 11 25H2 build 26200.9445, Ryzen 7 7840U, approximately 30.8 GB physical memory. Native compiler: w64devkit 2.10.1 / GCC 16.2.0; Go 1.27.1. The final complete probe ran while the other agents' native compilers were idle. Power mode was not independently recorded. Timings use Go's monotonic elapsed timer and include the complete worker request; load is recorded separately. These are individual observations, not percentiles or a device-wide performance guarantee.

## Actual results

The final `observations.jsonl` passes `TestNativeWindowsLifecycle`, including a new generation after cancellation with zero reused tokens. Load took about two seconds. The four cold-session requests completed within the explicit thirty-second Windows CPU deadline; the context-backed selector and formatter require about twenty seconds on this host. The hot repeat reuses the same active session and is not the normal cross-session latency.

| Probe | Evidence |
| --- | --- |
| Chinese chat | Complete original message and deadline preserved |
| English chat | Filler removed, restart prohibition preserved |
| Explicit list | Three numbered items, original order and constraints |
| Context-backed spelling | `chatter fly` selected as `Chatterfly`; formatter retained the service prohibition |
| Cancellation | Acknowledged in 20.585 ms; native child terminated |
| Next explicit request | Completed in 9.978 s including native reload; `ReusedTokens: 0` |
| Close | Worker no longer reports alive |

Read the original JSONL rows for exact text, tokens, timing and errors. The full test passed in 55.27 seconds. This is a text-stage observation, not speech accuracy or a semantic guarantee.

## Artifacts and failed attempts

- `cases.json` contains only authored request inputs, including the single independent screen-text clue. No answer labels enter inference.
- `manifest.json` records model/source archive SHA-256 and size, runtime feature gates, build settings, host, dirty Git revision, source identities, original scratch-path mapping and archive hashes. Weights and executable environments remain ignored under `.workspace/`.
- `observations.jsonl` retains final real output. `failed-upstream-10s.jsonl` and `failed-static-10s.jsonl` preserve the initial timeouts and cancellation-retry failure. The first upstream probes overlapped compilation and are not an idle-host latency comparison.
- `intermediate-tuned-30s.jsonl` completed generation but exposed 512 cached tokens after a generation change following cancellation. Native HTTP cancellation can race task enqueueing, so erasing the slot alone was insufficient. It is superseded by terminating the native child on Windows cancellation and loading the same pinned model only for the next explicit request. Cancelled text is never retried.
- The official Windows CPU archive failed clean-install DLL closure because it omits `vcruntime140.dll`. A source build with shared libraries and static MinGW compiler runtimes passed import closure but hung on exit and was rejected. The final runtime uses two fully static executables: baseline x64 and AVX2/FMA/F16C/BMI2, selected by CPUID and OSXSAVE/XGETBV. Windows uses four decoding threads, eight prefill threads on this host, 512-token batches/microbatches and one 4096-token slot. No GPU driver is required.
- `measured-source/*.txt` preserves the exact worker and engine source for the complete measured run. Canonical staging cleared the original worker path before binary hashing; its identity was reconstructed by rebuilding the unchanged source with the same `-trimpath` flags. The actually measured AVX2 executable remained available and was hashed directly. The unused baseline executable is not assigned an inference-measurement hash.

The final checkout additionally makes the native reload wait cancellation-aware, so cancelling again during an unready reload cannot hold the worker for its ninety-second load limit. `TestCancellationDuringNativeReloadIsBounded` starts a real helper executable which never opens its health socket and verifies cancellation in under three seconds; `TestFailedNativeReloadCannotContinueServing` requires fatal unavailability after a failed reload. These are lifecycle tests, not additional model inference. A failed reload makes the pipe worker exit after reporting failure.

## Reproduction

Run the offline archive check from a checkout:

```powershell
node benchmarks/reproduction/windows-local-rewrite-v1/verify.mjs
```

For actual inference on Windows, stage the canonical runtime and provision the exact catalog model outside Git. Use a new output path on each run; the test refuses to overwrite an existing result:

```powershell
node scripts/stage-llm-runtime.mjs windows-amd64
$env:ARION_GGUF_RUNTIME = (Resolve-Path apps/desktop/resources/windows-amd64/llm).Path
$env:ARION_GGUF_MODEL = '<verified directory containing Qwen3.5-2B-Q4_K_M.gguf>'
$env:ARION_GGUF_OBSERVATIONS = '<new absolute path ending in .jsonl>'
go test ./apps/desktop/infrastructure/llm/gguf -run '^TestNativeWindowsLifecycle$' -v -count=1 -timeout=5m
```

The local execution sandbox blocked AF_UNIX binding; the recorded actual-host inference used approved unsandboxed execution. Native packaging additionally checks both static executables' complete PE dependency closure and a baseline `--version` launch with only the Windows system directory on PATH. See [ADR 025](../../../docs/decisions/025-windows-local-polishing.md) and [local configuration](../../../docs/operations/local-rewrite.md).
