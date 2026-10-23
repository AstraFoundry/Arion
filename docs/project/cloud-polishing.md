# Cloud polishing

Optional polishing with a cloud model the user connects ([ADR 022](../decisions/022-cloud-polishing.md), [CLD-01–05](../../specs/009-contextual-rewrite.md#cloud-polishing)). The UI calls `runtime.Actions` and reads `runtime.State.Cloud`; it never imports `infrastructure/cloudpolish`. The composition root (`bootstrap`) owns the service and routes each rewrite request to the local worker or the cloud through `polishRouter`, by `Preferences.PolishEngine`.

## Where the user sets it up

One decision per page:

- **Onboarding.** The model step sets up recognition only. The **Polish** step offers Off, This device (with its download size) and Cloud. Choosing This device keeps the row compact and shows a primary yellow Download action at its trailing edge; selection alone never downloads or shows the local ready check. Confirming Download replaces the trailing action with Cancel download and smoothly expands progress inside the local option, before Cloud; cancellation does not occupy a separate row. Failures offer retry within the local option. Only a ready local worker replaces the action with a check; its progress panel then collapses. Continue waits for confirmation, then allows setup to continue while the confirmed download runs in the background. Cancelling turns polishing off; choosing local again requires confirmation. Only with Cloud, a **Cloud model** step follows: the provider list, then the chosen provider's page (Sign in with ChatGPT, or the masked API key and Connect), then, once connected, Disconnect and the model. Back leaves a provider page before the step; the footer says Skip until a model is chosen.
- **Settings › Model** keeps the settings on the section: a **Recognition model** row (opening the recognition model's page), then the Polish card with the polish switch, **Runs on** (This device / Cloud), **Punctuation** (a row like Runs on, enabled only while polish is on; English lists Smart, Complete, and Omit last mark, while Simplified Chinese also lists 空格代替标点; a saved spaces value stays until the user picks another mode), with Cloud one **Cloud model** row (the provider's mark and the model, opening the cloud pages), and Match the app; Context is its own group. Every explanation is a tooltip on the row's info hint. The title's back button steps out one page at a time.
- Providers are shown by their [LobeHub](https://icons.lobehub.com) marks.
- Enabling local polishing in Settings also offers Download before provisioning missing resources. Installed verified resources load automatically. Startup and switching back to local never resume an unconfirmed or cancelled download; queued work from an older choice is cancelled before it can download.
- The context switches (cursor, screen, history) hide while Cloud is chosen; the cloud never receives them. Match the app still chooses the scene locally.

## What views call

These return immediately; results arrive in `State.Cloud`.

| Method | Effect |
| --- | --- |
| `SignInChatGPT()` | Opens `auth.openai.com` in the browser; one callback on `127.0.0.1:1455`. Phase `signing_in`, then `connected` or back to the previous connection. |
| `ConnectCloudKey(provider, key)` | Lists the provider's models with the key (phase `verifying`); stores it only if that works. |
| `CancelCloud()` | Stops a sign-in or key check. |
| `DisconnectCloud()` | Deletes the credential item and forgets the connection. |
| `ListCloudModels()` | Lists the connected provider's models; views call it once when the rows show without a list. |

The chosen model is a preference (`UpdatePreferences` with `CloudModel`), applied to the service with `SetModel`.

## What views read

`State.Cloud`: `Phase` (`disconnected`, `signing_in`, `verifying`, `connected`), `Provider`, `Email` (ChatGPT only), `Models`, `Listing`, `Failure`. Failure codes (`runtime.CloudCodes`) equal `cloudpolish.Codes`; a test checks both lists. `CANCELLED` shows nothing; a failed model list offers **Retry**.

## Credential and startup

- One item in the OS credential store: service `Arion`, account `cloud-polish` (`keystore.OpenItem`). JSON with the provider and either the API key or the ChatGPT refresh token, account ID and email.
- At launch the service reads that item off the main thread, without network. The provider opens on first use: a ChatGPT connection trades its refresh token for an access token then and stores a rotated refresh token.
- Republic signs every request; ChatGPT requests go to the Codex backend with `originator: arion`. Requests use the system proxy through the default transport.

## Requests

- Final text only (`RewritePolicy.FinalOnly`), 15 s timeout, no context (`ContextPolicy` empty). The prompt is the shipped `live` template as chat turns (`prompt.Template.Turns`), versioned `cloud+<template>`.
- The proposal is validated like a local one; failures deliver the local cleanup and record `unavailable` or `timeout`.
- Provider IDs are in `cloudpolish.KeyProviders` and `runtime.CloudKeyProviders`; display names are in `ui/shared/config`.

## Verification

`go test ./infrastructure/cloudpolish ./ui/app` runs against local fakes, including the ChatGPT flow on port 1455. `bootstrap` tests need macOS or Windows. Sign-in against the real services remains open (CLD-A05).
