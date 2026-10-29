# Native prompt protocol smoke

This native Apple Silicon check exercises the production Swift worker with already provisioned Qwen3-ASR 1.7B affine 8-bit files. It verifies the ready capability, one contextual request, a following empty-context request in the same resident worker, and explicit rejection of an oversized context. It does not establish hotword accuracy, causal streaming, latency or Windows inference acceptance.

The exact worker and SwiftPM package sources are retained here and in the completed run. The worker tokenizes the complete prompt with actual preprocessed audio tokens and reserves its existing 128-token generation budget against the loaded checkpoint's `maxPositionEmbeddings`. The tested checkpoint declares 65,536 positions. The oversized request contains exactly `"x " * 70000`; its original JSONL bytes are retained compressed. The existing terminal envelope reports `type: "error"`; it does not expose a distinct context-limit reason.

The existing `asr-window-v1/ocr-work-term.wav` fixture is developer-authored synthetic development audio using the macOS Tingting voice. Its original corpus entry, license/consent, sample identity and SHA-256 are retained in the run metadata. Reference text is archival evaluator metadata and was never sent to inference. The only nonempty successful recognition context was `MLX Swift`.

## Retained observations

- [Complete byte-capture replay](../../../runs/asr-hotwords-v1/runtime-smoke-v1/provenance.json): executable and all eight model identities, original compressed requests, original stdout/stderr, source snapshots and checksums. Every request passed; process exit was zero.
- [Initial attempts](../../../runs/asr-hotwords-v1/runtime-smoke-initial-attempts/attempts.json): available Xcode build output and exit zero; failed sandbox launch with worker exit 255 and no ready/output; successful elevated startup; original smoke parsed responses. The initial process output was not retained as raw bytes, and the build output is a verbatim transcription of captured tool output. These limitations are explicit in its metadata.

The initial sandbox launch failed without emitting an OS rejection reason. The same executable and model loaded successfully with sandbox escalation; a GPU sandbox cause is inferred. The complete replay also used sandbox escalation. No weights were downloaded.

## Replay and verify

From the repository root, verification requires no model or native runtime:

```sh
python3 benchmarks/reproduction/asr-hotwords-v1/runtime-smoke/verify.py
```

Rebuild the native worker with the repository's pinned Xcode/SwiftPM command, provision the exact catalog model files, and supply a new output directory:

```sh
python3 benchmarks/reproduction/asr-hotwords-v1/runtime-smoke/replay.py \
  --worker /absolute/path/to/arion-asr \
  --model /absolute/path/to/qwen3-asr-1.7b-mlx-8bit \
  --out /absolute/path/to/new-output-directory
```

The directory must not exist. The replay checks every local model digest, captures raw requests/responses and retains failed state if a check fails. Native inference requires macOS Apple Silicon and Metal access. Original absolute paths in the archived `original-check.py` describe its initial invocation; `replay.py` resolves fixtures relative to the checkout.
