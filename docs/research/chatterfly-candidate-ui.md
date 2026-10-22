# Chatterfly candidate UI

Static reading of the installed **Chatterfly 1.0.7.13415** (`/Library/Input Methods/Chatterfly.app`, arm64 slice) for the visual spec of Arion's own candidate bar (`apps/desktop/ime/macos/candidate_panel.m`) and input-menu icon. No Chatterfly code, assets or fonts ship in Arion; only measured values are reused.

## Sources

| Question | Evidence |
| --- | --- |
| Candidate geometry | `+[YuanbaoProductProfile yuanbaoCandidateSizePresets]` builds seven `SGCandidateSizePreset` objects; per-level values come from constant tables, fixed values from immediates (Apple `objdump`/`otool` disassembly) |
| Bar chrome | `+[YuanbaoProductProfile yuanbaoLightCandidateTheme]` / `yuanbaoDarkCandidateTheme` → `SGCandidateTheme` |
| Colors, default level | `+[YuanbaoProductProfile yuanbaoCandidateUIConfig]` dictionary (CFString keys resolved from chained fixups) |
| Renderers | Class names `SGCandidateViewControllerRow`, `SGCandidateLayerChromeRenderer`, `SGCandidateRoundedSelectionRenderer`; a custom window, not `IMKCandidates` |
| Legacy skin | `Resources/Skins/输入法默认皮肤/{Skin,SkinDark}/skin.plist` and `skin@2x.png` |
| Menu icon | `Info.plist` names `menubarpinyin.pdf`, a 16×16 pt vector |

## Size presets

`defaultCandSizeLevel` is 2 (second preset); `enableCandSizeLevel` is true.

| Level | Bar height | Selection height | Body font | Number font | Number width | Selection radius | Outer radius |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | 30 | 22 | 14 | 12 | 8 | 6 | 10 |
| **2** | **32** | **24** | **16** | **13** | **8** | **6** | **10** |
| 3 | 35 | 27 | 18 | 14 | 10 | 8 | 12 |
| 4 | 38 | 30 | 20 | 15 | 10 | 8 | 12 |
| 5 | 41 | 33 | 22 | 16 | 10 | 8 | 12 |
| 6 | 44 | 36 | 24 | 17 | 10 | 10 | 14 |
| 7 | 46 | 38 | 26 | 18 | 10 | 10 | 14 |

Every level uses candidate spacing 23, selection horizontal padding 4, content leading inset 4, content trailing inset 28, number–text spacing 3, arrow size 20, arrow trailing inset 8 and background insets 4/24/4/4 (leading/trailing/top/bottom).

## Theme

| Property | Light | Dark |
| --- | --- | --- |
| Bar background | `#FFFFFF` | `#171717` |
| Border | 0.5 pt, black 20% | 0.5 pt, white 20% |
| Shadow | `#1E201F` at 10.2%, radius 8, offset (0, −4) | same |
| Secondary shadow | `#181A18` at 5.9%, radius 3, offset (0, −2) | same |
| Candidate text | `#1D1D1D` (`defaultCandColor`) | white 80% (dark skin `FontColor`) |
| Selection fill | `#0088FF` (`candidateFocusBoxColorLight`) | `#0099F2` (`candidateFocusBoxColorDark`) |
| Selected text | white | white |

Other config values: `defaultPageSize` 6, `defaultFontSize` 16, `numFontDelta` 3, `candidateVerticalCentered` true, `pageButtonIconStyle` `outlinedChevron`. The skin font is `PingFangSC-Regular`.

## Arion mapping and inferences

- Arion offers levels 1–3 as small/medium/large (medium by default) and the themes above, with its own drawing code. The selection fill defaults to Sunbeam yellow `#FFCC00` with ink `#14110F` text, the app's accent; Settings › Keyboard › Highlight color offers the blue fill and white text above among other presets and a custom color ([ADR 017](../decisions/017-offline-pinyin-keyboard.md)).
- **Inferred:** the 23 pt candidate spacing is measured between texts, so the gap between selection boxes is 23 − 2×4 = 15 pt. The unselected index color (text color at 50%/45%) is not in the config and was chosen to match the skin's gray page chevron.
- Arion defaults to 9 candidates per page rather than Chatterfly's 6, matching the benchmarks; Settings › Keyboard offers 5, 6, 7 or 9.
- The chevron turns Arion's menu to the next page; earlier pages stay on `-`/`[`.
- The input-menu icon must be a 16 pt image. Arion's previous single 64 px, 72 dpi TIFF was read as 64 pt and enlarged the "Arion" row in the input menu. The build now combines 16 px (72 dpi) and 32 px (144 dpi) slices with `tiffutil -cathidpicheck`.

## Spread (expanded) list

Pressing past the page spreads the candidates into rows (`SGCandViewControllerScroll`, `SGCandidatesTextViewScroll`, `isScrollSpreading`, `expandedRowCount` 6). The scroll view keeps per-column maxima (`candidateMaxs`) so rows align, a current row (`curRow`, `focusLineBgColor`), row separators (`SGCandidateLineSeparatorRenderer drawSeparatorsForCandidateFrames:pageSize:…`) and an expanded focus box. `-[SGCandidatesController updateWindow:]` animates the window frame with an `NSViewAnimation` of 0.3 s when spreading and 0.2 s when folding, default ease-in-out; `animationDidEnd:` refreshes the window. The bar itself appears without a fade. Arion's grid follows this structure; the row tint and separator colors are Arion's choice, since the focus-line color is skin data not read here.

## Limits

These are static reads of one build. A live screenshot of Chatterfly's bar was not captured: synthetic keystrokes did not reach a focused field, and the earlier REA native-UI request had already been unavailable ([installed keyboard](chatterfly-installed-keyboard.md)). Runtime size-level overrides from user preferences were not traced.
