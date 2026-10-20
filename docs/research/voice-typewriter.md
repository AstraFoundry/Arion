# Voice typewriter

How competitors make streaming dictation flow smoothly, and what Arion adopted. Static reads of installed bundles; no runtime behavior was observed.

## Doubao (DoubaoIme 1.0.2, macOS)

`OimeAudio` ships `Classes/Core/UI/Typewriter/ASRTypeWriter.swift` and `TypewriterSpring.swift`. `ASRTypeWriter`'s fields (Objective-C metadata of the Swift class) are an access and a dispatch queue, an `interval`, a `callback`, a `spring`, `justDidReplace`, `currentResult` and a lazily computed `currentIndex`: it emits the current recognition result a little at a time. Its log lines include `[ASR][Typewriter] replace_decision replaceLength=…`, so a revised partial replaces part of what was already shown. The commit path names `drainingTypeWriter`, `hasFinishedTypeWriterReset` and `continueASRFinalCommitAfterTypeWriterReset`: the final commit waits for the typewriter to drain or reset. A separate `textWindowTypeWriter` drives its floating text window.

`TypewriterSpring`'s fields are `ARG_BALANCE_POSITION`, `ARG_ELASTIC_COEFFICIENT`, `ARG_DEFAULT_INTERVAL`, `ARG_MAX_INTERVAL`, `ARG_MIN_INTERVAL`, `ARG_TICK_INTERVAL`, `ARG_DEFAULT_VELOCITY` and `ARG_UTF8_COUNT`, each eight bytes, behind the `isTypewriterSpringEnable` preference. That is a spring on the typing interval around a balance backlog. The values are filled at runtime (likely remote configuration); no static defaults were found, so they remain unknown.

## Chatterfly (1.0.7, macOS)

The speech bar animates its own chrome (`SGSpeechBarBarBackgroundView expandAnimated:`/`collapseAnimated:`, `SGSpeechBarWarningView playAppearFadeInAnimation`, a `ghostFadeout` string). No per-character text animation was found by name.

## Arion

The same spring paces both voice surfaces: the dictation pill (`ui/entities/dictation/typewriter.go`) and the input method's marked text (`ime/macos/composition.m`, [ADR 006](../decisions/006-macos-input-method-host.md)). The backlog settles around 6 characters at 45 ms each; every character above or below that shortens or lengthens the interval by 25%, within 12–90 ms. A revision keeps the shown characters that still agree and types the rest. When recognition ends, the pill drains at 12 ms per character, while the input method's commit inserts the final text immediately. In the pill, each new character fades in over 180 ms. These values are Arion's own choice, not Doubao's.
