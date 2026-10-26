# Cloud polishing

Optional polishing with a cloud model the user connects ([ADR 022](../decisions/022-cloud-polishing.md), [CLD-01–05](../../specs/009-contextual-rewrite.md#cloud-polishing)). The UI calls `runtime.Actions` and reads `runtime.State.Cloud`; it never imports `infrastructure/cloudpolish`. The composition root (`bootstrap`) owns the service and routes each rewrite request to the local worker or the cloud through `polishRouter`, by `Preferences.PolishEngine`.

## Where the user sets it up

One decision per page:

- **Onboarding.** The model step sets up recognition only. The **Polish** step offers Off, This device (with its download size) and Cloud. Choosing This device keeps the row compact and shows a primary yellow Download action at its trailing edge; selection alone never downloads or shows the local ready check. Confirming Download replaces the trailing action with Cancel download and smoothly expands progress inside the local option, before Cloud; cancellation does not occupy a separate row. Failures offer retry within the local option. Only a ready local worker replaces the action with a check; its progress panel then collapses. Continue waits for confirmation, then allows setup to continue while the confirmed download runs in the background. Cancelling turns polishing off; choosing local again requires confirmation. Only with Cloud, a **Cloud model** step follows: the provider list, then the chosen provider's page (Sign in with ChatGPT, or the masked API key and Connect), then, once connected, Disconnect and the model. Back leaves a provider page before the step; the footer says Skip until a model is chosen.
- **Settings › Model** keeps the settings on the section: a **Recognition model** row (opening the recognition model's page), then the Polish card with the polish switch (**Polish** / **润色**), **Runs on** (This device / Cloud), then, while Cloud is chosen, one **Cloud model** row (the provider's mark and the model, opening the cloud pages), then **Punctuation** (the label on the left and the mode menu on the right, enabled only while polish is on; English lists Smart, Complete, and Omit last mark, while Simplified Chinese also lists 空格代替标点; a saved spaces value stays until the user picks another mode), then Match the app. Context is its own group. Every explanation is a tooltip on the row's info hint. The title's back button steps out one page at a time.
- Providers are shown by their [LobeHub](https://icons.lobehub.com) marks.
- Enabling local polishing in Settings also offers Download before provisioning missing resources. Installed verified resources load automatically. Startup and switching back to local never resume an unconfirmed or cancelled download; queued work from an older choice is cancelled before it can download.
- The context switches (cursor, screen, history) are shared by both engines. Enabled sources reach the selected cloud provider for spelling selection ([ADR 024](../decisions/024-shared-polishing-pipeline.md)); the tooltip discloses that transmission. Match the app chooses the scene locally.

## What views call

These return immediately; results arrive in `State.Cloud`.

| Method | Effect |
| --- | --- |
| `SignInChatGPT()` | Opens `auth.openai.com` in the browser; one callback on `127.0.0.1:1455`. Phase `signing_in`, then `connected` or back to the previous connection. |
| `ConnectCloudKey(provider, key)` | Lists the provider's models with the key (phase `verifying`); stores it for that provider only if that works, and makes that provider active. Other providers' keys stay. |
| `ActivateCloud(provider)` | Makes a provider that already has a stored credential active, without asking for the key again. |
| `CancelCloud()` | Stops a sign-in or key check. |
| `DisconnectCloud()` | Deletes the active provider's credential and the active-provider pointer. Other providers' keys stay. |
| `ListCloudModels()` | Lists the connected provider's models; views call it once when the rows show without a list. |

The chosen model is a preference (`UpdatePreferences` with `CloudModel`), applied to the service with `SetModel`.

## What views read

`State.Cloud`: `Phase` (`disconnected`, `signing_in`, `verifying`, `connected`), `Provider`, `Email` (ChatGPT only), `Models`, `Listing`, `Saved` (provider ids that still have a credential), `Failure`. Failure codes (`runtime.CloudCodes`) equal `cloudpolish.Codes`; a test checks both lists. `CANCELLED` shows nothing; a failed model list offers **Retry**.

## Credential and startup

- One OS credential item per provider: service `Arion`, account `cloud-polish/<provider>` (`keystore.CloudPolishProviderAccount`). JSON with the provider and either the API key or the ChatGPT refresh token, account ID and email, plus the model last chosen for that provider.
- Account `cloud-polish` stores only which provider is active (`{"provider":"<id>"}`). It holds no key. A previous build stored the whole credential there. Load copies that secret to `cloud-polish/<provider>` when that provider has no item yet, then rewrites `cloud-polish` as the pointer.
- Opening a provider that already has an item makes it active and restores its model. The key is not copied into the UI. The provider list marks those rows saved.
- At launch the service reads the items off the main thread, without network. The active provider opens on first use: a ChatGPT connection trades its refresh token for an access token then and stores a rotated refresh token on the ChatGPT item only.
- Disconnect deletes the active provider's item and the pointer. The other items remain, and choosing one of those providers makes it active again.
- Republic signs every request; ChatGPT requests go to the Codex backend with `originator: arion`. Requests use the system proxy through the default transport.

## Requests

- Final-only scheduling (`RewritePolicy.FinalOnly`) with a 15 s deadline for all stages. `infrastructure/llm/rewrite` is shared with the local worker: speech cleanup, optional context-backed spelling selection, then scene formatting. The recipe is versioned `cloud+<SelectedVersion>`. Cursor, screen and history follow the same `ContextPolicy` switches as the local engine; recalled terms and prior recognized source also inform selection.
- A rewrite binds its provider/model once. The provider adapter converts chat messages and enforces completion/refusal checks. API models receive the stage output cap. The shared recipe leaves thinking off (`AllowThinking` zero value). The local worker writes that policy as `chat_template_kwargs.enable_thinking`. The cloud adapter sends Republic `reasoning_effort: "none"`, and Republic translates it: OpenAI chat uses `reasoning_effort`; OpenAI responses, ChatGPT Codex, OpenRouter and DeepSeek responses use `reasoning.effort`; Anthropic and DeepSeek chat use `thinking: {"type": "disabled"}`; Gemini uses `thinkingBudget: 0`. A model that rejects the off switch fails the request. Gemini 2.5 Pro cannot disable thinking. Gemini 3 and 4 have no hard off switch; a lone zero budget may map to a thinking level. The ChatGPT Codex transport rejects token caps and uses the overall deadline. Native grammar/cache fields stay in the local adapter; cloud choices pass the same domain validator.
- Only the selector receives enabled context; the formatter sees the edited source and scene/script profile. A rewrite makes up to two provider calls, with no live cloud previews.
- The proposal is validated like a local one; failures deliver the local cleanup and record `unavailable` or `timeout`.
- Provider IDs are in `cloudpolish.KeyProviders` and `runtime.CloudKeyProviders`; display names are in `ui/shared/config`.

## Verification

`go test ./infrastructure/cloudpolish ./ui/app` runs against local fakes, including the ChatGPT flow on port 1455. `bootstrap` tests need macOS or Windows. Sign-in against the real services remains open (CLD-A05).

[Shared-pipeline verification](../../benchmarks/runs/unified-polishing-v1/README.md) records complete desktop/core checks and actual local model/worker runs. One context fixture repeatedly times out, and its isolated recheck fails session erasure. Real cloud service and microphone-to-editor acceptance remain open. The staged runner snapshots the shared `llm/rewrite` package along with its adapters.

The [OpenAI gateway integration run](../../benchmarks/runs/cloud-polishing-gateway-v1/README.md) exercises both `/v1/responses` and `/v1/chat/completions` over loopback HTTP through the production Republic transport and shared pipeline. Its 60 cases cover all context-switch combinations through coordinator delivery, stage isolation, final-only scheduling, single commit, rejected/truncated responses, transient HTTP retries, cancellation and deadline recovery. Run from the repository root:

```sh
go test -race ./apps/desktop/infrastructure/cloudpolish -run '^TestOpenAIGateway' -count=1 -timeout=120s
```

Only the remote gateway replies and the coordinator's audio/recognition/editor ports are synthetic. These tests establish protocol and orchestration behavior; they do not establish real cloud inference quality, service latency, OS context capture or microphone-to-editor acceptance.

[Main-branch integration verification](../../benchmarks/runs/cloud-polishing-main-v1/README.md) additionally checks the current source-tag prompt format and punctuation forwarding, retaining final-only timing for both engines.
