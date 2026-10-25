#pragma once
#include <windows.h>
#include <string>
enum class NativeCopy { Voice,SwitchChinese,SwitchEnglish,FullWidth,HalfWidth,ChinesePunctuation,EnglishPunctuation,Reset,OpenApp,CopyPending,Unavailable,RankingUnavailable,OptionsFailed,ResetFailed,ChineseBadge,EnglishBadge,FullWidthBadge,HalfWidthBadge,ChinesePunctuationBadge,EnglishPunctuationBadge,RightAlt,RightCtrl,OpenAppFailed };
inline const wchar_t *nativeCopy(NativeCopy key) {
    static const wchar_t *const english[]={L"Voice Input",L"Switch to Chinese",L"Switch to English",L"Use Full-width Characters",L"Use Half-width Characters",L"Use Chinese Punctuation (Ctrl+.)",L"Use English Punctuation (Ctrl+.)",L"Reset Learned Words",L"Open Arion",L"Copy Pending Text",L"Arion keyboard unavailable",L"Context ranking unavailable",L"Could not save keyboard mode",L"Could not reset learned words",L"中",L"En",L"Ａ",L"A",L"，",L",",L"Right Alt",L"Right Ctrl",L"Could not open Arion"};
    static const wchar_t *const chinese[]={L"语音输入",L"切换为中文",L"切换为英文",L"使用全角字符",L"使用半角字符",L"使用中文标点 (Ctrl+.)",L"使用英文标点 (Ctrl+.)",L"重置学习词库",L"打开 Arion",L"复制未提交文字",L"Arion 键盘不可用",L"上下文调频不可用",L"无法保存键盘模式",L"无法重置学习词库",L"中",L"En",L"Ａ",L"A",L"，",L",",L"右 Alt",L"右 Ctrl",L"无法打开 Arion"};
    return (PRIMARYLANGID(GetUserDefaultUILanguage())==LANG_CHINESE?chinese:english)[static_cast<size_t>(key)];
}
inline std::wstring nativeVoiceTitle(const std::wstring &key) {
    std::wstring title=nativeCopy(NativeCopy::Voice);
    if (key==L"right_alt" || key==L"right_ctrl") title+=std::wstring(L" — ")+nativeCopy(key==L"right_alt"?NativeCopy::RightAlt:NativeCopy::RightCtrl);
    return title;
}
