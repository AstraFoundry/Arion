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
| `Keycap(s)`, `LevelBars`, `Press` | Shortcut display, live meter, press feedback |

Use these instead of `ui.Button` / `ui.PrimaryButton`. MyGo's stock switches, radios, fields, selects and progress bars are themed by `theme.Apply` and are used as is.

## Illustrations

`widgets/showcase` draws mock UI (another app's editor window and the dictation pill) floating on a photograph from `ui/shared/assets`. Its scenes follow the real flow: `Dictating` loops the draft streaming in the pill and then landing in the editor, `Listening` holds the pill open (with live levels when the microphone is allowed), and `Inserted` shows the outcome.

- Only onboarding (the left pane) and Settings › About use illustrations.
- Photos are already soft (motion blur or shallow focus), downscaled to 1000 px at most, under 200 KB, under the Unsplash License or equivalent, and credited in `copy.AboutCredits`. See `ui/shared/assets/README.md`.
- Mock surfaces are solid with real elevation; no glass panes, glows or tinted gradients. Scrims are ink gradients only where text or controls overlap the photo.
- Mock copy comes from `copy` (`ShowcaseSentence` mixes Chinese and English, as real dictation does), and the illustration is labeled for assistive technology (`ShowcaseLabel`).

## Review

Render every surface with `ARION_UI_SNAPSHOTS=<dir> go test ./ui/app` (light, plus `TestDarkAppearance` for dark) and look at the frames. The `kill-ai-slop` skill's taxonomy applies: one accent, hierarchy from scale and space, no decorative tiles, pulses or one-hue status boxes.
