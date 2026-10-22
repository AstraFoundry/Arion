# Design system

Arion's native UI uses Sunbeam, the system it shares with Lyra, rendered with MyGo and the coss ui component anatomy ported to Go ([ADR 007](../decisions/007-sunbeam-design-system.md)). Physical values live only in `ui/shared/theme`; views use the semantic tokens and the `ui/shared/kit` primitives.

## Color

One accent, warm neutrals, and semantic colors only where state differs.

| Token | Light | Dark | Use |
| --- | --- | --- | --- |
| `Canvas` | ink 50 `#faf8f5` | `#090706` | Window background |
| `Card` | white | ink 800 | Grouped surfaces |
| `CardBorder` / `Hairline` | ink 200 / ink 100 | `#322d29` / `#2a2622` | Card and control edges / dividers inside a card |
| `Text` / `Muted` / `Faint` | ink 900 / 500 / 400 | ink 50 / 300 / 400 | Hierarchy, alongside size and weight |
| `Accent` + `AccentText` | yellow 500 + ink 900 | same | Primary action fill, selection, the mark. Never light text on yellow |
| `AccentSoft` | yellow 100 | yellow at 14% | Hover wash, recording-field highlight |
| `AccentInk` | yellow 700 | yellow 400 | The accent as a line or text on a surface |
| `Live` / `Success` / `Warning` / `Danger` | Sunbeam 500 steps | 400 steps | Microphone open, outcomes, failures |
| `OnFill` | white | white | Marks on a saturated semantic fill |
| `Shade` / `Highlight` / `Scrim` | warm ink | black / white 6% | Shadows, 1px top light, photo scrims |
| `Pill*` | warm substrate `#161410` | same | The dictation HUD, dark in both appearances |

Rules:

- Yellow is never text or an icon color on a light surface; it is under 3:1 there. Use `AccentInk`, or ink.
- Navigation controls that tint icons with the accent are built inside `theme.Ink`, so the sidebar is ink and yellow stays for actions.
- State is carried in words first. `kit.Status` colors only the icon; the label stays `Text`.

## Type, radius, spacing

- **Type:** the system face (SF / Segoe UI), chosen so the app reads as native next to the OS. Steps: `Display` 28, `Title` 20, `Heading` 15, `Body` 13, `Caption` 11.5. Use tabular figures (`FontFeatures("tnum")`) for measurements.
- **Radius:**
  - `RadiusXS`–`RadiusXL` = 2/4/6/10/14.
  - Controls use 10, cards 14, keycaps 6, and the pill is full.
  - Nested corners are computed (inner = outer − padding) or left square.
- **Spacing:**
  - `GapTight` 4, `GapRelated` 8, `GapGroup` 28, `Gutter` 24.
  - A section title sits `GapRelated` from its card (`kit.Group`); groups stand `GapGroup` apart.

## Elevation

`kit.Raise` casts a tight contact shadow plus an ambient one whose blur tracks the height, in warm ink, never colored:

| Level | Use |
| --- | --- |
| `Raised` | Cards and outline buttons |
| `Lifted` | Popovers |
| `Floating` | Mock windows in illustrations |
| `Overlay` | Reserved for transient overlays |

A surface gets either a hairline with a contact shadow, or real elevation; never a hairline plus a wide diffuse glow.

The macOS consent guide is a whole window floating over System Settings: its card fills a flyout whose system shadow is the elevation, so it draws no `kit.Raise` or border of its own. Its app tile nests at `RadiusCard − padding`.

## Primitives (`ui/shared/kit`)

| Primitive | Anatomy |
| --- | --- |
| `Button(c, tokens, variant, label, enabled)` | coss `default`/`outline`/`ghost`, plus `Live` (the primary anatomy in the recording color while the microphone is open): 32 DIP, radius 10, 1px edge, 1px top light, contact shadow; flat when pressed; 0.96 press scale; 64% opacity when disabled |
| `IconButton` | 28 DIP ghost square with a tooltip naming it |
| `Style` | The button anatomy for custom content (the dictation toggle) |
| `Card`, `Group`, `Row`, `Divider`, `SectionTitle` | Grouped settings surfaces |
| `Status` | Colored icon plus a word in text ink |
| `Pill` | The dictation HUD surface, shared by the live pill and illustrations |
| `Mark` | The app icon bitmap; the brand mark is never a glyph in an accent tile |
| `Kbd`, `KbdGroup`, `LevelBars`, `Press` | coss `Kbd`/`KbdGroup` for keys (20 DIP, radius 4, `Subtle` fill, 12 DIP medium `Muted` text; pill variant on the substrate), live meter, press feedback |
| `SelectPopup`, `SelectItem`, `SelectSeparator` | coss `Select` popup over `ui.SelectBase`: a card face 4 DIP below the trigger, radius 10, hairline, top light and `Lifted` elevation, 4 DIP padding; 28 DIP options with a 16 DIP check column, washed in `AccentSoft` while highlighted; inset hairline separators. The trigger keeps the look of the control it is (the scene card) |

| `CardHeader` | coss `CardHeader`: a 15 DIP semibold title and an optional action (select, ghost button, figure) on one 44 DIP row |
| `Stat` | A card stating one measurement: icon and caption label over a `Display` figure (drawn by the caller, usually with `Figure`) and its unit in muted `Body` |
| `Figure`, `Rolls` | A number on odometer wheels: the text is laid out invisibly for size and assistive technology, and each digit is painted rolling up over it, clipped to the line, the leaving digit fading as the next arrives. Digits roll in from the left 35 ms apart on a 900 ms spring; `enter` starts them from 0. `Rolls` keeps each wheel's turns so a growing value rolls forward through 9 → 0, never back |
| `SelectTrigger` | coss `Select` trigger for a plain value: the `Outline` button anatomy at 28 DIP, the value and an up-down chevron; its popup is `SelectPopup` |
| `Empty` | coss `Empty`: one muted, centered line filling a card body that has nothing to show yet |

Use these instead of `ui.Button` / `ui.PrimaryButton`. MyGo's stock switches, radios, fields, selects and progress bars are themed by `theme.Apply` and are used as is. A choice made from a custom trigger uses the `kit` select, not a native `Element.Menu`.

## Icons

Interface glyphs are Lucide (`ui/shared/icons`). Cloud polishing providers are shown by their marks from [LobeHub Icons](https://icons.lobehub.com) (`@lobehub/icons-static-svg`, MIT, `svg/LICENSE-lobehub`), drawn by `features/cloud.Mark`: monochrome marks in `Text` ink with `ui.Icon`, the Gemini and DeepSeek marks in their own colors with `ui.Image`. ChatGPT uses the OpenAI mark. Marks stand alone at 16–32 DIP, never inside a tinted tile.

## Overview

Settings › Overview (`widgets/usage`) is the window's first section and its default. Cards are laid out in equal pairs: time dictating and words dictated, words polished and average speed (`Stat`), the daily-words heatmap (full width), then scenes and vocabulary.

The heatmap follows GitHub's contribution graph: a column per week, Monday on top, ending with the current week, as many weeks as the card fits (cells 11–15 DIP on a 3 DIP gap, up to 53 weeks), with Mon/Wed/Fri and month labels and a Less–More legend in the header. Five shades: the `Hairline` well, the accent at 30% and 62%, the accent, then `AccentInk`. A day's shade is its rank among the visible days with dictation (the busiest quarter is darkest), so one unusual day does not wash out the rest. Hovering a day rings it and names its date and words in a tooltip.

Entering the overview (opening the window on it, or choosing it) plays one entrance, about a second long:

- Cards rise 8 DIPs and fade in on a 520 ms spring without bounce, 45 ms apart in reading order. The motion is a pose (transform and opacity), which the GPU renderer composites; MyGo's software rasterizer, used by the headless tester, draws transformed cards far slower, so filmstrip timings from `ARION_UI_SNAPSHOTS` are not real frame times.
- Each stat's `Figure` rolls up from 0 starting 90 ms after its card.
- Scene bars grow from 0 after their card, 50 ms apart; later range changes ease them with `Settle`, and their counts change with MyGo's numeric content transition.
- The heatmap's cells light in a wave from the oldest week once its card has landed (the 1.2 s run starts at −0.18): each grows from 40% and fades in over a quarter of the run.

A dictation committed while the overview shows rolls the figures and reshades the map without replaying the entrance. With Reduce Motion, MyGo only fades: every card fades at once without stagger, and figures, bars and cells take their values immediately. Scene bars are accent on a hairline track; their counts are always written beside them. Each card shows `Empty` when its period has no dictation.

## Illustrations

`widgets/showcase` draws mock UI (another app's editor window and the dictation pill) floating on a photograph from `ui/shared/assets`. Its scenes follow the real flow: `Dictating` loops the draft streaming in the pill and then landing in the editor, `Listening` holds the pill open (with live levels when the microphone is allowed), and `Inserted` shows the outcome.

- Only onboarding (the left pane) and Settings › About use illustrations. Settings › About loops `Dictating`; with Reduce Motion every `Dictating` scene holds the `Inserted` frame.
- Photos are already soft (motion blur or shallow focus), downscaled to 1000 px at most, under 200 KB, and under the Unsplash License or equivalent. See `ui/shared/assets/README.md`.
- Settings › About holds, in order: the illustration with name and version, software update (`features/update`), the Privacy group, and the team, source, X account, bug reports and license. Their URLs and proper names live in `ui/shared/config`; only the row titles are copy keys.
- Mock surfaces are solid with real elevation; no glass panes, glows or tinted gradients. Scrims are ink gradients only where text or controls overlap the photo.
- Mock copy comes from `copy` (`ShowcaseSentence` mixes Chinese and English, as real dictation does), and the illustration is labeled for assistive technology (`ShowcaseLabel`).

## Review

Render every surface with `ARION_UI_SNAPSHOTS=<dir> go test ./ui/app` (light, plus `TestDarkAppearance` for dark) and look at the frames. The `kill-ai-slop` skill's taxonomy applies: one accent, hierarchy from scale and space, no decorative tiles, pulses or one-hue status boxes.
