# 001 Desktop input

## Requirements

- DSK-01: Users MUST choose `shortcut` or `system_ime` mode. Changing mode cancels the current session and invalidates its generation.
- DSK-02: Shortcut mode MUST offer explicit toggle activation first. Hold-to-talk requires a tested native key-down/key-up adapter; MyGo activation callbacks alone are insufficient.
- DSK-03: The user MUST see recording, processing, candidate preview, permission failure and insertion failure states. Permission requests occur only for enabled capabilities.
- DSK-04: Capture starts only after microphone consent and ends on stop, cancellation, OS suspend or device removal. Quit releases microphone, hotkeys and owned IME composition.
- DSK-05: A target snapshot MUST include stable platform identity, editor capability, selection/range ownership, generation and sensitive-field status. The portable token is opaque; native handles stay in the adapter.
- DSK-06: Shortcut drafts stay in Arion's preview. Commit MUST revalidate target and selection and insert once. A focus change, uncertain write result or unsupported target MUST require user action rather than a second automatic attempt.
- DSK-07: IME mode MUST follow the host's composition/candidate lifecycle and release all owned ranges on cancellation/deactivation. See [007](007-ime-ranking.md).
- DSK-08: Block context extraction and memory learning from secure/password fields. If the adapter cannot establish field safety, it cannot read its content.
- DSK-09: Model missing/unsupported hardware/permission denied MUST be actionable explicit errors; never silently use cloud inference.
- DSK-10: The interface MUST offer English (default) and Simplified Chinese in settings and onboarding, persist the choice, and update existing windows without restarting recognition or altering user content. UI language MUST remain independent of recognition/model selection.

## Acceptance

| ID | Scenario and required result |
| --- | --- |
| DSK-A01 | Fresh install → permissions denied → clear status; no microphone or target read |
| DSK-A02 | Toggle record → receive revised preview → stop → final commit exactly once |
| DSK-A03 | Start in editor A → focus editor B → stop → neither receives an unreviewed insertion |
| DSK-A04 | Escape/cancel during ASR or correction → late results cannot commit or resurrect composition |
| DSK-A05 | Existing clipboard changes during explicit paste → do not overwrite the new clipboard or retry insertion |
| DSK-A06 | Switch modes during capture → old generation closed; new mode requires a new session |
| DSK-A07 | Secure field or unknown field safety → no context, no automatic memory learning |
| DSK-A08 | Sleep, microphone unplug, hotkey conflict or adapter loss → capture stops with explicit reason |
| DSK-A09 | Choose Simplified Chinese → settings/onboarding/panel/overlay and native window titles update → restart retains the choice → switch back to English; existing drafts, forms and recognition preferences remain intact |

Compatibility must be recorded per app/version/input mode on both OSes: native editor, browser editable field, Electron editor, office document, terminal, noneditable field and elevated Windows app. These are test categories, not established support claims. Never auto-send Enter or execute a terminal command.
