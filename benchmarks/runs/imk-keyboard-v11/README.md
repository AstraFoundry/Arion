# Input-method consent and payload follow-up

The user supplied an Arion Dev consent prompt screenshot and reported choosing Allow. This is user-reported action, not evidence that native authorization persisted. The retained original source query shows Chatterfly current, the Arion parent disabled and unselected, the Pinyin child enabled and unselected, and no learning key. Public signature output shows that repeated development installation replaced the previously staged Developer ID host with an ad hoc host. Neither observation establishes the cause of activation failure.

`observed/` preserves the original five allowlisted records. `observations.json` is explicitly derived; the source query, source metadata, signature output and bounded installation events remain separate. Screenshots, application payloads, provisioning profiles, environment files, private control receipts, certificates and key material are excluded. There are no new system mutations in archiving or verification.

The later fix rejects invalid Developer ID payloads and inconsistent Data Protection identities before replacing installed bytes, retains copied-but-undiscovered bundles as pending, removes immediate automatic settings navigation after native consent, and records observed source-state changes. The preflight verifies signature and entitlement consistency; it does not establish a pinned publisher team or independently validate profile CMS authorization. Corrected source snapshots describe the post-observation fix; they are not represented as the exact producer of the earlier installation events.

The completed new formal build passed signing, notarization, ticket, Gatekeeper and package checks. Its actual bundled input host passed the read-only `TestInputBundleConfiguredDeveloperIDPayload` test. Public artifact digests and both original completed logs are retained. The updater archive signature was not configured. Neither build validation nor payload preflight invokes source installation or proves native consent. Other focused checks reported by the parent have no saved raw logs and are recorded only as a derived provenance note.

Actual native grant persistence, discovery in the add-source dialog, selection and installed host typing remain unaccepted. Previous controls are preserved in [v10](../imk-keyboard-v10/README.md).

Run `node benchmarks/runs/imk-keyboard-v11/verify.mjs` to verify archived bytes and bounded assertions without OS, input-source, signing, Keychain or UI operations.
