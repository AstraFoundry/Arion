# Typeless iOS keyboard

## Target and scope

App Store IPA `Typeless-2.11.0.ipa`, SHA-256 `07a252b50b15a7d22ca387c00634b04f889e61f15a96649c20381ed785cd83eb`: app `com.typeless.mobile` 2.11.0 (build 6, SDK iphoneos26.5), keyboard extension `com.typeless.mobile.keyboard`, Live Activity extension `com.typeless.mobile.dynamicisland`. The archive was extracted into the private evaluation workspace and read with `plutil`, `strings`, REA and a parser for the Hermes v96 string tables of both `main.jsbundle` files; nothing was executed. Everything below is **static evidence** of what the bundle contains, not verified runtime behavior. The full report is `.workspace/mobile/typeless/report.md` in the private repository.

## Observations

- **Architecture.** App and keyboard are React Native (Hermes bytecode 96). The keyboard bundle has about 34,600 strings; the host about 85,700.
- **Keyboard declaration.** `com.apple.keyboard-service`, principal class `keyboard.KeyboardViewController`, `RequestsOpenAccess = true`, `PrimaryLanguage = mis` (one keyboard for every language), `IsASCIICapable = false`. The keyboard plist has no microphone usage description; the host has one.
- **Recording in the app.** Host copy says the keyboard must switch apps to turn on the microphone and attributes this to Apple. The keyboard carries `typeless://` routes (among them `swipe-back`), and the host teaches the return gesture (“Swipe back to continue”, an explanation screen). The host declares `UIBackgroundModes` `audio`, imports `AVAudioSession` play-and-record categories, and offers how long the microphone stays available after a dictation: immediately, 10 seconds, 5 minutes, 12 hours or never. Picture-in-picture and Dynamic Island are offered as ways to keep the session.
- **Keyboard ↔ host.** All three binaries import the Darwin notification center (`TLDarwinNotificationModule`, `DarwinEventBusService`) and share the App Group `group.com.typeless`. No notification names or defaults keys were recovered as literals.
- **Recognition is cloud.** Neither binary links `Speech.framework`, SpeechAnalyzer or a Whisper runtime. The bundles name `api.typeless.com`, `/ai/voice_flow`, `/rt_voice_flow` and WebSocket audio chunk sending.
- **Keyboard UI.** The extension is a voice panel and a full Rime IME: eight schemas, including English QWERTY, Pinyin (full and 9-key), Zhuyin, Kana and Hangul. Voice gestures: “Tap to speak”, “Release to finish / cancel / translate”, “Swipe to translate”, and a second recorder for “Speak to edit”. States include preparing, session ending and ended, replacing text, translating, no Full Access, microphone denied or in use, interrupted by another app, no network, weekly limit reached and signed out.
- **Insertion.** Keyboard JS names `insertText`, `setMarkedText`, `deleteBackward` and `adjustTextPosition`, with undo and rewrite of the last transcript (`TranscriptTracker`).
- **Host.** Onboarding: add keyboard, Full Access (seven help steps), microphone, practice dictate, edit and translate, three survey questions, switch keyboard, invite code. History has cloud sync and retention choices; dictionary, per-app tone, referral, paywall with a weekly free-word cap.
- **Tracking.** `NSUserTrackingUsageDescription`, Adjust, Firebase and Sentry.

## Arion design choices

These are Arion decisions informed by, not copied from, the evidence ([ADR 051](../decisions/051-native-mobile-keyboards.md)):

- Arion records inside the keyboard through WebKit instead of the containing app, so dictation does not leave the editor. Typeless's copy claims Apple requires the switch; whether the WebKit path holds on current iOS devices is the main open verification in [mobile apps](../project/mobile.md#verification).
- Recognition and polishing stay on the device; Typeless sends audio to its servers.
- Tap and hold both dictate; holding offers cancel and translate as release targets ([012](../../specs/012-mobile-input.md) MOB-01). Voice editing of a selection and a typing layout are not implemented.
- The microphone is held only while dictating; there is no keep-alive setting.
- No accounts, quotas or tracking SDKs.

## Limitations

Native code was not decompiled (no Ghidra, Hopper or IDA available), and Hermes call sites were not recovered, so a string shows that text exists, not which function uses it. The app-opening API, the Darwin notification names, the audio session lifetime and the network protocol were not traced or observed.
