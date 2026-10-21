# Local development

Current optional rewriting uses the [selected staged GGUF configuration](local-rewrite.md), with a verified profile weight and bundled private-socket runtime.

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

macOS packages require macOS 14 or later. Speech recognition still stages ONNX Runtime, whose Mach-O minimum is 14.0; rewriting stages llama.cpp as described in [local rewrite](local-rewrite.md). Building needs Xcode Command Line Tools and SDK. Windows builds must run on Windows with the native toolchain requirements declared by pinned MyGo. Check [MyGo research](../research/mygo.md) and inspect actual compiler errors rather than adding cross-platform stubs.

```sh
make build desktop
make desktop-dev
```

These commands call `scripts/desktop-command.mjs`, the same build/dev entry point used by CI. It runs the pinned `go tool mygo` from `apps/desktop` with `CGO_ENABLED=1`; on macOS it sets the configured minimum OS in the deployment target and cgo compiler/linker flags with no dependency on a build-machine library path. Both call `scripts/stage-desktop-resources.mjs` (`make desktop-libs` runs that preparation alone). It generates the English/Simplified Chinese macOS permission localizations from the [copy catalogs](../frontend/i18n.md) and builds/stages native ASR into `apps/desktop/resources/<platform>/asr` with pinned dependencies, shaders and licenses; they are build artifacts and stay ignored. Windows installer creation uses NSIS; cross-packaging cgo for Windows additionally needs a MinGW-w64 toolchain (`CC=x86_64-w64-mingw32-gcc`).

`desktop-dev` launches the menu bar app. On first run it shows onboarding, asks for microphone and Accessibility access, and downloads the automatically selected model bundle (about 270 MB) into the per-user model store; after that dictation works offline. `make build desktop` produces `Arion.app` and a DMG under ignored `apps/desktop/build/`. macOS ties Accessibility consent to the app's code signature: an ad hoc rebuild needs the permission granted again.

### Offline keyboard checks

On macOS, `make keyboard-check` / `pnpm keyboard:check` stages the pinned public librime runtime/data and build-only private SDK, then runs voice composition, encrypted-learning and actual-engine/AppKit typing regressions in isolated fixtures. This is included in `pnpm desktop:check`. The first preparation downloads immutable public archives, including build-only Boost headers; repeat preparations verify the same sizes/hashes. `pnpm keyboard:stage` prepares assets separately. These standard checks do not provision the user's Keychain or modify the installed input-source list. The separately authorized `scripts/test-keyboard-keychain.mjs` real-key runner refuses an existing production key, records ownership before engine work, deletes only its exact owned item and verifies restoration. Its full Data Protection lifecycle requires a profile-authorized signing configuration. That configuration now passes all six phases with device-only/nonsynchronizing attributes verified from the OS and both stores restored absent. The earlier unprofiled `-34018` failure remains historical evidence. Legacy rejection and synthetic policy tests are independently runnable. The source-test runners fail explicitly when called without staged artifacts.

`make keyboard-verify` / `pnpm keyboard:verify` checks the staged nested IMK bundle's universal binaries, relocatable dependencies, compiled data, source/license ledger and localizations. A packaged signed bundle also receives signature checks. This does not establish actual input-source selection or external-editor behavior. The native bundle is compiled under a temporary sibling and published only after its executable, data and localized identity files are complete. Installation checks the declared source identity, registers the installed URL with LaunchServices before TIS, and queries actual parent/child enablement. Successful registration and enable requests with a still-disabled parent retain the bundle as pending setup with an explicit Keyboard settings action; native authorization is allowed to finish without automatic navigation; explicit copy/register/enable API failures still roll back. The [competitor onboarding trace](../research/input-source-onboarding.md) records the separate static enable/retry/select stages and manual guidance. Settings distinguishes pending, ready and active sources, shows progress during copying and refreshes after OS changes. First installation continues through an explicit, cancellable enable-and-switch attempt. Existing pending sources expose Enable Arion, and enabled sources expose Use Arion; both reuse installed bytes. Setup waits up to 30 seconds, with stage-specific errors and Keyboard settings help. A system Allow prompt stays visible; no automatic Settings navigation displaces it. Manual recovery is Keyboard → Text Input → Edit… → +, add Arion, then select it. Copied-but-undiscovered and registered-but-disabled are distinct states. Cancelling or leaving setup prevents later automatic selection and retains installed files. Short standalone Shift release switches Chinese/English; Shift+Space and the input menu remain available. Switching briefly shows the selected mode beside the caret. Shift chords and long holds do not switch. Ordinary symbols commit directly without a symbol candidate menu. The input menu also offers learned-word reset; a learning failure preserves already-produced text and exposes pending text for explicit copying.

### Data locations

Fully quit Arion before a reset. These commands delete data without creating a backup:

| Command | Result |
| --- | --- |
| `make reset DRY_RUN=1` | Print the profile entries and system scopes that would be cleared; no signing, compilation or mutation |
| `make reset` | Clear every entry in the local profile, including preferences, downloaded models, vocabulary, keyboard learning, usage, diagnostics and old onboarding backups; also clear the macOS system state described below |
| `make reset keep-models` | Full reset while preserving the entire downloaded recognition/rewrite `models/` directory, including partial downloads |
| `make reset keep-models keep-preferences` | Preserve downloaded models and `preferences.json` / `keyboard.json`; onboarding completion stays as stored |
| `make reset keep-system` | Clear the profile while retaining OS state and the Windows DPAPI key blob |
| `make reset desktop onboarding` | Replay onboarding only, with a preferences backup |

Preservation suffixes combine freely: `keep-vocabulary` retains `vocabulary.bin` and its encryption key; `keep-keyboard` retains `keyboard/` and both legacy/protected keyboard keys; `keep-usage` retains `usage.json`; `keep-credentials` retains all Arion account refresh tokens; `keep-permissions` retains system consent; `keep-input-method` retains the installed input method. `keep-system` retains all system scopes, including encryption keys. `keep-preferences` preserves the two profile preference files. Unknown suffixes and unrelated extra goals fail before any recipe runs. `DRY_RUN=1` works with every full-reset suffix combination.

On macOS, system reset deletes only service `Arion`'s `vocabulary-key` and `account-refresh-token` / `account-refresh-token/<profile fingerprint>` items, plus service `si.astrafoundry.arion.keyboard.learning`'s exact `aes256-gcm-v1` item in both legacy and Data Protection stores. No secret bytes are read or exported. The native helper reuses the production input-source disable/switch/termination functions, and removes `~/Library/Input Methods/ArionInput.app` after checking its bundle identifier. Permissions are reset with `tccutil reset All` for `si.astrafoundry.arion` and `si.astrafoundry.inputmethod.arion`. Input-source discovery can remain cached by macOS until logout/login; removal does not claim to purge the OS registration cache. Retaining input-method installation or all system state requires stopping `ArionInput` yourself before resetting its data.

Deleting the protected keyboard key requires Xcode command-line tools, the configured installed Developer ID signing identity and matching `MACOS_IME_PROVISIONING_PROFILE` from the existing [signing configuration](deployment.md#macos-signing-configuration). The command reads root `.env` with shell variables taking precedence, builds a temporary helper with the IMK identity, signs it using the existing provisioning validator, and checks Keychain access before mutation. It does not import certificates, register the helper as an input source, or notarize it. `keep-keyboard` avoids the protected-key signing requirement; `keep-system` avoids all native system tooling. Keychain interaction is disabled; locked/inaccessible items fail explicitly. System operations are not transactional: an error stops the command and can leave earlier system changes applied, while profile deletion starts only after system cleanup succeeds. Retry after resolving the reported failure. Windows currently supports profile-only reset with `keep-system`; full system reset fails before mutation.

The profile is `ARION_DATA_DIR` when explicitly set, otherwise the platform's default location below. Full reset refuses relative, symlinked or broad protected directories. A custom `ARION_DATA_DIR` requires `keep-system`, because it does not isolate the shared OS credentials/permissions/input source. Repository `.workspace` models, packaged application resources, exported files and server data are outside the reset scope. Tests run with temporary profile fixtures via `node --test scripts/lib/reset-desktop.test.mjs` and the standard `pnpm test` entry point. Destructive production Keychain/permission removal is not run by these tests.

To replay onboarding only, run `make reset desktop onboarding`, then reopen the app (or run `make desktop-dev`). This existing command sets only `onboardingCompleted` to `false`, saving a uniquely named `preferences.json.<id>.bak` beside the preferences first. Other profile and system state stays intact. Missing preferences or an already reset flag require no change; invalid JSON fails without mutation. `ARION_DATA_DIR` selects its profile. Without Make, run `node scripts/reset-desktop-onboarding.mjs` for onboarding, or `node scripts/reset-desktop.mjs keep-models --dry-run` to preview a full reset; omit `--dry-run` to execute.

| Data | macOS | Windows |
| --- | --- | --- |
| Preferences | `~/Library/Application Support/Arion/preferences.json` | `%AppData%\Arion\preferences.json` |
| Verified models | `~/Library/Application Support/Arion/models/<artifact>/` | `%AppData%\Arion\models\` |
| Keyboard learning | `~/Library/Application Support/Arion/keyboard/learning/` ciphertext and lock files; separate Keychain key | Not implemented |
| Usage counts | `~/Library/Application Support/Arion/usage.json` | `%AppData%\Arion\usage.json` |
| Vocabulary (AES-GCM) | `~/Library/Application Support/Arion/vocabulary.bin`, key in the login Keychain | same folder, key protected by DPAPI |
| Hosted-sync refresh token | login Keychain, service `Arion`, account `account-refresh-token/<profile fingerprint>` | Credential Manager target `Arion/account-refresh-token/<profile fingerprint>` |

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

`apps/desktop/desktop.json` is public package metadata; `mygo.config.ts` applies environment-driven signing. `make build desktop` reads the root `.env` (shell variables take precedence), uses ad hoc signing without Apple configuration, and automatically signs/notarizes with complete credentials and a matching Developer ID IMK provisioning profile. Partial configuration fails. Ad hoc development has no Data Protection Keychain authorization, so protected keyboard learning stays unavailable. See [signing configuration](deployment.md#macos-signing-configuration) and [`.env.example`](../../.env.example). The controlled identifier is `si.astrafoundry.arion`; the IMK bundle uses `si.astrafoundry.inputmethod.arion`. Windows publisher signing still requires integration. Local preferences and encrypted vocabulary are implemented; desktop sync enrollment/outbox remain open. Credentials must use OS key stores; never add fake environment settings that appear operational.

SYNC-11 selects AstraFoundry as the standard packaged center and build-time `ARION_CENTER_*` overrides for source builds. The [center profile contract](../project/central-sync-design.md#distribution-and-build-time-center-profile) is validated and compiled by `desktop-command.mjs` for both build and dev. Omit all center variables for the standard profile; empty/unknown/incomplete overrides fail. A packaged app ignores runtime center ENV. Enrollment and outbox remain pending, so this configures destinations and account behavior rather than delivering an integrated sync client. Desktop build variables are public configuration and separate from the relay's secret-bearing deployment `ARION_SYNC_*` variables.

## Troubleshooting

- Standalone Shift switching: duplicate modifier notifications from Codex/Cursor preserve the original short-press deadline and switch once on release. Another physical key/modifier or a long hold still cancels the switch; Shift+Space remains available.

- If macOS Input Monitoring shows Arion enabled but the current signed app still reports denied after relaunch, verify the entry belongs to `/Applications/Arion.app`. Old development identities such as `local.arion.desktop` can leave stale entries. Remove only the obsolete Arion entry and add the current app again through System Settings with the user’s explicit consent, then use the OS Quit & Reopen action; never modify the TCC database directly.
- macOS dictation key: Right Option is side-specific and requires Input Monitoring for Arion. A non-null event tap is insufficient: startup checks consent and actual enabled state before reporting readiness. Missing consent leaves the key selected with a permission failure; granting consent restarts it. Revocation stops the listener and cancels the active session. Listener replacement waits for its native thread to exit and rejects old queued transitions.

- Missing toolchain/network: provision matching Go and module cache explicitly; do not lower MyGo's required version.
- Model unavailable: open the menu bar panel or Settings › Model; a failed download or checksum shows its reason and a retry. `go run ./cmd/arion-models import` installs an already-downloaded archive into a store without network.
- Platform build failure: capture compiler/SDK errors and compare pinned upstream source before choosing an adapter.
- macOS shortcut mode uses disclosed system paste for every target; system IME uses its owned IMK client. Previous delivery preferences remain readable but no longer select a macOS shortcut route. Windows delivery selection is unchanged.
- Clipboard contention/unavailability stops before posting. Restoration only occurs while Arion owns the board. A restoration error after posting preserves the uncertain outcome and warns the user to check the editor; never retry automatically.

## Automation and experiments

`make help` lists fixed flows. `make check` includes docs, portable core and evaluation harness checks. `make e2e` uses a real desktop session and briefly opens/closes a production fixture window. `make ablation-plan`, `make stream-plan` and `make datasets-check` prepare experiments; the last only fetches pinned HF metadata. Follow [evaluation commands](../../tools/evaluation/README.md) for live corpus validation, runs, reports and product scenarios. Windows users without GNU Make use the pnpm scripts directly. `.workspace/` contains private artifacts; automated benchmark CI uploads only aggregates.

System input-source installation requires the profile-bearing Developer ID host. Runtime preflight rejects unsigned/ad hoc payloads before replacing installed files; `desktop-dev` remains suitable for shell/shortcut development but must not install its ad hoc IMK payload. Use the signed DMG for the complete installation/authorization scenario. A valid copied host that is temporarily absent from TIS stays pending; it is not treated as missing installation. The installer still does not automatically select the input source.

Native MLX builds require Xcode 26.4.1 / Swift 6.3.1, while their deployment target remains macOS 14.0. Intel and Windows CPU builds require CMake and a native C++ compiler; the Windows Vulkan executable is digest-pinned. See [native ASR packaging](../decisions/020-native-qwen-asr.md).
