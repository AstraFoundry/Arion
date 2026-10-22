# Input menu items

The input menu (menu bar input source menu) shows a selected IMK source's own items between the source list and the system's items (Edit Text Replacements, Emoji & Symbols, Keyboard Viewer, Show Input Source Name, Open Keyboard Settings). This compares the items Chatterfly and Doubao add with Arion's.

## Evidence

- User screenshots of the macOS 26.6.2 input menu with each source selected.
- Static strings and imports from the installed binaries. REA had no configured decompiler provider (Ghidra/Hopper/IDA), so no method bodies were read. Selector names and log strings below are observed text, not traced behavior.
  - Chatterfly 1.0.8.13459 (`com.tencent.inputmethod.chatterfly`), `/Library/Input Methods/Chatterfly.app/Contents/MacOS/Chatterfly`.
  - Doubao 1.0.2 (build 1.0.2.7, `com.bytedance.inputmethod.doubaoime`), `/Library/Input Methods/DoubaoIme.app/Contents/MacOS/DoubaoIme`.

## Items

| Group | Chatterfly | Doubao | Arion |
| --- | --- | --- | --- |
| Voice | 语音输入 (fn); 语音指令 (双击 fn); Skills 设置 | 语音输入 (长按 fn/单击右 option) | — |
| Mode | 切换至英文输入 (⇧) / 切换至中文输入 (⇧) | 切换至英文输入, with ⇧ shown as the key equivalent | 切换至英文输入 (⇧) / 切换至中文输入 (⇧) |
| Width | 使用全角输入 | 切换至全角输入 | — |
| Punctuation | 使用英文标点输入 (^.) | 中文下使用英文标点 | — |
| Keyboard data | — | — | 复制待输入文本 (only with recovery text); 重置已学词语 |
| App | 个人主页; 偏好设置 | 豆包输入法设置 | 打开 Arion |

Chatterfly logs `[YBMenu] item… clicked` per action (`itemVoiceInput`, `itemVoiceCommand`, `itemSkillsSettings:`, `itemSwitchCnEnMenu:`, `itemSwitchQuanBanMenu:`, `itemEnPunc`, `itemPersonalHome`, `itemPreference`), plus debug/log-upload items not shown in the screenshot. Doubao exposes `toggleFullWidthInput:` and `toggleEnglishPunctuation:` and builds its voice title from the configured hotkey. Both put the shortcut in the title, so the menu teaches it. Both name items by the action, not the current state.

Doubao also declares `TICapsLockLanguageSwitchCapable` and `tsInputMethodCharacterRepertoireKey` Latn/Hans/Hant. Neither competitor sets `TISIntendedLanguage`.

## Arion alignment

Implemented with existing capabilities:

- Action wording with the standalone-Shift shortcut (KBD-04): **Switch to English (⇧)** / **Switch to Chinese (⇧)**. The old label was **English — Shift**, which did not say whether it named the current mode or the action.
- Separators group the mode switch, keyboard data and the app.
- **Open Arion** activates the app that shipped the input method. The build writes its bundle ID to `ArionAppIdentifier` in the input method's `Info.plist`. The item is omitted when LaunchServices cannot resolve that app.

Not implemented, because each needs a product decision and a spec:

- **Voice input item.** It needs a new IME → app bridge operation that starts dictation, and a title that follows the configured dictation key.
- **Full-width and English punctuation toggles.** The schema defines one punctuation table for `half_shape` and `full_shape` and no switches. Rime's `full_shape`/`ascii_punct` options would need schema switches, persisted state in `keyboard.json` and caret-badge feedback.
- **Caps Lock language switching** (`TICapsLockLanguageSwitchCapable`). It changes KBD-04's switch contract.
