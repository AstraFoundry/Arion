# Code Review

## Checklist

### Architecture

- [ ] Does the change respect FSD/DDD layer boundaries?
- [ ] Is business logic in the domain layer, not in controllers or components?
- [ ] Are repository/ports defined in domain and implemented in infrastructure?

### Code quality

- [ ] Is all code and commentary in English?
- [ ] Are there no hardcoded strings?
- [ ] Is copy concise while preserving accessible names and necessary action guidance?
- [ ] Does frontend UI follow the installed craft skills (motion, polish, no AI slop)?
- [ ] Are existing implementations of the same business rule reused without abstracting merely similar code?
- [ ] Are failures exposed rather than silently masked, and are spec-approved recovery paths tested without bypassing privacy, cancellation or target-identity guarantees?

### Testing

- [ ] Are domain rules covered by unit tests?
- [ ] Are integration points covered by integration tests?
- [ ] Do tests fail for the right reason?

### Documentation

- [ ] If behavior or conventions changed, is `docs/` updated in the same change set?
- [ ] Are new public APIs documented?

### Security

- [ ] Are user inputs validated at the boundary?
- [ ] Are speech, context, memory and secrets excluded from logs?
- [ ] Are error responses safe for production?

## Review culture

- Reviews should be constructive and specific.
- Prefer asking questions over making demands.
- Approve only when the checklist is satisfied.

### Arion input and model boundaries

- [ ] Are session/generation/revision and host composition ownership validated?
- [ ] Can cancellation/focus/mode changes invalidate pending writes?
- [ ] Does local ranking preserve candidate IDs/text and user selection?
- [ ] Does the adapter expose real alternatives rather than invented n-best?
- [ ] Are checkpoint/export/tokenizer licenses, hashes and benchmarks recorded?
- [ ] Do OS-specific claims include actual target-host evidence?
- [ ] Is self-hosted relay content encrypted before leaving the client?

## Pull request conventions

- **Title:** the same [Conventional Commits form](../../CLAUDE.md#commit-conventions) as a commit subject.
- **Body:** English, in these sections, in this order:
  1. `## Summary` — what changes and why, one numbered item per independent change. Cite applicable spec acceptance IDs and ADRs; omit these references when the change has no relevant acceptance requirement or decision.
  2. `## Changes` — grouped by area (e.g. **Desktop IME**, **Site**). Name the files and what changed in each. Include doc, spec and ADR updates, and corrections made in passing (e.g. a doc that disagreed with the code).
  3. `## Screenshots` — required for any visible UI change, on the site or in the desktop app. Show every changed state: each locale whose layout differs, mobile width for the site, and intermediate animation frames when the change is motion. Lay out pairs in Markdown tables with a bold label per cell.
  4. `## Testing` — every check that ran and its result (e.g. `pnpm site:check`, native tests, a browser check). Under **Not run**, list each check that could not run here (e.g. native tests that need macOS) and the exact command or manual step a reviewer should use.
- **Screenshot hosting:** never commit screenshots to the feature branch. Push them to a separate image-only branch (`claude/pr-screenshots-<topic>`, under `pr-screenshots/`). Link them as `https://github.com/AstraFoundry/Arion/blob/<branch>/pr-screenshots/<file>.png?raw=true`. Say in the body that the branch can be deleted after merge.
- **Claims:** describe only verified results. Do not present unverified behavior as working.
