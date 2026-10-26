#include <iostream>
#include "candidate_window.h"
int main() {
    CandidateWindow panel(GetModuleHandleW(nullptr));
    RECT caret{10,10,11,30};
    if (panel.show(nullptr,caret,{{L"\u4f60\u597d",L"",0}},0,false,false,1,CandidatePalette{})) return 1;
    if (panel.visible()) return 2;
    panel.hide();
    if (panel.visible()) return 3;
    std::cout<<"PASS: invalid owner cannot show or retain a candidate surface; native surface compiles without engine mocks.\n";
}
