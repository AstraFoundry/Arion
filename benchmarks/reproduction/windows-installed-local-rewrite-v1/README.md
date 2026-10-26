# Installed Windows local polishing repeat

The existing `TestNativeWindowsLifecycle` passed against the actual installed runtime in `%LOCALAPPDATA%/Programs/Arion/llm` and the model downloaded by the application into `%APPDATA%/Arion/models/qwen3.5-2b-q4-k-m-metal`. This repeat covers the final worker, including cancellation-aware native reload. No inference was rerun while archiving these results.

The original test log reports a pass in 60.81 seconds. All six completion requests returned complete text within the explicit thirty-second Windows CPU request budget. Cancellation was acknowledged in 20.399 ms; the next explicit request completed with `ReusedTokens: 0`. The context-backed spelling request selected `Chatterfly`, then formatted the full message while retaining the service prohibition. Read [the original observations](observations.jsonl) for exact outputs, requests and per-call timings.

This is installed-artifact and lifecycle validation. The installed application was available and GUI checks were ongoing; background application/download activity was not controlled or independently recorded. The weights were present before this probe. The recorded times are individual observations, not an isolated latency benchmark, a comparison against the earlier run, or hardware qualification. There was no microphone capture, GUI assertion or editor insertion in this test. Separate ongoing GUI checks do not become passing evidence through this archive.

## Artifact identity

`manifest.json` records actual SHA-256 and byte size for the installed worker, baseline executable, accelerated executable and installed model. All three executable identities match the current canonical staged resources. The model matches the catalog's 1,280,835,840-byte artifact and SHA-256 `aaf42c8b7c3cab2bf3d69c355048d4a0ee9973d48f16c731c0520ee914699223`. No executable identity was reconstructed for this repeat.

`source/` holds exact source snapshots captured during archival, with their original repository paths and identities recorded in the manifest. It includes the cancellation-aware engine, worker, fixture test, pipeline/templates, process ownership, CPU checks and canonical build pins/configuration. Installed binaries match the canonical staged files; the snapshots' capture time remains explicitly distinguished from the execution.

`observations.jsonl`, `run.log` and `cases.json` preserve original bytes in this new directory. The manifest records their scratch-source mappings and all archived-file hashes. Earlier successful, failed and superseded attempts remain untouched in [the first Windows validation archive](../windows-local-rewrite-v1/README.md). Weights, executable environments and application profile data remain outside Git.

## Verification and reproduction

```powershell
node benchmarks/reproduction/windows-installed-local-rewrite-v1/verify.mjs
```

This verifies archived bytes, model/catalog identity, completion/cancellation results, the original passing log and installed/canonical executable hash agreement. It reads no host-profile files and reruns no inference.

To repeat actual installed-runtime inference, use the existing [native test procedure](../windows-local-rewrite-v1/README.md#reproduction) with `ARION_GGUF_RUNTIME` pointing at the installed `llm` directory and `ARION_GGUF_MODEL` at the installed verified model directory. Always choose a new output filename. [ADR 025](../../../docs/decisions/025-windows-local-polishing.md) defines the Windows CPU runtime and deadline.
