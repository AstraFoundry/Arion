# Desktop release status

This file accompanies each release as `RELEASE-STATUS.md`. Build success verifies packaging; the evidence below limits the supported product claims.

| Capability | macOS Apple silicon | macOS Intel | Windows x64 |
| --- | --- | --- | --- |
| Native package | App and DMG | App and DMG | Per-user NSIS installer |
| Local speech recognition | Implemented; fixture replay measured | Runtime packaged; host acceptance pending | Runtime packaged; complete packaged dictation acceptance pending |
| Local polishing | Worker and pinned runtime bundled | Unsupported; no approved runtime/model pair | Unsupported; no approved runtime/model pair |
| System input method | IMK host bundled; install/select/remove and host edits need acceptance | Same IMK host; host acceptance pending | TSF not implemented |
| Encrypted sync | Desktop enrollment and outbox pending | Desktop enrollment and outbox pending | Desktop enrollment and outbox pending |

Windows is an early preview. Polishing on unsupported platforms reports its unavailable state; no content is routed to cloud inference. Model weights are downloaded separately from the pinned catalog and are not embedded in installers.

The macOS package requires macOS 14 or later. Its bundled polishing ONNX Runtime has a Mach-O minimum of 14.0; package verification rejects any executable whose minimum exceeds the advertised package minimum. Minimum-version host acceptance and minimum RAM measurements remain open.

## Signing and update trust

The package identifiers are `si.astrafoundry.arion` and `si.astrafoundry.arion.inputmethod`; the IMK input-source ID adds `.voice`. Changing from development identifiers requires a fresh permission grant and removal of the old input source through the older build before testing the new one. Local user data paths have not been migrated.

macOS packages currently use ad hoc signing. Developer ID signing, Apple notarization and Windows publisher signing are pending credentials and integration; these packages are for preview testing. Updating an ad hoc build can require granting privacy permissions again. A valid ad hoc signature does not establish Gatekeeper approval.

Tagged release jobs require a matching Ed25519 update key and verify every platform's archive size and signature before upload. That key is independent of Apple certificates. Without it, ordinary CI still produces installer artifacts; the release job fails before creating a draft. Drafts and prereleases are not offered by the app's updater. Publishing remains a manual decision after acceptance.

## Acceptance before broad distribution

- Run real microphone capture, recognition and insertion on clean macOS and Windows systems, including offline use after provisioning, cancellation, focus changes and device loss.
- Exercise input-source install/select/remove and composition cleanup in owned host applications before claiming IME support.
- Verify install, uninstall and an old-to-new published update on each supported platform; check preferences, vocabulary, credentials and consent, plus a failed replacement.
- Validate minimum OS/RAM and the supported editor compatibility matrix on actual hosts. Native shell E2E and compile checks do not replace these runs.
- Complete held-out meaning/style review before making polishing quality claims. The model catalog preserves conversion and evaluation limitations; shipping its provenance does not resolve them.

Detailed evidence and remaining limits are in [validation](../quality/validation.md) and [desktop release gates](deployment.md#desktop-release-gates).
