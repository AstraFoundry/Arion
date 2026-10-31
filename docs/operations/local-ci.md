# Local CI

Portable checks and release orchestration can run on a repository-scoped GitHub Actions runner in a local Ubuntu VM. This changes execution location only; GitHub retains event handling, job results, caches, artifacts and Actions secrets. Public product CI never checks out the private evaluation repository.

## Routing and rollback

The repository Actions variable `LOCAL_CI_ENABLED=true` routes Linux jobs to `[self-hosted, Linux, ARM64, af-arion-linux]` for pushes, release callers, manual signing callers and pull requests whose head repository is this repository. Fork pull requests use `ubuntu-latest`. With the variable absent or set to `false`, all Linux jobs use `ubuntu-latest`.

The variable affects `portable`, `scope`, `required-checks`, `cancel-on-portable-failure`, and the Linux `preflight` and `draft-release` jobs. The required-check aggregation and native scope decision keep their existing behavior. The runner must be online before enabling local routing. To roll back, set the variable to `false`; cancel and rerun already queued local jobs because they retain their selected runner labels.

Fork routing is an operational default, not an isolation boundary against a modified PR workflow. Every local job, including same-repository PRs, runs in a disposable VM. Repository maintainers must continue reviewing external workflow execution approvals.

## Runner lifecycle

The operator manages the local installation outside the checkout, under `~/ci/astrafoundry`. Each repository has an independent Lima VM. The clean Ubuntu 24.04 ARM64 base includes Docker, build tools, AWS CLI and the official Actions runner, without repository data or runner credentials. A host controller clones it, registers an `--ephemeral` runner using a short-lived token, waits for one job, exports diagnostics and destroys the complete VM disk. Re-registration always starts from the clean base. The controller retains the latest 30 diagnostic generations locally. GitHub retains uploaded artifacts and job logs under its existing policy.

Plain mode disables host directory mounts, SSH agent forwarding and dynamic application port forwarding. The host's GitHub login and private repository credentials are never copied into the guest. Workflow secrets arrive through GitHub's job token and secret mechanism. The VM does not enforce an outbound network allowlist; use network isolation when local-network access must be prohibited.

The controller runs as a macOS user LaunchAgent, so login, power, sleep and network availability determine runner availability. Each repository handles one local Linux job at a time. Keep runner automatic updates enabled and rebuild the base regularly for OS/tool updates. Existing pinned Node, pnpm and Go versions remain provisioned by the workflow actions; caching remains architecture-specific.

## Native coverage

The macOS and Windows jobs stay on their existing GitHub-hosted runners. A local Linux VM cannot replace Metal/Swift, AppKit, Windows UI, TSF, DPAPI or installer checks. The first host has macOS 15.1 and Command Line Tools only, without the full Xcode app; it does not meet the [native toolchain requirements](local-dev.md). Do not relabel it as `macos-26` or remove native checks to make the migration appear complete. Moving those jobs requires suitable dedicated macOS and Windows machines and verification of their interactive desktop and disposable-user behavior.

## References

- [GitHub self-hosted runner scope and public-repository warning](https://docs.github.com/en/actions/how-tos/manage-runners/self-hosted-runners/add-runners)
- [Ephemeral runners, logs and updates](https://docs.github.com/en/actions/reference/runners/self-hosted-runners)
- [Lima plain mode](https://lima-vm.io/docs/config/plain/)
