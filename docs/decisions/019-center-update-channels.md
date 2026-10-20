# ADR 019: Update channels served by the Arion center

## Status

Accepted. Supersedes the **Source of truth** decision of [ADR 010](010-in-app-updates.md); its mechanism, signing key, install, checks and disclosure decisions stand.

## Context

The updater listed GitHub releases through the GitHub REST API on every check. Anonymous API requests share a limit of 60 per hour per address; behind a shared proxy egress the limit was exhausted, the API answered `403`, and **Check for updates** reported that it could not check. The same dependency also made GitHub the only download origin and gave maintainers no publication step separate from GitHub's own release state.

## Decision

- **Origin.** Apps check and download from the Arion center, `https://api.arion.astrafoundry.si` (`desktop.json` `updates.url` = `…/v1/updates`). The center's `services/updates` reads channel objects from an S3-compatible release bucket and redirects file downloads to short-lived presigned URLs. No client request reaches the GitHub API.
- **Channels.** `stable` and `beta`. MyGo's `url-channel-v1` patch sends `channel=stable`, or `channel=beta` when **Allow beta updates** is on; `404` is the existing normal no-release state. `beta` offers the newest published release of either kind; a prerelease never enters `stable`.
- **Staging and publication are separate.** Tagged CI verifies every installer, archive and delta signature, then stages them under `files/` and `releases/<version>/release.json` in the bucket and still creates the GitHub draft. Staging offers nothing. A staff member publishes a staged release to a channel, or withdraws one, in the AstraFoundry admin, which writes the channel manifests and index ([bucket contract](../../packages/contracts/releases/README.md)).
- **Trust is unchanged.** The manifest is served verbatim and the app verifies size and the Ed25519 signature of each archive or delta with its built-in key. The bucket, center and admin can withhold or delay an update but cannot make an app install unsigned code.

## Consequences

- An update check is an HTTPS GET to the center with the app name and version in its User-Agent and the channel in the query; the center and bucket see the IP address and timing. Nothing else is sent.
- Installed versions built with the GitHub feed (through `0.1.0-alpha.5`) keep checking GitHub. The first center-feed release must also be published on GitHub so those installs can move; later releases need only a channel publication.
- The center becomes an availability dependency for updates; a failed check is reported, never shown as an empty channel.
- Deltas are built against the `stable` channel manifest the center serves at build time.

## Alternatives considered

- **Authenticated GitHub API calls:** a token cannot be shipped in a public app.
- **A static bucket website as the feed:** no channel selection without two feeds, and the bucket would need public listing or public objects.
- **Publishing from CI:** removes the human acceptance step that release status requires.

## References

- [ADR 010](010-in-app-updates.md), [services/updates](../../services/updates/README.md), [deploy/center](../../deploy/center/README.md), [third_party](../../third_party/README.md)
