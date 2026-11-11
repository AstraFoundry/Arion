# 011 Selection actions

The selection popover's data, URL rules, native read/copy/vocabulary path, the popover view, its onboarding page and its settings card. [ADR 048](../docs/decisions/048-selection-action-urls.md) owns the URL boundary.

## Requirements

SEL-01: Selection preferences live on the existing preferences record. `Enabled` is off when missing, including profiles written before the field existed. The two built-in clients are `claude` (`claude://claude.ai/new?q={{text}}`) and `chatgpt` (`codex://new?prompt={{text}}`). Both stay off until enabled. A nil action list means the built-in list, in menu order: translate-zh, explain, summarize, rewrite, ask, ask-chatgpt, copy, add-vocabulary. Titles are not stored. Normalize drops an unknown client, resets a template that is not the built-in for that id, and keeps `PreferencesVersion` at 1.

SEL-02: `{{text}}` is the encoded prompt or the raw selection. Prompt bodies are the built-in English instructions. A client action sends the selection unchanged. The filled URL is percent-encoded. Only `https`, `claude`, and `codex` are allowed, and an `https` URL must not carry userinfo. `http`, `javascript`, `file`, `chatgpt`, `doubao`, `feishu`, and any extra `{{...}}` placeholder are refused. The Claude query is capped at 14,000 Unicode characters before encoding. ChatGPT is not capped. Filling a URL does not launch a process, send a key, or run a shell or Apple event.

SEL-03: A mouse-up reads the focused selection only while the master switch is on. macOS uses the existing Accessibility selected-text read. Windows uses UI Automation `TextPattern.GetSelection` and returns that text. The dictation hold key and the Windows visible-range script are not this reader. A secure field produces no offer and no text. An empty selection produces no offer. The selected text is not logged. A permission denial may be recorded as `selection_watch` with a stable code only.

SEL-04: Copy writes the captured selection with the existing clipboard writer. Vocabulary saves it with `AddHotword`. Both refuse a secure field before writing. The action rechecks the field at perform time and still writes the captured offer text.

SEL-05: Launch uses the operating system's URL open (`/usr/bin/open` on macOS, `ShellExecuteW` on Windows) on a URL that already passed SEL-02. The opener's error does not include the URL.

SEL-06: The popover renders the offer's actions as given and builds no URL. It is a dark menu in both appearances. Prompt rows come first in offer order; the first one is highlighted and carries a Return glyph named Return. Up and Down move the highlight, and Return runs the highlighted row, or copy when no prompt row is shown. `ask` is a question field; submitting a non-empty question opens that action with the question. A bottom row holds the client actions (Ask ChatGPT), copy as a glyph named Copy, and Add to vocabulary. Prompt and client rows are absent when the offer has none, which is the case when no client is enabled. A failed action shows its reason in the menu: an invalid hotword shows the vocabulary term rule, a secure field the secure-field message.

SEL-07: Onboarding ends with a selection page after translation, where the platform has a selection reader (DSK-14). Its master switch starts off, and the Claude and ChatGPT rows appear beneath it once it is on. Leaving the switch off is a valid answer: Finish stays enabled and completes setup. Settings › General carries the same switch and client rows. Both write `Preferences.Selection.Enabled` and the client flags through `UpdatePreferences`. The ChatGPT row states that it prefills a local chat through `codex://`.

## Not yet wired

Bootstrap does not yet host the popover in a window beside the selection. The view is `uiapp.Selection`, sized by `SelectionWidth` and `SelectionHeightFor`.

## Acceptance

- SEL-A01: Defaults stay off. An old preferences file without `selection` stays off. Normalize repairs a foreign client id and a non-built-in template without changing the preferences version.
- SEL-A02: Encoding, the Claude length cap, a refused scheme, and a missing client fail in portable tests. Those tests do not launch Claude, ChatGPT, Doubao, or Feishu.
- SEL-A03: On macOS, with `ARION_E2E_SELECTION=1` and existing Accessibility access, the production reader returns a known selection from the owned editor, copy leaves it on the clipboard, vocabulary reopens with that hotword, a secure field produces no offer and no entry, the client URL equals the filled template, and the master switch calls no reader. Missing access fails the test. The popover action list is not asserted.
- SEL-A05: `go test ./ui/app` renders, in both locales, a fresh profile reaching the selection page as the last step with the switch off and Finish enabled, Finish completing setup with the switch left off, the page omitted without a reader, the menu highlighting `translate-zh` with the Return glyph and running it on Return, the `ask` field, Ask ChatGPT, copy and vocabulary, the prompt and client rows absent when no client is enabled, and the General card writing `Selection.Enabled` and a client flag.
- SEL-A04: On Windows, the same outcomes use `GetSelection` text from the owned editor. The test does not post Ctrl+C and does not launch a client.
