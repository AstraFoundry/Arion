# Local development

## Foundation checks

Use Node from `.nvmrc` (24) and exact pnpm 10.33.0 from `package.json`. The workspace has dependency-free evaluation tooling and a committed lockfile; install with `make install` / `pnpm install --frozen-lockfile`. npm/yarn installation is rejected. Native dependencies remain Go modules.

```sh
make install
make check
make desktop-check
make e2e
```

`pnpm check` validates local Markdown links, runtime configuration JSON, active-doc placeholders, Go layer imports and JavaScript syntax. `make core-check` runs portable core tests/vet/format validation in standalone module mode. These do not claim OS adapter or inference coverage.

## Native desktop

MyGo v0.4.6 requires Go 1.27.1. The workspace/tool directive uses Go automatic toolchain selection when available; a first native build may download it. Use an installed matching Go toolchain with `GOTOOLCHAIN=local` for explicitly offline development after dependencies are provisioned.

macOS needs Xcode Command Line Tools and SDK. Windows builds must run on Windows with the native toolchain requirements declared by pinned MyGo. Check [MyGo research](../research/mygo.md) and inspect actual compiler errors rather than adding cross-platform stubs.

```sh
make desktop-build
make desktop-dev
```

These commands use the pinned `go tool mygo` from `apps/desktop`. Windows installer creation uses NSIS: the pinned CLI provisions it on Windows when missing; cross-packaging on macOS without NSIS emits the executable and explicitly skips the installer. `desktop-dev` launches the minimal window and is an interactive action; it does not attach a microphone, register an IME or download model weights. Close the window normally when done. `desktop-build` produces native packaging for the current host under ignored `apps/desktop/build/`.

For direct compile validation:

```sh
cd apps/desktop
go test ./...
go vet ./...
```

Run `go mod tidy` in the module whose imports changed and retain `go.sum`. Keep core independent of desktop: run `GOWORK=off go test ./...` from `packages/core`. Do not use `go test ./...` at repository root: this is a multi-module workspace.

## Configuration

`apps/desktop/mygo.json` is public development package metadata, not secrets. The identifier `local.arion.desktop` requires replacement with a controlled production identifier before release. Runtime settings/storage and sync config are not implemented. Credentials must use OS key stores; never add fake environment settings that appear operational.

## Troubleshooting

- Missing toolchain/network: provision matching Go and module cache explicitly; do not lower MyGo's required version.
- Model unavailable: current shell has no model adapter. Follow ASR-01/RNK-01 rather than interpreting it as an inference error.
- Platform build failure: capture compiler/SDK errors and compare pinned upstream source before choosing an adapter.
- Unsupported field: future adapters must return a typed unsupported result; clipboard is not a silent bypass.

## Automation and experiments

`make help` lists fixed flows. `make check` includes docs, portable core and evaluation harness checks. `make e2e` uses a real desktop session and briefly opens/closes a production fixture window. `make ablation-plan`, `make stream-plan` and `make datasets-check` prepare experiments; the last only fetches pinned HF metadata. Follow [evaluation commands](../../tools/evaluation/README.md) for live corpus validation, runs, reports and product scenarios. Windows users without GNU Make use the pnpm scripts directly. `.workspace/` contains private artifacts; automated benchmark CI uploads only aggregates.
