# Code Review

## Checklist

### Architecture

- [ ] Does the change respect FSD/DDD layer boundaries?
- [ ] Is business logic in the domain layer, not in controllers or components?
- [ ] Are repository/ports defined in domain and implemented in infrastructure?

### Code quality

- [ ] Is all code and commentary in English?
- [ ] Are there no hardcoded strings?
- [ ] Is there no redundant UI copy?
- [ ] Does frontend UI follow the installed craft skills (motion, polish, no AI slop)?
- [ ] Is there no duplicated logic that could be extracted?
- [ ] Is there no fallback/clever bypass logic?

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
