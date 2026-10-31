# Product driver contracts

Arion's public driver package owns the JSONL process transport, artifact identity checks and product E2E scenario assertions. It has no corpus, research scoring or private-repository dependency. Native drivers remain in `apps/desktop/cmd` and product scenarios remain in `tests/e2e`.

Run a product scenario suite with an explicitly provisioned native driver:

```sh
node tools/driver/cli.mjs e2e --suite tests/e2e/desktop-scenarios.json --driver <driver.json> --out <new-directory>
```

The output directory must not exist. The command verifies pinned runtime/model artifacts before launching the driver, requires actual platform and capability attestations, checks the bytes of every returned evidence file, evaluates the suite assertions and confirms owned-fixture cleanup. A failed scenario retains a failed `results.json`. Missing drivers, evidence, capabilities or cleanup fail explicitly.

## Public modules

| Module | Exports |
| --- | --- |
| `src/protocol.mjs` | `Driver`, `requireValue` |
| `src/artifacts.mjs` | `sha256`, `digest`, `hashFile`, `readJson`, `verifyDriverArtifacts` |
| `src/e2e.mjs` | `runScenarios`, `assertObservations` |

`Driver` sends sequential protocol-1 requests with process-owned request IDs. It rejects mismatched IDs, invalid JSON, unsolicited stdout, oversized lines, timeouts and process failures. Stderr is discarded to keep fixture content out of transport logs. Protocol clients perform their domain response validation after receiving an envelope.

`await driver.close()` waits for the child process to exit before callers remove its owned working directory.

The internal evaluation repository consumes these modules from its selected public code revision. Research experiment manifests, corpus isolation, metrics and report replay belong to that repository.

## Conformance tests

```sh
node --test tools/driver/test/*.test.mjs
```

Tests use temporary synthetic subprocesses and files. They exercise transport, failure records, artifact/evidence checks and cleanup. A synthetic process reporting a platform is a protocol fixture; these tests do not establish native editor dispatch, speech inference or product OS acceptance.

## Local completion research probe

`go build ./apps/desktop/cmd/arion-completion-probe` builds a protocol-1 adapter
for the existing local LLM client. Its `load` request accepts `client.Config`;
`complete` accepts the existing `prompt.ChatRequest`. Each request requires a
positive `timeoutMs` no greater than 120000. Responses preserve the request ID,
actual result, error and monotonic wall duration. It has no cloud path, candidate
insertion or automatic text commitment. Private studies own prompts, fixtures,
artifact verification and scoring; this probe does not establish native IME
acceptance or validate proposed pinyin completions.
