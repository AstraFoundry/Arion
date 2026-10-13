# Portable core

Standalone Go module with no native or third-party dependencies. Run `GOWORK=off go test ./...` here.

- `domain`: the dictation session aggregate (state machine, monotonic revisions, exactly-once commit claim), stable error codes, conservative correction rules (`arion-rules-v1`), candidate permutation validation and evaluation-driven model selection.
- `contracts`: versioned transport-neutral DTOs and UI events.
- `domain/ports`: audio capture, recognizer, finalizer, corrector, ranker, target and composition boundaries.
- `application`: the dictation coordinator (generations, bounded queues, cancellation, finalization, correction and single commit) and persisted preferences.
