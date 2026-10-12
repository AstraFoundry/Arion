# Distribution and deployment

## Current capability

The native shell can be built with the pinned MyGo CLI. No microphone, model, IME installer, updater or sync service is implemented. A successful shell build is not a releasable input method.

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
