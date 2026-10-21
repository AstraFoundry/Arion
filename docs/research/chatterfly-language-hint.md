# Chatterfly Chinese/English caret hint

## Scope

Installed macOS Chatterfly **1.0.8.13459 (13459)**, arm64. This investigation follows the actual Shift-switch call chain to the caret hint, rather than attributing an unrelated toast or candidate-bar animation to it. The [ledger](evidence/chatterfly-language-hint.json) records artifact identity, REA Evidence IDs, verified instruction listings and limitations. Complete local outputs remain in `.workspace/research/chatterfly-language-hint/`.

REA 6.4.0 inspected Mach-O identity, signature and packaged skin metadata. The connected native decompiler was unavailable: doctor found no Hopper/Ghidra/IDA provider. Function observations below use ordinary Apple LLVM listings, with 3,959 instruction words in 23 focused listings checked against the installed arm64 bytes. They are not REA pseudocode or observed execution.

## Actual path

`SGInputControllerBase.funcSwitchCnEn:` calls `SGInputStatusNotification.updateSGInputStatus:andGrowlSwitch:andTouchBar:`. At `0x1002f9b58–0x1002f9bd0`, the notification checks the technical `keyGrowlSwitch` preference and dispatches a status type to `SGNotification.showGrowWithType:`. Chatterfly's brand branch reaches `SGInputHummingWindow.showHummingWithType:` (`0x100308ccc–0x100308d68`). This establishes a caret-hint path, distinct from the desktop status bar and speech tooltip.

The humming window obtains the host caret through `attributesForCharacterIndex:lineHeightRectangle:`, checks screens and can use the mouse location when its caret path is unusable. The pure-image branch uses the selected image's point size and disables the window shadow; the generic view branch is **36 × 36 points**. The shipped Chinese/English skin images are **23 × 23 pixels at 1×**, with matching 2× assets. Arion now uses an original **23 × 23 point** solid rounded badge with no window shadow. A selected custom skin or missing image can change which branch renders; this investigation did not read the user's active skin preference.

## Timing and animation

- `addTimer` at `0x1002e9c44` invalidates the previous timer and schedules dismissal after **1.0 second** (`0x1002e9cb4`).
- `orderFront:` calls `showAnimation`. A non-null entry group runs before its completion starts the timer; a null entry group starts the timer immediately. Exit similarly runs a configured group or immediately calls `stopAnimation`, which sets view/window alpha to zero.
- Built-in `sharedDefaultNodes` sets Chinese and English `animIn = 0` and `animOut = 0`.
- The packaged default skin's `HummingNotif` instead declares `301` and `302`. `SGGrowlStateNode.initWithDataDict:` reads these as ordinary integer values.
- In this binary, `SGSkinSourceLoader.animationGroupByType:withTarget:andBackgroundTarget:` has no implemented branch for **0, 301 or 302**. Following each through the dispatch reaches the nil-group return at `0x1002518d8–0x1002518e8`. Supported animation families elsewhere in the same function do not establish that this hint fades, springs or scales.
- The brand profile explicitly sets `hummingPureImage = true` (`0x10044d2d8`), and the pure-image renderer rejoins the same node/group lifecycle.

For the inspected built-in nodes and shipped default-skin values, the static result is therefore **immediate appearance, a one-second timer and immediate hiding**. This is a version-bound static conclusion, not a recorded frame timing or a claim about every custom skin.

## Comparison with Arion

| Property | Inspected Chatterfly path/assets | Current Arion |
| --- | --- | --- |
| Entry/exit with default node values | No non-null animation group | Immediate show/hide |
| Dismissal timer | 1.0 s | 1.0 s |
| Pure-image packaged badge | 23 × 23 at 1×, shadow disabled | 23 × 23 solid surface, no window shadow |
| Generic renderer | 36 × 36 | Uses the compact badge |
| Rapid switches | Clear old animation and invalidate timer | Hide old badge and replace timer |

Arion now matches the inspected default badge geometry, shadow policy and one-second lifetime. The badge is rendered with native text and a solid rounded layer, without competitor artwork. Window animation is explicitly disabled; rapid switches replace the timer. No spring, scale or fade is added. Font rasterization and custom skins are not claimed as pixel-identical.

## Visual verification boundary

A temporary native preview instantiated the updated production `ArionLanguagePanel`, with its actual labels and one-second timer. CUA captured its real window, confirming the compact solid rounded badge and centered Chinese glyph. The screenshot is retained at `.workspace/research/chatterfly-language-hint/arion-aligned-hint.png`. Native AppKit integration assertions cover the 23 × 23 point size, disabled shadow/window animation, visibility beyond 0.8 seconds, dismissal after one second, replacement timer, focus preservation and edge placement. The screenshot API can return a hidden window's backing image, so repeated identical screenshots do not measure visible duration.

CUA refused modifier-only Shift with `keyPressIncludedNoNonModifierKeys`; capture of the Chatterfly background app failed. No Chatterfly Shift animation movie or frame-by-frame parity was obtained. The skin images were inspected directly as packaged assets, not presented as live screenshots. Temporary source navigation ended on ABC; the original Arion source remained enabled/selectable, but restoration through the available UI shortcuts was not achieved. No input-source installation, preference editing or security changes were performed.
