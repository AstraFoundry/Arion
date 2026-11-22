# ADR 052: Mobile account, overview and cloud polishing

Status: Accepted. Amends [ADR 051](051-native-mobile-keyboards.md): the mobile apps gain the desktop's account, Overview and cloud polishing. Recognition stays on the device.

## Context

ADR 051 kept the first mobile apps to dictation: onboarding for permissions and the keyboard, settings, and on-device polishing only. Compared with the desktop, the apps had no Overview of what the user has dictated, no AstraFoundry account, and no way to polish with a model the user chose. Phones without Apple Intelligence or Gemini Nano could not polish at all.

## Decision

- **Overview.** Both apps open on an Overview built from per-day counts, as the desktop's: time dictating, words, words polished, average speed, daily words for recent weeks, words by scene, and recent vocabulary. Before the first dictation it shows a first-run card. The keyboard records each committed dictation as numbers and a scene name only ([005](../../specs/005-privacy.md)); no text is kept for it.
- **Account.** The apps sign in to AstraFoundry with the desktop's issuer, scopes and resource, as public clients `arion-ios` and `arion-android` with the redirect `arion://auth/callback` (authorization code with PKCE in the system browser sheet: `ASWebAuthenticationSession`, Custom Tabs). Sign-in is optional. Onboarding offers it with the terms checkbox, the default-off usage-sharing checkbox ([ADR 044](044-onboarding-consent.md)) and **Continue locally**. Signed in, the apps show the profile, the usage-sharing switch, the account portal links and Sign out, and report consented daily usage as the desktop does. Sync is not on mobile.
- **Cloud polishing.** Polishing and translation can run on a cloud model the user connects with an API key: OpenAI, Anthropic, Google Gemini, DeepSeek, OpenRouter, and OpenAI- or Anthropic-compatible endpoints ([ADR 022](022-cloud-polishing.md)). The ChatGPT subscription sign-in is desktop-only, because its callback is a fixed loopback port. Requests carry the rule-corrected transcript and the shared recipe, with thinking off and the 15-second bound. A failure inserts the rule-corrected text with the existing notice. Keys are in the iOS keychain group shared with the keyboard, or encrypted with an Android Keystore key.
- **Onboarding.** One decision per step, with progress shown: welcome (with the account), microphone, keyboard, polishing (this device or cloud), and practice.

## Consequences

- The keyboard sends the transcript over the network when, and only when, the user chose a cloud model; on iOS this uses the Full Access the keyboard already needs. Audio never leaves the device.
- Real sign-in needs `arion-ios` and `arion-android` registered at the issuer with the `arion://auth/callback` redirect. Until then the flow reaches the issuer and fails there with the authorization notice.
- The account, consent, usage-report and cloud adapters exist in Swift and Kotlin as well as Go. Their pure parts are tested on each platform against the desktop's rules.

## Alternatives considered

- **Reuse `arion-desktop` with an added redirect.** One client registration, but the desktop's client would accept a custom-scheme redirect that any app could claim. Rejected for per-platform clients.
- **Keep mobile on-device only.** Rejected by the product owner: phones without an on-device model could not polish, and the apps would differ from the desktop.

## References

- [012 Mobile input](../../specs/012-mobile-input.md)
- [Account sign-in](../project/account-signin.md), [cloud polishing](../project/cloud-polishing.md)
- [Mobile apps](../project/mobile.md)
