# IME lifecycle implementation and local acceptance

## Implemented

- Stable source IDs and installed path; complete staged payload signature validation before stopping a host.
- Same-volume atomic bundle exchange, reverse-exchange rollback, resource identity checks and a persistent transaction journal. Ambiguous recovery preserves both copies for explicit repair.
- A per-user cross-process maintenance lock serializes installation, activation and removal.
- Removal uses a signed maintenance-only host entry, public TIS disablement and an independent fresh observer. Denied or unknown removal retains files. Parent-disabled/child-enabled is unavailable for selection.
- Startup detects a bundled update. Polling schedules automatic replacement only for an enabled, unselected source with no active dictation/maintenance. Selection is checked again after staging. Normal pending updates show status without an Update button; failures expose Retry update and stop repeated automatic attempts for this app run. Missing or disabled sources are not silently installed/enabled.

## Verified before automatic scheduling

The signed local lifecycle build passed real unselected and selected update, injected readiness failure with rollback, verified removal after System Settings removed the source, reinstallation and restored activation. Public disable requests from both the resource helper and installed host returned statuses that did not establish persistence: fresh observations remained available and focused system logs reported a preferences policy denial. Files were retained as intended.

The user confirmed physical keyboard input produced “你好” and repeated switching worked in the owned Arion E2E Editor. Automated synthetic letter events did not establish Chinese candidate input; they are not counted as a typing pass. No chat message was sent. A subsequent WeChat login was started, but a new end-to-end WeChat switching result has not been established for the automatic-update build.

## Automatic update verification

Focused tests cover waiting for an unused source, active dictation exclusion, single-flight execution, no automatic activation and stopping retries after failure. English and Simplified Chinese native UI tests cover pending automatic update without a manual action and explicit retry after failure. The final local package passed Developer ID signature, notarization, Gatekeeper, resource and architecture verification. The signed main app was installed with its previous bytes retained. Startup with an enabled, unselected source completed automatic replacement without clicking Update: the installed IME executable SHA-256 changed from `2aa1fb166e3efdfb74eb08a7ede35fba979691cba775191a3a6dd06eab72af2b` to the bundled `9cff1fc5f6a00f51a98710a66c57b6a4f6e0636c8fbc74f54ad6f06ba4cca07b`; the same run recorded `ime_install_started` followed by `ime_install_completed`. Selected-source automatic deferral is unit-tested; it has not been separately exercised in the final signed app.

WeChat remained alive through three synthetic source-shortcut attempts after the automatic update. The source probe stayed enabled/unselected (`state: 2`), so these attempts do not establish an actual switch to Arion and are not counted as a WeChat switching pass. No text or message was submitted. Native UI automation could not retrieve the menu-only Arion window after restart; the visible states were checked with the bilingual native UI tests instead.

## Limits

Old released hosts have no composition-idle handshake. A selected source therefore waits for normal switching rather than being restarted during typing. TIS observation is not an atomic reservation against a simultaneous system source switch; the post-staging check narrows this race but cannot prove its absence. A future cooperative host handoff needs separate protocol and old-version migration validation.

Atomic replacement removes the missing-directory window; it does not prove all application-local input-source caches refresh. Global service termination and other applications' cache deletion are not included. One-account verification is not universal cross-application or cross-macOS coverage.

Local execution evidence is under `.workspace/notes/ime-lifecycle-native/`, with automatic-update test/build logs under `.workspace/notes/ime-auto-update-*`. Private application caches and personal screen contents are not release fixtures.
