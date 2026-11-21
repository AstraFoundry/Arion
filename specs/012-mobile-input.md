# 012 Mobile input

The iOS keyboard extension and the Android input method ([ADR 051](../docs/decisions/051-native-mobile-keyboards.md)). Product behavior follows the desktop where a phone allows it; the differences are stated here. Implementation status is in [mobile apps](../docs/project/mobile.md).

## Requirements

- MOB-01: The keyboard offers one microphone, the language key when more than one recognition language is chosen, delete, space, return and, where the platform asks for it, the next-keyboard key. Tapping the microphone starts dictation; tapping again finishes. Holding the microphone dictates until release; releasing over cancel discards the dictation and releasing over translate translates it.
- MOB-02: While listening, the recognized text appears in the keyboard: finalized text in ink, the tail the recognizer may still revise muted. The editor receives nothing until the dictation ends, and then exactly one insertion at the caret focused at that time. A space is added between Latin words; none between CJK text.
- MOB-03: Cancel, dismissing the keyboard (on Android also leaving the field) or starting another dictation invalidates the dictation in flight. On iOS a caret move inside the same keyboard session keeps it, and the text goes to the caret focused when it ends. A late recognition or model result cannot insert text.
- MOB-04: Recognition runs on the device's on-device recognizer, for the chosen language. Audio never leaves the device. A missing model, an unsupported language, a busy recognizer, lost microphone and empty speech each end the dictation with their own message; a missing model starts its download.
- MOB-05: With polishing on (the default), the rule-corrected transcript is rewritten by the on-device model with the desktop's formatting recipe and scene rules, bounded to 15 seconds, and the output must pass the desktop's delivery check. If the model is unavailable, refuses, fails the check or times out, the rule-corrected transcript is inserted with the notice “Inserted without polishing.” Settings explains when the device cannot polish.
- MOB-06: Translate produces the chosen target language (the desktop's twelve targets) from the same recipe. When translation cannot run, the source-language text is inserted with a notice.
- MOB-07: The scene follows the field: a search field gets search finishing; an e-mail subject (Android) gets e-mail; everything else chat.
- MOB-08: Vocabulary terms bias recognition (`contextualStrings` on iOS, biasing strings on Android) and are kept as written by polishing. History is off by default, keeps at most 100 dictations on the device, and turning it off erases it. Android fields that request no personalized learning are never kept. Neither is included in device backups.
- MOB-09: iOS: dictation requires Full Access, explained in onboarding; without it the microphone shows how to turn it on. Microphone and speech permissions are requested by the app. Android: password fields disable the microphone; RECORD_AUDIO is requested by the app.
- MOB-10: Onboarding asks for one thing per step: what Arion does, the microphone, adding (and on Android selecting) the keyboard, then a practice field. Android skips steps already done when the user returns from Settings.
- MOB-11: Copy comes from `packages/mobile-strings` in English and Simplified Chinese. Colors, radii and spacing come from Sunbeam's generated tokens; keyboard keys are raised and app surfaces flat.

## Acceptance

| ID | Scenario and required result |
| --- | --- |
| MOB-A01 | Practice field → Arion keyboard → tap microphone → speak → draft visible in the keyboard and not in the field → tap again → one insertion |
| MOB-A02 | Hold microphone → speak → slide over cancel and release → nothing inserted; repeat, release over translate → the target-language text is inserted |
| MOB-A03 | Start listening → dismiss the keyboard (Android: or change fields) → the dictation ends; nothing arrives later |
| MOB-A04 | iOS without Full Access → tap microphone → Full Access message, no capture |
| MOB-A05 | Model unavailable or polishing refused → rule-corrected text inserted, notice shown |
| MOB-A06 | Silence → “Didn't catch that.”; no insertion |
| MOB-A07 | History off → dictate → nothing recorded; on → entry with copy and delete; off → entries erased |
| MOB-A08 | Android: Arion enabled and selected in Settings → return → onboarding advances to practice |
| MOB-A09 | `pnpm mobile:check` passes; `pnpm ios:test` and `pnpm android:test` pass |
