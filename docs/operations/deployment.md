# Distribution and deployment

## Current capability

`make desktop-build` packages `Arion.app` (menu bar app, `LSUIElement`), its icon, the microphone usage description, the audio-input entitlement and the speech runtime libraries in `Contents/Resources/lib`, signed ad hoc. Models are not bundled: the app downloads the bundle selected by evaluation from pinned URLs and verifies every file. There is no IME installer, updater or sync service yet. The development identifier `local.arion.desktop` and ad hoc signing are not release-ready.

```sh
make desktop-build
```

## Desktop release gates

1. Validate the compatibility matrix, model artifact checksums/licenses and privacy/latency gates on macOS and Windows.
2. Assign controlled bundle/package identifiers and secret signing identities outside the repository.
3. Package native inference libraries/tokenizers/approved weights consistently for each supported architecture; load verified assets from controlled paths.
4. Declare microphone and relevant entitlements/permission descriptions only for implemented features. Sign all executable hosts/frameworks.
5. macOS: sign/notarize app and IMK bundle, validate input-source registration and clean removal. Windows: sign app/TSF native host, validate architecture/registry registration and clean removal under actual user privilege boundaries.
6. Prove mode switching, host crashes, shutdown and update/uninstall clean owned composition and don't strand OS registrations.
7. Provide update trust, rollback and model migration checks before shipping automatic updates.

## Self-hosted relay gates

Implement SYNC-01 before deploying anything. Specify server auth/enrollment, TLS/reverse proxy, ciphertext database migrations, size/rate limits, durable cursor/revision behavior, backup/restore and key-epoch tests. Keep private keys on clients; operators back up encrypted records/routing metadata. A restore cannot invent plaintext or silently discard client conflicts.

Docker/Kubernetes assets are reserved under [deploy](../../deploy/README.md); no image, host, ingress or database credentials are invented by this foundation.
