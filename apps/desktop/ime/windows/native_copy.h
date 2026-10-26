#pragma once
#include <windows.h>
#include <string>
enum class NativeCopy { Voice,SwitchChinese,SwitchEnglish,FullWidth,HalfWidth,ChinesePunctuation,EnglishPunctuation,Reset,OpenApp,CopyPending,Unavailable,RankingUnavailable,OptionsFailed,ResetFailed,ChineseBadge,EnglishBadge,FullWidthBadge,HalfWidthBadge,ChinesePunctuationBadge,EnglishPunctuationBadge,RightAlt,RightCtrl,OpenAppFailed };
inline const wchar_t *nativeCopy(NativeCopy key) {
    static const wchar_t *const english[]={L"Voice input",L"Switch to Chinese (Shift)",L"Switch to English (Shift)",L"Use full-width characters",L"Use half-width characters",L"Use Chinese punctuation (Ctrl+.)",L"Use English punctuation (Ctrl+.)",L"Reset learned words",L"Open Arion",L"Copy pending text",L"Keyboard unavailable",L"Context ranking unavailable",L"Could not save keyboard mode",L"Could not reset learned words",L"中",L"En",L"Ａ",L"A",L"，",L",",L"Right Alt",L"Right Ctrl",L"Could not open Arion"};
    static const wchar_t *const chinese[]={L"语音输入",L"切换至中文输入 (Shift)",L"切换至英文输入 (Shift)",L"切换至全角输入",L"切换至半角输入",L"中文下使用中文标点 (Ctrl+.)",L"中文下使用英文标点 (Ctrl+.)",L"重置已学词语",L"打开 Arion",L"复制待输入文本",L"键盘不可用",L"上下文调频不可用",L"无法保存键盘模式",L"无法重置已学词语",L"中",L"En",L"Ａ",L"A",L"，",L",",L"右 Alt",L"右 Ctrl",L"无法打开 Arion"};
    return (PRIMARYLANGID(GetUserDefaultUILanguage())==LANG_CHINESE?chinese:english)[static_cast<size_t>(key)];
}
inline std::wstring nativeVoiceTitle(const std::wstring &key) {
    std::wstring title=nativeCopy(NativeCopy::Voice);
    if (key==L"right_alt" || key==L"right_ctrl") title+=std::wstring(L" (")+nativeCopy(key==L"right_alt"?NativeCopy::RightAlt:NativeCopy::RightCtrl)+L")";
    return title;
}
