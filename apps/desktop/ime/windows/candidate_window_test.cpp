#include <iostream>
#include <filesystem>
#include <fstream>
#include <stdexcept>
#include "candidate_window.h"
static void require(bool ok,const char *message) { if (!ok) throw std::runtime_error(message); }
struct SurfaceCapture {
    RECT area{};
    HDC dc=nullptr;
    HBITMAP bitmap=nullptr;
    HGDIOBJ previous=nullptr;
    unsigned char *pixels=nullptr;
    explicit SurfaceCapture(HWND window) {
        GetClientRect(window,&area);
        BITMAPINFO info{}; info.bmiHeader.biSize=sizeof(BITMAPINFOHEADER); info.bmiHeader.biWidth=area.right;
        info.bmiHeader.biHeight=-area.bottom; info.bmiHeader.biPlanes=1; info.bmiHeader.biBitCount=32;
        dc=CreateCompatibleDC(nullptr);
        bitmap=CreateDIBSection(dc,&info,DIB_RGB_COLORS,reinterpret_cast<void**>(&pixels),nullptr,0);
        require(bitmap && pixels,"create native candidate capture");
        previous=SelectObject(dc,bitmap);
        SendMessageW(window,WM_PRINTCLIENT,reinterpret_cast<WPARAM>(dc),PRF_CLIENT);
        GdiFlush();
    }
    ~SurfaceCapture() { SelectObject(dc,previous); DeleteObject(bitmap); DeleteDC(dc); }
    unsigned char alpha(int x,int y) const { return pixels[(y*area.right+x)*4+3]; }
    size_t colored() const {
        size_t count=0,bytes=static_cast<size_t>(area.right)*area.bottom*4;
        for (size_t i=0;i<bytes;i+=4) if (pixels[i+2]>140 && pixels[i+1]>80 && pixels[i+2]-pixels[i]>60 && pixels[i+1]-pixels[i]>40) ++count;
        return count;
    }
    void save(const std::filesystem::path &file) const {
        if (file.empty()) return;
        BITMAPV5HEADER info{}; info.bV5Size=sizeof(info); info.bV5Width=area.right; info.bV5Height=-area.bottom;
        info.bV5Planes=1; info.bV5BitCount=32; info.bV5Compression=BI_BITFIELDS;
        info.bV5RedMask=0x00ff0000; info.bV5GreenMask=0x0000ff00; info.bV5BlueMask=0x000000ff; info.bV5AlphaMask=0xff000000; info.bV5CSType=0x73524742;
        DWORD bytes=static_cast<DWORD>(area.right*area.bottom*4);
        BITMAPFILEHEADER header{}; header.bfType=0x4d42; header.bfOffBits=sizeof(header)+sizeof(info); header.bfSize=header.bfOffBits+bytes;
        std::ofstream out(file,std::ios::binary);
        out.write(reinterpret_cast<const char*>(&header),sizeof(header)); out.write(reinterpret_cast<const char*>(&info),sizeof(info)); out.write(reinterpret_cast<const char*>(pixels),bytes);
        require(out.good(),"save native rendering evidence");
    }
};
static size_t capture(HWND window,const std::filesystem::path &file) {
    SurfaceCapture surface(window); surface.save(file); return surface.colored();
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
        colors.surface=dark?RGB(32,32,32):RGB(255,255,255); colors.text=dark?RGB(245,245,245):RGB(20,17,15); colors.border=dark?RGB(70,70,70):RGB(215,215,215); colors.selected=RGB(255,204,0);
        require(panel.show(owner,caret,{{L"\u7b11\u8138",L"",0},{L"\u5fae\u7b11",L"",1},{L"\u5f00\u5fc3",L"",2},{L"\U0001f60a",L"",3,true}},3,false,true,3,colors,9,false,size),"show typed Emoji among Chinese candidates");
        auto file=output.empty()?output:output/((dark?"emoji-dark-":"emoji-light-")+std::to_string(size)+".bmp");
        capture(panel.handle(),file); require(SUCCEEDED(panel.error()),"render each theme and text size");
        RECT bounds{}; GetClientRect(panel.handle(),&bounds); clicked=false;
        SurfaceCapture surface(panel.handle());
        COLORREF surrounding=GetPixel(surface.dc,2,bounds.bottom/2);
        require(surrounding==colors.surface,"color-font drawing preserves the surrounding alpha surface");
        SendMessageW(panel.handle(),WM_LBUTTONUP,0,MAKELPARAM(bounds.right-MulDiv(40,static_cast<int>(GetDpiForWindow(owner)),96),bounds.bottom/2));
        require(clicked && chosen.kind==CandidateAction::Select && chosen.index==3 && chosen.revision==3,"Emoji mouse hit carries exact engine identity and revision");
        require(GetFocus()==focused,"candidate paint never steals editor focus");
    }
    for (int size:{14,16,20}) {
        colors.surface=RGB(32,32,32); colors.selected=RGB(255,204,0); colors.selectedText=RGB(20,17,15);
        require(panel.show(owner,caret,{{L"\u7684",L"",0}},0,false,false,4,colors,9,false,size),"show independent glyph alignment fixture");
        SurfaceCapture surface(panel.handle());
        int dpi=static_cast<int>(GetDpiForWindow(owner));
        CandidatePainter measurement;
        int numberWidth=0;
        for (wchar_t digit=L'1';digit<=L'9';++digit) {
            int width=0;
            require(SUCCEEDED(measurement.measure(std::wstring(1,digit),CandidateNumberFontFamily,static_cast<FLOAT>(MulDiv(std::max(11,size-3),dpi,96)),width)),"measure displayed label width");
            numberWidth=std::max(numberWidth,width);
        }
        int numberLeft=MulDiv(10,dpi,96),textLeft=numberLeft+numberWidth+MulDiv(3,dpi,96);
        auto inkCenter=[&](int left,int right) {
            int top=surface.area.bottom,bottom=-1;
            for (int y=MulDiv(4,dpi,96);y<surface.area.bottom-MulDiv(4,dpi,96);++y) for (int x=left;x<right;++x) {
                COLORREF pixel=GetPixel(surface.dc,x,y);
                if (GetRValue(pixel)<210 && GetGValue(pixel)<170 && GetBValue(pixel)<100) { top=std::min(top,y); bottom=std::max(bottom,y); }
            }
            require(bottom>=top,"find actual visible label glyph pixels");
            return (top+bottom)/2.0;
        };
        if (!output.empty()) surface.save(output/("alignment-"+std::to_string(size)+".bmp"));
        double numberCenter=inkCenter(numberLeft,numberLeft+numberWidth);
        double textCenter=inkCenter(textLeft,surface.area.right-MulDiv(10,dpi,96));
        std::cout<<"Glyph centers at "<<size<<": "<<numberCenter<<", "<<textCenter<<"\n";
        require(std::abs(numberCenter-textCenter)<=1.0,"number and Chinese visible glyphs share a vertical center within one pixel");
    }
    std::vector<NativeCandidate> compact;
    const std::wstring words=L"\u7684\u90fd\u5230\u5bf9\u5171\u591a\u70b9\u4f46\u6253";
    for (size_t i=0;i<words.size();++i) compact.push_back({words.substr(i,1),L"",100+i,false});
    for (bool dark:{false,true}) {
        colors.surface=dark?RGB(32,32,32):RGB(255,255,255); colors.text=dark?RGB(245,245,245):RGB(20,17,15); colors.border=dark?RGB(70,70,70):RGB(215,215,215);
        colors.muted=dark?RGB(145,145,145):RGB(110,110,110);
        for (size_t selected=0;selected<compact.size();++selected) {
            require(panel.show(owner,caret,compact,selected,false,true,10+selected,colors),"show compact nine-candidate row");
            RECT bounds{}; GetClientRect(panel.handle(),&bounds);
            require(bounds.right<MulDiv(500,static_cast<int>(GetDpiForWindow(owner)),96),"single-character row no longer reserves oversized cells");
            SurfaceCapture surface(panel.handle());
            require((GetWindowLongPtrW(panel.handle(),GWL_EXSTYLE)&WS_EX_LAYERED)!=0,"candidate uses a per-pixel alpha surface");
            require(surface.alpha(0,0)==0 && surface.alpha(bounds.right-1,0)==0 && surface.alpha(0,bounds.bottom-1)==0 && surface.alpha(bounds.right-1,bounds.bottom-1)==0 && surface.alpha(bounds.right/2,bounds.bottom/2)==255,"transparent corners retain an opaque content center");
            size_t fractional=0,selectionEdge=0;
            POINT hit{-1,-1};
            for (int y=0;y<bounds.bottom;++y) for (int x=0;x<bounds.right;++x) {
                auto alpha=surface.alpha(x,y);
                if (alpha>0 && alpha<255) ++fractional;
                auto pixel=GetPixel(surface.dc,x,y);
                if (pixel==colors.selected && hit.x<0) hit={x,y};
                // The highlight's upper edge contains opaque blended coverage,
                // distinct from both its solid fill and the panel background.
                if (y==MulDiv(4,static_cast<int>(GetDpiForWindow(owner)),96) && alpha==255 && pixel!=colors.selected && pixel!=colors.surface && GetRValue(pixel)>200 && GetBValue(pixel)<200) ++selectionEdge;
            }
            require(fractional>10,"outer round corners have fractional alpha coverage");
            require(selectionEdge>0,"highlight corners have antialiased blended coverage");
            require(hit.x>=0,"highlight has visible selectable pixels");
            clicked=false; SendMessageW(panel.handle(),WM_LBUTTONUP,0,MAKELPARAM(hit.x,hit.y));
            require(clicked && chosen.index==100+selected && chosen.revision==10+selected,"compact spacing preserves exact candidate hit identity");
            require(GetFocus()==focused,"compact candidate update never steals focus");
            if (!selected && !output.empty()) capture(panel.handle(),output/(dark?"compact-dark.bmp":"compact-light.bmp"));
        }
        std::vector<NativeCandidate> grid=compact; grid.insert(grid.end(),compact.begin(),compact.end());
        require(panel.show(owner,caret,grid,10,true,true,30,colors,9,true),"show compact expanded grid");
        require(SUCCEEDED(panel.error()),"expanded grid and shared typography render correctly");
        if (!output.empty()) capture(panel.handle(),output/(dark?"grid-dark.bmp":"grid-light.bmp"));
    }
    panel.hide(); DestroyWindow(owner);
    std::cout<<"PASS: native system Emoji shaping, unsupported-sequence suppression, color pixels, themes, sizes and nonactivating candidate surface.\n";
    } catch (const std::exception &error) { std::cerr<<error.what()<<"\n"; return 1; }
}
