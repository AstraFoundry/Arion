# Distribution and deployment

## Current capability

`make desktop-build` packages `Arion.app` (menu bar app, `LSUIElement`), its icon, the microphone usage description, the audio-input entitlement and the speech runtime libraries in `Contents/Resources/lib`, signed ad hoc, and its disk image. `scripts/cc-relocatable.sh` links without the build machine's module-cache rpath, so `mygo build` signs the binary that ships; `scripts/verify-macos-app.mjs` checks rpaths and the signature without changing the app. Models are not bundled: the app downloads the bundle selected by evaluation from pinned URLs and verifies every file. There is no IME installer or sync service yet. The development identifier `local.arion.desktop` and ad hoc signing are not release-ready.

```sh
make desktop-build
```

## Releases and updates

The app updates itself from the latest published GitHub release of `AstraFoundry/Arion` ([ADR 010](../decisions/010-in-app-updates.md)). With `MYGO_UPDATER_PRIVATE_KEY` set, `mygo build` also writes the signed update archive and `update-<os>-<arch>.json` for the platform.

1. Once: store the secret key from `mygo keygen` in the password manager and as the repository secret (`gh secret set MYGO_UPDATER_PRIVATE_KEY --repo AstraFoundry/Arion < mygo-update.key`). Its public key is `updates.publicKey` in `apps/desktop/mygo.json`.
2. Bump `version` in `apps/desktop/mygo.json`, commit, and push the tag `v<version>`.
3. [Release](../../.github/workflows/release.yml) checks the tag against the version, creates a draft release and builds macOS arm64, macOS x86_64 and Windows x86_64 on their own runners, each uploading installers and signed updates to the draft.
4. Review the draft (every platform's manifest present, notes edited) and publish it. Installed apps see it on their next check; drafts are never offered.

## Desktop release gates

1. Validate the compatibility matrix, model artifact checksums/licenses and privacy/latency gates on macOS and Windows.
2. Assign controlled bundle/package identifiers and secret signing identities outside the repository.
3. Package native inference libraries/tokenizers/approved weights consistently for each supported architecture; load verified assets from controlled paths.
4. Declare microphone and relevant entitlements/permission descriptions only for implemented features. Sign all executable hosts/frameworks.
5. macOS: sign/notarize app and IMK bundle, validate input-source registration and clean removal. Windows: sign app/TSF native host, validate architecture/registry registration and clean removal under actual user privilege boundaries.
6. Prove mode switching, host crashes, shutdown and update/uninstall clean owned composition and don't strand OS registrations.
7. Update trust (Ed25519-signed archives) and swap rollback are in place ([ADR 010](../decisions/010-in-app-updates.md)); model migration checks and Developer ID signing, which keeps privacy consent across updates, remain before broad distribution.

## Self-hosted relay gates

Implement SYNC-01 before deploying anything. Specify server auth/enrollment, TLS/reverse proxy, ciphertext database migrations, size/rate limits, durable cursor/revision behavior, backup/restore and key-epoch tests. Keep private keys on clients; operators back up encrypted records/routing metadata. A restore cannot invent plaintext or silently discard client conflicts.

Docker/Kubernetes assets are reserved under [deploy](../../deploy/README.md); no image, host, ingress or database credentials are invented by this foundation.

## Marketing site

`apps/site` is a static export served by Cloudflare Workers static assets at `arion.astrafoundry.si` ([ADR 016](../decisions/016-marketing-site.md)); `apps/site/wrangler.jsonc` has no Worker script and binds the custom domain.

```sh
make site-check   # dictionaries, types, lint, tests, static export
make site-deploy  # build apps/site/out, then wrangler deploy
```

`wrangler deploy` reads Cloudflare credentials (`wrangler login` or `CLOUDFLARE_API_TOKEN`) from the shell; none are stored in the repository. The site's origin and the Astra Foundry platform origin are build-time values in `apps/site/.env.production` ([site README](../../apps/site/README.md#configuration)); a different deployment overrides them in the environment and rebuilds.
