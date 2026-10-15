# Local development

## Foundation checks

Use Node from `.nvmrc` (24) and exact pnpm 10.33.0 from `package.json`. The workspace has dependency-free evaluation tooling and a committed lockfile; install with `make install` / `pnpm install --frozen-lockfile`. npm/yarn installation is rejected. Native dependencies remain Go modules.

```sh
make install
make check
make desktop-check
make e2e
```

`pnpm check` validates local Markdown links, runtime configuration JSON, active-doc placeholders, Go layer imports and JavaScript syntax. `make core-check` runs portable core tests/vet/format validation in standalone module mode. `make sync-check` does the same for `services/sync`. `pnpm test` includes both. `make sync-integration` needs `ARION_SYNC_TEST_DATABASE_URL` and is not part of the default run. `make i18n-check` / `pnpm i18n:check` validates translation keys, templates, formatting and direct hardcoded copy. These do not claim OS adapter or inference coverage.

`make site-check` / `pnpm site:check` checks the marketing site in `apps/site`: dictionary completeness, TypeScript, ESLint, Vitest and the Next.js static export into `apps/site/out`. `pnpm --filter @arion/site dev` serves it on `127.0.0.1` for local work; see the [site README](../../apps/site/README.md) and [site conventions](../frontend/site.md).

## Native desktop

MyGo v0.4.6 requires Go 1.27.1. The workspace/tool directive uses Go automatic toolchain selection when available; a first native build may download it. Use an installed matching Go toolchain with `GOTOOLCHAIN=local` for explicitly offline development after dependencies are provisioned.

macOS packages require macOS 14 or later because the bundled polishing ONNX Runtime declares a 14.0 minimum. Building needs Xcode Command Line Tools and SDK. Windows builds must run on Windows with the native toolchain requirements declared by pinned MyGo. Check [MyGo research](../research/mygo.md) and inspect actual compiler errors rather than adding cross-platform stubs.

```sh
make desktop-build
make desktop-dev
```

These commands call `scripts/desktop-command.mjs`, the same build/dev entry point used by CI. It runs the pinned `go tool mygo` from `apps/desktop` with `CGO_ENABLED=1`; on macOS it sets the configured minimum OS in the deployment target and cgo compiler/linker flags and uses a relocatable rpath to `Contents/Resources/lib`. Both call `scripts/stage-desktop-resources.mjs` (`make desktop-libs` runs that preparation alone). It generates the English/Simplified Chinese macOS permission localizations from the [copy catalogs](../frontend/i18n.md) and copies the pinned sherpa-onnx and ONNX Runtime libraries from the Go module cache into `apps/desktop/resources/<platform>/`; they are build artifacts and stay ignored. Windows installer creation uses NSIS; cross-packaging cgo for Windows additionally needs a MinGW-w64 toolchain (`CC=x86_64-w64-mingw32-gcc`).

`desktop-dev` launches the menu bar app. On first run it shows onboarding, asks for microphone and Accessibility access, and downloads the automatically selected model bundle (about 270 MB) into the per-user model store; after that dictation works offline. `desktop-build` produces `Arion.app` and a DMG under ignored `apps/desktop/build/`. macOS ties Accessibility consent to the app's code signature: an ad hoc rebuild needs the permission granted again.

### Data locations

| Data | macOS | Windows |
| --- | --- | --- |
| Preferences | `~/Library/Application Support/Arion/preferences.json` | `%AppData%\Arion\preferences.json` |
| Verified models | `~/Library/Application Support/Arion/models/<artifact>/` | `%AppData%\Arion\models\` |
| Usage counts | `~/Library/Application Support/Arion/usage.json` | `%AppData%\Arion\usage.json` |
| Vocabulary (AES-GCM) | `~/Library/Application Support/Arion/vocabulary.bin`, key in the login Keychain | same folder, key protected by DPAPI |
| Hosted-sync refresh token | login Keychain, service `Arion`, account `account-refresh-token` | Credential Manager target `Arion/account-refresh-token` |

`ARION_DATA_DIR` points the app at another profile directory; `ARION_OFFLINE=1` refuses model downloads; `ARION_E2E_CONTROL=<socket>` enables the owned-scenario control mode (fixture audio instead of the microphone). These exist for isolated E2E profiles, not for users.

The refresh-token item is created only after the user signs in ([account sign-in](../project/account-signin.md)). It is a different Keychain account and a different Credential Manager target from the vocabulary key. Signing out deletes the refresh token. The vocabulary file and its key stay.

For direct compile validation:

```sh
cd apps/desktop
go test ./...
go vet ./...
```

Run `go mod tidy` in the module whose imports changed and retain `go.sum`. Keep core independent of desktop: run `GOWORK=off go test ./...` from `packages/core`. Do not use `go test ./...` at repository root: this is a multi-module workspace.

## Configuration

`apps/desktop/mygo.json` is public development package metadata, not secrets. The controlled identifier is `si.astrafoundry.arion`; the IMK bundle uses `si.astrafoundry.arion.inputmethod`. Developer ID/notarization and Windows publisher signing still require credentials. Local preferences and encrypted vocabulary are implemented; desktop sync enrollment/outbox remain open. Credentials must use OS key stores; never add fake environment settings that appear operational.

## Troubleshooting

- Missing toolchain/network: provision matching Go and module cache explicitly; do not lower MyGo's required version.
- Model unavailable: open the menu bar panel or Settings › Model; a failed download or checksum shows its reason and a retry. `go run ./cmd/arion-models import` installs an already-downloaded archive into a store without network.
- Platform build failure: capture compiler/SDK errors and compare pinned upstream source before choosing an adapter.
- Unsupported field: future adapters must return a typed unsupported result; clipboard is not a silent bypass.

## Automation and experiments

`make help` lists fixed flows. `make check` includes docs, portable core and evaluation harness checks. `make e2e` uses a real desktop session and briefly opens/closes a production fixture window. `make ablation-plan`, `make stream-plan` and `make datasets-check` prepare experiments; the last only fetches pinned HF metadata. Follow [evaluation commands](../../tools/evaluation/README.md) for live corpus validation, runs, reports and product scenarios. Windows users without GNU Make use the pnpm scripts directly. `.workspace/` contains private artifacts; automated benchmark CI uploads only aggregates.
