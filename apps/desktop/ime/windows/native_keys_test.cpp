#include <iostream>
#include "native_keys.h"
int main() {
    if (windowsNamedKey(VK_LEFT)!=0xff51 || windowsNamedKey(VK_DELETE)!=0xffff || windowsNamedKey(VK_F1)!=-1) return 1;
    HKL layout=GetKeyboardLayout(0); BYTE state[256]{};
    state[VK_CONTROL]=0x80;
    auto shortcut=windowsKey('C',0,state,layout);
    if (shortcut.symbol!='c' || arion_key_action(shortcut.symbol,shortcut.modifiers,0)!=ARION_KEY_PASS) return 2;
    auto punctuation=windowsKey(VK_OEM_PERIOD,0,state,layout);
    if (arion_key_action(punctuation.symbol,punctuation.modifiers,0)!=ARION_KEY_TOGGLE_PUNCTUATION) return 3;
    state[VK_CONTROL]=0; state[VK_SHIFT]=0x80;
    auto space=windowsKey(VK_SPACE,0,state,layout);
    if (arion_key_action(space.symbol,space.modifiers,0)!=ARION_KEY_TOGGLE_ASCII) return 4;
    auto left=windowsKey(VK_SHIFT,0x002a0000,state,layout),right=windowsKey(VK_SHIFT,0x00360000,state,layout);
    if (!left.shift || left.shiftSide!=1 || !right.shift || right.shiftSide!=2) return 5;
    auto caps=windowsKey(VK_CAPITAL,0,state,layout);
    if (!caps.capsLock || caps.symbol!=-1 || caps.shift) return 6;
    state[VK_SHIFT]=0; state[VK_CONTROL]=0x80;
    auto lctrl=windowsKey(VK_LCONTROL,0,state,layout),rctrl=windowsKey(VK_RCONTROL,0,state,layout);
    if (!lctrl.control || lctrl.controlSide!=1 || lctrl.symbol!=-1 || !rctrl.control || rctrl.controlSide!=2) return 7;
    if (arion_caps_lock_language(1)!=1 || arion_caps_lock_language(0)!=0) return 8;
    std::cout<<"PASS: native named keys/Shift sides and shared shortcut/mode policy; no engine processing or input injection.\n";
}
