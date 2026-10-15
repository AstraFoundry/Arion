# Distribution and deployment

## Current capability

`make desktop-build` packages `Arion.app` (menu bar app, `LSUIElement`), localized microphone descriptions, the audio-input entitlement, speech runtime libraries, the IMK host, and its disk image. Apple silicon also ships the pinned local-polishing worker and GenAI runtime. The macOS package minimum is 14.0, matching the highest minimum in the bundled native libraries.

`scripts/desktop-command.mjs` is the shared local/CI build entry point and sets the configured minimum OS in cgo compiler/linker flags. Its `scripts/stage-desktop-resources.mjs` preparation step includes Go module licenses, native runtime licenses/notices and model provenance. `scripts/cc-relocatable.sh` drops build-machine rpaths at link time. The read-only `scripts/verify-desktop-package.mjs` checks required resources, architectures, package identities/versions, native minimum OS versions and macOS signatures/rpaths. Models are downloaded separately from pinned URLs with verified digests.

The identifiers are `si.astrafoundry.arion` and `si.astrafoundry.arion.inputmethod`. macOS still uses ad hoc signing; Developer ID/notarization and Windows publisher signing remain credential-dependent work. The [release status](release-status.md) distinguishes packaged capabilities from actual host acceptance. The sync relay image and compose stack live in [deploy/sync](../../deploy/sync/README.md).

```sh
make desktop-build
```

## Releases and updates

The app updates itself from the latest published GitHub release of `AstraFoundry/Arion` ([ADR 010](../decisions/010-in-app-updates.md)). With `MYGO_UPDATER_PRIVATE_KEY` set, `mygo build` also writes the signed update archive and `update-<os>-<arch>.json` for the platform.

1. Once: store the matching Ed25519 secret key in the password manager and as `MYGO_UPDATER_PRIVATE_KEY` in repository secrets. It is independent of Apple Developer certificates. If the original private key is lost before the first release, deliberately replace the key pair and the configured public key before shipping; after shipping, rotation needs the old signing key. Do not commit keys.
2. Bump `version` in `apps/desktop/mygo.json`, commit, and push `v<version>`. The [release workflow](../../.github/workflows/release.yml) validates the tag, configured repository and private/public key match before building or creating a draft. A missing key fails explicitly.
3. The release calls the [same CI workflow](../../.github/workflows/ci.yml) at the tagged commit: portable/repository/site gates and native tests on macOS arm64 (`macos-15`), macOS Intel (`macos-15-intel`) and Windows x64 (`windows-2022`). Each runner stages identical resources, builds once, verifies its package, and verifies its update archive and any deltas. Ordinary PR/main CI needs no signing secrets and retains installer artifacts for preview testing.
4. Only after every job passes, download all three artifact sets and validate their installer, manifest version, archive URL, byte size and Ed25519 signature. Write `SHA256SUMS` and attach [RELEASE-STATUS.md](release-status.md). Then create/update the draft and upload the verified set. Existing published releases cannot be overwritten. No partial build set is uploaded; an interrupted upload can be retried against its draft.
5. Review notes, checksums and platform acceptance before publishing the draft. Drafts and prereleases are never offered to installed apps. Tagged release builds remain blocked until the update key is provisioned; ordinary CI packaging is available without it.

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
