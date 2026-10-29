#pragma once
#include "json.h"
#include "../shared/spelling_options.h"
#include "candidate_window.h"

struct KeyboardSettings {
    size_t pageSize=9;
    int textSize=16;
    bool emojiSuggestions=true,typoCorrection=false;
    uint32_t fuzzyPinyin=0;
    CandidatePalette palette;
    std::wstring dictationKey;
};
inline bool keyboardColor(const Json *value,COLORREF &color) {
    if (!value || value->type!=Json::String || value->string.size()!=7 || value->string[0]!=L'#') return false;
    unsigned number=0;
    for (size_t i=1;i<7;++i) { wchar_t c=value->string[i]; int digit=c>=L'0'&&c<=L'9'?c-L'0':c>=L'a'&&c<=L'f'?c-L'a'+10:c>=L'A'&&c<=L'F'?c-L'A'+10:-1; if (digit<0) return false; number=number*16+digit; }
    color=RGB(number>>16,(number>>8)&255,number&255); return true;
}
inline KeyboardSettings readKeyboardSettings(const std::wstring &path) {
    KeyboardSettings settings;
    DWORD light=1,size=sizeof(light);
    RegGetValueW(HKEY_CURRENT_USER,L"Software\\Microsoft\\Windows\\CurrentVersion\\Themes\\Personalize",L"AppsUseLightTheme",RRF_RT_REG_DWORD,nullptr,&light,&size);
    bool dark=!light;
    Json value;
    if (readJsonFile(path,value) && value.type==Json::Object) {
        const Json *field=value.get(L"pageSize"); if (field && field->type==Json::Number && field->number>=1 && field->number<=9) settings.pageSize=static_cast<size_t>(field->number);
        field=value.get(L"candidateSize");
        if (field && field->type==Json::String) { if (field->string==L"small") settings.textSize=14; else if (field->string==L"large") settings.textSize=20; }
        field=value.get(L"emojiSuggestions"); if (field && field->type==Json::Boolean) settings.emojiSuggestions=field->boolean;
        field=value.get(L"typoCorrection"); if (field && field->type==Json::Boolean) settings.typoCorrection=field->boolean;
        field=value.get(L"fuzzyPinyin");
        if (field && field->type==Json::Number && field->number<=ARION_FUZZY_MASK &&
            field->number==static_cast<uint32_t>(field->number)) settings.fuzzyPinyin=static_cast<uint32_t>(field->number);
        field=value.get(L"dictationKey"); if (field && field->type==Json::String) settings.dictationKey=field->string;
        field=value.get(L"appearance"); if (field && field->type==Json::String) { if (field->string==L"dark") dark=true; else if (field->string==L"light") dark=false; }
    }
    // Base colors are platform surfaces; the app supplies the resolved accent
    // and contrast text for each appearance, including custom accent colors.
    settings.palette.surface=dark?RGB(32,32,32):RGB(255,255,255);
    settings.palette.text=dark?RGB(245,245,245):RGB(20,17,15);
    settings.palette.muted=dark?RGB(170,170,170):RGB(105,105,105);
    settings.palette.border=dark?RGB(70,70,70):RGB(215,215,215);
    if (const Json *highlight=value.get(L"highlight")) if (const Json *theme=highlight->get(dark?L"dark":L"light")) {
        keyboardColor(theme->get(L"fill"),settings.palette.selected);
        keyboardColor(theme->get(L"text"),settings.palette.selectedText);
    }
    HIGHCONTRASTW contrast{}; contrast.cbSize=sizeof(contrast);
    if (SystemParametersInfoW(SPI_GETHIGHCONTRAST,sizeof(contrast),&contrast,0) && (contrast.dwFlags&HCF_HIGHCONTRASTON)) {
        settings.palette.surface=GetSysColor(COLOR_WINDOW); settings.palette.text=GetSysColor(COLOR_WINDOWTEXT);
        settings.palette.selected=GetSysColor(COLOR_HIGHLIGHT); settings.palette.selectedText=GetSysColor(COLOR_HIGHLIGHTTEXT);
        settings.palette.muted=GetSysColor(COLOR_WINDOWTEXT); settings.palette.border=GetSysColor(COLOR_WINDOWTEXT);
    }
    return settings;
}
