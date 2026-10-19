# Translated x86 execution and final control evidence

This immutable supplement preserves completed **22 / 22** x86_64 real-engine/AppKit cases executed through **Rosetta 2 on Apple Silicon**, a separately tested later metadata patch, completed final control/build logs and a restored application-UI attempt that did not reach installation. It does not establish physical Intel testing or installed IMK typing dispatch. Earlier [v1](../imk-keyboard-v1/README.md) and [v2](../imk-keyboard-v2/README.md) remain unchanged.

```sh
node benchmarks/runs/imk-keyboard-v3/verify.mjs
```

The verifier is read-only and requires no runtime, network, Keychain or system mutation. `archive.json` binds every included original file and source snapshot by original path, size and SHA-256. Native binaries, SDKs, user profiles, keys and preference files are excluded.

## Execution architecture and exact source revisions

`engine/typing-attempt-13.json` records a thin x86_64 executable launched with `/usr/bin/arch -x86_64`; the physical host and Node runner were arm64. Already-installed Rosetta translated the executable. All 22 real conversion, AppKit, encrypted persistence/failure, rollback, revision and cleanup cases passed. The runtime and production native sources match attempt 12; the architecture-selecting runner snapshot has its own recorded identity. `sources/` preserves the exact originals recorded for attempt 13.

`metadata/runner-metadata-after-13.mjs` is a **later runner-only patch** with SHA-256 `75cf875b1deab8c245243866e66e27869dc1d89b89d8583dff1df7435a18b263`. Its separate JSON records native and translated `sysctl.proc_translated` probes, the absent-OID/native case and malformed/error rejection. This patch follows [Apple's documented Rosetta probe](https://developer.apple.com/documentation/apple-silicon/about-the-rosetta-translation-environment). These metadata checks did not rerun the 22 functional cases; the original attempt-13 runner and later patch are intentionally preserved as distinct source revisions.

Reproduce the original functional run by copying `sources/` to a fresh `.workspace/` directory, staging its pinned runtime/data, then using `node scripts/test-keyboard-input.mjs --arch x86_64 --report <new-path>` there. On Apple Silicon this requires Rosetta already available. The archived run is translated x86 execution, not a claim that the same test ran on physical Intel. Never provision a runtime or SDK inside this archive.

## Completed final controls

The original `checks/desktop-keyboard-check-final.log` records desktop Go checks and the actual native AppKit/encrypted-learning/22-case keyboard checks. The referenced native report is retained as `checks/keyboard-input.json`. Separate original logs retain the final IME race checks and i18n check. These are control observations, not keyboard latency measurements or a new external-editor acceptance run.

`checks/signed-desktop-build-final.log` is completed original output reporting Developer ID signing, notarization and signature/ticket/Gatekeeper/package verification. It contains public build/profile provenance and configuration variable names, with no credential values. No application or DMG binary is copied. A successful signed package still does not establish OS source enablement or text dispatch.

## Restored UI attempt

Run 07 launched the owned signed application with `NSWorkspace` and an isolated offline profile. The main window appeared, but automation could not access the menulet/install button; UI installation was therefore **not exercised**, and actual IMK dispatch was not reached. Its observation file preserves the stated evidence type, and its controller state confirms restoration with ABC current, no own input-source entries, no test learning key and no restoration errors.

Only allowlisted own application/source status, owned process metadata and synthetic observations are archived. The profile, application copies, credentials and unrelated UI content are excluded. Exact historical controller/launcher sources are preserved as evidence and must not be mistaken for an automatically authorized install/replay command.
