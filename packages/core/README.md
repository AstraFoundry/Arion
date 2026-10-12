# Portable core

This standalone Go module contains session/candidate invariants, portable DTOs and domain-owned ports. It has no native or third-party dependencies. Run `GOWORK=off go test ./...` here for portable checks.

Only lifecycle transitions and candidate permutation validation are implemented. Streaming orchestration, cancellation/revision reducers, memory and use cases remain explicit tasks in the [handoff](../../docs/project/agent-handoff.md). No microphone or model is attached.
