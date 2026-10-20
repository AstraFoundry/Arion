# Voice typewriter

How competitors make streaming dictation flow smoothly, and what Arion adopted. Static reads of installed bundles; no runtime behavior was observed.

## Doubao (DoubaoIme 1.0.2, macOS)

`OimeAudio` ships `Classes/Core/UI/Typewriter/ASRTypeWriter.swift` and `TypewriterSpring.swift`. `ASRTypeWriter`'s fields (Objective-C metadata of the Swift class) are an access and a dispatch queue, an `interval`, a `callback`, a `spring`, `justDidReplace`, `currentResult` and a lazily computed `currentIndex`: it emits the current recognition result a little at a time. Its log lines include `[ASR][Typewriter] replace_decision replaceLength=…`, so a revised partial replaces part of what was already shown. The commit path names `drainingTypeWriter`, `hasFinishedTypeWriterReset` and `continueASRFinalCommitAfterTypeWriterReset`: the final commit waits for the typewriter to drain or reset. A separate `textWindowTypeWriter` drives its floating text window.

`TypewriterSpring`'s fields are `ARG_BALANCE_POSITION`, `ARG_ELASTIC_COEFFICIENT`, `ARG_DEFAULT_INTERVAL`, `ARG_MAX_INTERVAL`, `ARG_MIN_INTERVAL`, `ARG_TICK_INTERVAL`, `ARG_DEFAULT_VELOCITY` and `ARG_UTF8_COUNT`, each eight bytes, behind the `isTypewriterSpringEnable` preference. That is a spring on the typing interval around a balance backlog. The values are filled at runtime (likely remote configuration); no static defaults were found, so they remain unknown.

## Chatterfly (1.0.7, macOS)

The speech bar animates its own chrome (`SGSpeechBarBarBackgroundView expandAnimated:`/`collapseAnimated:`, `SGSpeechBarWarningView playAppearFadeInAnimation`, a `ghostFadeout` string). No per-character text animation was found by name.

## Arion

The same spring paces both voice surfaces: the dictation pill (`ui/entities/dictation/typewriter.go`) and the input method's marked text (`ime/macos/composition.m`, [ADR 006](../decisions/006-macos-input-method-host.md)). The backlog settles around 6 characters at 45 ms each; every character above or below that shortens or lengthens the interval by 25%, within 12–90 ms. When recognition ends, the pill drains at 12 ms per character, while the input method's commit inserts the final text immediately. These values are Arion's own choice, not Doubao's.

Arion's draft is not append-only: closed segments are re-decoded, a live rewrite can restate them, and the streaming tail is revised. An early version rolled the display back to the last agreeing character and retyped the rest, so a rewrite of an early word erased most of the line and faded it in again; users saw the text flash and step backward. Shown characters now follow each revision in place; only new characters are typed, and only they fade (from 35% to full over 150 ms). A revision shorter than what is shown cuts the display to it.

In the pill, a long line glides left with a 180 ms spring (lagging its target by at most one character) instead of jumping a character at a time, and characters crossing the left edge fade out over 28 pt. Those edge characters are drawn as single-character spans at cached single-character widths, so their fades sit exactly on their glyphs; a span with zero alpha would draw in the full text color, so the faintest keeps 1%. The live phases share one row, so the draft no longer leaves and re-enters when speech ends, and its color eases to the muted tone.
