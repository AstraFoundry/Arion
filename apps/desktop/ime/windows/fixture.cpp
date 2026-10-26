#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <msctf.h>
#include <richedit.h>
#include <string>
#include <vector>
#include <iostream>
#include <thread>
#include "identity.h"
#include "wire.h"

static HWND editor, password;
static LRESULT CALLBACK dispatch(HWND window, UINT message, WPARAM wp, LPARAM lp) {
    if (message == WM_APP + 1) {
        auto command = reinterpret_cast<std::string*>(lp);
        if (*command == "text") {
            int length = GetWindowTextLengthW(editor); std::wstring value(length + 1, L'\0'); GetWindowTextW(editor, value.data(), length + 1); value.resize(length);
            std::cout << "{\"text\":" << quoted(value) << "}" << std::endl;
        } else if (*command == "password") { SetFocus(password); std::cout << "{\"done\":true}" << std::endl; }
        else if (*command == "edit") { SetFocus(editor); std::cout << "{\"done\":true}" << std::endl; }
        else if (*command == "select") {
            if (GetForegroundWindow()!=window || GetFocus()!=editor) { std::cout << "{\"error\":\"focus\"}" << std::endl; }
            else { INPUT keys[2]{}; keys[0].type=keys[1].type=INPUT_KEYBOARD; keys[0].ki.wVk=keys[1].ki.wVk=VK_HOME; keys[1].ki.dwFlags=KEYEVENTF_KEYUP; if (SendInput(2,keys,sizeof(INPUT))!=2) std::cout << "{\"error\":\"input\"}" << std::endl; else std::cout << "{\"done\":true}" << std::endl; }
        }
        else if (*command == "reset") { SetFocus(editor); SetWindowTextW(editor,L"A\U0001f642\u4e2dZ"); SendMessageW(editor, EM_SETSEL, 3, 4); std::cout << "{\"done\":true}" << std::endl; }
        else if (*command == "readonly") { SendMessageW(editor,EM_SETREADONLY,TRUE,0); std::cout << "{\"done\":true}" << std::endl; }
        else if (*command == "writable") { SendMessageW(editor,EM_SETREADONLY,FALSE,0); std::cout << "{\"done\":true}" << std::endl; }
        else if (*command == "user") { SendMessageW(editor,EM_REPLACESEL,TRUE,reinterpret_cast<LPARAM>(L"user")); std::cout << "{\"done\":true}" << std::endl; }
        else if (*command == "quit") DestroyWindow(window);
        delete command; return 0;
    }
    if (message == WM_DESTROY) { PostQuitMessage(0); return 0; }
    return DefWindowProcW(window, message, wp, lp);
}
int wmain(int count, wchar_t **args) {
    if (count != 2 || FAILED(CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED))) return 2;
    auto rich = LoadLibraryW(L"Msftedit.dll"); auto dll = LoadLibraryW(args[1]);
    if (!rich || !dll) return 3;
    auto create = reinterpret_cast<HRESULT(__stdcall*)(REFCLSID, REFIID, void**)>(GetProcAddress(dll, "DllGetClassObject"));
    IClassFactory *factory = nullptr; ITfTextInputProcessor *service = nullptr; ITfThreadMgr *manager = nullptr;
    if (!create || FAILED(create(ArionService, IID_IClassFactory, reinterpret_cast<void**>(&factory)))) return 4;
    HRESULT hr = factory->CreateInstance(nullptr, IID_ITfTextInputProcessor, reinterpret_cast<void**>(&service)); factory->Release();
    if (FAILED(hr)) return 5;
    hr = CoCreateInstance(CLSID_TF_ThreadMgr, nullptr, CLSCTX_INPROC_SERVER, IID_ITfThreadMgr, reinterpret_cast<void**>(&manager));
    TfClientId client = 0; if (FAILED(hr) || FAILED(manager->Activate(&client))) return 6;
    ITfClientId *clients = nullptr;
    hr = manager->QueryInterface(IID_ITfClientId, reinterpret_cast<void**>(&clients));
    if (SUCCEEDED(hr)) { hr = clients->GetClientId(ArionService, &client); clients->Release(); }
    if (FAILED(hr)) return 8;
    HINSTANCE instance = GetModuleHandleW(nullptr); WNDCLASSW definition{}; definition.lpfnWndProc = dispatch; definition.hInstance = instance; definition.lpszClassName = L"ArionTSFOwnedFixture"; definition.hbrBackground = reinterpret_cast<HBRUSH>(COLOR_WINDOW + 1); RegisterClassW(&definition);
    HWND window = CreateWindowExW(0, definition.lpszClassName, L"Arion TSF owned test fixture", WS_OVERLAPPEDWINDOW | WS_VISIBLE, 150, 150, 600, 250, nullptr, nullptr, instance, nullptr);
    editor = CreateWindowExW(WS_EX_CLIENTEDGE, L"RICHEDIT50W", L"A\U0001f642\u4e2dZ", WS_CHILD | WS_VISIBLE | WS_TABSTOP | ES_MULTILINE, 20, 20, 540, 80, window, nullptr, instance, nullptr);
    SendMessageW(editor,EM_SETEDITSTYLE,SES_USECTF,SES_USECTF);
    password = CreateWindowExW(0, L"EDIT", L"", WS_CHILD | WS_VISIBLE | WS_TABSTOP | ES_PASSWORD, 20, 120, 300, 30, window, nullptr, instance, nullptr);
    SetForegroundWindow(window); SetFocus(editor); SendMessageW(editor, EM_SETSEL, 3, 3);
    hr = service->Activate(manager, client);
    if (FAILED(hr)) { std::cerr << "TSF activation failed: 0x" << std::hex << static_cast<unsigned long>(hr) << std::endl; return 7; }
    std::cout << "{\"ready\":true}" << std::endl;
    std::thread input([window] { std::string line; while (std::getline(std::cin, line)) { auto command = new std::string(line); if (!PostMessageW(window, WM_APP + 1, 0, reinterpret_cast<LPARAM>(command))) delete command; if (line == "quit") return; } PostMessageW(window, WM_CLOSE, 0, 0); });
    MSG message; while (GetMessageW(&message, nullptr, 0, 0) > 0) { TranslateMessage(&message); DispatchMessageW(&message); }
    service->Deactivate(); service->Release(); manager->Deactivate(); manager->Release();
    if (input.joinable()) input.join();
    FreeLibrary(dll); FreeLibrary(rich); CoUninitialize(); return 0;
}
