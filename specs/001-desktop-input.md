# 001 Desktop input

## Requirements

- DSK-01: Users MUST choose `shortcut` or `system_ime` mode. Changing mode cancels the current session and invalidates its generation.
- DSK-02: Shortcut mode MUST offer explicit toggle activation. An optional hold key, off by default, MAY dictate while held through a tested native key-down/key-up adapter (macOS listen-only event tap with Input Monitoring consent; Windows low-level keyboard hook); MyGo activation callbacks alone are insufficient. Holding past the threshold (300 ms) stops on release; a shorter tap keeps dictating until the next press; another key during a hold that started the session cancels it. A hold key that cannot listen MUST report why without disabling the toggle shortcut.
- DSK-03: The user MUST see recording, processing, candidate preview, permission failure and insertion failure states. Permission requests occur only for enabled capabilities.
- DSK-04: Capture starts only after microphone consent and ends on stop, cancellation, OS suspend or device removal. Quit releases microphone, hotkeys and owned IME composition.
- DSK-05: A target snapshot MUST include stable platform identity, editor capability, selection/range ownership, generation and sensitive-field status. The portable token is opaque; native handles stay in the adapter.
- DSK-06: Shortcut drafts stay in Arion's preview. Commit MUST revalidate target and selection and insert once. A focus change, uncertain write result or unsupported target MUST require user action rather than a second automatic attempt.
- DSK-07: IME mode MUST follow the host's composition/candidate lifecycle and release all owned ranges on cancellation/deactivation. See [007](007-ime-ranking.md).
- DSK-08: Block context extraction and memory learning from secure/password fields. If the adapter cannot establish field safety, it cannot read its content.
- DSK-09: Model missing/unsupported hardware/permission denied MUST be actionable explicit errors; never silently use cloud inference.
- DSK-10: The interface MUST offer English (default) and Simplified Chinese in settings and onboarding, persist the choice, and update existing windows without restarting recognition or altering user content. UI language MUST remain independent of recognition/model selection.
- DSK-11: On macOS, a packaged Arion asking for Accessibility or Input Monitoring MUST open that privacy pane in System Settings with a nonactivating guide under its window that offers Arion's app to drag into the list. The guide follows the window, shows only while System Settings is frontmost, and closes on consent (returning focus to the Arion window that asked), on its close button, or when System Settings quits. Microphone keeps the system prompt; an unbundled build keeps the system prompts. The guide never changes consent itself.

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
| DSK-A10 | Choose a hold key → hold, speak, release → final commit exactly once; tap → dictation continues until the next press; hold key + another key → session cancelled, nothing inserted; missing Input Monitoring on macOS → explicit status and a grant action, toggle shortcut unaffected |
| DSK-A11 | macOS, packaged app, consent missing → grant Accessibility (or Input Monitoring) → pane opens with the guide under it → switch apps: guide hides; return: it reappears → drag Arion into the list and switch it on → guide closes and the asking window is frontmost; quitting System Settings closes the guide |

Compatibility must be recorded per app/version/input mode on both OSes: native editor, browser editable field, Electron editor, office document, terminal, noneditable field and elevated Windows app. These are test categories, not established support claims. Never auto-send Enter or execute a terminal command.
