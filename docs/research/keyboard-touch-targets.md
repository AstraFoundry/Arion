# Keyboard touch targets

What decides which key a touch presses on the system keyboard, Rime frontends and other open-source keyboards, and what Arion's phone keyboard adopted (`KeyTargets`, `PinyinSpelling` in `ArionKit/Typing/KeyTargets.swift`). Sources were read for this record; nothing here was measured on Apple's keyboard.

## Apple

- [US 8,232,973](https://patents.google.com/patent/US8232973B2/en), figs. 8–9, separates the drawn keys from invisible hit regions. Likely continuations enlarge a region without redrawing it: after GO, a touch on the drawn F can select D. It also shows a touch between keys resolved to a key, and a preview on touch-down before selection on lift. Apple does not publish the shipping hit map or its parameters.
- A CHI 2018 study ("Typing on an Invisible Keyboard") reports touches landing below visual key centers, more on upper rows. It is not Apple data.

## Rime frontends and fcitx

| Project | Behavior (code) |
| --- | --- |
| Hamster / 仓输入法 ([`65693706`](https://github.com/imfuxiao/Hamster/tree/65693706d01fc6c19ed6071968e542b0a2ef3f36), `KeyboardButton.swift`, `KeyboardButton+Touch.swift`) | The drawing is inset inside the button, so the visual margin belongs to the key; preview on press, release checked with an outside tolerance |
| Trime (`KeyboardView.kt`) | Each key's bounds include half of each gap; edge keys extend to the border |
| fcitx5-android (`BaseKeyboard.kt` `findTouchTarget`) | Rectangular row and key bounds, per-pointer dispatch |
| fcitx5-ios (`uipanel/KeyModifier.swift`) | A new touch first sends older pending presses, so overlapping taps keep their order; the flushed key's release is not sent again |

None adjusts targets by what Rime expects next; librime's prism (`ExpandSearch`) can say which continuations are feasible, not how likely they are.

## Other keyboards

- AOSP LatinIME (`ProximityInfo`, `KeyDetector`): proximity search within 1.2 key widths, row-dependent touch-position correction in its spatial model, 8 dp hysteresis for modifiers, 0 dp vertical correction by default.
- FlorisBoard: keeps a moving touch on its key within the visible bounds widened by 10% of width and 35% of height per side; a second pointer releases the first popup key.
- AnySoftKeyboard: containment first, then the nearest eligible key within 1.4× the mean key size plus gap; older pending pointers release before a younger one.
- KeyboardKit: per-button press and release with a release tolerance; its `clearInteractable` color exists because a keyboard extension receives no touches on fully clear pixels.

## Arion

- Every pixel of a panel is touchable (a 1% fill, `KeyboardMetrics.touchableBackground`). Before it, taps in the 6 pt gaps and on the clear parts of the top icon keys were dropped.
- A touch that misses every key face goes to the nearest key: gaps split at their middle, the outer keys own the border, the last row owns the space below.
- Within 10% of a letter key's width from its edge toward a neighbor, the touch goes to the neighbor when only the neighbor's letter continues the input: Pinyin from the shipped syllable list (complete syllables, then the start of one; apostrophes separate), English from the desktop lexicon once the word has three letters. Elsewhere geometry wins, so the bias cannot be noticed as a jump.
- Feedback (haptic and click) comes on touch-down; a key types on release. When a second finger lands while a character key is held, the held key types at once and its release is skipped.
- Not adopted: a vertical offset (no shipping value is published; LatinIME defaults to none), probability weights (librime gives feasibility only), and a key preview bubble.
