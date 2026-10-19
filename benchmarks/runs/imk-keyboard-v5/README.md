# Strict key policy and GUI registration evidence

This immutable supplement preserves the failed Data Protection capability probe, two completed legacy-item rejection runs, native and translated x86_64 synthetic-key regressions, and the failed/restored GUI caller diagnostic. The [pre-fix lifecycle evidence](../imk-keyboard-v4/README.md) remains unchanged. **A successful full Data Protection Keychain lifecycle and installed IMK typing dispatch remain unverified.**

```sh
node benchmarks/runs/imk-keyboard-v5/verify.mjs
```

Verification reads frozen files only. It checks every size/SHA-256 mapping, executed source identities, actual failures, policy boundaries and restoration. It performs no network, runtime, Keychain, signing, input-source or UI operation. Historical drivers are evidence, not authorized replay commands. No binaries, provisioning profiles, keys, persistent-reference bytes, ownership receipts, ciphertext/profile directories or personal settings are included.

## Actual key policy results

`dp-capability/` retains run `666720cc-0cc0-4f93-b102-24e41819aa0c`: an explicit Data Protection add returned **-34018** under the existing unprofiled Developer ID signature. The capability phase and overall run failed; guard/cleanup passed and key absence was restored. This is not a successful policy-backed lifecycle. Original failure logs and the exact executed harness/production source revision are retained.

`legacy-rejection/` retains the initial strict-provider test `50c25cf1-37b9-4100-8e41-dbc5ace5fe40`. A synthetic owned legacy item was actually created; production registration rejected it with migration-required error **11**, created no Data Protection item and wrote no vault bytes. Cleanup deleted the owned item and restored absence. This verifies rejection, not implicit migration or full Data Protection provisioning.

`app-legacy-rejection/` retains the later app-like signed run `14cdf21e-8493-4133-97be-05c364449e3b`, with backend-specific ownership/deletion handling and a distinct exact-source snapshot. Legacy rejection again passed; explicit legacy, Data Protection and default final queries all reported absence. The manifest records `legacyOnly = true` and `provisioning = null`. Additional source identities bind the runner, build/provisioning/signing helpers, learning tests and privacy/specification documentation; documentation bytes use `.md.txt` names to preserve their original content without treating archive-local links as active documentation.

The strict provider explicitly selects Data Protection, checks required attributes and rejects existing legacy items. A full policy-backed preparation now requires a profile authorized for the IMK host application identifier. The archived missing-profile rejection is explicit; the full Data Protection lifecycle was not rerun with an authorizing profile. Preparation 05 also retains the earlier overly broad signing-parser rejection, separately from successful legacy-only preparation 06. Later sources do not masquerade as the earlier parser failure's exact revision.

The policy snapshots cite Apple's [TN3137](https://developer.apple.com/documentation/technotes/tn3137-on-mac-keychains) and [Keychain access groups](https://developer.apple.com/documentation/security/sharing-access-to-keychain-items-among-a-collection-of-apps). They explain why a Developer ID certificate alone does not establish entitlement authorization. The archive contains no actual profile or entitlement credential asset.

## Synthetic-key functional checks

The completed strict-provider native arm64 and x86_64-under-Rosetta reports each pass **22/22** actual librime/AppKit cases. They use isolated injected synthetic keys and exercise real encrypted storage, restart learning, cancellation, callback failure, commit preservation and cleanup. They do not invoke the production OS-kept-key creation path. The x86_64 executable ran through Rosetta 2 on an arm64 host; no physical Intel claim is made.

`functional/source-identities.json` binds 13 runtime/source identities captured after the strict native run, before engine/typing source changes. The same production revision served the subsequent translated run. Exact source/manifests/schema bytes are retained; the native runtime binary is excluded with its digest. Original report copies were preserved before later test runs could overwrite the default report. The separate synthetic learning log passed, but that log alone is not evidence of successful real Data Protection provisioning.

## Actual GUI caller, failed activation

Run 11 used an actual `NSApplication` button action on the main thread. Raw `gui-registration.json` reports LS registration, TIS registration and enable statuses `0`, but the third-party parent remained disabled and the child enabled; no installed IMK event dispatch occurred. This narrows the caller-context hypothesis without identifying the activation cause. This diagnostic is not the production install button.

`ui-observations.json` is explicitly **transcribed actual CUA actions/AX text**, not an original screenshot: button element 1 was clicked, the result label appeared, and closing element 3 returned a timeout. Do not infer successful termination from that timeout alone. The separate original `final-process-query.json` records `pgrep` exit 1 with empty output.

The original controller `state.json` records restoration to ABC, absent key, disabled cached validation entries and no errors. The separately executed later `final-source-query.json` records ABC, absent key and no own entries. Both originals are preserved; the later query does not rewrite the earlier cached state. No system preferences, unrelated sources or application/profile binaries are retained.

To investigate reproduction, copy the relevant exact revision into a new `.workspace/` checkout, stage separately pinned dependencies and obtain the authorization/assets required for any OS-kept-key or source operation. Never replay mutating archive sources in place or provision environments under this archive.
