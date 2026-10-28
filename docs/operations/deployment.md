# Distribution and deployment

## Current capability

`make build desktop` packages `Arion.app` (menu bar app, `LSUIElement`), localized microphone descriptions, the audio-input entitlement, native Qwen3-ASR workers and their resource closure, the IMK host, and its disk image. Both Apple silicon and Windows x64 also ship the pinned local-polishing worker (`arion-llm`) and GGUF runtime ([ADR 018](../decisions/018-staged-gguf-rewriting.md), [ADR 025](../decisions/025-windows-local-polishing.md)). The macOS package minimum is 14.0, matching the highest minimum in the bundled native libraries.

`scripts/desktop-command.mjs` is the shared local/CI build entry point and sets the configured minimum OS in cgo compiler/linker flags. Its `scripts/stage-desktop-resources.mjs` preparation step includes Go module licenses, native runtime licenses/notices and model provenance. `scripts/cc-relocatable.sh` drops build-machine rpaths at link time. The read-only `scripts/verify-desktop-package.mjs` checks required resources, architectures, package identities/versions, native minimum OS versions and macOS signatures/rpaths. Models are downloaded separately from pinned URLs with verified digests.

The identifiers are `si.astrafoundry.arion` and `si.astrafoundry.inputmethod.arion`. macOS uses ad hoc signing without Apple configuration; complete configuration enables Developer ID signing and notarization through the same build command. Windows publisher signing remains open. The [release status](release-status.md) distinguishes packaged capabilities from actual host acceptance. The sync relay image and compose stack live in [deploy/sync](../../deploy/sync/README.md); the complete center (relay plus update channels behind one TLS origin) is [deploy/center](../../deploy/center/README.md).

Windows CPU ASR disables OpenMP and retains ggml's native thread pool. MSVC uses the static C/C++ runtime (`MultiThreaded`); `/MT` alone does not remove OpenMP's `VCOMP140.DLL` dependency. Staging and final package verification read normal and delay-loaded PE imports recursively, reject unbundled DLL dependencies, and execute the CPU worker's `--version` with only Windows system directories on `PATH`. The optional Vulkan worker may import the GPU driver's `vulkan-1.dll`; CPU recognition must run without that driver or compiler redistributables.

The Windows EXE installer stages its complete payload on the destination volume before replacing installed entries. It rejects a running application, serializes installers for the same product, and rolls back renamed entries if replacement fails. It never treats an ignored extraction error as a successful installation. Unrelated top-level files remain untouched; model and keyboard user data remain in the separate user profile. A failed rollback retains its backup path for recovery. This handles ordinary installation errors, not abrupt power loss. The isolated NSIS regression installs, rejects a running app and a locked resource, checks rollback, retries the upgrade and uninstalls the test application. Windows `pnpm updater:check` runs this real installer regression alongside the shared signed-update checks in CI.

```sh
make build desktop
```

Apple silicon `make build desktop` and `make desktop-dev` compile `ArionASR` with `xcodebuild`. Install the full Xcode 26.4.1 / Swift 6.3.1 application, point `xcode-select` at `Xcode.app/Contents/Developer`, complete first-launch initialization, and install the Metal toolchain; Command Line Tools alone are not enough:

```sh
sudo xcode-select -s /Applications/Xcode.app/Contents/Developer
sudo xcodebuild -license accept && sudo xcodebuild -runFirstLaunch
xcodebuild -downloadComponent MetalToolchain
```

On a clean checkout, the first `ArionASR` build runs `xcodebuild -quiet` and takes 2–5 minutes without per-line output while fetching SwiftPM packages and compiling `mlx-swift` and Metal shaders. Without Apple signing variables in `.env` or the shell, `make build desktop` produces an ad hoc signed preview (`apps/desktop/build/Arion.app` and `.dmg`): shortcut dictation works after granting permissions, but macOS TCC consent does not persist across ad hoc rebuilds and system input-method (`ArionInput.app`) installation is refused with `ARTIFACT_UNVERIFIED` until signed with `MACOS_IME_PROVISIONING_PROFILE`.

When running `make desktop-dev` for local macOS development (`Arion Dev`, `si.astrafoundry.arion.dev`, output under `~/Library/Caches/Arion/desktop-dev/<checkout-hash>/darwin-arm64/Arion Dev.app`), see the diagnostic table in [macOS development build and TCC permission diagnostics](local-dev.md#macos-development-build-and-tcc-permission-diagnostics) (covering `xcodebuild` / `MetalToolchain` setup, the 2–5 minute silent cold build, avoiding the ad hoc `CDHash` permission invalidation loop by launching the cached bundle with `open` instead of rerunning `make desktop-dev`, resetting Microphone / Accessibility / Input Monitoring TCC records with `tccutil reset`, adding Screen Recording via `+` and `⌘⇧G`, `ARTIFACT_UNVERIFIED` on ad hoc IME installation, and testing `AXValue` edit learning in Notes/TextEdit). See also [ADR 020](../decisions/020-native-qwen-asr.md).

## Center configuration

`make build desktop` and `make desktop-dev` read the same public center profile from shell ENV or the root `.env`. Omit all `ARION_CENTER_*` variables for AstraFoundry defaults. For a device-authenticated self-hosted center:

```dotenv
ARION_CENTER_URL=https://sync.example.org/prefix
ARION_CENTER_AUTH_MODE=device
```

Custom `oidc` requires all five `ARION_CENTER_OIDC_*` fields in [`.env.example`](../../.env.example). The HTTP base URL and token resource are separate settings. Empty, unknown, conflicting and incomplete values fail; URLs must use HTTPS, except explicit loopback HTTP for development, and cannot contain credentials/query/fragment.

The app embeds the validated public profile and ignores launch-time center ENV. Package verification checks the executable's `--print-center-profile` result without center ENV. Custom profiles isolate account refresh tokens and disable the official update feed; device mode offers no account sign-in, and custom OIDC never calls the official waitlist API. Official tagged releases require the exact standard profile. Existing legacy account tokens are retained but no longer read: sign in again. Local vocabulary is unchanged. See the [profile contract and remaining sync limits](../project/central-sync-design.md#implemented-profile-boundaries).

## Manual signed macOS installer

Run [Signed macOS build](../../.github/workflows/signed-macos.yml) from GitHub Actions on the intended branch or tag when signing credentials are held by the team. It reuses the existing CI checks and signed packaging with the macOS ARM64 matrix only, using the configured organization/repository secrets. Download the `desktop-darwin-arm64` artifact after success. The independent concurrency group lets this build finish while cross-platform releases are retried. This workflow creates installer artifacts; it does not create tags, stage release storage or publish an update channel. The source commit is recorded by the workflow run.

## macOS signing configuration

The build entry point reads the root `.env` when present; existing shell variables take precedence, including deliberately empty values. Copy [`.env.example`](../../.env.example) for the settings. `make build desktop` is the only packaging command: no Apple variables produces a preview, partial Apple configuration fails, and complete configuration signs and notarizes. `MYGO_UPDATER_PRIVATE_KEY` independently enables signed updates. `desktop-dev` keeps development signing and does not import release credentials.

For a local Mac, install a **Developer ID Application** certificate with its private key, store notarization credentials once with `xcrun notarytool store-credentials astrafoundry`, and configure:

```dotenv
MACOS_SIGNING_IDENTITY="Developer ID Application: Your Company (ABCDE12345)"
MACOS_NOTARY_PROFILE=astrafoundry
MACOS_IME_PROVISIONING_PROFILE=/private/path/ArionInput.provisionprofile
```

The signed desktop build additionally requires `MACOS_IME_PROVISIONING_PROFILE`, an existing absolute path to an Apple **Developer ID distribution provisioning profile** for `si.astrafoundry.inputmethod.arion`. Its macOS platform, expiration, Team ID, permitted application identifier and signing certificate must match. The build decodes the CMS profile, extracts selected JSON-compatible fields through XML for macOS 15 compatibility, validates those constraints before resource compilation, embeds it as `ArionInput.app/Contents/embedded.provisionprofile`, and gives only that nested host `com.apple.application-identifier` and `com.apple.developer.team-identifier` through MyGo's `helperEntitlements`. No shared Keychain group is requested. Package verification checks the embedded authorization and actual signed host entitlements. Keep this account asset outside the repository; setting a profile path does not acquire or create one.

This profile authorizes the Data Protection Keychain default access group ([Apple TN3137](https://developer.apple.com/documentation/technotes/tn3137-on-mac-keychains), [Keychain access groups](https://developer.apple.com/documentation/security/sharing-access-to-keychain-items-among-a-collection-of-apps), [profile model](https://developer.apple.com/documentation/technotes/tn3125-inside-code-signing-provisioning-profiles)). It is not an established fix for the separate TIS parent-enable failure. The current local profile has passed certificate/application authorization checks and the real protected-key lifecycle with device-only/nonsynchronizing OS attributes. This does not complete installed IMK acceptance. Ad hoc development builds do not add these entitlements or embed a profile; system input-source installation now rejects that payload before changing installed files. Use the signed release app for the native authorization scenario. Existing ad hoc hosts from earlier Dev builds can be removed explicitly from the signed app before installing its bundled trusted host. Protected keyboard learning never uses a legacy Keychain fallback.

`MACOS_NOTARY_KEYCHAIN` optionally selects the existing keychain holding that profile. The build checks the identity/private key and access to the notary service before staging resources. Keep certificates and passwords outside the repository; a `.cer` alone does not contain the private key. Export the certificate and private key together as a password-protected `.p12` for CI.

Apple credentials use application-independent `MACOS_*` and `APPLE_*` names so the same Developer ID certificate can sign other AstraFoundry applications. The former `ARION_MACOS_*` names have been replaced by `MACOS_*`; migrate local environment and GitHub secret names together. Use `astrafoundry` as the label for new notary profiles, CSR key labels and exported certificate files. Existing profiles such as `arion-notary` remain usable when explicitly configured. Certificate signing identities must retain the exact Apple-issued name; changing a local label does not change the certificate's legal holder. Keep each application's update signing key separate.

For CI or importing credentials locally, configure the following instead of an existing profile:

| Environment variable / GitHub repository secret | Value |
| --- | --- |
| `MACOS_SIGNING_IDENTITY` | Full Developer ID Application identity, including Team ID |
| `MACOS_IME_PROVISIONING_PROFILE_BASE64` | CI secret: Base64 of the matching Developer ID IMK profile; macOS CI decodes it to a private temporary file and sets `MACOS_IME_PROVISIONING_PROFILE` to its absolute path |
| `MACOS_CERTIFICATE_P12_BASE64` | Base64 of the `.p12` containing certificate and private key |
| `MACOS_CERTIFICATE_PASSWORD` | Password protecting the `.p12` |
| `APPLE_ID` | Apple account used for notarization |
| `APPLE_TEAM_ID` | Team ID matching the signing identity |
| `APPLE_APP_SPECIFIC_PASSWORD` | App-specific password for that account |
| `MYGO_UPDATER_PRIVATE_KEY` | Separate Ed25519 update key matching `desktop.json` |

The script creates a temporary keychain with a random password, imports the certificate for Apple signing tools, adds it to the user's search list, and stores a validated `astrafoundry` profile there. It restores the original search list and deletes temporary credentials on completion or failure. Raw Apple credentials are removed from build subprocess environments. Existing certificates and profiles are left in place. GitHub-hosted runners are disposable; self-hosted signing jobs require a dedicated user because the keychain search list is user-wide.

Prerelease App and IMK bundles use the numeric base version in `CFBundleShortVersionString` and `CFBundleVersion`, set before signing; installer names and signed update manifests retain the complete prerelease version. Public package metadata lives in `apps/desktop/desktop.json`; `mygo.config.ts` adds environment-driven signing through the pinned MyGo API without rewriting source files. MyGo signs nested executables/libraries and the IMK bundle before the outer app, enables hardened runtime and secure timestamps, signs and notarizes the DMG, and staples tickets to both DMG and App **before** producing Ed25519-signed updates. The verifier checks code identities, runtime flags, timestamps, App/DMG tickets and Gatekeeper acceptance; with updates enabled it also verifies the final archive signature and the extracted App's signature/ticket. No signed archive is modified afterward.

The current profile-bearing macOS arm64 canonical build completes Developer ID signing, Accepted notarization, embedded authorization checks and Gatekeeper validation. A retained build before this profile gate also passed packaging trust checks. Its installed nested IMK host also passed Gatekeeper, but source selection still failed; packaging trust does not establish IMK acceptance. Clean-machine microphone, IMK, inference and published-update acceptance remain separate gates. See [release status](release-status.md).

## Releases and updates

Release assets retain MyGo's native architecture names. Windows installers build as `Arion Setup <version> amd64.exe` and are uploaded as `Arion.Setup.<version>.amd64.exe`; macOS installers build as `Arion <version> arm64.dmg` and are uploaded as `Arion.<version>.arm64.dmg`. Update archives use `arion-<version>-windows-amd64.tar.gz` and `arion-<version>-darwin-arm64.tar.gz`, with `update-windows-amd64.json` and `update-darwin-arm64.json`. The build does not rename update archives or rewrite manifest URLs. GitHub Releases and download pages should describe these targets as **Windows x64 (Intel / AMD)** and **macOS Apple Silicon (M series)**. Existing published assets keep their original names.

The app checks the Arion center's `stable` channel, or `beta` with **Allow beta updates** on, at `https://api.arion.astrafoundry.si/v1/updates` (`desktop.json` `updates.url`, [ADR 019](../decisions/019-center-update-channels.md)); no client request reaches the GitHub API. With `MYGO_UPDATER_PRIVATE_KEY` set, `mygo build` also writes the signed update archive and `update-<os>-<arch>.json`, whose URLs point at the center.

1. Once: configure the repository secrets above, including the matching Ed25519 secret key as `MYGO_UPDATER_PRIVATE_KEY`, and keep a backup in the password manager. It is independent of Apple Developer certificates. If the original update private key is lost before the first release, deliberately replace the key pair and the configured public key before shipping; after shipping, rotation needs the old signing key. Do not commit keys. Also configure release staging: `ARION_RELEASES_S3_ENDPOINT`, `ARION_RELEASES_S3_REGION`, `ARION_RELEASES_S3_BUCKET`, `ARION_RELEASES_S3_PREFIX` (`arion`), `ARION_RELEASES_S3_ACCESS_KEY_ID` and `ARION_RELEASES_S3_SECRET_ACCESS_KEY`, a token that can write only `files/` and `releases/` under the prefix.
2. Run `pnpm contributors:update` (or `make contributors-update`) so Settings › About lists the current GitHub contributors, bump `version` in `apps/desktop/desktop.json` and add the matching `## <version>` section to `apps/desktop/CHANGELOG.md`, commit, and push `v<version>`. The [release workflow](../../.github/workflows/release.yml) validates the changelog heading, tag, `release.repository`, private/public key match, staging settings, required Apple configuration and the encoded IMK provisioning profile before building. Missing notes or partial credentials fail explicitly; actual certificate validity and notary access are checked on macOS.
3. All tags, including alpha previews, build and require macOS ARM64 and Windows x64 ([ADR 021](../decisions/021-apple-silicon-only-macos.md)). The selected matrix and asset verification use the same policy in `scripts/lib/release.mjs`; ordinary CI checks those two supported platforms. The release calls the [same CI workflow](../../.github/workflows/ci.yml) at the tagged commit: portable/repository/site gates and native tests on the selected platforms: macOS arm64 (`macos-26`, Xcode 26.4.1) and Windows x64 (`windows-2022`). Each runner stages identical resources, builds once, verifies its package, and verifies its update archive and any deltas. Ordinary PR/main CI needs no signing secrets and retains installer artifacts for preview testing.
   The `v0.1.0-alpha.17` tag has an explicit user-requested exception: the reusable workflow receives `skip-e2e: true` because native window and isolated updater E2E were already completed. Other tags and ordinary CI default to running them. Portable, desktop, keyboard, updater regression, signing, notarization and asset verification gates remain enabled.
4. Only after every job passes, download every selected platform artifact set and validate their installer, manifest version, center archive URL, byte size and Ed25519 signature. Write `SHA256SUMS` using download filenames (installer spaces become dots) and attach [RELEASE-STATUS.md](release-status.md). `release.mjs stage` then uploads the files flat under `files/`, the documents under `releases/<version>/`, and the staged record `releases/<version>/release.json` last ([bucket contract](../../packages/contracts/releases/README.md)); it refuses to replace an existing record and treats a failed lookup as a failure. The GitHub draft is still created as the public archive. No partial build set is uploaded; an interrupted staging can be rerun because the record is written last.
5. Review notes, checksums and platform acceptance, then publish the staged release to `beta` or `stable` in the AstraFoundry admin (**Admin › Releases › Arion**). Publication writes the channel manifests and index that the [center](../../deploy/center/README.md) serves within its cache time (default 30 s); withdrawal returns each target to the previous published release. Staged releases are never offered. Prerelease versions can only be published to `beta`. Tagged releases require both update and Apple credentials; ordinary CI packaging remains available without them. Only tagged macOS build steps receive Apple secrets; PR/main and Windows builds do not import them.

Builds through `0.1.0-alpha.5` contain the GitHub feed. Publish the first center-feed release on GitHub as well as on a channel, so those installs update once to a build that uses the center.

For Tencent COS, use the regional endpoint `https://cos.<region>.myqcloud.com`, the matching region (for example `ap-hongkong`), and the complete bucket name including its APPID suffix. Release CI configures virtual-hosted-style S3 addressing; COS rejects path-style requests with `PathStyleDomainForbidden`. Keep anonymous writes disabled. Use a dedicated API-only CAM user whose `PutObject`, `HeadObject` and `GetObject` permissions cover only `<prefix>/files/*` and `<prefix>/releases/*`; channel writes belong to the platform publisher. A missing staged record must return HTTP 404, rather than an authorization error. Validate upload/readiness and denial of channel writes before enabling CI credentials. Store the dedicated key only in Actions Secrets, independently of local TCCLI login credentials.

## Desktop release gates

1. Validate the compatibility matrix, model artifact checksums/licenses and privacy/latency gates on macOS and Windows.
2. Assign controlled bundle/package identifiers and secret signing identities outside the repository.
3. Package native inference libraries/tokenizers/approved weights consistently for each supported architecture; load verified assets from controlled paths.
4. Declare microphone and relevant entitlements/permission descriptions only for implemented features. Sign all executable hosts/frameworks.
5. macOS: sign/notarize app and IMK bundle, validate input-source registration and clean removal. Windows: sign app/TSF native host, validate architecture/registry registration and clean removal under actual user privilege boundaries.
6. Prove mode switching, host crashes, shutdown and update/uninstall clean owned composition and don't strand OS registrations.
7. Update trust (Ed25519-signed archives) and swap rollback are in place ([ADR 010](../decisions/010-in-app-updates.md)); model migration checks and Developer ID signing, which keeps privacy consent across updates, remain before broad distribution.

## Self-hosted center and sync relay

- **Complete center (`deploy/center`)**: [deploy/center](../../deploy/center/README.md) runs Caddy (automatic TLS for `ARION_CENTER_DOMAIN`), `services/updates` (`/v1/updates/*`, `/v1/releases/*`, `/v1/downloads/*`), `services/sync` (`/v1/sync/*`, `/v1/security-events`, `oidc` gate by default), and PostgreSQL 17 with one-shot migrations. Copy `deploy/center/.env.example` to `deploy/center/.env`, place the read-only S3/COS secret in `deploy/center/secrets/updates-s3-secret`, and run `docker compose up -d --build`.
- **Standalone ciphertext relay (`deploy/sync`)**: [deploy/sync](../../deploy/sync/README.md) builds only the `services/sync` relay and runs PostgreSQL 17 plus a one-shot migration. Put TLS on a reverse proxy in front of `127.0.0.1:8080`. `ARION_SYNC_ACCOUNT_GATE=noop` is the self-host default (device signatures only); `oidc` is the hosted gate for `api.arion.astrafoundry.si` and must not share the platform database. Kubernetes packaging under [deploy/k8s](../../deploy/k8s/README.md) is still unused.
- **Desktop build-time binding (`ARION_CENTER_*`)**: Configuring `deploy/sync/.env` or `deploy/center/.env` only configures the server containers and does not retarget an already-built desktop app. Packaged desktop apps ignore runtime `ARION_CENTER_*` environment variables; to point a desktop build at a self-hosted center or relay, set `ARION_CENTER_URL` and `ARION_CENTER_AUTH_MODE` (plus the five `ARION_CENTER_OIDC_*` variables when using `oidc`) in the root `.env` or shell **before** running `make build desktop` or `make desktop-dev` ([Center configuration](#center-configuration)).
- **Key and backup boundaries**: Keep private encryption keys on clients; operators back up ciphertext and routing metadata. A restore cannot invent plaintext or silently discard client conflicts. Revocation rotates the key epoch; clients re-encrypt, and previously downloaded plaintext cannot be remotely erased.

## Marketing site

`apps/site` is a static export served by Cloudflare Workers static assets at `arion.astrafoundry.si` ([ADR 016](../decisions/016-marketing-site.md)); `apps/site/wrangler.jsonc` has no Worker script and binds the custom domain.

```sh
make site-check   # dictionaries, types, lint, tests, static export
make site-deploy  # build apps/site/out, then wrangler deploy
```

`wrangler deploy` reads Cloudflare credentials (`wrangler login` or `CLOUDFLARE_API_TOKEN`) from the shell; none are stored in the repository. The site's origin and the AstraFoundry platform origin are build-time values in `apps/site/.env.production` ([site README](../../apps/site/README.md#configuration)); a different deployment overrides them in the environment and rebuilds.

## Native ASR migration

[ADR 020](../decisions/020-native-qwen-asr.md) defines the platform packages and open native-host gates. Apple silicon includes the Swift compatibility library closure collected by `swift-stdlib-tool`, colocated Metal shaders and SwiftPM resource bundles. The verifier rejects ONNX/sherpa/X-ASR runtime files in the final package. Runtime source and model provenance are pinned separately; model downloads remain outside the signed application.

Packages with ASR hotword support must include the rebuilt `arion-mlx-2` worker, which advertises the additive protocol-3 `prompt` capability. Rebuilding only the Go shell leaves nonempty hotword requests unsupported. The existing pinned Windows CPU/Vulkan workers expose the corresponding steering string `prompt` option in their model capability metadata; this change does not replace their pinned binaries. Runtime capability checks reject missing support before submitting contextual audio. The [Apple Silicon smoke archive](../../benchmarks/reproduction/asr-hotwords-v1/runtime-smoke/README.md) verifies native capability, per-request empty-context reuse and context-limit refusal; actual Windows inference and packaged-host acceptance remain separate gates.

### Windows speech typography

Source Han Sans SC 2.005R is bundled unchanged under SIL OFL 1.1. The original regular and medium OTF files, copyright/license and hash manifest are in apps/desktop/ui/shared/typography/fonts. Windows speech views embed both weights in Arion.exe; candidate_fonts.rc embeds the regular font in ArionTSF.dll. Production and native fixtures share scripts/lib/windows-candidate-fonts.mjs for resource compilation. The package verifier checks font bytes in both binaries and exact license/provenance copies under licenses. Font changes therefore update the normal protected TSF core payload and never register a system font. macOS retains its existing typography.
