# Contract ownership

Normative meanings are in [004 Contracts](../../specs/004-contracts.md). Shared Go DTOs live in [core contracts](../core/contracts/contracts.go); this directory is reserved for reviewed wire fixtures and schemas when an actual IPC or relay transport is implemented. Do not duplicate domain behavior in generated clients. Every wire fixture must carry a contract version and pass an implementation-backed round-trip check.
