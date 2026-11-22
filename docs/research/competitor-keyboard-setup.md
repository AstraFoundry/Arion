# Competitor keyboard setup on iOS

How Doubao Input Method and WeChat Input Method guide a user through adding their keyboard on iOS, and what Arion takes from it. Static evidence from App Store IPAs; nothing was executed.

## Sources and limits

| App | IPA | SHA-256 |
| --- | --- | --- |
| Doubao Input Method (豆包输入法) | `doubao-keyboard.ipa`, app `oime_iOS`, keyboard `KeyboardExtension.appex` | `1ea5d0ae956c68a224ad6420daa94919615b9260eb4e973c33eb0eaf133a5e47` |
| WeChat Input Method (微信输入法) | `wechat-keyboard.ipa`, app `wxkb`, keyboard `wxkb_plugin.appex` | `03957beb314adfd147ed9019a48ae196f57e1227db0089d9cda34861baa1599d` |

Both main executables and both keyboard executables are FairPlay-encrypted (`cryptid 1`), so how each app opens Settings, detects that its keyboard was added, and advances its onboarding could not be read. The evidence below is the unencrypted Settings bundles, extension property lists and guide resources. A Codex study with REA of both IPAs stopped at its usage limit before reporting.

## Settings-app page

Both apps ship a `Settings.bundle` with a single `PSGroupSpecifier`, so their page in the Settings app (the page their onboarding opens, under the system's Keyboards row) carries a note. The two notes have the same shape, in Chinese only and written into `Root.plist`:

- What to do when the Keyboards row is missing: leave Settings from the app switcher, open it again and find the app (Doubao: search for it), then turn the keyboard on.
- Why Full Access is asked for (voice input among the reasons).
- That the privacy policy protects the user's data.

The note means the user does not have to remember the app's instructions after the switch to Settings.

## Visual guides

| App | Resources | What they show |
| --- | --- | --- |
| Doubao | Lottie files in `OimeUIAssets.bundle`: `setting_lottie`, `full_access_guide_lottie`, `switch_lottie`, `reboot_lottie`, `guide_microphone` (light and dark) | Embedded screenshots of the real Settings pages (Settings › Apps › Doubao, the Keyboards page with the keyboard and Allow Full Access switches), animated, 2–4 s |
| WeChat | GIFs in `res/image`: `fullaccess_guide`, `notfound_guide`, `voiceinput_microphone_guide` (light and dark) | Tapping Keyboards, turning on the keyboard, then Allow Full Access; for a missing keyboard, restarting the phone |

## Keyboard declaration

| | Doubao | WeChat | Arion |
| --- | --- | --- | --- |
| `PrimaryLanguage` | `zh-CN` | `zh-Hans` | `en-US` |
| `IsASCIICapable` | true | true | false (voice only) |
| `RequestsOpenAccess` | true | true | true |
| Minimum iOS | 15.0 | 15.1 | 26 |
| Other extensions | notification, Live Activity | share, action, widgets (one a voice-input widget) | none |

## Arion design choices

- Arion's page in Settings carries the setup steps under the Keyboards row: turn on Arion, Allow Full Access for the microphone, and what to do when the row is missing. Unlike both competitors, the copy is localized (English and Simplified Chinese) and generated from the mobile string catalog.
- The onboarding keyboard step says where the steps are and, like WeChat's guide, shows the whole path as the Settings app draws it: Arion's page with the note, a tap on Keyboards, then the Arion and Allow Full Access switches. It is drawn with system colors, switches and localized labels rather than screenshots, which go stale with each iOS release and match only one language.
- The privacy reassurance and the list of Full Access benefits are not repeated: the note states the one reason Arion needs it.
- Not adopted: restart-the-phone guidance, a second recorder or widget, and Live Activities.

## Open

- How either competitor detects that the keyboard was added and Full Access granted, and whether onboarding advances by itself; this needs decrypted binaries or a device run.
- Arion's Settings page shows the note on Settings › Apps › Arion in the simulator, but not the Keyboards row for the ad-hoc-signed build; the full path needs a device run ([mobile apps](../project/mobile.md#verification)).
