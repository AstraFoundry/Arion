# 011 Selection actions

The selection popover's data, URL rules, and native read/copy/vocabulary path. The popover, onboarding page, and settings card are not part of this contract yet. [ADR 048](../docs/decisions/048-selection-action-urls.md) owns the URL boundary.

## Requirements

SEL-01: Selection preferences live on the existing preferences record. `Enabled` is off when missing, including profiles written before the field existed. The two built-in clients are `claude` (`claude://claude.ai/new?q={{text}}`) and `chatgpt` (`codex://new?prompt={{text}}`). Both stay off until enabled. A nil action list means the built-in list, in menu order: translate-zh, explain, summarize, rewrite, ask, ask-chatgpt, copy, add-vocabulary. Titles are not stored. Normalize drops an unknown client, resets a template that is not the built-in for that id, and keeps `PreferencesVersion` at 1.

SEL-02: `{{text}}` is the encoded prompt or the raw selection. Prompt bodies are the built-in English instructions. A client action sends the selection unchanged. The filled URL is percent-encoded. Only `https`, `claude`, and `codex` are allowed, and an `https` URL must not carry userinfo. `http`, `javascript`, `file`, `chatgpt`, `doubao`, `feishu`, and any extra `{{...}}` placeholder are refused. The Claude query is capped at 14,000 Unicode characters before encoding. ChatGPT is not capped. Filling a URL does not launch a process, send a key, or run a shell or Apple event.

SEL-03: A mouse-up reads the focused selection only while the master switch is on. macOS uses the existing Accessibility selected-text read. Windows uses UI Automation `TextPattern.GetSelection` and returns that text. The dictation hold key and the Windows visible-range script are not this reader. A secure field produces no offer and no text. An empty selection produces no offer. The selected text is not logged. A permission denial may be recorded as `selection_watch` with a stable code only.

SEL-04: Copy writes the captured selection with the existing clipboard writer. Vocabulary saves it with `AddHotword`. Both refuse a secure field before writing. The action rechecks the field at perform time and still writes the captured offer text.

SEL-05: Launch uses the operating system's URL open (`/usr/bin/open` on macOS, `ShellExecuteW` on Windows) on a URL that already passed SEL-02. The opener's error does not include the URL.

## Deferred

The popover, its action list, the onboarding step, and the settings card are intentionally absent. Onboarding order in DSK-14 stays unchanged until that page exists.

## Acceptance

- SEL-A01: Defaults stay off. An old preferences file without `selection` stays off. Normalize repairs a foreign client id and a non-built-in template without changing the preferences version.
- SEL-A02: Encoding, the Claude length cap, a refused scheme, and a missing client fail in portable tests. Those tests do not launch Claude, ChatGPT, Doubao, or Feishu.
- SEL-A03: On macOS, with `ARION_E2E_SELECTION=1` and existing Accessibility access, the production reader returns a known selection from the owned editor, copy leaves it on the clipboard, vocabulary reopens with that hotword, a secure field produces no offer and no entry, the client URL equals the filled template, and the master switch calls no reader. Missing access fails the test. The popover action list is not asserted.
- SEL-A04: On Windows, the same outcomes use `GetSelection` text from the owned editor. The test does not post Ctrl+C and does not launch a client.
