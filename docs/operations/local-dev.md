# Local development

Research commands qualified with `/path/to/Arion-evaluation` run in the private [Arion evaluation repository](https://github.com/AstraFoundry/Arion-evaluation). Product checks run in this public checkout.

Current optional rewriting uses the [selected staged GGUF configuration](local-rewrite.md), with a verified profile weight and bundled private-socket runtime.

## Foundation checks

Use Node from `.nvmrc` (24) and exact pnpm 10.33.0 from `package.json`. The public workspace has dependency-free product driver tooling and a committed lockfile; install with `make install` / `pnpm install --frozen-lockfile`. npm/yarn installation is rejected. Native dependencies remain Go modules. `pnpm install` and `make hooks` set `core.hooksPath` to `.githooks`. The pre-push hook runs `gofmt` across the Go modules and `node scripts/check-repository.mjs` (the same work as `pnpm check`). It does not run desktop tests, native E2E, or `go test`. GitHub Actions does not install the hook.

```sh
make install
make check
make desktop-check
make e2e
```

`pnpm check` validates local Markdown links (including `README.zh-CN.md` and `README.ja.md`), runtime configuration JSON, active-doc placeholders, Go layer imports and JavaScript syntax. It also fails on a documented `pnpm` script or `make` target that does not exist , a document under `docs/` that no other document links, a duplicate or unlisted ADR number, a translated README whose local links differ from `README.md`, and an `AGENTS.md` that is not a mirror of `CLAUDE.md`. `make core-check` runs portable core tests/vet/format validation in standalone module mode. `make sync-check` and `make updates-check` do the same for `services/sync` and `services/updates`. `pnpm test` runs the script tests, the core, sync and updates checks, the i18n gate and the public driver contract tests. Private study and historical archive tests run in Arion evaluation. `make sync-integration` needs `ARION_SYNC_TEST_DATABASE_URL` and is not part of the default run. `make i18n-check` / `pnpm i18n:check` validates translation keys, templates, formatting and direct hardcoded copy. These do not claim OS adapter or inference coverage.

`make site-check` / `pnpm site:check` checks the marketing site in `apps/site`: dictionary completeness, TypeScript, ESLint, Vitest and the Next.js static export into `apps/site/out`. `pnpm --filter @arion/site dev` serves it on `127.0.0.1` for local work; see the [site README](../../apps/site/README.md) and [site conventions](../frontend/site.md).

## Native desktop

MyGo v0.4.6 requires Go 1.27.1; the workspace selects the patched `go1.27.3` toolchain. The workspace/tool directive uses Go automatic toolchain selection when available; a first native build may download it. Use an installed matching Go toolchain with `GOTOOLCHAIN=local` for explicitly offline development after dependencies are provisioned.

Both supported desktop targets also need a native Rust/Cargo toolchain for the locked keyboard ranking worker; see [resource preparation](README.md).

macOS builds also require CMake on `PATH` to compile the pinned Rime keyboard runtime from source. Check `cmake --version` before running either desktop command. A `spawnSync cmake ENOENT` error in `stage-keyboard-runtime.mjs` means CMake is missing or unavailable on that shell's `PATH`; it is not an input-method signing failure.

macOS packages require macOS 14 or later. Apple silicon speech recognition stages the native MLX worker with `xcodebuild` (`ArionASR`). That needs the full **Xcode 26.4.1** app, its Metal toolchain (`xcodebuild -downloadComponent MetalToolchain`) and Swift 6.3.1, not Command Line Tools alone: `xcode-select -p` must be `/Applications/Xcode.app/Contents/Developer`. Open Xcode once (or run `sudo xcodebuild -license accept && sudo xcodebuild -runFirstLaunch`). The first cold `ArionASR` build invokes `xcodebuild -quiet`, which fetches SwiftPM dependencies and compiles `mlx-swift` and Metal shaders for 2–5 minutes without per-line output before printing `Staged native ASR for darwin-arm64.`. The deployment target remains macOS 14.0. Rewriting stages llama.cpp as described in [local rewrite](local-rewrite.md). Windows builds must run on Windows with the native toolchain requirements declared by pinned MyGo. Check [MyGo research](../research/mygo.md) and inspect actual compiler errors rather than adding cross-platform stubs.

```sh
make build desktop
make desktop-dev
```

These commands call `scripts/desktop-command.mjs`, the same build/dev entry point used by CI. It runs the pinned `go tool mygo` from `apps/desktop` with `CGO_ENABLED=1`; on macOS it sets the configured minimum OS in the deployment target and cgo compiler/linker flags with no dependency on a build-machine library path. Both call `scripts/stage-desktop-resources.mjs` (`make desktop-libs` runs that preparation alone). It generates the English/Simplified Chinese macOS permission localizations from the [copy catalogs](../frontend/i18n.md) and builds/stages native ASR into `apps/desktop/resources/<platform>/asr` with pinned dependencies, shaders and licenses; they are build artifacts and stay ignored. Windows installer creation uses NSIS; cross-packaging cgo for Windows additionally needs a MinGW-w64 toolchain (`CC=x86_64-w64-mingw32-gcc`).

### Switch between the release and development

Daily use runs the signed release in `/Applications/Arion.app`. On macOS, use this sequence for development:

1. Quit Arion from its menu. Run `make desktop-status` to confirm that no desktop process remains.
2. Configure a persistent signing identity as described below.
3. Run `make desktop-dev`. The command refuses an already-running release or development app before staging resources.
4. Test the `Arion Dev` bundle printed by the command. Run `make desktop-status` to verify the actual executable path and SHA-256.
5. Quit `Arion Dev`. Run `make desktop-production` to open the installed release, then verify `make desktop-status` again.

On macOS, the command builds and signs once, then launches the bundle through LaunchServices. It does not run a direct-process live-reload watcher. Quit the app before each explicit rebuild. This keeps privacy requests attributed to Arion Dev instead of the terminal or coding agent that ran the build. The switching commands do not terminate processes or change permissions. Do not open an ad hoc `make build desktop` preview beside the installed release: both have the production identifier, so OS reopen requests can select a different copy. Use `desktop-dev` for shortcut development and the signed release workflow for installation, IME, and update acceptance.

`Arion Dev` uses identifier `si.astrafoundry.arion.dev`. Mutable state, including preferences, model downloads, vocabulary, clipboard history, usage, logs, and update records, lives in the `Arion Dev` profile under the OS configuration directory. The release keeps its `Arion` profile. Credential service names follow the same separation. No production credentials or preferences are migrated into development. The first development launch therefore requires its own setup and model provisioning.

Development does not connect to or manage the installed system input method. This prevents it from changing the release's keyboard settings, learning, or vocabulary projection. Signed IME acceptance uses the release workflow. The development updater is unavailable and does not load, recover, or rewrite saved update transactions. See [ADR 035](../decisions/035-development-runtime-isolation.md).

On macOS, generated development bundles live under `~/Library/Caches/Arion/desktop-dev/<checkout-hash>/darwin-arm64/`. This keeps them outside file-provider synchronization. Source resources stay in the checkout. Each checkout has its own output directory, and the launch preflight checks running Arion bundles across checkouts.

The login-Keychain dialog naming “Arion” is separate from TCC. Vocabulary and optional clipboard history read their OS-kept encryption keys at startup. The ad hoc development bundle's designated requirement is tied to its code hash, so a rebuild/live reload can invalidate a previously trusted identity and prompt again, even after “Always Allow”. To avoid repeated prompts while editing, stop the `make desktop-dev` watcher and open the already-built cached bundle; the durable solution is development signing with a stable certificate requirement. Do not delete encryption keys or change a keychain item's access to all applications. Clipboard capture uses the in-memory initialized store, not a Keychain read on every copy. [Apple TN3127](https://developer.apple.com/documentation/technotes/tn3127-inside-code-signing-requirements) explains the signing requirement boundary.

### macOS development build and TCC permission diagnostics

Set `MACOS_DEV_SIGNING_IDENTITY` in the root `.env` to a persistent certificate name or its SHA-1 fingerprint. The command resolves exactly one valid installed identity with `security find-identity -v -p codesigning`, then passes that fingerprint to MyGo's `dev -sign` option for each build. The maintained `-build-only` option leaves launch and lifetime to LaunchServices. Missing, ad hoc, unavailable, or ambiguous identities fail before compilation. Development signing does not require notarization or an IME provisioning profile.

To create an Apple development identity in Xcode:

1. Open **Xcode > Settings > Apple Accounts** and sign in.
2. Select your account and team, then open **Manage Certificates**.
3. Click **+ > Apple Development**. Keep this identity for subsequent builds.
4. Run `security find-identity -v -p codesigning` and copy its fingerprint into `MACOS_DEV_SIGNING_IDENTITY`.

Xcode installs the certificate and private key in the login Keychain. If the identity exists but is not valid, verify its issuing intermediate certificate. Apple Development uses WWDR G3; install the matching intermediate from [Apple PKI](https://www.apple.com/certificateauthority/) when missing, without adding custom trust overrides. If an existing certificate lacks its private key, import a password-protected `.p12` exported from the original Mac. A downloaded `.cer` alone cannot sign code. Do not revoke shared team certificates to repair one development machine. See [Apple's certificate instructions](https://help.apple.com/xcode/mac/current/en.lproj/dev154b28f09.html).

Grant microphone, Accessibility, and Input Monitoring to **Arion Dev** once after establishing its fixed signing identity. Permission retention across changed builds must be checked on the actual Mac. If macOS requests a reopen, reopen the already-built bundle at the printed path. Verify the executable with `make desktop-status` before testing. Do not reset TCC records as part of a normal rebuild.

| Symptom | Check and action |
| --- | --- |
| Xcode or Metal compiler missing | Select the full Xcode application and install its Metal toolchain using the commands above. |
| Permission switch is on but access is denied | Check the running path, signing identity, and designated requirement with `codesign -d -r- <bundle>`. An old ad hoc record can remain after switching identities. Remove only the obsolete development entry through System Settings with the user's consent, then grant the final signed bundle. |
| Microphone entry is absent | Use **Settings > General > Authorize** in Arion Dev to request the native microphone dialog. A previously denied request requires the user to change the OS setting. |
| Screen Recording picker cannot find the bundle | In the OS picker, use **+**, then **⌘⇧G**, and enter the printed absolute bundle directory. Screen content can remain disabled for shortcut-only tests. |
| No Dock icon after launch | Arion Dev is a menu bar agent. Check `make desktop-status` and its menu bar item. |
| System input-method setup is unavailable | Development does not own the installed input method. Use the signed release workflow for IME acceptance. |
| Dictation corrections are not learned in an editor | Edit learning needs readable `AXValue`. Test in Notes or TextEdit. Electron editors, terminals, and many browsers do not expose the required value. |

For shortcut delivery acceptance, verify both polishing and translation: no editable focus copies the processed result; an empty editable field accepts normal paste; moving the caret or focusing another editable field still pastes there. Check the editor and actual clipboard contents as well as the completion label. On macOS and Windows, also focus a Feishu (Lark) chat composer and a Claude composer (Claude Desktop, and Claude in a browser). Each should insert once while the caret is in the composer, and copy when focus is a button or a read-only transcript. Then change code, rebuild, reopen the same signed development identity, and repeat without changing OS authorization.

### Offline keyboard checks

On macOS, `make keyboard-check` / `pnpm keyboard:check` stages the pinned public librime runtime/data and build-only private SDK, then runs voice composition, encrypted-learning and actual-engine/AppKit typing regressions in isolated fixtures. This is included in `pnpm desktop:check`. The first preparation downloads immutable public archives, including build-only Boost headers; repeat preparations verify the same sizes/hashes. `pnpm keyboard:stage` prepares assets separately. These standard checks do not provision the user's Keychain or modify the installed input-source list. The separately authorized `scripts/test-keyboard-keychain.mjs` real-key runner refuses an existing production key, records ownership before engine work, deletes only its exact owned item and verifies restoration. Its full Data Protection lifecycle requires a profile-authorized signing configuration. That configuration now passes all six phases with device-only/nonsynchronizing attributes verified from the OS and both stores restored absent. The earlier unprofiled `-34018` failure remains historical evidence. Legacy rejection and synthetic policy tests are independently runnable. The source-test runners fail explicitly when called without staged artifacts.

`make keyboard-verify` / `pnpm keyboard:verify` checks the staged nested IMK bundle's universal binaries, relocatable dependencies, compiled data, source/license ledger and localizations. A packaged signed bundle also receives signature checks. This does not establish actual input-source selection or external-editor behavior. The native bundle is compiled under a temporary sibling and published only after its executable, data and localized identity files are complete. Installation checks the declared source identity, registers the installed URL with LaunchServices before TIS, and queries actual parent/child enablement. Successful registration and enable requests with a still-disabled parent retain the bundle as pending setup with an explicit Keyboard settings action; native authorization is allowed to finish without automatic navigation; explicit copy/register/enable API failures still roll back. The [competitor onboarding trace](../research/input-source-onboarding.md) records the separate static enable/retry/select stages and manual guidance. Settings distinguishes pending, ready and active sources, shows progress during copying and refreshes after OS changes. First installation continues through an explicit, cancellable enable-and-switch attempt. Existing pending sources expose Enable Arion, and enabled sources expose Use Arion; both reuse installed bytes. Setup waits up to 30 seconds, with stage-specific errors and Keyboard settings help. A system Allow prompt stays visible; no automatic Settings navigation displaces it. Manual recovery is Keyboard → Text Input → Edit… → +, add Arion, then select it. Copied-but-undiscovered and registered-but-disabled are distinct states. Cancelling or leaving setup prevents later automatic selection and retains installed files. Short standalone Shift release switches Chinese/English; Shift+Space and the input menu remain available. Switching briefly shows the selected mode beside the caret. Shift chords and long holds do not switch. Ordinary symbols commit directly without a symbol candidate menu. The input menu starts with Voice input, titled with the dictation key, which toggles dictation through the running app and is disabled without it. It also offers learned-word reset; a learning failure preserves already-produced text and exposes pending text for explicit copying.

### Data locations

These reset commands default to the release profile. Fully quit the target app before a reset. The full-reset commands delete data without creating a backup:

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

To replay onboarding only, run `make reset desktop onboarding`, then reopen the release. For development onboarding, set `ARION_DATA_DIR` to the `Arion Dev` profile before running that command, then reopen the development bundle. This existing command sets only `onboardingCompleted` to `false`, saving a uniquely named `preferences.json.<id>.bak` beside the preferences first. Other profile and system state stays intact. Missing preferences or an already reset flag require no change; invalid JSON fails without mutation. `ARION_DATA_DIR` selects its profile. Without Make, run `node scripts/reset-desktop-onboarding.mjs` for onboarding, or `node scripts/reset-desktop.mjs keep-models --dry-run` to preview a full reset; omit `--dry-run` to execute.

The table lists release locations. Development application data and credentials use the separate `Arion Dev` namespace described above; installed keyboard data remains release-owned.

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

`apps/desktop/desktop.json` is public package metadata; `mygo.config.ts` applies environment-driven signing. `make build desktop` reads the root `.env` (shell variables take precedence), uses ad hoc signing without Apple configuration, and automatically signs/notarizes with complete credentials and a matching Developer ID IMK provisioning profile. Partial configuration fails. Development does not own the installed input method or its protected keyboard learning. See [signing configuration](deployment.md#macos-signing-configuration) and [`.env.example`](../../.env.example). The controlled identifier is `si.astrafoundry.arion`; the IMK bundle uses `si.astrafoundry.inputmethod.arion`. Windows publisher signing still requires integration. Local preferences and encrypted vocabulary are implemented; desktop sync enrollment/outbox remain open. Credentials must use OS key stores; never add fake environment settings that appear operational.

SYNC-11 selects AstraFoundry as the standard packaged center and build-time `ARION_CENTER_*` overrides for source builds. The [center profile contract](../project/central-sync-design.md#distribution-and-build-time-center-profile) is validated and compiled by `desktop-command.mjs` for both build and dev. Omit all center variables for the standard profile; empty/unknown/incomplete overrides fail. A packaged app ignores runtime center ENV. Enrollment and outbox remain pending, so this configures destinations and account behavior rather than delivering an integrated sync client. Desktop build variables are public configuration and separate from the relay's secret-bearing deployment `ARION_SYNC_*` variables.

## Troubleshooting

- Standalone Shift switching: duplicate modifier notifications from Codex/Cursor preserve the original short-press deadline and switch once on release. Another physical key/modifier or a long hold still cancels the switch; Shift+Space remains available.

- If macOS Input Monitoring shows Arion enabled but the current signed app still reports denied after relaunch, verify the entry belongs to `/Applications/Arion.app`. Old development identities such as `local.arion.desktop` can leave stale entries. Remove only the obsolete Arion entry and add the current app again through System Settings with the user’s explicit consent, then use the OS Quit & Reopen action; never modify the TCC database directly.
- macOS dictation key: Right Option is side-specific and requires Input Monitoring for Arion. A non-null event tap is insufficient: startup checks consent and actual enabled state before reporting readiness. Missing consent leaves the key selected with a permission failure; granting consent restarts it. Revocation stops the listener and cancels the active session. Listener replacement waits for its native thread to exit and rejects old queued transitions.

- Missing toolchain/network: provision matching Go and module cache explicitly; do not lower MyGo's required version. `make workflows-check` installs pinned actionlint v1.7.11 into `.workspace/tools/actionlint/` on its first run, using Go checksum verification and the terminal's `GOPROXY`/`GOSUMDB` configuration. If `proxy.golang.org` is unreachable, explicitly set `GOPROXY` to a trusted reachable module proxy for that first invocation (or configure the terminal's HTTPS proxy). Subsequent checks verify the cached tool's module/version and run it without module downloads, including with `GOPROXY=off`. Installation failures stop the gate; workflows are never silently skipped. Delete the version/platform cache directory to provision it again.
- `xcodebuild` / `tool 'xcodebuild' requires Xcode` or `cannot execute tool 'metal' due to missing Metal Toolchain` during `make desktop-dev` or `make build desktop`: install the full Xcode 26.4.1 app, run `sudo xcode-select -s /Applications/Xcode.app/Contents/Developer && sudo xcodebuild -license accept && sudo xcodebuild -runFirstLaunch`, and install the Metal shader compiler with `xcodebuild -downloadComponent MetalToolchain`. Command Line Tools cannot compile `ArionASR`. On the first run, `scripts/stage-asr-runtime.mjs` runs `xcodebuild -quiet`, which takes 2–5 minutes with no terminal output while fetching SwiftPM packages and compiling `mlx-swift` + Metal shaders; wait for `Staged native ASR for darwin-arm64.` instead of interrupting with `Ctrl+C`.
- macOS signing failure `resource fork, Finder information, or similar detritus not allowed`: Finder/file-provider metadata can appear on generated `.app` and Swift resource `.bundle` directories. The maintained MyGo signing path removes only `com.apple.FinderInfo` and `com.apple.ResourceFork` before nested signing and again before the outer signature; it preserves unrelated metadata and original resource modes, and does not follow symlinks. Both build and dev use this path. `make desktop-dev` also keeps the running bundle in the local cache to prevent metadata being reattached by Desktop synchronization. Restart an already-running `desktop-dev` after changing the MyGo CLI so it loads the patched tool.
- Model unavailable: open the menu bar panel or Settings › Speech & Polish; a failed download or checksum shows its reason and a retry. `go run ./cmd/arion-models import` installs an already-downloaded archive into a store without network.
- Platform build failure: capture compiler/SDK errors and compare pinned upstream source before choosing an adapter. On macOS, `scripts/cc-relocatable.sh` uses `xcrun --sdk macosx clang` so cgo compilation and linking use the selected Xcode's macOS SDK. An `unknown architecture` error in a Command Line Tools `.tbd` file can indicate that a bare compiler selected an SDK newer than its linker supports; verify `xcode-select -p` and `xcrun --sdk macosx --show-sdk-path`.
- macOS shortcut mode uses disclosed system paste for every target; system IME uses its owned IMK client. Previous delivery preferences remain readable but no longer select a macOS shortcut route. Windows delivery selection is unchanged.
- Clipboard contention/unavailability stops before posting. Restoration only occurs while Arion owns the board. A restoration error after posting preserves the uncertain outcome and warns the user to check the editor; never retry automatically.

## Automation and experiments

`make help` lists fixed flows. `make check` runs the repository checks (`make docs`), the GitHub Actions lint (`make workflows-check`) and `pnpm test`. `make e2e` uses a real desktop session and briefly opens/closes a production fixture window. `make -C /path/to/Arion-evaluation ablation-plan`, `make -C /path/to/Arion-evaluation stream-plan` and `make -C /path/to/Arion-evaluation datasets-check` prepare experiments; the last only fetches pinned HF metadata. Follow the [public driver contract](../../tools/driver/README.md) for product scenarios, and the private repository README for corpus validation, study runs and historical report replay. The [repository split](../project/repository-split.md) defines their ownership. Windows users without GNU Make use the pnpm scripts directly. `.workspace/` contains private artifacts. Private studies run locally; review aggregate summaries before publishing them.

System input-source installation requires the profile-bearing Developer ID host. Runtime preflight rejects unsigned/ad hoc payloads before replacing installed files; `desktop-dev` owns shell and shortcut development and does not manage the installed IMK payload. Use the signed DMG for the complete installation/authorization scenario. A valid copied host that is temporarily absent from TIS stays pending; it is not treated as missing installation. The installer still does not automatically select the input source.

Native MLX builds require the Xcode 26.4.1 application / Swift 6.3.1 (`xcode-select` pointed at `Xcode.app`), while their deployment target remains macOS 14.0. Windows CPU builds require CMake and a native C++ compiler; the Windows Vulkan executable is digest-pinned. See [native ASR packaging](../decisions/020-native-qwen-asr.md).

ASR hotword changes require rebuilding the native worker as well as Go. `make desktop-dev` and `make build desktop` stage the current `ArionASR` automatically; `make desktop-libs` prepares native resources alone. A custom `ARION_ASR_WORKER` must point to the rebuilt worker: protocol 3 alone does not prove prompt support. `arion-mlx-2` advertises `ready.capabilities: ["prompt"]`; the pinned speech.cpp worker advertises `prompt` as a steering string option in `ready.model.options`. Nonempty hotword context fails explicitly when the selected worker lacks this capability. No model re-download or speech.cpp dependency update is needed for this option.

Each transcription carries its own space-separated canonical `prompt`, including an explicit empty string for an empty snapshot. The native workers apply the loaded model's token/position limit to the complete prompt, audio and generation budget; this is separate from the Go hotword selection budget. Reproduce or verify the [native prompt smoke](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/reproduction/asr-hotwords-v1/runtime-smoke/README.md`) for capability, resident-worker isolation and overflow evidence. Apple Silicon execution needs Metal access; the retained sandbox failure emitted no diagnostic reason and passed after sandbox escalation. This smoke does not establish native Windows hotword inference.
