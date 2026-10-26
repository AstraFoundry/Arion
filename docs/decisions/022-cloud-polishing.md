# ADR 022: Optional cloud polishing through Republic

## Status

Accepted (product decision), amended by [ADR 024](024-shared-polishing-pipeline.md) for shared stages and enabled context transmission. The original context exclusion and single-call recipe below are superseded. Amends PRV-01 of [005](../../specs/005-privacy.md) and adds the cloud engine to [009](../../specs/009-contextual-rewrite.md#cloud-polishing). Extends [ADR 011](011-live-local-rewriting.md); [ADR 013](013-rewrite-context-on-device.md) is amended by ADR 024 for explicitly enabled cloud context.

## Context

Local polishing needs the bundled GGUF runtime, which ships only on macOS on Apple silicon ([ADR 018](018-staged-gguf-rewriting.md)); Windows has no approved rewrite runtime. People who already pay for ChatGPT or hold a provider API key asked to polish with that model instead, during onboarding or in Settings, while keeping the local model as a choice.

[Republic](https://github.com/AstraFoundry/republic) is AstraFoundry's Go provider library: one chat interface over OpenAI, Anthropic, Google, DeepSeek, OpenRouter and the ChatGPT Codex backend, with Codex OAuth token refresh. It stops at providers and starts nothing on its own.

## Decision

- **Engine choice.** Polishing has an engine preference, `local` (default) or `cloud` (`Preferences.PolishEngine`). Choosing the cloud is explicit and visible on the polish card ("Runs on"); nothing is routed to the cloud otherwise. The cloud engine makes polishing available where the local runtime is not shipped.
- **One connection.** Either a ChatGPT sign-in or one API key (OpenAI, Anthropic, Google Gemini, DeepSeek, OpenRouter). Connecting replaces the previous connection. The credential is one OS credential item (service `Arion`, account `cloud-polish`): the API key, or the ChatGPT refresh token with its account ID and email. ChatGPT access tokens stay in memory; a rotated refresh token is stored again. Disconnect deletes the item.
- **ChatGPT sign-in.** An in-app OAuth authorization code flow with PKCE against `auth.openai.com`, using the public client the Codex CLI registers (the one Republic refreshes). Its redirect URI is fixed to `http://localhost:1455/auth/callback`, so the loopback listener takes port 1455 on `127.0.0.1`; a busy port is an explicit failure. The account must carry a ChatGPT account ID. Requests go to the Codex backend with `originator: arion` and `store: false`.
- **API keys** are checked by listing the provider's models before they are stored; a refused key is not stored.
- **Model choice.** The user chooses among the models the connected provider lists (`Preferences.CloudModel`). ChatGPT models come from the Codex backend's model list, which Republic leaves unsupported; Arion supplies that listing through a Republic provider spec.
- **Payload.** Only the final rule-corrected text after stop, its scene and the recalled vocabulary terms (at most eight, as for the local model), inside the shipped `live` prompt as chat turns. No audio, no text around the cursor, no window text, no dictation history, no app or site identity. Context switches hide while the cloud engine is chosen, and the target adapter does not read field or window text for it.
- **Final only.** No live previews: one request per dictation (`RewritePolicy.FinalOnly`), bounded by a 15 s timeout. Local polishing uses the same timing ([ADR 023](023-final-only-polishing.md)). The result passes `domain.ValidateRewriteDelivery` like any local proposal; failures, refusals and timeouts deliver the local cleanup with an explicit outcome (LRW-08).

## Consequences

- Dictated text leaves the device when, and only when, the user chose the cloud engine and connected a provider. The provider's own retention and terms apply; Arion sets `store: false` where the protocol has it.
- Recognition, rules, scene choice, cleanup and validation stay local; offline dictation is unchanged. Without a connection or a model, a cloud session records `unavailable` and inserts the local cleanup.
- Windows users can polish without a local runtime.
- The ChatGPT flow depends on OpenAI's Codex client registration and backend, which are not a published API; a change there surfaces as a sign-in or model-list failure, not a fallback. The model-list `client_version` is a fixed value Arion sends.
- Credential items on Windows are limited to 2,560 bytes, which is why the access token is not stored.
- Cloud output has not been evaluated against the local benchmark; no quality or latency claim is made.

## Alternatives considered

- **Running `codex login` in a terminal** (Republic's `LoginCodex`): needs the CLI installed and a terminal; not a desktop flow.
- **Reading `~/.codex/auth.json`:** couples Arion to another tool's file and its sign-out; rejected.
- **Live previews in the cloud:** one request per pause multiplies traffic and cost and sends partial speech; rejected for now.
- **Sending context to the cloud** (as Typeless does): ADR 013 keeps context on the device; rejected.

## References

- [Cloud polishing](../project/cloud-polishing.md) — call surface and states.
- Republic [authentication](https://github.com/AstraFoundry/republic/blob/main/docs/guides/authentication.md) and [codex provider](https://github.com/AstraFoundry/republic/blob/main/docs/providers/codex.md).
