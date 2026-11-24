# Typeless mobile typing

Static reading of the Typeless 2.11.0 iOS IPA and Android APKM (build 386), done with Codex CLI. Nothing was executed; the iOS Mach-O files are FairPlay-encrypted, so iOS findings rest on bundled data and keyboard-bundle strings.

- **Engine.** Android ships `librime.so` (strings `1.16.1`, exports `rime_get_api`) with a JNI bridge (`RimeNativeBridge`: sessions, candidates from an index, workspace changes). The iOS keyboard bundle names `RimeEngineModule`; both ship the same schemas with prebuilt `build/` output (`rime_version: 1.16.0`).
- **Schemas.** `zh_CN_pinyin_full`, `zh_CN_pinyin_9key` (digit derivation over the full dictionary, own prism), `zh_TW_pinyin_full`, `zh_TW_zhuyin_full`, `en_US_qwerty`, Japanese kana 12-key, Korean dubeolsik. Chinese schemas enable the user dictionary. About 60 MB grammar models per language ship beside them.
- **Layout choice.** The app labels the Pinyin variants 拼音 - 全键盘 and 拼音 - 九宫格: a setting, not a key on the keyboard.
- **Switching.** The keyboard copy reads 滑动切换键盘 (swipe to switch keyboards). Direction, button placement and persistence were not established.
- **Candidates.** An expandable candidate panel (`toggleCandidatePanel`, grid) and `menu.page_size: 16`.

Not established: exact key rows, number and symbol pages, the voice/typing toolbar order, what happens to a pending composition when dictation starts, and runtime storage paths.
