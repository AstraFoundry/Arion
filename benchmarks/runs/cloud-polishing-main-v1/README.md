# Cloud polishing main-branch integration

The shared pipeline and mock gateway tests were integrated onto `c5531aa9`, preserving the newer final-only timing, punctuation setting and `<source>` formatter envelope. The shared-context decision is ADR 024 because ADR 023 already defines final-only timing. This run is synthetic HTTP verification, not inference evidence.

- All 68 gateway cases pass: the prior 60 cases plus both protocols × four punctuation modes. Punctuation cases inspect the serialized system instruction because an echo gateway cannot establish model punctuation behavior.
- Race-enabled cloudpolish, shared rewrite, native adapter unit tests, core application and bootstrap packages pass. Opt-in actual native inference tests are not enabled by this command.
- Complete UI package, vet and repository check results are retained in `checks/`.
- The first complete UI run found an obsolete assertion requiring hidden cloud context and an off-screen history switch in the new layout. The assertion now requires visible context, and the interaction uses the existing scroll-to-row helper; targeted rechecks passed.
- The initial merge check exposed the new punctuation type and source-envelope contract; these were integrated, and the original failed log is retained.
- Earlier native timeout and session-erasure observations remain unresolved; see [original verification](../unified-polishing-v1/README.md).

Commands, run from the repository root:

```sh
go test -race ./apps/desktop/infrastructure/cloudpolish ./apps/desktop/infrastructure/llm/rewrite ./apps/desktop/infrastructure/llm/gguf ./packages/core/application ./apps/desktop/bootstrap -count=1
go test ./apps/desktop/ui/app -count=1
go vet ./apps/desktop/infrastructure/cloudpolish ./apps/desktop/infrastructure/llm/rewrite ./apps/desktop/infrastructure/llm/gguf ./apps/desktop/bootstrap ./packages/core/application
pnpm check
```

`source-hashes.json` identifies the changed implementation and tests. `manifest.json` hashes the completed logs and this report; historical gateway and inference archives remain unchanged.

Final results: `pnpm desktop:check` passed, including the complete UI package (157.703 s), desktop vet, native hold-listener checks and native keyboard checks. Related race tests, vet and repository checks passed. Initial failed checks above remain in the archive.
