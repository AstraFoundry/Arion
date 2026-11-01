---
title: "Arion 0.1.1: our first release"
description: Arion 0.1.1 is available today for Apple silicon Macs and Windows x64. Dictation, polishing, translation, an offline Pinyin keyboard and clipboard history all run on your computer, open source under AGPL-3.0.
date: 2026-11-01
authors: stvlynn
category: announcement
tags: [release, dictation, pinyin keyboard, offline]
image: /blog/arion-0-1-1/cover.webp
imageAlt: A mock screen recording zoomed in on a chat message box, with Arion's dictation pill showing an English draft and a key overlay showing Right Option held
draft: true
---

Today we're releasing Arion 0.1.1, the first stable version of Arion. It runs on Apple silicon Macs with macOS 14 or later and on Windows x64, it is open source under AGPL-3.0, and it is free to download.

Arion is a local-first voice and Pinyin input method. Hold a key and speak, in Chinese, English or both in one sentence; your computer recognizes the speech, and when you let go the sentence is typed where your cursor is. The same app adds optional polishing and voice translation, an offline full-Pinyin keyboard, and encrypted clipboard history.

Between the first alpha and 0.1.1 we shipped 21 preview builds, and early users helped us tune shortcuts, the candidate bar, polishing rules and the update flow in the editors they actually use. 0.1.1 brings that work together into the first version we recommend for everyday use. Here is what it does, organized by what you'll use it for, followed by what it doesn't do yet.

<embed-frame src="/blog/arion-0-1-1/stage.en.html#dictation" title="Demo: holding Right Option to dictate into a chat box" ratio="16/10" caption="Hold Right Option in a chat box and speak. The draft stays in the pill; on release the sentence is typed once. A mock recording: no sound is captured."></embed-frame>

## Hold to talk, release to type

Anywhere you can type, hold the dictation key, speak, and let go. The default is Right Option on macOS and Right Alt on Windows. A short tap starts hands-free mode, which keeps listening until your next press; <kbd>Esc</kbd> cancels without typing anything. In Settings you can record any single key or key combination instead, and Arion tells you right away if it clashes with your translation or clipboard shortcut.

While you speak, the streaming draft appears only in the pill at the bottom of the screen; nothing touches your document yet. When you release the key, Arion types the final result once, at the cursor. If you switch windows, move focus or cancel in the meantime, the stale result is discarded instead of landing somewhere else. When nothing editable has focus, the result stays on the clipboard so you can paste it where you want.

Speech recognition uses Qwen3-ASR 1.7B: 8-bit through MLX on Apple silicon, and Q8_0 through speech.cpp on Windows, on the CPU or Vulkan. The model downloads once before first use (about 2.5 GB on a Mac, 2.2 GB on Windows), each file is checked against a pinned SHA-256 digest, and from then on dictation works offline. You can switch languages mid-sentence, as in "把这个 PR merge 到 main 之前，先跑一下 CI", without choosing a language first.

If you dictate quietly at a shared desk, Settings has a software input gain from −6 dB to +12 dB. In password fields, Arion reads no context and learns nothing.

## Polish after you speak

Polishing is off by default. When you turn it on, a local language model tidies each dictation after you release the key: it drops fillers like "um" and "so", keeps only the correction when you change your mind mid-sentence ("Wednesday, no wait, Thursday" becomes "Thursday"), and formats the text for the app you're in.

<embed-frame src="/blog/arion-0-1-1/stage.en.html#polish" title="Demo: dictating an email with polishing on" ratio="16/10" caption="Dictating an email with fillers and a correction. After release, the local model drops the struck-through words and adds a greeting and sign-off in email form. A mock recording."></embed-frame>

The local polishing model is Qwen3.5-2B (Q4_K_M GGUF, about 1.3 GB), run through llama.cpp on macOS and Windows. With **Match the app** on, Arion picks a scene for each dictation from the kind of field you're in, the web site's host and the app's identifier: chat, email, a work message, a prompt for an AI assistant, search, or a list. The scene sets the layout and leaves the content alone, and choosing it reads only that metadata, never the text in the field.[^scene]

Punctuation has four modes: smart, full, omitting the final period, and, in the Chinese interface, spaces in place of punctuation. You can also let polishing consult the text around your cursor and in the current window to get names and terms that sound alike right. Arion reads that text through the OS accessibility interface, takes no screenshots, and uses it only on the device.[^context]

Polishing runs once, after you let go. Earlier builds showed a live polished preview while you spoke, but in our measurements the final pass always replaced it and never changed what was typed, so we removed it and gave that compute back to recognition.[^final]

If you'd rather use a cloud model, sign in with ChatGPT or add an API key for OpenAI, Anthropic, Google Gemini, DeepSeek or OpenRouter. Arion sends a request only after you choose the cloud engine, connect a provider and pick a model, and it sends only the recognized text, the scene and any recalled vocabulary. Audio, cursor and window context, and your dictation history stay on the device. Each provider's key is kept separately in the OS credential store, so switching providers never overwrites one.

## Translate with a second key

Voice translation has its own shortcut: Right Command on macOS, Right Ctrl on Windows. Hold it and speak in your language; Arion recognizes what you said, fixes sound-alike words and proper nouns, and types the sentence in your target language.

<embed-frame src="/blog/arion-0-1-1/stage.en.html#translate" title="Demo: holding Right Command to answer in Chinese" ratio="16/10" caption="A teammate asks in Chinese. Hold Right Command and answer in English: the pill shows the English draft, and the Chinese translation is typed on release. A mock recording."></embed-frame>

There are 12 target languages: English, 简体中文, 繁體中文, 日本語, 한국어, Español, Français, Deutsch, Português, Italiano, Русский and العربية. Translation uses the same local or cloud model as polishing, so it runs even when polishing is off. If the model is missing, times out or returns output that fails validation, the translation fails with a notice in the pill and nothing untranslated is typed into the conversation; the source text stays only in that session's memory. Edits you make to a translation are not learned as corrections, so target-language spellings never leak into your vocabulary.

## Gets better as you use it

Vocabulary is organized by the correct spelling: each entry lists the wrong forms recognition tends to produce, and with the Arion input method installed you can edit the word's Pinyin reading directly. Entries that only give the correct spelling become hotwords for local speech recognition, and eligible learned corrections flow into both recognition and the Pinyin keyboard's vocabulary. New entries appear first.

In native macOS editors, if you fix a word Arion just typed, it learns the correction, uses your spelling next time, and offers an undo in the pill.

<dictation-pill state="learned" term="Arion" label="Arion's dictation pill reporting that it learned the word Arion, with an undo button" caption="After a correction, the pill reports the learned word, with a one-click undo."></dictation-pill>

Vocabulary, learned corrections and clipboard history are encrypted with AES-256-GCM before they reach the disk. Keys are generated on the device and kept in the Keychain on macOS or protected by DPAPI on Windows. If a key is missing or fails its integrity check, Arion stops with an error rather than falling back to plaintext.[^privacy]

## A Pinyin keyboard in the same input method

Arion includes an offline full-Pinyin input method: an IMK input method on macOS and a TSF input method on Windows, sharing one Rime schema, candidate ranking and learning logic. Tap <kbd>Shift</kbd> to switch between Chinese and English, and the badge by the cursor flips to show the mode. On macOS, the input menu also toggles full-width characters and English punctuation, and <kbd>Ctrl</kbd> <kbd>.</kbd> switches punctuation.

<embed-frame src="/blog/arion-0-1-1/stage.en.html#keyboard" title="Demo: Pinyin candidates reordered by context, then Shift to English" ratio="16/10" caption="Typing shi yan after 我们在婚礼上许下 (“we made at our wedding”). The candidate bar opens in its usual order, then local ranking moves 誓言 (“vows”) to the front; tapping Shift flips the badge from 中 to A. A mock recording."></embed-frame>

Candidate ranking pays attention to what you just wrote. The base layer only remembers which words you committed next to each other in the same app within the last 120 seconds, and one confirmation can move a candidate up by at most two places, so a single slip can't take over the first slot. Across 50 homophone pairs, from the second occurrence on, frequency learning alone put the intended word first 32.2% of the time; with these word pairings it was 88.7%.[^imk]

On top of that, you can turn on local neural ranking in Settings. It runs in two passes: a small model of about 4.5 MB reranks on every change to your input with a 100 ms budget, and after a 150 ms pause the local polishing model takes a second look using the text before and after the cursor and in the current window, with a 900 ms budget. The candidate bar always appears immediately in its native order, and both passes can only reorder existing candidates, never invent new ones. On macOS the context comes through the accessibility interface and is never read from password fields or denylisted apps; on Windows, ranking currently uses only text you recently committed.[^ranking]

The keyboard also completes English words and prefixes in Chinese mode, offers Emoji candidates for complete Pinyin words, and highlights the selected candidate in one of eight colors or a custom one. On macOS, the input menu has a **Voice input** item that starts dictation.

## Clipboard history

Clipboard history is off by default. Turn it on and Arion keeps the text, links, images and file references you copy. Press <kbd>⌘</kbd> <kbd>⇧</kbd> <kbd>V</kbd> (<kbd>Ctrl</kbd> <kbd>Shift</kbd> <kbd>V</kbd> on Windows) to open a compact picker at your cursor, where you can search and pin items, and hovering over an image or long text shows a preview. Press Return to paste the selection into the field you came from; when there is no field, it is copied instead.

<embed-frame src="/blog/arion-0-1-1/stage.en.html#clipboard" title="Demo: searching clipboard history and pasting a command" ratio="16/10" caption="Opening clipboard history at the cursor, hovering over a screenshot to preview it, then searching for make and pressing Return to paste the command into the document. A mock recording."></embed-frame>

For files, only the reference is kept, never the file's contents, and folders are skipped. By default the history holds 200 items and 100 MB in total; past that, the oldest unpinned items go first, and nothing expires with age. The whole history is encrypted with its own key. The clipboard Arion borrows to type a dictation never enters the history.[^clipboard]

## Updates

Every Arion update is signed with Ed25519 and verified on your computer before it installs. You get stable releases by default; prereleases arrive only after you turn on **Allow beta updates** in Settings › About. Automatic checking and automatic downloading are separate switches, and automatic downloading starts off. A downloaded update waits for you to choose Restart and never interrupts input in progress. When a new version is available, the reminder appears only in an idle moment after a dictation, and you can postpone or dismiss it; system notifications follow your OS settings.

## Where your data stays

Audio lives in memory only while you dictate and is released when you finish; it is never written to a temporary file. Recognition drafts and polished text are kept out of diagnostic logs.

Arion has no cloud speech recognition. Once the models are downloaded, dictation and Pinyin typing don't need the network. The only requests Arion makes are model downloads, update checks, cloud polishing if you choose it, and product notifications once you sign in to an account. If you sign in and explicitly agree, Arion also reports daily usage totals: the date, the number of dictations, microphone time and word count, with no recordings, text, scenes or app information. The full data flow is in [Where your dictation goes](/blog/where-your-dictation-goes/).

## Get started

Choose your platform on the [download page](/#download). The Mac version needs macOS 14 or later on Apple silicon; the Windows version is an x64 installer. On first launch, onboarding asks for the system permissions it needs, helps you set the dictation key, downloads the recognition model, and lets you turn on polishing and voice translation.

If you already run an alpha, check for updates in Settings › About; if 0.1.1 isn't offered, download and install it once by hand.

To build from source, clone the repository and run:

```sh
make install
make build desktop
```

## What isn't there yet

Every release note we've written lists its acceptance limits, and this one does too:

- **Multi-device sync hasn't shipped.** Its encryption boundary is in the spec: each vocabulary record is encrypted on your device before upload, and the relay only ever sees ciphertext. Desktop device enrollment isn't finished.
- **Learning from your edits works only in native macOS editors.** On Windows you can maintain vocabulary by hand.
- **Context ranking on the Windows keyboard uses only recently committed text**; it doesn't read the cursor or window yet.
- **The 10 translation targets other than English and Simplified Chinese have no evaluation of their own**; they share the English rules. Cloud polishing hasn't been benchmarked for quality or latency either.
- **Intel Macs are not supported** ([ADR 021](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/021-apple-silicon-only-macos.md)).

If something goes wrong, tell us in [GitHub Issues](https://github.com/AstraFoundry/Arion/issues), or join us on [Discord](https://discord.com/invite/72kh465mMm) or [Feishu](https://applink.feishu.cn/client/chat/chatter/add_by_link?link_token=640t6cde-d38e-4a81-875b-7799e44b77ae). How we decide what ships is in [Why we build our voice input method like an experiment](/blog/building-like-an-experiment/).

Thank you to everyone who filed issues, recorded reproductions and sent suggestions during the alpha.

[^scene]: Scene selection in [Spec 009: Contextual rewrite](https://github.com/AstraFoundry/Arion/blob/main/specs/009-contextual-rewrite.md#scene-selection) (`SCN-01`, `SCN-02`).

[^context]: [ADR 013: Rewrite context on device](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/013-rewrite-context-on-device.md).

[^final]: [ADR 023: Final-only polishing](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/023-final-only-polishing.md).

[^privacy]: [Spec 005: Privacy and trust boundaries](https://github.com/AstraFoundry/Arion/blob/main/specs/005-privacy.md) and [ADR 005: Local encryption](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/005-local-encryption.md).

[^imk]: Method and results are in the section "Measure quality and latency in the same run" of [Why we build our voice input method like an experiment](/blog/building-like-an-experiment/).

[^ranking]: [ADR 036: Two-stage local keyboard ranking](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/036-two-stage-keyboard-ranking.md) and [ADR 037](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/037-keyboard-context-and-developer-diagnostics.md). The 100 ms and 900 ms figures are configured budgets; the actual latencies have not been measured.

[^clipboard]: [Spec 010: Clipboard history](https://github.com/AstraFoundry/Arion/blob/main/specs/010-clipboard.md) and [ADR 034](https://github.com/AstraFoundry/Arion/blob/main/docs/decisions/034-clipboard-history.md).
