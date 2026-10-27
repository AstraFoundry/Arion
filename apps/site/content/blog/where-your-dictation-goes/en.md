---
title: Where your dictation goes
summary: Arion recognizes speech on your computer. Here is what stays on the device, the few things that use the network, and how to check it yourself.
date: 2026-10-27
category: product
cover: /backdrops/flowers-wide-1600.webp
coverAlt: A soft, motion-blurred photograph of falling flowers
---

Voice input hears everything you say while it is on. That makes one question matter more than any feature: where does the audio go? For Arion the answer is short. It stays on your computer, and so does the text it becomes.

## What stays on your device

When you press <kbd>⌥</kbd> <kbd>Space</kbd>, Arion records into memory, recognizes the speech with a model on your computer, and types the result where your cursor is. Audio is held only for the dictation in progress and dropped when it ends. Drafts and corrections are kept the same way.

<embed-frame src="/blog/where-your-dictation-goes/shortcut.html" title="Try the shortcut" ratio="16/7" caption="Press ⌥ Space in the frame, or click it, to start and stop a pretend dictation. Nothing is recorded."></embed-frame>

Words you teach Arion stay too. Correct a word it typed and it applies the fix next time, with an undo:

<dictation-pill state="learned" term="Arion" label="Arion's dictation pill reporting that it learned the word Arion, with an undo button" caption="The pill after a correction. The learned word is stored on the device, encrypted."></dictation-pill>

Approved and learned vocabulary is sealed with AES-256-GCM. The key is generated on the device and kept in the login Keychain on macOS, or with DPAPI on Windows.[^keys]

## What uses the network

Three things, and none of them carries what you said:

| Request | What it sends | When |
| --- | --- | --- |
| Model download | A request for public model files, each checked against a pinned SHA-256 digest | Once, before first use |
| Update check | No content or device identifier | Automatically, unless you turn it off |
| Sign-in and cloud polishing | Recognized text and enabled context, never audio | Only if you sign in or connect a provider |

After the model download, dictation needs no network at all. Arion has no cloud speech recognition and no telemetry.

<aside>

**Sync is not available yet.** Its design is fixed in the specification: each record is encrypted on the device before upload, and only your devices can read it.

</aside>

## Check it yourself

Arion is open source under AGPL-3.0, so you do not have to take this post's word for it. Build it from a fresh clone:

```sh
git clone https://github.com/AstraFoundry/Arion.git
cd Arion
make install
make desktop-build
```

Then turn off Wi-Fi and dictate. If anything here stops being true, it is a bug, and the [privacy page](/privacy/) and the specification[^spec] change in the same change set as the code.

[^keys]: A missing key or a failed integrity check is an error. Arion never falls back to storing vocabulary in plaintext.

[^spec]: [Specification 005: Privacy](https://github.com/AstraFoundry/Arion/blob/main/specs/005-privacy.md) in the repository.
