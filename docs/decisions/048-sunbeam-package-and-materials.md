# ADR 048: Sunbeam from its MyGo package, with flat and raised materials

Status: Accepted. Supersedes the token ownership and button and card anatomy of [ADR 007](007-sunbeam-design-system.md); the rest of ADR 007 stands.

## Context

[ADR 007](007-sunbeam-design-system.md) ported Sunbeam's tokens into `ui/shared/theme` by hand and gave every button and card a 1px edge, a top light and a contact shadow. The AstraFoundry design repository now publishes the same tokens and components for MyGo as the Go module `github.com/AstraFoundry/design/packages/mygo` (package `sunbeam`). It generates them from the same `sunbeam.json` the marketing site uses through `@astrafoundry/design`. Sunbeam's specification (the design repository's `packages/web/skills/sunbeam/SKILL.md`, section 4) has since split surfaces into two materials. The window and its chrome are flat. A layer that floats above it, such as a popover, menu or tooltip, is raised, and so is every control in it.

The hand-kept copy had drifted from the published palette: muted text, the accent as ink, dark-mode semantic colors, and no warning token in the package. The package needs MyGo v0.8, which the desktop's MyGo snapshot now is (see [third_party](../../third_party/README.md)).

## Decision

- `ui/shared/theme` resolves Arion's semantic `Tokens` from `sunbeam.Light()` and `sunbeam.Dark()`, and `theme.Apply` takes its MyGo theme colors from `sunbeam.Theme`. Physical values come from the package. Arion adopts the package's values where they differ from the old copy.
- The theme keeps only colors Sunbeam has no token for: the warning orange and the shortcut recording blue. It also keeps Arion's type scale, row geometry, spacing and motion.
- The settings window, the menu-bar panel, onboarding and their cards are **flat**. Buttons take `sunbeam.Face` and `sunbeam.PrimaryFace`: a fill with no edge, gradient or shadow. Cards keep a hairline and drop their shadow. `kit.Live` is the flat primary face in the recording color.
- Select popups, tooltips and field-error callouts are **raised**. They use `sunbeam.PanelSurface` at Arion's control and keycap radii, and build their content in `sunbeam.Layer(sunbeam.Raised, …)`.
- Arion keeps its control geometry: 32 DIP buttons at radius 10, 28 DIP icon buttons and select triggers, 52 DIP rows, and the menu-bar panel's documented size. The package's 40 DIP pill buttons are not used.
- Arion-specific primitives stay in `kit` and draw only with package tokens and materials: the dictation pill, `Kbd`, `Notice`, `FieldError`, the select popups, `Status`, `Figure` and the voice mark. The pill uses Sunbeam's substrate scale.
- `kit.Raise` maps `Raised`, `Lifted`, `Floating` and `Overlay` onto the package's `shadow-1` to `shadow-4`, for illustrations.

## Consequences

- A change to `sunbeam.json` reaches the desktop by updating the module version in `apps/desktop/go.mod`. The marketing site and the desktop follow one source.
- Muted text and the accent as ink are slightly darker on light surfaces. In dark mode, success, danger and the live microphone use the same colors as in light mode. Primary hover darkens toward ink.
- MyGo's stock controls (switches, radios, fields, progress) and the sidebar change only through their theme colors.
- The module has no release tag yet. `go.mod` pins a pseudo-version of the design repository's `main`.
- Visual changes are reviewed through `ARION_UI_SNAPSHOTS` in both appearances. Native macOS and Windows acceptance follows [native E2E](../quality/e2e.md).

## Alternatives considered

- **Keep the hand-kept copy and sync values by review.** That is how the copy drifted.
- **Use the package's components as they are.** Its 40 DIP pill buttons would change the documented row and panel geometry.
- **Raise the settings cards.** Sunbeam reserves the raised material for layers above the window. Cards in the flow are flat.

## References

- [Design system](../frontend/design-system.md)
- AstraFoundry design repository: `packages/mygo/README.md`, `packages/web/skills/sunbeam/SKILL.md` §4
