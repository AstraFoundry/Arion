# Mobile voice keyboards

How other voice keyboards on iOS and Android start dictation, what their keyboards show, and what the platforms allow an input method to do. [Typeless iOS keyboard](typeless-ios-keyboard.md) covers Typeless separately. Full reports, with artifact hashes, are under `.workspace/mobile/competitors/report.md` and `.workspace/mobile/android/report.md` in the private evaluation repository.

## Sources and limits

- **iOS IPAs** (free App Store downloads, static inspection, nothing executed): Wispr Flow 2.0.4 (`com.wispr.flowapp`), Willow 2.13.3 (`com.willowvoice.ios`), Superwhisper 3.5.1 (`com.superduper.superwhisper-ios`). All three keyboard executables are FairPlay-encrypted, so quotes come from unencrypted `.strings`, plists, Settings bundles and asset names. No Darwin notification names were recovered.
- **Mobbin** screens of the Wispr Flow iOS setup flow and voice recording states in other apps.
- **Android**: official Android and ML Kit documentation and store listings. No APK was inspected, and nothing was run on a device.

These are static and documentary observations, not verified runtime behavior.

## iOS

| | Wispr Flow | Willow | Superwhisper |
| --- | --- | --- | --- |
| Microphone | Opens the app; copy says Apple requires the switch, then “Swipe back” | Host-only microphone description; opens Willow | Opens the app or uses the Action Button / Control Center; “Swipe to return to the keyboard” |
| Recognition | Cloud, with the phone's on-device English model as backup | Links ggml Whisper in the keyboard | Cloud or local models in the app; keyboard text arrives when recording ends |
| Session | “Until you quit the app” | Not in clear text | Session timeout, “release mic immediately”, and a control to end the audio session |
| Typing layout | Optional “ABC” keyboard | KeyboardKit linked | System layout family requested |
| Live Activity | Required to start, by its error copy | Extension present | Declared; user asked to enable it |
| Own zh-Hans | No | No | No (Japanese only) |

Shared patterns:

- All three, like Typeless, need Full Access, share an App Group with the app, and keep the system keyboard height.
- Keyboard states are short words: idle, recording, processing, error, no Full Access, session ended, microphone busy, nothing heard.
- Wispr's listening keyboard (Mobbin) shows cancel on the left, confirm on the right, “Listening” and a dotted level meter between them, and the globe below. Its setup flow is: welcome, microphone, keyboard, a practice field with the keyboard open, done.
- Accounts, weekly word caps, writing-style drawers, notetaker banners and dictionaries live in the host apps, and some leak onto the keyboard (Wispr's notetaker and quota states).

Superwhisper's keyboard links WebKit; whether it captures through it was not determinable from the encrypted binary.

## Android

- **Recognizer.** `SpeechRecognizer.createOnDeviceSpeechRecognizer` captures the microphone itself; the IME needs `RECORD_AUDIO`, requested from an activity. On-device language packs are per device; `zh-CN` must be checked with `checkRecognitionSupport` and may need `triggerModelDownload`.
- **Own capture.** An IME that records with `AudioRecord` needs a `microphone` foreground service, and the documentation does not list a visible input method among the while-in-use exemptions. Arion does not record itself on Android.
- **ML Kit GenAI.** Inference is allowed only for the top foreground app and returns `BACKGROUND_USE_BLOCKED` otherwise, including from a foreground service; the documentation does not mention input methods. The Rewriting API supports English, Japanese, French, German, Italian, Spanish and Korean, not Chinese; the Prompt API publishes no language list.
- **Competitors** (store listings and press): Typeless for Android is a microphone-only IME plus an accessibility bubble; Wispr Flow is a floating bubble, not an IME, and needs a network; Gboard puts the microphone above its keys; Doubao's input method is a full keyboard with tap and long-press voice and a separate cleanup pass.

## Arion design choices

- The keyboard keeps one microphone, the system height, short state messages, an explicit end and a single insertion ([012](../../specs/012-mobile-input.md)). Accounts, quotas and settings stay out of it.
- iOS captures inside the keyboard through WebKit instead of switching apps ([ADR 051](../decisions/051-native-mobile-keyboards.md)). Every product studied switches apps, so this remains the main device verification.
- Android polishing falls back to the rule-corrected transcript with a notice when ML Kit refuses, which the documentation suggests may happen for every input method, and for Chinese text.
- Copy ships in English and Simplified Chinese; none of the iOS competitors studied ship Simplified Chinese keyboard copy.
