# ADR 007: Sunbeam design system and coss anatomy in Go

Status: Accepted.

## Context

Arion's native UI followed the desktop's own theme: the system accent, cool gray surfaces and MyGo's stock controls. It looked generic, its brand (a blue icon) and its UI disagreed whenever the accent was not blue, and some patterns read as machine defaults (accent tiles behind glyphs as the brand mark, one-hue green status text, one spacing value between unrelated groups).

Lyra, the sibling product, uses Sunbeam: a yellow (#FFCC00) accent that always carries dark ink text, warm low-chroma "ink" neutrals, a restrained radius scale and two-layer warm shadows. Its component library takes its anatomy from [coss ui](https://coss.com/ui) (Base UI primitives with Tailwind classes) and layers Sunbeam tokens on top.

The question was whether Arion should adopt coss ui as its component baseline.

## Decision

- **coss ui is not a dependency.** It is React, Base UI and Tailwind; [ADR 001](001-desktop-native-boundaries.md) keeps Arion a MyGo native app with no React or webview. Its *anatomy* is the baseline instead, ported to Go in `ui/shared/kit`:
  - buttons in coss's `Primary`, `Outline` and `Ghost` variants, plus `Live` for the dictation toggle while recording, 32 DIP tall, radius 10, with a 1px edge, a 1px top light and a contact shadow that press flat, and 64% opacity when disabled;
  - cards with a hairline edge, a contact shadow in the light and a top light in the dark;
  - the `Select` popup, with its check column, highlight wash and separators, built on MyGo's headless `SelectBase`.
- **Sunbeam tokens** live only in `ui/shared/theme`. `theme.Apply` installs them as MyGo's theme too, so stock controls (switches, radios, fields, progress) match. The app icon is redrawn in Sunbeam yellow with ink bars (`cmd/arion-icon`).
- **The accent is pinned**, not taken from the system: Arion's brand is yellow on every desktop. Yellow never carries light text, and it is not used as text or icon color on light surfaces (`AccentInk` exists for that). Navigation uses ink as its accent (`theme.Ink`) because MyGo tints every sidebar icon with the accent.
- **Illustrations** are mock UI (an editor window and the dictation pill) floating on a soft, motion-blurred photograph, drawn by `widgets/showcase`. They appear only in onboarding and Settings › About. The onboarding dictation key step instead shows an isometric figure of the keyboard and microphone (`widgets/keyfigure`, see the design system).

## Consequences

- Views use `kit.Button`/`kit.IconButton` rather than `ui.Button`/`ui.PrimaryButton`, and colors only through `theme.Tokens`.
- Users who chose a different system accent see Arion's yellow. Increase Contrast still darkens borders and secondary text.
- The binary embeds about 150 KB of photograph.
- Any new primitive is ported from coss anatomy into `kit` by hand, and visual changes are reviewed through `ARION_UI_SNAPSHOTS` in both appearances.

## Alternatives considered

- **Embed coss ui in a webview.** This contradicts ADR 001 and adds a second rendering stack to a menu bar utility.
- **Keep the system accent.** This loses the shared brand with Lyra, and the icon and UI would still disagree.
- **Blue brand with Sunbeam neutrals.** This was offered and declined in favor of matching Lyra.

## References

- [Design system](../frontend/design-system.md)
- Lyra `packages/ui/src/styles.css` (Sunbeam tokens) and `packages/ui/src/components/button.tsx` (coss variants)
