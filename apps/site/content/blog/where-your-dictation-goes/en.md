---
title: Where your dictation goes
description: Most voice input tools stream audio to the cloud, and the OS permission prompt looks the same either way. Here is the local memory lifecycle behind a dictation in Arion, the three places that touch the network, and how to verify it with Wi-Fi off.
date: 2026-10-27
authors: stvlynn
category: product
tags: [privacy, offline, encryption]
image: /blog/where-your-dictation-goes/cover.webp
imageAlt: A motion-blurred photograph of pink and white flowers in green grass
---

When you grant an input method microphone and accessibility permissions, the operating system shows the same dialog regardless of how the app is built. Behind that prompt, voice input tools take two very different paths: most stream your microphone audio to a remote server for transcription, while a few run the models on your own machine.

Because a voice input method has an open microphone and access to the text field where your cursor sits, one question matters before any feature: during the few seconds you hold the shortcut and speak, where does the audio go, and what happens to the text it becomes?

In Arion, both stay on your computer. Here is the exact lifecycle on your device, and how to verify it yourself.

## The lifecycle of a dictation on your device

When you press <kbd>⌥</kbd> <kbd>Space</kbd>, audio capture, speech recognition and optional polishing all run inside local processes on your machine. Raw audio (PCM) is recorded into process memory only for the dictation in progress; as soon as the text is inserted, or if you cancel partway through, the buffer is dropped, leaving no temporary audio files on disk (`PRV-A03`). Recognition drafts and local rewrites stay in session memory as well and are never written to diagnostic logs. When window context is enabled, Arion reads bounded text from the focused control through the OS accessibility API rather than taking screenshots, and skips password fields and denylisted apps (`PRV-A05`).

<embed-frame src="/blog/where-your-dictation-goes/shortcut.html" title="Try the shortcut" ratio="16/7" caption="Press ⌥ Space in the frame, or click it, to start and stop a pretend dictation. Nothing is recorded."></embed-frame>

The few pieces of data that need to persist across sessions (words you add to your vocabulary, or word pairs Arion learns when you edit a mistyped word) are encrypted with AES-256-GCM before they touch disk:

<dictation-pill state="learned" term="Arion" label="Arion's dictation pill reporting that it learned the word Arion, with an undo button" caption="The pill after a correction. The learned word is stored on the device, encrypted."></dictation-pill>

The encryption key is generated on the device and stored in the login Keychain on macOS, or protected with DPAPI on Windows. If the key is missing or an integrity check fails, Arion fails closed rather than falling back to plaintext.[^keys]

Local diagnostic bundles (`Settings › About`) follow the same boundary: they log only random correlation IDs, coarse model IDs, numeric durations and error codes. Automated tests (`PRV-A02`, `PRV-A10`) inject synthetic transcript, prompt, path and token canaries during test runs and fail the build if any canary byte appears in logs or exported archives.

## The only three network requests

Once the local models are downloaded, dictation and Pinyin typing generate no network traffic (`PRV-A01`). Only three things in the application ever use the network, and none of them sends your audio:

| Request | What it sends | When |
| --- | --- | --- |
| Model download | A request for public model files, each verified against a pinned SHA-256 digest | Once, before first use |
| Update check | Only the selected channel (`stable` or `beta`), with no text, vocabulary or device identifier; archives are verified with an Ed25519 signature | Automatically, unless turned off |
| Sign-in and cloud polishing | Recognized text and enabled context, never audio | Only if you sign in or connect a provider |

Arion has no cloud speech recognition and no telemetry.

<aside>

**Sync is not available yet.** Its boundary is already defined in the specification: each record is encrypted on the device before upload, so the relay sees only ciphertext sizes and routing metadata, and only your devices hold the key (`PRV-A04`).

</aside>

## Verify it yourself

The point of an open-source local tool is that you do not have to take a blog post's word for it. Arion is licensed under AGPL-3.0; you can build it from a clean checkout:

```sh
git clone https://github.com/AstraFoundry/Arion.git
cd Arion
make install
make build desktop
```

Download the models once, turn off Wi-Fi (or block outbound traffic in your firewall), and dictate. If anything described here stops being true, it is a bug, and the [privacy page](/privacy/) and the specification[^spec] are updated in the same change set as the code.

[^keys]: A missing key or a failed integrity check is an error (`PRV-A06`). Arion never falls back to storing vocabulary in plaintext or silently replaces an unreadable key.

[^spec]: [Specification 005: Privacy and trust boundaries](https://github.com/AstraFoundry/Arion/blob/main/specs/005-privacy.md) in the repository.
