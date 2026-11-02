# ADR 040: Return account, polish and translation to onboarding

## Status

Accepted. Supersedes the Settings-only placement of account, polishing and translation in [ADR 039](039-focused-onboarding.md).

## Context

[ADR 039](039-focused-onboarding.md) shortened first-run setup so optional pages did not delay the first dictation. That cut also removed account sign-in, the polish choice and cloud configuration, and the translation switch and shortcut. Those three belong in the first-run flow. The combined practice page, the explicit recognition download, and the rule that sign-in does not grant usage consent should stay.

## Decision

Follow the sequence and gates in [DSK-14](../../specs/001-desktop-input.md).

The order is Welcome (with account sign-in), recognition download confirmation, polish choice, Microphone and Accessibility together, the polish shortcut with a practice field, then translation with its switch, shortcut, practice field and completion. Cloud setup opens inside the polish step when that engine is selected.

Both microphone and Accessibility stay on one page. Each row uses the settings permission row. After a grant, the row remains and shows the granted status. Continue locally on the welcome page moves on without a sign-in. When accounts are disabled, the welcome page omits sign-in. Choosing cloud polishing opens its setup inside the polish step, and that layer can be skipped before a model is connected. The polish shortcut page labels that shortcut and offers a practice field. Translation is the next page: the enable switch, and when it is on, the target language, shortcut and another practice field. It stays available when polishing is off, because a translation session forces the rewrite stage. That page also completes setup.

Account sign-in on the welcome page shows localized product terms before opening the browser. Google, GitHub and email share one sign-in; a custom issuer has one Sign in button. None of them grants usage consent. After sign-in, the page shows the same usage-sharing switch as Settings › Account.

Local polishing uses the Settings choice. Continue waits until a confirmed download is provisioning, loading or ready. Choosing cloud opens its setup in that same step. Formatting and context stay in Settings.

Practice and completion keep ADR 039's readiness gates. Losing readiness keeps the practice page and disables completion. Keyboard setup stays in Settings › Keyboard.

## Consequences

First-run setup again includes account, polish and translation without restoring the separate key-test page, the second translation page, or the usage-consent checkbox. Tests cover the skips, the download gates and the visible steps. Native OS checks remain necessary for actual editor insertion and permission dialogs.

## Alternatives considered

Restoring the previous thirteen-step flow would bring back the input-method page and the separate translation-key page. Those were the delay ADR 039 removed on purpose. Leaving the three steps only in Settings left first-run setup without the choices the user makes before dictating.
