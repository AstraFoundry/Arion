# macOS input-source activation alignment

Implementation and known-authorization acceptance pass on macOS arm64. Clean-account first Allow/denial is **not verified**. The [result](result.json), [package identities](package-identity.json), [provenance](provenance.json) and [SHA-256 manifest](manifest.json) bind this run. Sources are focused snapshots; runtimes and signed application binaries remain separately provisioned.

| Check | Observed result |
| --- | --- |
| Existing enabled, unselected source | Production `ActivateIME` selected Arion and verified the expected executable hello. |
| Already selected source | Activation completed again without replacing bytes. |
| Cancellation | An immediately cancelled attempt failed explicitly and retained the prior input source. |
| Retry file preservation | Executable, Info.plist and signature-resource hashes remained unchanged. |
| Signed upgrade | An unselected source remained unselected; new expected-build connection completed. |
| Remove/reinstall | State 0 after removal, state 1 after installation, state 3 after activation. This account retained earlier OS authorization. |
| Actual IMK Pinyin | CUA key events `n i h a o` produced marked `ni hao`; Space committed `你好`, edits=1, marked=false. |
| Actual IMK cancellation | Subsequent `n i Escape` retained `你好`, edits=1, marked=false. |
| Isolation/cleanup | Editor typing used a temporary keyboard profile; owned host/editor stopped and profile removed. Original source restored. |
| Final installation | Main and nested host match the final signed package; previous main app retained in ignored `.workspace/ime-activation-app-backup/Arion.app`. |

`native/isolated-host-activation.json` retains a failed direct-launch attempt: source selection succeeded, but the expected-host connection timed out. The successful isolated attempt used LaunchServices `open --env ARION_DATA_DIR=<temporary profile>` with an observation listener already present, verified the actual hello, then used the production activation adapter with that socket. The failed attempt is not relabelled as success. The failure's precise launch-race cause was not independently recovered.

`checks/desktop-check.log` contains the full desktop test/vet and native keyboard checks; the captured engine/AppKit report has 34 passing cases. `checks/race-final.log` covers setup, filesystem/rollback and diagnostics; `checks/ui-final.log` covers bilingual enable/switch/cancel/recovery and onboarding; `checks/native-e2e.log` covers real window lifecycles. Opt-in context, learning, diagnostic-save and paste cases are explicitly skipped there and are not claimed passed. `checks/signed-build-final.log` records signatures, provisioning, notarization and Gatekeeper acceptance. Repository and i18n checks are retained separately. UI PNGs are view-test frames, not screenshots of a completed system consent wizard.

Actual OS activation uses the production adapter through `arion-ime-probe`; GUI action wiring uses separate controller/view tests. CUA could not attach the background main app for live GUI clicks. The actual editor observations come from its socket, not inferred from UI text alone. No microphone, personal editor contents, cloud recognition or permission database reset was used. Physical Intel and Windows TSF are outside this result.

Replay the retained assertions without changing system state:

```sh
node benchmarks/runs/ime-activation-v1/replay.mjs
```

To reproduce native activation, use the signed production host and rebuilt `arion-ime-probe --op activate --timeout 30s`. A probe cannot compete with the desktop for the same listener. `--cancel-after` tests cancellation; `--socket` selects an explicitly configured test host's socket. Stop the listener-owning desktop only for this authorized scenario, save/restore the selected source with the retained native source-control helper, and query actual source state rather than accepting exit codes alone. Reinstall tests require an authorized source installation/removal environment. First-consent/denial requires a separate clean macOS account or VM.
