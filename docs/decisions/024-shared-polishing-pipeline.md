# ADR 024: Shared polishing pipeline and context controls

## Status

Accepted (product decision). Amends ADR 013's on-device-only context rule and ADR 022's context exclusion and single-call cloud recipe. Updates CTX-02–05, CLD-03–04 and PRV-01. The user chose equivalent context controls for local and cloud polishing.

## Context

Local polishing uses the selected spelling and formatting stages, while cloud polishing used the older `live` template. Hiding context controls for the cloud produced different behavior and required maintaining separate rewrite recipes.

## Decision

- `infrastructure/llm/rewrite` owns one immutable pipeline: local speech cleanup, bounded context-backed spelling choices, and scene formatting. Both the GGUF worker and cloud service invoke it; neither adapter owns a separate polishing recipe.
- The cursor, screen and recent-dictation switches apply to both engines. Existing persisted values and defaults are shared. Disabled sources are not collected for rewriting or included in the request. Secure-field exclusions, sanitization, history expiry and context bounds remain in the shared collection/domain code.
- Selecting cloud polishing allows enabled context sources, recalled terms and earlier recognized source to reach the connected provider's spelling-selection stage. The formatter receives only the edited source and its scene/script profile. Audio and app/site identity are not sent. Context is never logged or synced by Arion.
- Model adapters own transport and runtime behavior: local slot erasure, native token-budget checks and preparation stay in GGUF; provider credentials, wire conversion and response completeness stay in cloudpolish. The native selector's grammar is not a portable provider option; cloud selector replies pass the same bounded-choice validator.
- Cloud polishing remains final-only with a 15-second deadline for the complete pipeline. A final rewrite can make a selector call followed by a formatter call; no selector call is needed when there is no spelling decision. Local polishing is also final-only ([ADR 023](023-final-only-polishing.md)); neither engine produces polished previews during speech.
- Settings show the same context card for either active engine. Cloud and privacy tooltips disclose that enabled context is sent to the selected provider. Provider retention policies apply to that context as well as dictated text.

## Verification

Shared pipeline tests cover context/formatter isolation, earlier-source evidence, invalid and truncated choices, cancellation between stages, preparation and ambiguous candidates. Cloud transport tests inspect selector and formatter request bodies; bootstrap tests cover every combination of the three switches for both engines. UI tests cover both locales and shared preferences across engine changes.

These tests use synthetic model responses. They establish routing and protocol behavior, not cloud model quality, provider availability, latency or actual OS context coverage. Real-service acceptance remains CLD-A05.
