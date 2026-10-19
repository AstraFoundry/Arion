# Keyboard callback and signed-source supplement

This immutable supplement adds **22 / 22** completed real-engine/AppKit cases and the completed **failed/restored** signed system-source attempt. The [v1 archive](../imk-keyboard-v1/README.md) retains attempt 11, all earlier failures and system runs 03/04 unchanged.

Verify the supplement without a runtime, network access, Keychain access or OS mutation:

```sh
node benchmarks/runs/imk-keyboard-v2/verify.mjs
```

`archive.json` maps every archived file to original path, size and SHA-256. Attempt 12's 13 recorded source/runtime identities were checked against the immediately preserved source originals. Twelve source/schema/manifest records are copied; the native dylib is excluded with its exact digest retained. Runtime binaries, SDKs, user profiles, real keys and unfinished final-build execution remain outside this archive.

## Actual engine and AppKit evidence

`engine/typing-attempt-12.json` and `.log` preserve the completed 22 cases. All previous 20 gates continue to pass. The added destruction case uses the actual Rime API to establish that releasing a typing owner removes its real engine session while leaving client mark/insert and UI-callback counts unchanged.

The added callback-failure case exercises separate mouse-selection and mouse-highlight operations against actual shared encrypted userdb transaction failures. A regular file obstructs the isolated vault directory, causing real filesystem persistence errors. The production adapter retains pending text for recovery, revokes candidate data/revision and rejects both stale and current callbacks after becoming unavailable. No fake engine, error injection return or generated conversion is used.

The same production encrypted backend uses an isolated synthetic test key; no real Keychain or user profile is accessed. This is actual librime 1.17.0 plus an AppKit `NSTextView` protocol adapter. It does not establish installed IMK dispatch, external-editor acceptance, latency or general dictionary quality.

For a new functional run, copy `sources/` into a fresh `.workspace/` directory, then use its `scripts/stage-keyboard-runtime.mjs` and `scripts/test-keyboard-input.mjs` as described in [v1 reproduction](../imk-keyboard-v1/README.md#reproduce-the-final-functional-harness). Supporting provisioning scripts/profile were preserved from v1; the runtime/data identity is unchanged. Never provision runtime or SDK files inside this archive.

## Signed system attempt remains failed

`system/signed-desktop-build.log` is the completed original build output, reporting Developer ID signing, notarization, notarization-ticket verification and Gatekeeper acceptance. It was reviewed to exclude credentials; the public service URL/profile digest, build paths and signing configuration variable names are technical provenance, not secrets. No application or DMG binary is archived.

`system/run-05/observations.json` explicitly labels the production installation and Gatekeeper summaries as **transcriptions of actual tool/controller observations**. These are not raw API response captures. Original raw standalone helper outputs are separately preserved:

- `diagnostic-registration.txt`: LaunchServices/TIS registration returns 0, parent remains disabled and child enabled.
- `diagnostic-transition.txt`: explicit disable/enable calls return 0, but requery still shows the parent disabled.
- `select-status.txt`: selection returns **-50**.
- `before-restoration.json`: ABC remains current; only the own Arion child is enabled.
- `state.json`: restoration completed without errors, original own-bundle byte inventory restored, ABC selected, own sources disabled and test learning key absent.

The exact `controller-05.mjs`, `sources.swift`, `diagnose-complete.swift` and `diagnose-transition.swift` driver sources are retained. Their historical paths and installation side effects mean they must not be treated as an automatic replay command. No personal preference files, unrelated input-source inventories, key bytes or private data are included.

A signed/notarized artifact and Gatekeeper acceptance did not resolve the observed TIS parent-enable failure. Actual typing dispatch was never reached. The unfinished subsequent final build is excluded, and the overall OS acceptance remains open.
