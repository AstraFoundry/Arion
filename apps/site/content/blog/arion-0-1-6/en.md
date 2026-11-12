---
title: "Arion 0.1.5 to 0.1.6: what changed"
description: A second dictation key, polishing that says why it fell back, and copied files as an opt-in in 0.1.5. In 0.1.6, a bar that sends selected text to Claude or ChatGPT, image thumbnails in clipboard history, and steadier paste on Windows and in WeChat.
date: 2026-11-12
authors: stvlynn
category: announcement
tags: [release, changelog, dictation, selection actions, clipboard]
image: ./cover.webp
imageAlt: Arion's dark selection bar with Claude, ChatGPT, Copy, Add to vocabulary and an X. Below it, Claude's menu lists Translate to Chinese, Explain, Summarize key points, Rewrite and Ask a custom question…
draft: true
---

0.1.5 and 0.1.6 are the two updates since [Arion 0.1.4](/blog/arion-0-1-4/), for Apple silicon Macs and Windows x64. 0.1.5 adds a second dictation key, opens the microphone sooner, explains polishing fallbacks, lets you pin the candidate text color and makes copied files an opt-in. 0.1.6 adds selection actions, a first-run overview and image thumbnails in clipboard history, and fixes paste in slow Windows apps and after copying from WeChat on macOS.

If you are on an earlier version, open Settings › About and click **Check for Updates**.

## A second dictation key (0.1.5)

Settings › Dictation now has a **+** beside **Dictation key**. Click it, press the key you want, and either key starts dictation. The second key holds and taps like the first and starts ordinary dictation. **−** removes it.

<embed-frame src="/blog/arion-0-1-6/stage.en.html#shortcut" poster="./posters/shortcut.en.webp" title="Demo: adding a second dictation key" ratio="16/10" caption="Clicking + beside Dictation key and pressing Right Command. Right Option and Right Command now both start dictation. A mock recording."></embed-frame>

You can add one extra key, not more. A key that another shortcut already uses is rejected.[^shortcut]

## The microphone opens first (0.1.5)

When you pressed the key and started talking at once, the first words could fall into setup. Arion now opens the microphone first and queues what you say while it finds the text field and prepares the recognizer, then feeds that audio to the recognizer in order. Thanks to [@earmer](https://github.com/earmer) for [the change](https://github.com/AstraFoundry/Arion/pull/117).

<embed-frame src="/blog/arion-0-1-6/stage.en.html#capture" poster="./posters/capture.en.webp" title="Demo: speaking as soon as the key goes down" ratio="16/10" caption="Holding Right Option and speaking at once. The draft starts with the first word, and the end of a long draft stays at the pill's right edge. A mock recording: no sound is captured."></embed-frame>

A missing model still ends the session before the microphone opens. If setup fails or you cancel, the queued audio is dropped without being recognized, and so is audio for a password field.[^capture]

The same change fixes long drafts in the pill. In a long draft that mixed Chinese and Latin text, the end drifted left of the line's edge, because Arion mixed per-character widths with the width of the shaped line. The end of the draft now stays at the edge.

## Polishing says why it fell back (0.1.5)

When cloud polishing cannot be used and Arion inserts the unpolished text, the pill now shows a short note after the dictation completes: what happened and what to do next. Each note appears once per provider, model and kind of problem while Arion runs.

<embed-frame src="/blog/arion-0-1-6/stage.en.html#polish" poster="./posters/polish.en.webp" title="Demo: a polishing note and Allow reasoning when required" ratio="16/10" caption="A model refuses to polish without reasoning. The text is pasted unpolished, the pill says Model needs reasoning, and Allow reasoning when required is turned on in the cloud model's settings. A mock recording: no provider is contacted."></embed-frame>

Some models refuse a request with reasoning turned off, including newer Gemini models and gpt-5-mini on OpenRouter. The cloud model page has a new switch, **Allow reasoning when required**, off by default. When it is on and a model refuses, Arion retries with minimal reasoning, then low, and remembers the level that model accepted. This can make polishing slower for those models only. Thanks to [@earmer](https://github.com/earmer) for [this change too](https://github.com/AstraFoundry/Arion/pull/107).[^polish]

## Candidate text color (0.1.5)

The keyboard's candidate bar has a **Text color** setting under Settings › Keyboard. **Automatic** picks white or black for the highlighted candidate from the **Highlight color**: white where it keeps 3:1 contrast against the fill, black otherwise. The menu shows which one it chose, such as **Automatic (White)**.

<embed-frame src="/blog/arion-0-1-6/stage.en.html#textcolor" poster="./posters/textcolor.en.webp" title="Demo: choosing a highlight color and pinning the text color" ratio="16/10" caption="Choosing Purple as the highlight color turns the automatic text white. Choosing Black pins it, and the highlighted candidate follows. A mock recording."></embed-frame>

**Black** and **White** pin the color in both light and dark appearances. Choosing **Automatic** or another highlight color clears the pin. Thanks to [@cyrus-cai](https://github.com/cyrus-cai) for [adding it](https://github.com/AstraFoundry/Arion/pull/121).[^textcolor]

## Copied files are an opt-in (0.1.5)

Clipboard history now records copied files only when you turn on **Record copied files** in Settings › Clipboard. The switch is off by default, including on profiles from earlier versions, and turning it on asks you to confirm once with **Enable file history**.

<embed-frame src="/blog/arion-0-1-6/stage.en.html#files" poster="./posters/files.en.webp" title="Demo: turning on Record copied files" ratio="16/10" caption="Turning on Record copied files and confirming once. History keeps references to copied files; their contents are not uploaded. A mock recording."></embed-frame>

File records you already had are hidden while the switch is off, not deleted. History keeps references to the files rather than copies, and macOS may ask for access to the folders that hold them. Thanks to [@gavi102803](https://github.com/gavi102803) for [the change](https://github.com/AstraFoundry/Arion/commit/9fba8413).[^files]

## Cleaner Ctrl+V on Windows (0.1.5)

[@gialoc668](https://github.com/gialoc668) [reported](https://github.com/AstraFoundry/Arion/issues/110) that dictated text did not appear in Feishu and WeChat on Windows 11. One cause was keys still held during the paste: Windows does not release Shift, Alt or the Windows key for a posted Ctrl+V, so the chord could type a stray V or paste nothing. Arion now releases those keys first and posts Left Ctrl+V once. The pill says **Pasted** for a posted paste, because Arion cannot confirm that the app accepted it.[^paste]

## Selection actions (0.1.6)

Select text in any app and a small bar appears beside it, with the assistants you enabled, **Copy**, **Add to vocabulary** and an X. Rest the pointer on Claude or ChatGPT and its menu opens: **Translate to Chinese**, **Explain**, **Summarize key points**, **Rewrite** and **Ask a custom question…**, plus **Ask ChatGPT** under ChatGPT.

<embed-frame src="/blog/arion-0-1-6/stage.en.html#selection" poster="./posters/selection.en.webp" title="Demo: asking Claude to explain selected text" ratio="16/10" caption="Selecting a sentence, resting on Claude and choosing Explain. The Claude app opens with the prompt and the selection filled in, ready to send. A mock recording: nothing is sent to Claude."></embed-frame>

Choosing a prompt opens Claude through `claude://` or ChatGPT through `codex://`, which prefill a local chat with the prompt and your selection. You send it from there; Arion sends nothing itself. The prompts are built-in English instructions, and a Claude request is capped at 14,000 characters. Selection actions are off by default. Turn them on at the new last step of setup, **Act on selected text**, or under Settings › General, where you also choose Claude, ChatGPT or both.

Arion reads the selection without touching the clipboard: through Accessibility on macOS and UI Automation on Windows, never by posting Command-C. Password fields produce no bar, and the selected text is not written to the logs.[^selection]

The X opens a menu to stop the bar: **Disable for 1 hour**, on the current site, in the current app, or **Turn off selection actions**.

<embed-frame src="/blog/arion-0-1-6/stage.en.html#disable" poster="./posters/disable.en.webp" title="Demo: disabling the bar on one site" ratio="16/10" caption="The X opens the disable menu. After Disable on docs.example.com, selecting text on that site shows no bar. A mock recording."></embed-frame>

## A first-run overview (0.1.6)

Until your first dictation, the **Overview** page shows one card with a microphone and the key to hold, followed by the vocabulary card, in place of the totals, heatmap and scenes.

![Arion's Overview page before the first dictation: "No dictations yet. Say your first sentence." above a drawing of a microphone, the line "Speaking time, words and polishing scenes add up here as you dictate.", the hint "Hold in any app and speak" beside the Option key, and a Vocabulary card that says "No terms yet. Add names and terms Arion should spell your way."](./overview.en.webp "The Overview page on a fresh install. A render of the desktop UI.")

Settings, the menu bar panel and setup also use flatter surfaces in 0.1.6: cards have a hairline border and no shadow, and only menus and popups float.[^overview]

## Image thumbnails in clipboard history (0.1.6)

A copied PNG, JPEG, GIF, WebP or BMP file now shows a thumbnail beside its name in clipboard history, and resting on it shows the image larger. Thanks to [@gavi102803](https://github.com/gavi102803) for [adding it](https://github.com/AstraFoundry/Arion/commit/73b4f0b2).

<embed-frame src="/blog/arion-0-1-6/stage.en.html#thumbnails" poster="./posters/thumbnails.en.webp" title="Demo: an image file in clipboard history" ratio="16/10" caption="Opening clipboard history with Command-Shift-V. The copied beach.jpg shows its thumbnail, and resting on it shows the image larger. A mock recording."></embed-frame>

The entry is still a file reference: choosing it pastes the file, as before. An image that cannot be read shows no thumbnail and no error.[^thumbnails]

## Paste waits for slow Windows apps (0.1.6)

The same [Windows report](https://github.com/AstraFoundry/Arion/issues/110) had a second cause. Arion restored your previous clipboard 400 ms after the paste, and an app such as WeChat that read the clipboard later found the old contents: the field stayed empty while the pill said **Pasted**.

Arion now measures the field before pasting, checks it again after 300 ms and restores the clipboard at 600 ms. If the field has not changed, it checks again 500 ms later and restores at 1100 ms. If no check sees the text arrive, Arion leaves your dictated text on the clipboard and the pill says **Copied**, so pressing Ctrl+V pastes it. The issue is still open.[^hold]

## Paste after copying from WeChat on macOS (0.1.6)

After you copied text from a WeChat input box, the next dictation could fail with "Could not paste" and "The clipboard changed or was unavailable." [@huwan](https://github.com/huwan) [traced it](https://github.com/AstraFoundry/Arion/issues/122): WeChat adds two rich-text clipboard types whose names macOS cannot read, and Arion gave up its clipboard snapshot when one type returned no data. Arion now skips those types, and the paste goes through.[^wechat]

## Smaller fixes (0.1.6)

- A saved update that no longer matches the installed app, for example one prepared before you installed Arion again from the disk image, no longer stops the dictation key and the menu bar button from working. Arion forgets it and keeps dictation available.[^update]
- The menu bar panel no longer says "Ready" when everything is fine; it shows a state only when Arion is not ready.

## What isn't there yet

- On Windows, Arion cannot tell yet which app or site a selection came from, so the disable menu there has no site or app row.
- The Windows paste changes have passed portable tests but have not been tried by hand in WeChat, QQ or Feishu on Windows 11, so [issue #110](https://github.com/AstraFoundry/Arion/issues/110) stays open.
- Everything listed in [0.1.1's open items](/blog/arion-0-1-1/#what-isnt-there-yet) still applies: no multi-device sync, edit learning only in native macOS editors, and no Intel Mac support.

The full notes for each version are in the [desktop changelog](https://github.com/AstraFoundry/Arion/blob/main/apps/desktop/CHANGELOG.md). If something breaks, use **Report a problem**, open a [GitHub issue](https://github.com/AstraFoundry/Arion/issues), or find us on [Discord](https://discord.com/invite/72kh465mMm) or [Feishu](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=e05vef49-cce2-404f-9740-2fb99d17d444).

<github-people label="Contributors" logins="stvlynn, earmer, cyrus-cai, gavi102803"></github-people>

<github-people label="Issue reporters" logins="gialoc668, huwan"></github-people>

[^shortcut]: [Dictation shortcuts](https://github.com/AstraFoundry/Arion/blob/main/docs/project/dictation-shortcuts.md) and [Spec 001: Desktop input](https://github.com/AstraFoundry/Arion/blob/main/specs/001-desktop-input.md).

[^capture]: [Spec 001: Desktop input](https://github.com/AstraFoundry/Arion/blob/main/specs/001-desktop-input.md) (`DSK-08`, `DSK-A01`) and the Start step in [the architecture overview](https://github.com/AstraFoundry/Arion/blob/main/docs/project/architecture.md).

[^polish]: [Cloud polishing](https://github.com/AstraFoundry/Arion/blob/main/docs/project/cloud-polishing.md). Notes name a category of problem and never include your text.

[^textcolor]: [ADR 017: Shared offline Pinyin keyboard](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/017-offline-pinyin-keyboard.md).

[^files]: [Clipboard history](https://github.com/AstraFoundry/Arion/blob/main/docs/project/clipboard.md#file-access-guidance), [ADR 047: Clipboard file access guidance](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/047-clipboard-file-access-guidance.md) and [Spec 010: Clipboard history](https://github.com/AstraFoundry/Arion/blob/main/specs/010-clipboard.md) (`CLP-09`).

[^paste]: [ADR 003: Shortcut-mode insertion routes](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/003-insertion-routes.md) and [pull request #118](https://github.com/AstraFoundry/Arion/pull/118).

[^selection]: [Spec 011: Selection actions](https://github.com/AstraFoundry/Arion/blob/main/specs/011-selection-actions.md) (`SEL-02`, `SEL-03`) and [ADR 048: Selection action URLs](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/048-selection-action-urls.md).

[^overview]: The Overview section of the [design system](https://github.com/AstraFoundry/Arion/blob/main/docs/frontend/design-system.md#overview), [pull request #127](https://github.com/AstraFoundry/Arion/pull/127) and [ADR 049: flat and raised materials](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/049-sunbeam-package-and-materials.md).

[^thumbnails]: [Spec 010: Clipboard history](https://github.com/AstraFoundry/Arion/blob/main/specs/010-clipboard.md) (`CLP-06`).

[^hold]: [ADR 003: Shortcut-mode insertion routes](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/003-insertion-routes.md) and [pull request #126](https://github.com/AstraFoundry/Arion/pull/126). The field's length decides only when the clipboard is restored; it is not proof that the paste landed.

[^wechat]: [Pull request #123](https://github.com/AstraFoundry/Arion/pull/123).

[^update]: [Update lifecycle](https://github.com/AstraFoundry/Arion/blob/main/docs/project/update-lifecycle.md) and [pull request #137](https://github.com/AstraFoundry/Arion/pull/137).
