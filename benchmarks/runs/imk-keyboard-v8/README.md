# Profile-authorized Data Protection lifecycle

This immutable supplement preserves the public provisioning-validation metadata, failed preparation, two original real-key lifecycle runs, exact source identities, **87/87** tooling checks and the completed profile-authorized signed/notarized desktop build. The final six-phase run verifies the observed Data Protection Keychain backend and attributes, then restores key absence. It does not establish installed IMK event dispatch, hardware protection behavior, backup restoration or cross-device migration.

```sh
node benchmarks/runs/imk-keyboard-v8/verify.mjs
```

The verifier reads frozen reports/sources and referenced earlier supplements only. It performs no network, account operation, signing, native execution, Keychain or input-source operation. Profile files, `.env`, private backups, credentials, account screenshots, signing assets, application/runtime binaries, key/reference bytes, ownership receipts and ciphertext/profile directories are excluded. Public profile UUID, authorization, digest and local path metadata remain in the original report; the asset itself was not read for archiving.

## Authorized asset and preparation failures

The IMK host App ID was registered and its Developer ID profile created using the existing certificate. `controls/profile-validation.json` records validated application/team authorization and a profile digest; local configuration was completed outside this archive. No Apple Developer account screenshot or private account details are copied.

Run `2b30897a-cb61-4dee-af03-d428e3b1aac5` failed preparation before execution because certificate extraction used separate `codesign --extract-certificates` arguments. Its original manifest retains `prepared = false`, `executed = false`; the exact original error log remains. The final helper uses the equals-prefix form and reads native `plutil` Boolean scalars with an explicit expected Boolean type. An actual native-parser regression is included in the tooling tests.

The earlier native CMS/parser failure existed only as tool output, not a retained original log, and is not recreated here. Intermediate helper bytes for that failure are not available; final helper bytes are not attributed to the failed revision. Exact executed native/harness/runner identities for both lifecycle runs do match frozen originals, including earlier immutable source snapshots.

## First lifecycle and reporter-format limit

Run `a9e75cb0-f7b7-427c-ba4d-b5f73d8aac24` actually passed guard, writer, reader, delete, missing-key and cleanup, and restored absence. Its original assurance remains **`not-established`**. Raw OS results show the owned Data Protection item, absent legacy item, accessibility `cku` and Synchronizable NSNumber `0`. The runner's strict comparison against JSON `false` did not recognize that numeric report value. The archive does not rewrite this original result.

Its six production/harness source identities and runner digest match exact bytes in earlier supplements. Those hash-bound originals are referenced rather than reconstructed from the later normalized reporter.

## Final verified lifecycle

Run `5a63b57a-a11b-4a5c-be69-f67721ac7a29` uses the same production key backend, with the test reporter preserving **`synchronizableRaw = 0`** alongside normalized **`synchronizable = false`**. All six phases pass, restoration passes and the original assurance is **`verified`**. OS queries establish a matching owned Data Protection item, no legacy item, `AfterFirstUnlockThisDeviceOnly` accessibility (`cku`) and present non-synchronizable state.

Actual `shishi` selection commits `事实`, and a fresh process promotes it ahead of `实施`. Deleting the exact owned key makes actual engine startup fail closed with ciphertext inventory unchanged. Final default, legacy and Data Protection queries all report absence; interaction policy is restored. Ciphertext contents, key data and persistent-reference bytes are excluded. These observed attributes/lifecycle results do not simulate device theft, backup restoration or migration.

## Completed signed package and limits

The original completed build log records Developer ID signing, DMG notarization, hardened runtime, notarization tickets, Gatekeeper acceptance and package resources/architectures, including the profile-authorized nested IMK host. It contains public signing/configuration provenance and variable names, with no credential values. The final tooling suite passes **87/87**. Exact focused helper/harness/test/build source bindings are retained; unchanged sources reference earlier immutable supplements.

This build supersedes the earlier profile-blocked build gate. It does not activate/register/select the source or prove external-editor typing. The existing 23-case native/Rosetta synthetic-key evidence and the 600-process historical quality baseline are separate experiments and were not rerun by these key-lifecycle or signing controls. Historical failed activation and unverified active `IMKCandidates` round trips remain explicit.

Historical mutating sources are evidence, not replay authorization. Reproduction requires a separately supplied authorized profile/certificate/runtime and a reviewed new owned run in `.workspace/`; never provision private assets or execute mutating sources inside this archive.
