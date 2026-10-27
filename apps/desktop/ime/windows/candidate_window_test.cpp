#include <iostream>
#include <filesystem>
#include <fstream>
#include <stdexcept>
#include "candidate_window.h"
static void require(bool ok,const char *message) { if (!ok) throw std::runtime_error(message); }
static size_t capture(HWND window,const std::filesystem::path &file) {
    UpdateWindow(window);
    RECT area{}; GetClientRect(window,&area);
    BITMAPINFO info{}; info.bmiHeader.biSize=sizeof(BITMAPINFOHEADER); info.bmiHeader.biWidth=area.right;
    info.bmiHeader.biHeight=-area.bottom; info.bmiHeader.biPlanes=1; info.bmiHeader.biBitCount=32; info.bmiHeader.biCompression=BI_RGB;
    HDC source=GetDC(window),copy=CreateCompatibleDC(source); void *pixels=nullptr;
    HBITMAP image=CreateDIBSection(source,&info,DIB_RGB_COLORS,&pixels,nullptr,0);
    require(image && pixels,"create native Emoji capture");
    auto previous=SelectObject(copy,image);
    require(BitBlt(copy,0,0,area.right,area.bottom,source,0,0,SRCCOPY),"capture actual candidate pixels");
    size_t colored=0,bytes=static_cast<size_t>(area.right)*area.bottom*4;
    auto data=static_cast<unsigned char*>(pixels);
    for (size_t i=0;i<bytes;i+=4) {
        // Smile's yellow interior, not ClearType fringes or grayscale text.
        if (data[i+2]>140 && data[i+1]>80 && data[i+2]-data[i]>60 && data[i+1]-data[i]>40) ++colored;
    }
    if (!file.empty()) {
        BITMAPFILEHEADER header{}; header.bfType=0x4d42; header.bfOffBits=sizeof(header)+sizeof(info.bmiHeader); header.bfSize=header.bfOffBits+static_cast<DWORD>(bytes);
        std::ofstream out(file,std::ios::binary);
        out.write(reinterpret_cast<const char*>(&header),sizeof(header)); out.write(reinterpret_cast<const char*>(&info.bmiHeader),sizeof(info.bmiHeader)); out.write(reinterpret_cast<const char*>(pixels),static_cast<std::streamsize>(bytes));
        require(out.good(),"save native rendering evidence");
    }
    SelectObject(copy,previous); DeleteObject(image); DeleteDC(copy); ReleaseDC(window,source); return colored;
}
int main(int count,char **arguments) {
    try {
    CandidateWindow panel(GetModuleHandleW(nullptr));
    RECT caret{10,10,11,30};
    require(!panel.show(nullptr,caret,{{L"\u4f60\u597d",L"",0}},0,false,false,1,CandidatePalette{}),"refuse missing owner");
    require(!panel.visible(),"no surface without owner");
    panel.hide();
    require(!panel.visible(),"hide is idempotent");
    WindowsEmojiFont font; bool supported=false;
    require(SUCCEEDED(font.supports(L"\U0001f60a",supported)) && supported,"system font shapes smile as one color glyph");
    require(SUCCEEDED(font.supports(L"\u2764\ufe0f",supported)) && supported,"variation selector retains native Emoji presentation");
    require(SUCCEEDED(font.supports(L"\U0001f60a\U0001f60a",supported)) && !supported,"two Emoji are not one supported sequence");
    require(SUCCEEDED(font.supports(L"\U0010ffff",supported)) && !supported,"missing system glyph is omitted");
    for (const auto *sequence:{L"\U0001f1e8\U0001f1f3",L"\U0001f469\u200d\U0001f4bb",L"\U0001fae8"}) {
        require(SUCCEEDED(font.supports(sequence,supported)),"query actual system sequence coverage");
        std::cout<<"Observed native sequence support: "<<(supported?"supported":"omitted")<<"\n";
    }
    HWND owner=CreateWindowExW(0,L"STATIC",L"Arion Emoji render fixture",WS_OVERLAPPEDWINDOW,40,40,500,200,nullptr,nullptr,GetModuleHandleW(nullptr),nullptr);
    require(owner!=nullptr,"create owned native editor"); ShowWindow(owner,SW_SHOW); SetFocus(owner);
    HWND focused=GetFocus();
    CandidateAction chosen; bool clicked=false;
    panel.onAction([&](const CandidateAction &value){chosen=value;clicked=true;});
    std::filesystem::path output=count>1?std::filesystem::u8path(arguments[1]):std::filesystem::path{};
    CandidatePalette colors; colors.surface=RGB(255,255,255); colors.selected=colors.surface;
    require(panel.show(owner,caret,{{L"\U0001f60a",L"",0,true}},0,false,false,2,colors,9,false,20,false),"show actual native Emoji");
    require(capture(panel.handle(),{})>30,"Emoji has colored interior pixels without a colored background");
    require(SUCCEEDED(panel.error()),"Direct2D color-font paint succeeds");
    for (bool dark:{false,true}) for (int size:{14,16,20}) {
        colors.surface=dark?RGB(32,32,32):RGB(255,255,255); colors.text=dark?RGB(245,245,245):RGB(20,17,15); colors.selected=RGB(255,204,0);
        require(panel.show(owner,caret,{{L"\u7b11\u8138",L"",0},{L"\u5fae\u7b11",L"",1},{L"\u5f00\u5fc3",L"",2},{L"\U0001f60a",L"",3,true}},3,false,true,3,colors,9,false,size),"show typed Emoji among Chinese candidates");
        auto file=output.empty()?output:output/((dark?"emoji-dark-":"emoji-light-")+std::to_string(size)+".bmp");
        capture(panel.handle(),file); require(SUCCEEDED(panel.error()),"render each theme and text size");
        RECT bounds{}; GetClientRect(panel.handle(),&bounds); clicked=false;
        SendMessageW(panel.handle(),WM_LBUTTONUP,0,MAKELPARAM(bounds.right-MulDiv(40,static_cast<int>(GetDpiForWindow(owner)),96),bounds.bottom/2));
        require(clicked && chosen.kind==CandidateAction::Select && chosen.index==3 && chosen.revision==3,"Emoji mouse hit carries exact engine identity and revision");
        require(GetFocus()==focused,"candidate paint never steals editor focus");
    }
    panel.hide(); DestroyWindow(owner);
    std::cout<<"PASS: native system Emoji shaping, unsupported-sequence suppression, color pixels, themes, sizes and nonactivating candidate surface.\n";
    } catch (const std::exception &error) { std::cerr<<error.what()<<"\n"; return 1; }
}
