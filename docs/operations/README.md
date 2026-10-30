# Operations

- [Local rewriting](local-rewrite.md) — selected GGUF model, two-stage prompts, private runtime and delivery policy.
- [Windows input method](windows-input-method.md) — TSF setup, shared Pinyin integration, security boundary and native verification.
- [Local development](local-dev.md) — actual repository commands, module toolchains, and [macOS development build and TCC permission diagnostics](local-dev.md#macos-development-build-and-tcc-permission-diagnostics).
- [Product drivers](../../tools/driver/README.md) — public driver protocol and owned product scenarios.
- [Evaluation ownership](../project/repository-split.md) — private research commands, version binding and Git archive integrity.
- [Deployment](deployment.md) — desktop/model/IME and self-hosted relay release gates, and the marketing site.
- [Release status](release-status.md) — shipped platform capabilities, preview signing status and host acceptance limits.

## Build components

Use `make build <component>` with exactly one component per invocation:

| Command | Output |
| --- | --- |
| `make build desktop` | Native app and installer for the current host; configured ENV enables macOS signing/notarization and signed updates |
| `make build site` | Static marketing site in `apps/site/out` |

Both dispatch through `scripts/build.mjs`, also used by desktop CI. Missing, unsupported or extra components fail before building. Mobile and server builds are not registered yet. `make site-deploy` uses the same site build before deployment. The former `desktop-build` and `site-build` targets have been removed.

The foundation uses Node 24 and pinned pnpm 10.33.0 for workspace/check/driver tooling, Go 1.25.7+ for isolated portable core checks, Go 1.27.1+ for the sync relay and MyGo desktop. `go.work` and the desktop, sync and updates modules select the patched `go1.27.3` toolchain (standard-library security fixes); CI and the service images build with it. Go automatic toolchain selection may download the pinned newer toolchain. Native shell compilation is distinct from functional speech/IME support.

The desktop shell does not need a database. Optional [clipboard history](../project/clipboard.md) uses its own bounded encrypted snapshot and OS-kept key. The marketing site builds from committed public configuration (`apps/site/.env.production`); only `make site-deploy` needs Cloudflare credentials, supplied by the shell running it. The sync relay has its own PostgreSQL, configuration and runbook under [deploy/sync](../../deploy/sync/README.md). Do not point it at another product's database.

Native CI invokes the Windows keyboard checks through `scripts/check-windows-keyboard-ci.ps1`, and `pnpm keyboard:check` on macOS before packaging. [GitHub-hosted Windows runners run as administrators](https://docs.github.com/en/actions/reference/runners/github-hosted-runners#administrative-privileges), so the Windows launcher creates a disposable standard user, loads its profile for DPAPI and runs the complete `check-windows-keyboard.mjs` suite with that identity. It restores workspace permissions and removes the account afterwards, refuses developer machines and persistent runners, and never weakens production integrity checks. Local Windows checks remain `pnpm keyboard:windows:check` from an ordinary desktop session. Both platforms include shared checks; Windows adds real conversion, module and owned-ACP tests, while macOS runs its native adapters. These isolated gates do not register an input source or replace installed-editor acceptance.

The credentialed Windows child receives the parent's resolved Go executable explicitly and prepends that SDK directory after its environment is created. It verifies the same actual SDK version reported by the parent and the PATH identity, records the executable hash in `windows-keyboard-ci/go-toolchain.json`, and uses `GOTOOLCHAIN=local`. Toolchain provisioning belongs to `actions/setup-go`; test execution must not pick an older machine SDK and download another compiler inside a bounded native gate. The `go` directive is a language/minimum-version requirement, not an exact SDK pin; the selected parent SDK can be a newer approved patch. Profiles and Go caches remain owned by the temporary test user; no release credentials are forwarded.

The hosted launcher grants only the temporary SID the required window-station/desktop access, then restores and reads back the original DACLs during cleanup. A native preflight verifies the expected SID, medium integrity, desktop, GUI creation, TSF and DPAPI before the suite. The child archives its private fixture reports into `.workspace/keyboard-evidence.zip` before account removal so the parent artifact upload can read them; the disposable VM reclaims the user profile. Hosted execution has verified this preflight, the complete Go desktop suite and cleanup under the temporary identity. Actual foreground activation and global key-hook E2E run separately in the runner's original interactive session; desktop access for the temporary user alone does not establish foreground-input ownership.

CI installs with a frozen pnpm lockfile. `make updater-check` / `pnpm updater:check` verifies the maintained MyGo update-channel patch and runs in both the portable and Windows native CI jobs. The portable job also runs `pnpm site:check` (site dictionaries, types, lint, tests and static export). Native jobs run actual window E2E and verify packages on macOS arm64 and Windows x64. Tagged releases require Apple signing/notarization and update credentials, reuse these jobs at the tag commit and upload only a complete verified set for their selected platforms (all tags, including alpha: macOS ARM64 and Windows x64); ordinary CI retains installers without signing keys. Run studies and archive checks locally in the private [evaluation repository](https://github.com/AstraFoundry/Arion-evaluation), with a reviewed local driver and licensed corpus. GitHub Actions is disabled there. Keep raw traces private and follow the [repository split](../project/repository-split.md) for version binding.

Update and notification workflows are documented in [update lifecycle](../project/update-lifecycle.md). `make updater-e2e` uses owned temporary application bundles; `make notifications-check` checks native transport/registration code and `make notifications-prepare` builds a separate consent probe.
