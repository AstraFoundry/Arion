#pragma once
#include <windows.h>
#include "../shared/input.h"

struct NativeKey {
    int symbol=-1;
    int modifiers=0;
    unsigned shiftSide=0;
    bool shift=false;
    bool repeated=false;
};

inline int windowsNamedKey(WPARAM key) {
    switch (key) {
    case VK_BACK: return 0xff08;
    case VK_TAB: return 0xff09;
    case VK_RETURN: return 0xff0d;
    case VK_ESCAPE: return 0xff1b;
    case VK_HOME: return 0xff50;
    case VK_LEFT: return 0xff51;
    case VK_UP: return 0xff52;
    case VK_RIGHT: return 0xff53;
    case VK_DOWN: return 0xff54;
    case VK_PRIOR: return 0xff55;
    case VK_NEXT: return 0xff56;
    case VK_END: return 0xff57;
    case VK_INSERT: return 0xff63;
    case VK_DELETE: return 0xffff;
    default: return -1;
    }
}

inline NativeKey windowsKey(WPARAM key,LPARAM details,const BYTE state[256],HKL layout) {
    NativeKey value;
    if (state[VK_SHIFT]&0x80) value.modifiers|=1;
    if (state[VK_CONTROL]&0x80) value.modifiers|=4;
    if (state[VK_MENU]&0x80) value.modifiers|=8;
    if ((state[VK_LWIN]|state[VK_RWIN])&0x80) value.modifiers|=1<<26;
    value.repeated=(details&(1ull<<30))!=0;
    value.shift=key==VK_SHIFT || key==VK_LSHIFT || key==VK_RSHIFT;
    if (value.shift) {
        UINT side=key==VK_SHIFT ? MapVirtualKeyExW(static_cast<UINT>((details>>16)&0xff),MAPVK_VSC_TO_VK_EX,layout) : static_cast<UINT>(key);
        value.shiftSide=side==VK_LSHIFT ? 1 : side==VK_RSHIFT ? 2 : 0;
        return value;
    }
    value.symbol=windowsNamedKey(key);
    if (value.symbol>=0) return value;
    // Shortcut classification uses the printable base key, not a control code.
    // The shared policy passes shortcuts through without processing the engine.
    if (value.modifiers&(4|8|(1<<26))) {
        if (key>='A' && key<='Z') value.symbol=static_cast<int>(key-'A'+'a');
        else if (key==VK_OEM_PERIOD) value.symbol='.';
        else if (key==VK_SPACE) value.symbol=' ';
        return value;
    }
    wchar_t text[8]{};
    // Windows 10 1607+: bit2 prevents a test callback from mutating dead-key state.
    int count=ToUnicodeEx(static_cast<UINT>(key),static_cast<UINT>((details>>16)&0xff),state,text,8,4,layout);
    if (count==1 && text[0]>=0x20 && (text[0]<0xd800 || text[0]>0xdfff)) value.symbol=text[0];
    return value;
}
