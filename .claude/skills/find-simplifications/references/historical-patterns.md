# Historical simplification patterns

Use these patterns to calibrate a candidate whose value is not established by an unused-symbol search. They describe *kinds of decisions*, not a live deletion inventory for any one repo. Before applying a pattern, verify current owners and released-data or compatibility obligations in the target project.

(Generalized from DeepSeek Harness `dsh-find-simplifications/references/historical-patterns.md`. Concrete note links and package names removed; keep the decision shape.)

## Require a complete effect path

A declaration, serializer branch, and test can look complete without an effective feature. Trace both ends: something must produce the value, and an implementation must honor it.

Typical wins: reserved union members or cache hints with no emitters; request knobs rejected by every adapter while still forwarded through hooks. Deletion then includes declarations, forwarding, rejection-only tests, and generated docs.

Zero fixed callers is weaker than unreachability. Deliberate third-party registration or extension paths stay. For catalogued services, inspect generic plugin invocation and returned values before calling them dead. A working extension point and a field no provider can honor need different verdicts.

## Reduce distinctions where consumers decide

Map every public variant to the consumer action it enables. Multiple internal residency or routing states may project to one usable public target or availability signal; exact capability can stay in tool or API schemas instead of approximate registries.

Do not erase distinct durable facts merely because one renderer ignores them. Explicit start/completion facts can still be required for empty failed steps, crash repair, and inspection. Simplify the projection only when the underlying distinction remains necessary.

## Trade general behavior for a smaller explicit promise

A used feature can still be an expensive way to satisfy the required outcome. Examples of the shape: fixed-rate schedules instead of full calendar recurrence and missed-occurrence replay; failure reporting without automatic transactional rollback across plugins.

State the loss as clearly as the deletion. Convenience, smaller unions, or fewer tests alone do not justify losing required data or behavior. A survey may propose the trade-off without implying permission to implement it.

## Make assumptions explicit at their consuming operation

Look for a local default that has acquired shared storage, synchronization, or confirmation machinery. Prefer putting browser or request facts on the originating request and requiring explicit parameters at the consuming operation, rather than session-wide defaults and cross-tab conflict state. Leave framing with the caller instead of threading envelope controls through unrelated layers.

Do not move a requirement past the place that must enforce it. Removing duplicate client-side schema execution is fine if the host still validates before lookup and business invocation; the documented cost may be later malformed-input failure and loss of client parsing as accidental redaction.

## Derive at use or reuse an existing delivery owner

For each cache or event subscription, ask when the consumer needs the value and whether an authoritative source already retains it. Reading a canonical prefix at feedback time can delete a continuously copied pre-feedback buffer. Sorting current summaries can replace persisted promotion history when manual ordering is separate user state. Reusing a shared stream and projection store can remove feature notifications, repeated reads, and a second membership cache.

The timing promise determines whether this works. Feedback-time redaction differs from capture-time redaction; manual placement cannot be derived from activity timestamps; an initial historical read may remain necessary when a stream baseline is absent. Name residual obligations instead of declaring all caches redundant.

## Remove accidental composition owners

Inspect actual launch and resolution paths before trusting names such as demo, support, or example. Shared base config overlays can remove duplicate compositions; owner-local tests/guides/assets can remove resolver-only workspaces; explicit deployment policy can replace inferring policy from provider or sibling-tool presence, while optional dependencies stay internal.

A smaller package count is not enough. Separate caller intents may have different ownership and completion timing; a retained service definition may still serve independent providers. Prove the replacement's real composition and dependency behavior, and account for remaining configuration and lifecycle glue.

## Use rejected experiments to test parity claims

A library swap that deletes generic framing must retain provider-specific termination shims and document any stricter truncated-tail behavior. A nominally equivalent builtin that breaks deterministic fake-clock tests is not a simplification for a small deletion. Distinguish replaceable generic code from configured bounds, security rules, lifecycle ownership, and deployment behavior a library does not implement.

A past rejection is a scoped falsifier, not a permanent blacklist. Revisit only with new evidence that addresses the failed premise, measuring implementation, glue, tests, dependencies, and supported behavior together.
