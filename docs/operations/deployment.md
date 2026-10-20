# Distribution and deployment

## Current capability

`make build desktop` packages `Arion.app` (menu bar app, `LSUIElement`), localized microphone descriptions, the audio-input entitlement, speech runtime libraries, the IMK host, and its disk image. Apple silicon also ships the pinned local-polishing worker and GenAI runtime. The macOS package minimum is 14.0, matching the highest minimum in the bundled native libraries.

`scripts/desktop-command.mjs` is the shared local/CI build entry point and sets the configured minimum OS in cgo compiler/linker flags. Its `scripts/stage-desktop-resources.mjs` preparation step includes Go module licenses, native runtime licenses/notices and model provenance. `scripts/cc-relocatable.sh` drops build-machine rpaths at link time. The read-only `scripts/verify-desktop-package.mjs` checks required resources, architectures, package identities/versions, native minimum OS versions and macOS signatures/rpaths. Models are downloaded separately from pinned URLs with verified digests.

The identifiers are `si.astrafoundry.arion` and `si.astrafoundry.inputmethod.arion`. macOS uses ad hoc signing without Apple configuration; complete configuration enables Developer ID signing and notarization through the same build command. Windows publisher signing remains open. The [release status](release-status.md) distinguishes packaged capabilities from actual host acceptance. The sync relay image and compose stack live in [deploy/sync](../../deploy/sync/README.md).

```sh
make build desktop
```

## Center configuration

`make build desktop` and `make desktop-dev` read the same public center profile from shell ENV or the root `.env`. Omit all `ARION_CENTER_*` variables for AstraFoundry defaults. For a device-authenticated self-hosted center:

```dotenv
ARION_CENTER_URL=https://sync.example.org/prefix
ARION_CENTER_AUTH_MODE=device
```

Custom `oidc` requires all five `ARION_CENTER_OIDC_*` fields in [`.env.example`](../../.env.example). The HTTP base URL and token resource are separate settings. Empty, unknown, conflicting and incomplete values fail; URLs must use HTTPS, except explicit loopback HTTP for development, and cannot contain credentials/query/fragment.

The app embeds the validated public profile and ignores launch-time center ENV. Package verification checks the executable's `--print-center-profile` result without center ENV. Custom profiles isolate account refresh tokens and disable the official update feed; device mode offers no account sign-in, and custom OIDC never calls the official waitlist API. Official tagged releases require the exact standard profile. Existing legacy account tokens are retained but no longer read: sign in again. Local vocabulary is unchanged. See the [profile contract and remaining sync limits](../project/central-sync-design.md#implemented-profile-boundaries).

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

Public package metadata lives in `apps/desktop/desktop.json`; `mygo.config.ts` adds environment-driven signing through the pinned MyGo API without rewriting source files. MyGo signs nested executables/libraries and the IMK bundle before the outer app, enables hardened runtime and secure timestamps, signs and notarizes the DMG, and staples tickets to both DMG and App **before** producing Ed25519-signed updates. The verifier checks code identities, runtime flags, timestamps, App/DMG tickets and Gatekeeper acceptance; with updates enabled it also verifies the final archive signature and the extracted App's signature/ticket. No signed archive is modified afterward.

The current profile-bearing macOS arm64 canonical build completes Developer ID signing, Accepted notarization, embedded authorization checks and Gatekeeper validation. A retained build before this profile gate also passed packaging trust checks. Its installed nested IMK host also passed Gatekeeper, but source selection still failed; packaging trust does not establish IMK acceptance. Clean-machine microphone, IMK, inference and published-update acceptance remain separate gates. See [release status](release-status.md).

## Releases and updates

The app updates itself from the latest published GitHub release of `AstraFoundry/Arion` ([ADR 010](../decisions/010-in-app-updates.md)). With `MYGO_UPDATER_PRIVATE_KEY` set, `mygo build` also writes the signed update archive and `update-<os>-<arch>.json` for the platform.

1. Once: configure the repository secrets above, including the matching Ed25519 secret key as `MYGO_UPDATER_PRIVATE_KEY`, and keep a backup in the password manager. It is independent of Apple Developer certificates. If the original update private key is lost before the first release, deliberately replace the key pair and the configured public key before shipping; after shipping, rotation needs the old signing key. Do not commit keys.
2. Bump `version` in `apps/desktop/desktop.json`, commit, and push `v<version>`. The [release workflow](../../.github/workflows/release.yml) validates the tag, repository, private/public key match and required Apple configuration and the encoded IMK provisioning profile before building or creating a draft. Missing or partial credentials fail explicitly; actual certificate validity and notary access are checked on macOS.
3. Alpha tags (`v<version>-alpha` or `v<version>-alpha.<number>`) build and require only macOS ARM64. Stable and other prerelease tags require all three platforms. The selected matrix and asset verification use the same policy in `scripts/lib/release.mjs`; ordinary CI still checks all three platforms. The release calls the [same CI workflow](../../.github/workflows/ci.yml) at the tagged commit: portable/repository/site gates and native tests on the selected platforms: macOS arm64 (`macos-15`), macOS Intel (`macos-15-intel`) and Windows x64 (`windows-2022`). Each runner stages identical resources, builds once, verifies its package, and verifies its update archive and any deltas. Ordinary PR/main CI needs no signing secrets and retains installer artifacts for preview testing.
4. Only after every job passes, download every selected platform artifact set and validate their installer, manifest version, archive URL, byte size and Ed25519 signature. Write `SHA256SUMS` and attach [RELEASE-STATUS.md](release-status.md). Then create/update the draft and upload the verified set. Existing published releases cannot be overwritten. No partial build set is uploaded; an interrupted upload can be retried against its draft.
5. Review notes, checksums and platform acceptance before publishing the draft. Drafts and prereleases are never offered to installed apps. Tagged releases require both update and Apple credentials; ordinary CI packaging remains available without them. Only tagged macOS build steps receive Apple secrets; PR/main and Windows builds do not import them.

## Desktop release gates

1. Validate the compatibility matrix, model artifact checksums/licenses and privacy/latency gates on macOS and Windows.
2. Assign controlled bundle/package identifiers and secret signing identities outside the repository.
3. Package native inference libraries/tokenizers/approved weights consistently for each supported architecture; load verified assets from controlled paths.
4. Declare microphone and relevant entitlements/permission descriptions only for implemented features. Sign all executable hosts/frameworks.
5. macOS: sign/notarize app and IMK bundle, validate input-source registration and clean removal. Windows: sign app/TSF native host, validate architecture/registry registration and clean removal under actual user privilege boundaries.
6. Prove mode switching, host crashes, shutdown and update/uninstall clean owned composition and don't strand OS registrations.
7. Update trust (Ed25519-signed archives) and swap rollback are in place ([ADR 010](../decisions/010-in-app-updates.md)); model migration checks and Developer ID signing, which keeps privacy consent across updates, remain before broad distribution.

## Self-hosted relay

[deploy/sync](../../deploy/sync/README.md) builds the relay and runs PostgreSQL 17 plus a one-shot migration. Put TLS on a reverse proxy in front of `127.0.0.1:8080`. Keep private keys on clients; operators back up ciphertext and routing metadata. A restore cannot invent plaintext or silently discard client conflicts. Revocation rotates the key epoch; clients re-encrypt, and previously downloaded plaintext cannot be remotely erased.

`noop` is the self-host default. `oidc` is the hosted gate for `api.arion.astrafoundry.si` and must not share the platform database. Kubernetes packaging under [deploy/k8s](../../deploy/k8s/README.md) is still unused.

## Marketing site

`apps/site` is a static export served by Cloudflare Workers static assets at `arion.astrafoundry.si` ([ADR 016](../decisions/016-marketing-site.md)); `apps/site/wrangler.jsonc` has no Worker script and binds the custom domain.

```sh
make site-check   # dictionaries, types, lint, tests, static export
make site-deploy  # build apps/site/out, then wrangler deploy
```

`wrangler deploy` reads Cloudflare credentials (`wrangler login` or `CLOUDFLARE_API_TOKEN`) from the shell; none are stored in the repository. The site's origin and the AstraFoundry platform origin are build-time values in `apps/site/.env.production` ([site README](../../apps/site/README.md#configuration)); a different deployment overrides them in the environment and rebuilds.
