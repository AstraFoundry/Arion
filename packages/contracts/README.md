# Contract ownership

Normative meanings are in [004 Contracts](../../specs/004-contracts.md). Shared Go DTOs live in [core contracts](../core/contracts/contracts.go). Relay wire examples live beside this file and are round-tripped by `services/sync` tests. Do not duplicate domain behavior in generated clients.

- [sync/push-v1.json](sync/push-v1.json) — `POST /v1/sync/push` body, schema version 1. The HTTP tests post these bytes and read the ciphertext back.
