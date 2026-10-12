# Foundation validation

## Observed locally

- Repository links/configuration/layer imports/JavaScript syntax pass `pnpm check`.
- Portable core tests, Go vet and formatting pass `pnpm core:check`. Tests cover review-before-commit/terminal session transitions and candidate permutation/finite scores.
- MyGo native Go module compiles and passes vet on macOS arm64 with selected Go 1.27.1.
- `go tool mygo build` successfully produces `apps/desktop/build/darwin-arm64/Arion.app` and `Arion 0.0.0 arm64.dmg` with local ad hoc signing. This is not Developer ID signing or notarization.
- Direct Windows amd64 cross-compilation and `go tool mygo build -platform windows/amd64` succeed; the CLI emits `apps/desktop/build/windows-amd64/Arion.exe`. The local macOS host has no NSIS, so its cross-build explicitly skipped the Windows installer. Windows execution, installer registration and UI behavior remain unverified.
- Frozen pnpm 10.33.0 workspace installation, portable core checks and 21 evaluation-tool tests pass locally.
- Actual macOS native E2E passes: production window visibility, Content capture, resize and exactly one close callback. Content capture is not an OS/GPU screenshot. Windows amd64 native E2E test binary also cross-compiles successfully; Windows execution remains unverified.
- Factorial and streaming plans generate deterministically; missing corpus/driver requests fail explicitly. All seven pinned HF metadata revisions resolve through `pnpm datasets:check`. Dataset research records access limits and provenance; no audio/model acquisition or speech score is claimed.
- Complete Typeless REA Evidence is retained locally; committed ledgers preserve IDs, digests and partial coverage. Doubao evidence is ordinary shell static evidence, not REA decompilation.

## Not verified / not implemented

No Windows execution, microphone capture, ASR/model benchmark, correction/ranking inference, IMK/TSF host registration, crypto/storage or sync deployment was performed. Full product scenario drivers remain unimplemented. CI jobs are configured and have not been run remotely. No production model weights or distribution licenses were approved. See [handoff](../project/agent-handoff.md) and [evaluation](../research/evaluation.md).
