# CI performance

Arion keeps portable validation on Linux and native acceptance on the actual macOS and Windows hosts. See [operations](README.md) for scope, concurrency and cache ownership, and [local CI](local-ci.md) for runner routing.

## Measurement

The baseline is the latest 50 workflow runs returned by `gh run list --limit 50`, including failures, cancellations, dispatches and retries. The job API supplies individual step timestamps. Successful-job medians exclude skipped, cancelled and failed jobs; those jobs remain in the detailed export. Reproduce the baseline with `node scripts/ci-timings.mjs .workspace/notes/ci-runs`; use `--reuse` to analyze saved API responses without fetching a newer sample. The tool exports every job as CSV and successful-job medians as JSON. API timestamp inversions, including retry records whose creation time is later than their start, are flagged and their invalid duration is left empty. Queue time starts at the job's `created_at`, so dependency waiting before job creation is separate from runner queueing.

Setup, dependency download, compilation outside checks, cache upload, artifact upload and runner cleanup count as preparation. Named validation steps count as checks. A combined check command can compile and execute tests internally; its duration is **not pure test execution time**. The API cannot split that further without command-level instrumentation. Windows keyboard logs provide per-script timings; Go package results provide test execution durations.

The baseline successful-job medians were 143 seconds for portable, 565.5 seconds for macOS native, 572 seconds for Windows native, and 70 seconds for Windows updater. Native job queue medians were 2 seconds; portable was 4 seconds. These are historical medians, not a matched comparison of identical source changes.

## Incremental work

Service Dockerfiles copy module manifests before source; the sync image downloads dependencies in its own layer. The updates service has no external module dependencies. Neither Dockerfile copies the entire repository before dependency resolution. Native runtime builders verify source, compiler, SDK and output identities before reuse; tests still execute even when compilation is cached.

Run the same build twice without changing inputs. Compare compilation and dependency work, separately from required test execution. A warm native check must still run assertions and OS lifecycle tests; it should not be described as a zero-cost build. Linux cannot establish native macOS/Windows build behavior. Local repeated service builds measured sync at 11.078 then 0.078 seconds and updates at 0.294 then 0.063 seconds. The updates Docker image also rebuilt with every build/COPY layer cached; the warm invocation took 1.305 seconds.

## Validation evidence

The first hosted iteration exposed a cache-version change: reducing the native path list made the old main cache unreadable, including prefix restores. The corrected workflow preserves that path list and trims unused build trees before saving. Keyboard shards also omit Go provisioning; the Windows launcher retains the same real standard-user and native preflight checks for each shard. Small control jobs use sparse blobless checkouts.

The implementation is validated locally and then through a pull request. Hosted run history is linked from [the optimization pull request](https://github.com/AstraFoundry/Arion/pull/82). Keep its cancelled cold-cache iteration separate from completed measurements. The timing tool emits the detailed phase export and summary for comparing the baseline with the final run. Results from a pull request cannot establish main cache saving or release signing; those paths remain explicitly scoped in the workflow.
