# Official Apple IMK source registration control

This immutable supplement preserves an independently sourced **Apple NumberInput stage-0 IMK control** rebuilt for the current SDK and arm64, signed/notarized, registered and restored. LS registration, TIS registration and enable returned `0`, but its parent source remained disabled. **No source selection, control-host launch or typing was attempted.** This is an activation control, not Arion typing acceptance.

```sh
node benchmarks/runs/imk-keyboard-v6/verify.mjs
```

The verifier reads frozen evidence only: sizes/hashes, original sample-member provenance, the exact connection-name substitution, failed preparation, signed-control result and restoration. It performs no network, compiler, signing, Keychain, input-source or UI operation. Earlier [v4](../imk-keyboard-v4/README.md) and [v5](../imk-keyboard-v5/README.md) remain unchanged.

## Source and license

`sample/` retains original [Apple NumberInput sample](https://developer.apple.com/library/archive/samplecode/NumberInput_IMKit_Sample/NumberInput_IMKit_Sample.zip) source/header/metadata, localization, text NIB metadata and `ReadMe`, with the Apple license notices intact in source headers. The fixed sample-carrier SHA-256 is `06b6f0052b7bb5bc584fb6d32a9d6fda4bba1746d29f56f00123466b60729228`. `system/artifact.json` maps every compiled/resource input's original member hash. Binary NIB/TIFF resources and ZIP/app/executable carriers are excluded with hashes retained.

The stage-0 controller deliberately returns `NO`, passing input to the client; it is not a conversion engine. The derived main changes only the unique connection name. Preparation also changes bundle ID/name, matching plist connection, minimum OS and marketing version. It uses current `xcrun clang`, arm64/macOS 14 target, Cocoa/InputMethodKit, Developer ID signing and notarization. The controller code is original. `sources/` retains the preparation/controller/TIS-query source and captured signing dependencies; executable and generated bundle metadata hashes remain in the artifact record. The verifier does not rebuild excluded resources or execute any diagnostic.

The original `ReadMe` describes an **initial-install logout** before input-source selection. That is historical official sample documentation, not evidence that logout is the necessary or sufficient modern fix. No logout, preference rewriting, agent termination or security-setting change was performed.

## Failed preparation, then completed control

`logs/number-control-preparation.log` preserves the initial run-12 failure: a temporary generic signing-parser rule required an IMK provisioning profile before compilation/signing. The subsequent fixed parser allowed independent control preparation; `number-control-preparation-02.log` records completion. The earlier failed source revision is not snapshotted, so the final preparation/dependency sources must not be attributed to that failure's exact bytes.

The run-13 control has a separate source ID `si.astrafoundry.inputmethod.numbercontrol` and connection. Notarization returned **Accepted**; Gatekeeper reported Notarized Developer ID. Despite LS/TIS/enable statuses `0`, the original registration output records the parent **disabled/unselected**. No host was launched and no selection or typing attempt followed.

The controller's original state records restoration with ABC current, absent learning key, no errors and a disabled cached control entry. Separate later actual queries record no control or Arion entries, ABC current and the learning key absent. The owned control bundle was removed. These are restored diagnostics, not successful activation.

The [first-party Korean control](../imk-keyboard-v4/README.md) does enable/disable, while this independently sourced third-party IMK control also fails enablement. The public TIS API is not universally inert, and an Arion-specific root cause remains unproven. The [strict key-policy evidence](../imk-keyboard-v5/README.md) is a separate gate: real Data Protection provisioning still needs its authorizing profile and a completed policy-backed lifecycle.

## Final strict-provider controls

`final-controls/` retains the completed **86/86** tooling-test output, the actual latest strict-provider universal unsigned-host build and its `unsealed-development` bundle verification. A separate signed-build gate records the expected missing-profile rejection before compilation. No new strict-provider signed artifact is claimed; the earlier notarized Arion artifact predates the strict key provider.

The manifest binds 30 focused native/build/signing/provisioning/configuration/test sources. Unchanged bytes reference earlier immutable keyboard supplements by path and hash; new bytes are in `final-sources/`. Verification checks these references without the original workspace. This avoids duplicating unchanged source modules. The full 86-test output is retained, but unrelated evaluation test modules are not included as a complete suite-reproduction snapshot. The focused signing/provisioning test sources and build dependencies are bound; runtime/SDK acquisition remains separate.

No application binaries, resource binaries, provisioning assets, runtime/SDK, profiles, credentials, keys, persistent references or personal preferences are archived. Historical sources are not an authorized replay command; review assets and OS-mutation authorization before any reproduction in a new `.workspace/` checkout.
