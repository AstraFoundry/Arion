# ADR 027: Speech & Polish settings and explicit cloud provider use

## Status

Accepted. Amends where [ADR 011](011-live-local-rewriting.md), [ADR 012](012-context-scene-selection.md) and [ADR 013](013-rewrite-context-on-device.md) place their controls (Settings › Model), and the activation rule of [ADR 022](022-cloud-polishing.md). Issue [#23](https://github.com/AstraFoundry/Arion/issues/23).

## Context

Settings › Model had grown into the settings of the whole path from speech to inserted text: recognition, polishing and where it runs, punctuation, Match the app and context. It kept the name of one part and an order shaped by when each feature arrived. Punctuation stayed visible but disabled while polishing was off.

Cloud setup had three levels: Model › Cloud model › provider › model. Opening a saved provider's page called `ActivateCloud`, so looking at a provider switched every following dictation to it, with no action and no confirmation. The model choice sat on the deepest page, and a new connection polished nothing until the user picked a model ID. ChatGPT (sign in) and the API key providers shared one list.

## Decision

- The section is **Speech & Polish** (**识别与润色**). Its ID stays `model`, so links that focus it keep working. The page follows the text's path: **Recognition**; **Polish** (the switch, Runs on, then the local status or the Cloud model row); **Formatting** (Punctuation, Match the app); **Context**. Formatting and Context show only while polishing runs.
- Cloud setup has two levels:
  - The list shows the provider **In use** first, with its account or key, its **Model** and Disconnect. Then come **Subscriptions** (ChatGPT) and **API keys** (OpenAI, Anthropic, Google Gemini, DeepSeek, OpenRouter).
  - A provider's page holds its sign-in or key form, or, for a saved provider, **Use**.
  - Opening a page never changes the provider in use. A saved provider is switched to only by its **Use** button, on its row or its page. Connecting a new provider is itself the action: it becomes the provider in use, and the page returns to the list.
- A provider connected without a chosen model starts on its fast tier, matched against its own model list (`cloudpolish.DefaultModel`):
  - GPT Luna, else mini, for ChatGPT and OpenAI.
  - Claude Haiku.
  - DeepSeek Flash, else `deepseek-chat`.
  - Gemini Flash-Lite, else Flash.
  - The same tiers under their vendor prefixes on OpenRouter.

  Among a tier's models, the newest version wins, and an undated alias wins over its dated snapshot. Preview, experimental, realtime, audio, image, search, transcription, embedding and vision variants are skipped. A chosen model is never replaced.
- No thinking or effort control is offered. Every request keeps thinking off (CLD-04), so the fast tier fits.
- Onboarding's cloud step reuses the pages. Once a provider is in use, it shows only that provider, with **Change provider**. Back leaves Change before the step.

## Consequences

- Model IDs are not pinned. A provider renaming a tier's model (`deepseek-v4-flash` → `deepseek-flash`) is followed without a release. A provider with no tier match leaves the choice to the user, as before.
- `ActivateCloud` keeps its contract; only the views changed. Removing a saved provider that is not in use still means using it, then Disconnect.
- Real cloud services remain unverified (CLD-A05). The tier choice is not a quality evaluation of those models.
