# ADR 051: Native iOS and Android voice keyboards

Status: Accepted. Changes the mobile scope of [ADR 021](021-apple-silicon-only-macos.md) and the [product scope](../project/README.md): iOS and Android apps now exist. The desktop decisions stand.

## Context

Arion's product scope kept mobile to portable contracts. Users dictate on phones as much as on computers, and the competitors studied in [research](../research/README.md) ship a phone keyboard as part of the product. A phone keyboard differs from the desktop in three ways that drive this decision:

- An iOS keyboard extension may not run AVAudioEngine reliably, and most competitors open their own app to record and ask the user to switch back. [stephancill/ios-keyboard-dictation](https://github.com/stephancill/ios-keyboard-dictation) records inside the keyboard through WebKit, which captures media in its own processes.
- Both platforms provide on-device speech recognition (Speech's `DictationTranscriber` on iOS 26; the on-device `SpeechRecognizer` on Android 13 and later) and an on-device language model (Apple's Foundation Models; Gemini Nano through ML Kit's Prompt API). Arion's own speech and rewrite runtimes are not packaged for phones.
- The desktop's MyGo shell does not target phones, and Go runtimes inside a keyboard extension compete for its small memory budget.

## Decision

- Arion ships a native iOS app with a custom keyboard extension (UIKit, Swift 6, iOS 26 or later) in `apps/ios`, and a native Android app with an input method service (Kotlin, Jetpack Compose, Android 13 or later) in `apps/android`. Each is written for its platform; neither embeds the Go core.
- Recognition uses the platform's on-device recognizer, and polishing uses the platform's on-device language model. Nothing is sent to a server, and no cloud engine is offered on mobile. When the model is missing or refuses, the rule-corrected transcript is delivered with an explicit notice ([009](../../specs/009-contextual-rewrite.md) LRW-08).
- The iOS keyboard captures the microphone with a hidden, nonpersistent `WKWebView` (`getUserMedia` on a secure placeholder origin) that posts 16 kHz PCM to the keyboard. This is the only web view in Arion's apps. It renders no content, loads no network resource, grants only the microphone, and stops itself when the keyboard's heartbeat stops for three seconds. It requires Full Access, which is requested in onboarding with the reason.
- Both platforms use the desktop's formatting recipe unchanged: `apps/desktop/infrastructure/llm/prompt/format.json` is bundled into each app, and each app builds requests the way `SelectedTemplates.Format` and `FormatTranslation` do, with the delivery check of `domain.ValidateRewriteDelivery` and the spacing rules of `domain.Correct`.
- Sunbeam reaches the apps through generated tokens. `apps/desktop/cmd/arion-mobile-tokens` reads the Sunbeam MyGo package that the desktop pins and writes Swift and Kotlin token files. The keyboard is a layer above the app, so its keys are raised; app pages and cards are flat ([ADR 049](049-sunbeam-package-and-materials.md)). Text uses the platform's system typeface at Sunbeam's sizes, with the system serif for headlines.
- User-facing copy for both apps lives in one catalog, `packages/mobile-strings/strings.json` (English and Simplified Chinese), from which the String Catalog and Android string resources are generated. `pnpm mobile:check` fails when generated files drift.
- Drafts stay in the keyboard's panel. The editor receives one insertion when a dictation ends, as on the desktop.

## Consequences

- The keyboard UI and the domain rules exist twice, in Swift and Kotlin. The rules ported are small (spacing, delivery validation, request building) and tested on both sides against the same recipe file; the speech-cleanup spans, contextual spelling selection, scene detection from app identity and edit learning are not ported.
- Recognition and polishing quality depend on the OS version, the device and its downloaded models, and cannot be pinned or evaluated like the desktop's bundles. Polishing is available only where Apple Intelligence or Gemini Nano is.
- ML Kit's GenAI APIs serve only the foreground app. An input method serves another app in front, so Android polishing may be refused on every device; the keyboard then inserts the rule-corrected text and says so. This is an open device verification.
- Every iOS voice keyboard studied (Typeless, Wispr Flow, Willow, Superwhisper) opens its containing app to record, and two say Apple requires it ([mobile voice keyboards](../research/mobile-voice-keyboards.md)). If WebKit capture fails on current devices, the fallback is that same app switch; it has not been built.
- The hidden web view adds WebKit's capture path to the iOS keyboard's failure modes (content-process termination, audio-session interruption); each ends the dictation with an explicit failure.
- Account, sync, cloud polishing, clipboard history, selection actions and the desktop's typing keyboards are not on mobile.

## Alternatives considered

- **Open the app to record, as most competitors do.** Every new session leaves the editor, and the return relies on the user swiping back. Rejected for the WebKit capture, which keeps the user in place.
- **Embed the Go core through gomobile.** One implementation of the rules, but a second runtime in a memory-limited extension and no access to the platform recognizers' streaming APIs. The shared parts are small; rejected.
- **Bundle Qwen3-ASR and the GGUF rewriter.** Pinned quality, but model sizes and memory exceed what a keyboard can hold. Rejected for now.
- **Cross-platform UI (Compose Multiplatform, React Native).** Rejected: each keyboard must feel native, and Arion does not use web UI frameworks.

## References

- [012 Mobile input](../../specs/012-mobile-input.md)
- [Mobile apps](../project/mobile.md)
- [Typeless iOS keyboard evidence](../research/typeless-ios-keyboard.md)
- [Mobile voice keyboards](../research/mobile-voice-keyboards.md)
