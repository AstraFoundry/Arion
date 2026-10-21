# 021: Support macOS on Apple silicon only

## Status

Accepted by explicit product decision. This supersedes the macOS Intel support commitments in ADR 020 and earlier platform plans.

## Context

The selected macOS recognition and rewriting stack targets Apple silicon. Intel macOS adds a separate runtime, packaging and acceptance matrix. The product will concentrate macOS development and verification on Apple silicon.

## Decision

- Supported desktop targets are macOS ARM64 (Apple silicon, macOS 14 or later) and Windows x64. Intel macOS is unsupported for future releases.
- Ordinary CI checks macOS ARM64 and Windows x64. Alpha releases continue to distribute only macOS ARM64; stable and other prereleases require the two supported targets. No Intel macOS runner or release artifact is required.
- The supported build/dev entry point rejects Intel macOS. The website offers only the supported download targets and must not advertise Intel macOS compatibility.
- Preserve historical benchmark evidence and existing published artifacts. Retained x86_64 test tooling, model provenance, universal dependency slices and legacy update-contract parsing do not establish continued product support. Removing that historical code or data is a separate maintenance change.
- Existing Intel installations receive no promise of future updates or compatibility. Restoring support requires a new product decision, a maintained runtime and native-host acceptance.

## Consequences

macOS work has one supported architecture and one native CI runner. Windows x64 remains in scope with its existing preview and acceptance limits. ARM64 tests under Rosetta may still be run explicitly as historical diagnostics; they are not a release gate or an Intel support claim. This decision does not alter privacy, signing, offline operation or model-download requirements.

## Alternatives considered

Continuing Intel packaging or keeping its CI job as a required compatibility target would retain the separate support obligation that this decision removes.

## References

- [Native Qwen3-ASR workers](020-native-qwen-asr.md).
- [Product scope](../project/README.md).
- [Release policy](../operations/deployment.md#releases-and-updates).
