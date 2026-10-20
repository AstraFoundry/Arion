# ADR 010: In-app updates from GitHub releases

Status: Accepted; signed macOS ARM64 alpha distribution is published. The **Source of truth** below is superseded by [ADR 019](019-center-update-channels.md): apps check the Arion center's `stable`/`beta` channels, not the GitHub API.

## Context

Users install Arion from a disk image or installer and had no way to learn about or apply a new version. Updates must keep the privacy promise of [005](../../specs/005-privacy.md): no content leaves the device, and every network surface is named and testable. The deployment gates require update trust and rollback before automatic updates ship.

## Decision

- **Source of truth.** Updates come only from the latest *published* release of [AstraFoundry/Arion](https://github.com/AstraFoundry/Arion/releases), tagged `v<version>`. Drafts are never offered. The independent **Allow beta updates** switch in Settings › About defaults to off and persists as `allowPrereleaseUpdates`; when on, published prereleases are eligible alongside stable releases. An empty channel is a normal no-release state. `desktop.json` `updates.github` names the repository.
- **Mechanism.** The MyGo updater (`mygo.Updater`), with the approved maintained [minimal channel patch](../../third_party/README.md), not a second implementation. Each platform's release carries `update-<os>-<arch>.json` and a `tar.gz` of the app as installed (plus binary deltas from earlier versions). The manifest gives size and an Ed25519 signature over the archive's SHA-256; the public key is built into the app from `desktop.json` `updates.publicKey`. An archive that fails size or signature is discarded.
- **Install.** The archive unpacks next to the app and replaces it by rename; the old bundle is renamed back if the swap fails. The running app is untouched until the user chooses Restart (`mygo.App.Relaunch`).
- **Channel changes.** Changing the switch clears pending results and cancels/invalidate checks from the old channel. Checks remain explicit when automatic checking is off. The switch is disabled during checking, download and pending restart. Complete linked semantic versions distinguish successive alpha builds from the numeric macOS bundle metadata.
- **Checks.** At launch and whenever 6 hours have passed since the last attempt (polled every 15 minutes, so a sleeping Mac catches up on wake). `Preferences.checkUpdates` defaults to **on** and is a switch in Settings › About. A failed automatic check is logged and not shown; a check the user asked for reports its failure. Offline (`Options.Offline`) and scripted E2E profiles never contact the feed. Builds that cannot replace themselves (development runs, read-only locations) show that and offer no check.
- **Disclosure.** Settings › About shows the state and the Privacy group names update checks as the second network use. The menu bar panel and tray menu show "Update to <version>" while an update is pending.
- **Signing key.** `mygo keygen` makes the pair; the secret key lives in the maintainers' password manager and the `MYGO_UPDATER_PRIVATE_KEY` repository secret, never in the repository (`mygo-update.key` is ignored).

## Consequences

- An update check lists public GitHub release metadata and downloads the selected platform manifest over HTTPS. An update request is an HTTPS GET to GitHub with the app name and version in its User-Agent; GitHub sees the IP address and timing. No audio, text, vocabulary, identifiers or preferences are sent.
- Losing the secret key strands installed apps on their version; a leaked key lets its holder ship code to users. Rotating it needs a release signed with the old key that carries the new public key.
- Tagged builds reuse the complete CI gates, verify every selected platform's installers and signed archives, and only then upload to a draft. Ordinary CI retains installers without a key; missing release keys fail before creating a draft.
- The build must produce the shipped binary in one `mygo build`, because the update archive is made there: `scripts/cc-relocatable.sh` drops the module-cache rpath at link time and `scripts/verify-macos-app.mjs` only checks the result.
- While releases are signed ad hoc, macOS keys privacy consent (Accessibility, Input Monitoring) to the exact build, so users grant it again after an update. Developer ID signing (REL-01) removes this.
- Rollback is limited to a failed swap; a published bad release is fixed by publishing a newer one.

## Alternatives considered

- **Notify and open the release page:** no install code, but every update is a manual download and re-grant of consent.
- **Own manifest or a GitHub API client:** duplicates the MyGo updater's signature, delta and swap logic.
- **Checks off by default:** strictest reading of the offline promise, but most users would never learn about fixes; the check carries no content and is disclosed with a switch.

## References

- [MyGo auto-updates](https://github.com/egoist/mygo) (`docs/updates.md`, `docs/github-actions.md` of the pinned module)
- [Release workflow](../../.github/workflows/release.yml), [deployment](../operations/deployment.md)
