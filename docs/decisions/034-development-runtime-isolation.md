# ADR 034: Isolated development sessions

Status: Accepted.

## Context

A locally packaged preview and the installed release share the production bundle identifier. OS reopen requests can select the installed copy. Development previously used ad hoc signing and stored application data under `Arion`, despite its separate `.dev` identifier. Rebuilds changed its signing requirement, and both variants could address the same preferences, credentials, and update transaction. An unavailable updater still attempted persisted recovery.

## Decision

- Daily use runs the installed release. macOS development starts explicitly through `make desktop-dev`, after quitting the running desktop. The launch preflight checks release and development executable paths across checkouts and refuses overlap. `make desktop-status` reports actual paths and hashes. `make desktop-production` verifies and opens the installed signed release after development stops.
- macOS development requires a persistent installed code-signing identity in `MACOS_DEV_SIGNING_IDENTITY`. Each build uses the resolved certificate fingerprint. macOS builds once with MyGo `dev -build-only`, then launches the bundle through LaunchServices. Development requires explicit quit and rebuild rather than direct-process live reload, so TCC attributes access to the app instead of its terminal or coding-agent parent. The entry point does not silently use ad hoc signing when this setting is absent.
- Development application state and OS credentials use `Arion Dev`; releases retain `Arion`. Existing production state is neither copied nor migrated. Explicit E2E data directories remain supported.
- Development does not create an installed-input-method adapter. It cannot take over the release's socket or alter installed keyboard preferences, learning, or vocabulary projection. Signed IME acceptance uses the release workflow.
- An unavailable update source has no recovery scheduler and does not load or modify persisted update state. Supported release recovery keeps its existing maintenance and signature checks.

The command preflight governs the supported switching workflow. It cannot stop an older installed release or an OS login item from launching independently during a development session. Quit unexpected copies and verify the running path before acceptance. Runtime exclusion across unrelated bundle identifiers is not provided by MyGo's single-instance lock.

## Verification

Bootstrap tests require unsupported builds to leave valid staged and malformed update records untouched. Credential tests require development operations to address a distinct service. Script tests reject missing, unavailable, or ambiguous signing identities and identify desktop processes without matching the input-method host or test fixtures.

Native acceptance must verify two edited rebuilds retain the intended signing requirement and OS permissions, actual shortcuts deliver processed text correctly, and returning to the release preserves its profile and update records. Compilation alone does not establish these properties. See [local development](../operations/local-dev.md#switch-between-the-release-and-development).
