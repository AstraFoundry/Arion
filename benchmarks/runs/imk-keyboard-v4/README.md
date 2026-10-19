# OS diagnostics and pre-fix Keychain lifecycle evidence

This immutable supplement retains completed OS runs 08–10, focused public metadata sources, and the completed six-phase real Keychain lifecycle from the **pre-fix legacy-Keychain implementation**. Earlier [v1](../imk-keyboard-v1/README.md), [v2](../imk-keyboard-v2/README.md) and [v3](../imk-keyboard-v3/README.md) remain unchanged. These observations do not pass installed IMK typing acceptance or verify device-bound key protection.

```sh
node benchmarks/runs/imk-keyboard-v4/verify.mjs
```

The verifier only reads frozen evidence. It checks every included file's size and SHA-256, exact tested Keychain source identities, failed selection/restoration states, first-party control transitions and lifecycle boundaries. It does not execute archived drivers, access Keychain, provision a runtime, change input sources or contact the network. `archive.json` records original paths, original versus derived evidence, exclusions and provenance. Historical driver sources are evidence, not an authorized replay command.

## Failed registration hypotheses and restored control

Run 08 used the same source identifier under a fresh bundle filename with unchanged final signed host bytes. Registration/enable returned `0`, but the parent remained disabled and selection returned `-50`. The original `observations.json` says **restore-failed** and is preserved unchanged. The final original controller state records subsequent recovery, its initial cleanup error, ABC current, absent learning key, disabled cached own entries and no remaining errors. `recovery.json` is explicitly a derived summary of that final state and the original disable output; it does not replace or relabel the initial failure.

Run 09 changed the parent/child identifiers in a fresh signed bundle. Its original notarization result is **Accepted**, and Gatekeeper reports Notarized Developer ID. Registration/enable still returned `0` with the parent disabled; selection again returned `-50`. The controller restored ABC, absence of the learning key and disabled cached own entries. Public signing hashes/results and preparation/controller sources are retained; application/ZIP bytes and signing credentials are excluded. No installed-source event dispatch or text insertion passed.

Run 10 exercised the preinstalled Apple Korean parent while ABC stayed current. Actual `TISEnableInputSource` changed the parent from disabled to enabled; disabling returned it to disabled. Both statuses were `0`, and the original full control state was restored. This establishes that the public TIS calls from this CLI context can change a first-party source. It does not explain the third-party activation failure or establish that a GUI caller would fix it. No unrelated source-state inventory is included.

## Public metadata provenance

`metadata/` preserves the downloaded Apple NumberInput introduction, five extracted official sample `Info.plist` stages, the exact failing Meltype revision's metadata and GitHub tree, the separately captured current branch metadata/tree, and the original bounded assessment. The assessment records source URLs, the fixed Meltype commit and sample-carrier SHA-256. The sample ZIP remains excluded with its digest retained. The Apple stages all declare `CFBundleInfoDictionaryVersion = 6.0` and omit `CFBundleSupportedPlatforms`; the failing Meltype revision also has `6.0`. These are static observations, not proof that any missing metadata causes or fixes modern macOS registration. The branch named `main` is a captured snapshot, not a permanently fixed revision.

## Pre-fix real OS-kept key lifecycle

The original `b03804bd-afea-4544-aaed-e898ad06e8f3` reports pass guard, writer, reader, delete, missing-key and cleanup phases. Production engine/encryption code used an OS-kept key and an owned synthetic profile. Selecting `事实` from `shishi` promoted it from second candidate to first after a new process. Deleting the owned key made actual engine startup fail closed while the encrypted file's digest remained unchanged. Cleanup restored key absence and interaction policy; no input-source changes occurred. Only ciphertext inventory sizes/hashes are retained, not the ciphertext/profile or key material.

**Device-bound protection was not established.** Explicit legacy-Keychain lookup returned the same owned reference; explicit Data Protection lookup returned `-25300`. Accessible and Synchronizable attributes were absent, which does not mean false. The requested accessibility policy cannot be inferred from lifecycle success. `keychain-policy-assessment-01.json` records this limit and short local SDK-header excerpts with header SHA-256. A later policy fix or its tests require separate evidence.

`keychain/sources/` contains the six exact production/harness sources named in the executed manifest, plus the captured runner and build helper. The runner digest also matches the manifest. The earlier preparation compiler failure is preserved separately from the completed preparation/execution logs; it does not have a matching failed-source snapshot. Preserve that limitation rather than treating the final sources as the failed attempt's exact revision. Native test/library hashes remain in the public manifest; no binaries, SDK, ownership receipt, persistent reference bytes, key bytes, personal profiles or preferences are archived.

To investigate reproduction, copy the exact sources to a new `.workspace/` checkout, supply the separately pinned runtime/data/build dependencies and review all required OS-mutation authorization first. Do not run the historical sources in place or provision environments under this archive.
