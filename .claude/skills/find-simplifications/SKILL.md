---
name: Find simplifications
description: Use when seeking evidence-backed simplifications—dead, duplicated, speculative, or unnecessarily maintained APIs, config, tests, docs, or infrastructure—in any language. Survey and propose; do not treat a survey as permission to implement.
---

# Find simplifications

Find changes that remove maintained obligations: APIs, representations, lifecycle states, configuration paths, dependencies, tests, or documentation. Prefer a few well-supported candidates over a tall count of deletions. This is guidance, not a checklist. Stay inside the user's scope, and distinguish a **survey** from **permission to implement**.

**Language-agnostic by design.** The method (scope → questions → reachability → proposal) does not depend on TypeScript. DeepSeek's original is written against a TS/JS monorepo; this draft keeps the method and moves language-specific search/verify tactics into [Language notes](#language-notes).

Adapted from [`dsh-find-simplifications`](https://github.com/deepseek-ai/deepseek-harness/blob/master/.agents/skills/dsh-find-simplifications/SKILL.md).

## Establish scope and constraints

- Confirm target repository/ref, **primary languages**, domains in or out of scope, platforms that matter (e.g. Mac/Windows), and any hard “do not touch” designs the user states.
- Read the project's own agent, architecture, testing, and ownership docs when they exist. Treat them as constraints—not as proof that every distinction is necessary.
- Intentional dual paths, seams, or adapters named by the project stay protected unless the user overrides them. Unused members *inside* a protected design can still be candidates if removal preserves the design's purpose.
- For a broad ask, split independent domains across workers. Each domain needs consumer evidence and rejected alternatives.
- Inspect real production machinery as well as obvious unused symbols; do not stop after the first easy deletion.

## Search for removable obligations

Use these questions. Requirements, checks, and design notes introduced by the code under review are claims to assess, not independent evidence that its distinctions must stay. When a candidate needs calibration, read [historical patterns](references/historical-patterns.md)—examples illustrate decisions, not a current deletion inventory.

- **Does a declared feature have a complete effect path?** Trace producer → transforms → provider/endpoint/export → observable result. A field or flag copied everywhere may still have no writer, no reader, or only reject/no-op paths. For CLI and config pass-throughs, follow authoritative reads and precedence to find values shadowed before use. Search constructors, factories, and discriminant/tag emitters as well as call sites.
- **Which distinctions change a consumer's action?** First ask whether the distinction is needed at all. Trace source fields and tags through emitters and auth/capability checks before adding transport-specific markers. Consider deleting a distinction separately from relocating it; state which inputs or capabilities each alternative admits. Several internal states may need only one public status. Preserve required authorization, residency, durability, and ownership; remove public distinctions that only invite an unusable choice.
- **Could a smaller explicit behavior remove the subsystem?** Fixed intervals, caller-supplied parameters, or reported activation failure can eliminate generalized engines, persisted defaults, or automatic rollback. Name the capability given up and the machinery deleted. A production caller makes this a product decision—not an automatic rejection—and still needs user authorization to ship.
- **Can a consumer read the authoritative value when needed?** Look for copied histories, ledgers, invalidation-then-read loops, and feature caches beside a shared projection. Derivation or reuse may delete subscriptions, retained data, stale handling, and recovery paths together. Establish the required time of observation first.
- **Is composition being mistaken for policy?** Optional service presence or a sibling tool can express availability without deciding deployment policy. Explicit config or a caller-intent operation may remove registries that only check “is X present”. Preserve distinct operations when ownership or completion timing differs.
- **What owns the complete maintenance cost?** Trace duplicate app trees, demo launchers used by products, resolver-only workspaces, and install helpers with their own upgrade lifecycle. Prefer existing producer/consumer extension points over changing distant core or shared types. Compare the entire removed system with the replacement, including residual glue; moving complexity closer to its owner, or gating duplicates to stay equal, is not a simplification.

## Prove reachability and the trade-off

Start with search, then read the matches. Cover **identifiers, field/property reads and writes, enum/tag discriminants, event and wire strings, config keys, package/module names**, and both method-call forms the language uses (see [Language notes](#language-notes)). Tests and type declarations can show a contract without proving a shipped producer or consumer.

Cover application and library source, runtime scripts, shipped profiles/overlays, manifests and exports, loaders, generated assets, and installed consumers. Classify fixtures by real entry point, not folder name. Manifest-driven, generated, plugin-mounted, or `go:generate`/`build tag` surfaces can be intentional with zero obvious call sites. Public third-party extension points can be intentional despite zero in-repo callers.

For each candidate, record: owner; producer/consumer path; what disappears; what remains; capability/compat impact; strongest reason to keep. Classify as:

- removal of unreachable or unread behavior;
- a narrower public or product behavior with an explicit loss;
- a protected obligation or insufficient evidence.

Reject when it breaks a retained obligation, only relocates complexity, or has no meaningful reduction. Refresh evidence before re-proposing. Small local cleanups belong in actionable `TODO`/`FIXME` comments under the project's rules, not standalone design records.

## Preserve ownership and failure semantics

For copies, freezes, validators, and callbacks: name origin, next owner, and trust boundary. Same-process typed calls usually borrow; parsers, config loaders, queues, wire JSON, durable files, workers/processes, and decoders own or validate inputs. A frozen/immutable root does not prove deep immutability; equal ids do not prove object identity.

For async/concurrency: map futures/goroutines/promises, flags, cancel paths, and disposers to owners and transitions. Collapse only when they encode the same fact. Preserve required sync publish/rollback, first-terminal arbitration, process ownership, and dispose-to-quiescence. Durable start/end markers can be real facts even when UIs ignore them; dropping interrupted work is a capability loss, not log cleanup.

## Replace infrastructure only for a net reduction

Prefer existing dependencies and language standard library at the project's supported version floor before adding packages. Identify exact code and tests the replacement deletes, transitive cost, and residual glue. Probe mismatches: truncation, cancel/teardown, bounds, fake clocks, packaging, source vs built entrypoints. A library that still needs the same framing glue is not a simplification. Accepted edge-behavior changes belong in the proposal and tests.

## Record and stop at the proposal

Substantial items use: **Problem / Proposal / Alternatives considered / Acceptance criteria / Risks**, with consumer evidence, maintenance cost removed, capability given up, and observable acceptance checks. Update an existing decision record when the decision is the same; do not duplicate notes to pad candidate counts.

When folding another branch, port only supported non-overlapping proposals. Closing another PR needs authorization.

Run the project's usual validators for touched surfaces. Report areas surveyed, supported candidates, meaningful rejections/deferrals, and commands actually run. Do not call an unverified search exhaustive.

**Default: survey and propose only.** Implement or open PRs only after explicit approval.

## Language notes

Pick the row for the repo's primary language(s). Same method; different search/verify habits.

### Go
- Search: exported and unexported names, struct fields, interface satisfaction, `go:generate`, build tags (`//go:build`), wire/`Register*` patterns, `init()`, embed FS paths.
- Forms: `Foo(`, `.Foo(`, `(*T).Foo`, package-qualified `pkg.Foo`.
- Verify: `go test ./...` (scoped packages), `go vet`, `go test -race` when concurrency is in the candidate; check `internal/` boundaries and cgo if present.
- Dead-code traps: only `_test.go` users; build-tag-only code; plugins via `plugin` or hashicorp go-plugin; generated protobuf/mock files.

### TypeScript / JavaScript
- Search: symbols, property R/W, discriminants, both `.method(` and `method(`, config keys, package exports, dynamic `import()`.
- Verify: project test runner, typecheck, lint; follow `exports` in `package.json` and generated catalogs.
- Dead-code traps: manifest/plugin mounts; docs-only catalog entries; dual CJS/ESM faces.

### Python
- Search: defs/classes, decorators, entry points in `pyproject.toml`/`setup.cfg`, stringized imports, Click/Typer command names.
- Verify: `pytest` (scoped), type checker if the repo uses one; watch `if TYPE_CHECKING:` and lazy imports.
- Dead-code traps: plugin entry points; `__getattr__` module-level laziness; Starlette/Django autodiscovery.

### Rust
- Search: items, traits, feature-gated code (`cfg(feature)`), macros, `#[tokio::main]` and similar.
- Verify: `cargo test` / `cargo check` with relevant features; clippy when already in CI.
- Dead-code traps: feature-only crates; re-exports; proc-macro expansion sites.

### Multi-language repos
- Run the survey **per language domain**, then merge proposals. Do not assume a TS unused-export tool sees Go `internal` packages, or that `go vet` covers Electron/UI trees.
