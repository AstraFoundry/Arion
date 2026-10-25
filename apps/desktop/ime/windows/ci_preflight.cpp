// Hosted CI only: prove the loaded standard-user profile can use GUI, TSF and
// current-user DPAPI before running the actual owned ACP acceptance fixtures.
#include <windows.h>
#include <msctf.h>
#include <sddl.h>
#include <wincrypt.h>
#include <cstdio>
#include <cstring>
#include <stdexcept>
#include "transport.h"

static void require(bool value, const char *stage) {
    if (!value) throw std::runtime_error(stage);
}
static void check(HRESULT result, const char *stage) { require(SUCCEEDED(result), stage); }
static std::wstring objectName(HANDLE object) {
    DWORD needed = 0;
    GetUserObjectInformationW(object, UOI_NAME, nullptr, 0, &needed);
    require(needed != 0, "user object name length");
    std::vector<wchar_t> name(needed / sizeof(wchar_t));
    require(GetUserObjectInformationW(object, UOI_NAME, name.data(), needed, &needed), "user object name");
    return name.data();
}

int wmain(int argc, wchar_t **argv) {
    HANDLE token = nullptr;
    HWND window = nullptr;
    ITfThreadMgr *manager = nullptr;
    ITfDocumentMgr *document = nullptr, *previous = nullptr, *focused = nullptr;
    ITfContext *context = nullptr;
    bool initialized = false, activated = false, pushed = false;
    DATA_BLOB encrypted{}, decrypted{};
    LPWSTR sidText = nullptr;
    int result = 1;
    try {
        require(argc == 4, "expected SID, window station and desktop arguments");
        require(OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &token), "OpenProcessToken");
        DWORD integrity = 0;
        require(ordinaryToken(token, integrity), "expected unelevated medium integrity token");
        std::vector<BYTE> user;
        require(tokenValue(token, TokenUser, user), "TokenUser");
        require(ConvertSidToStringSidW(reinterpret_cast<TOKEN_USER*>(user.data())->User.Sid, &sidText), "SID conversion");
        require(std::wstring(sidText) == argv[1], "unexpected test user SID");
        auto station = objectName(GetProcessWindowStation());
        auto desktop = objectName(GetThreadDesktop(GetCurrentThreadId()));
        require(station == argv[2] && desktop == argv[3], "unexpected inherited window station or desktop");
        DWORD session = 0;
        require(ProcessIdToSessionId(GetCurrentProcessId(), &session), "process session");
        check(CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED), "CoInitializeEx"); initialized = true;
        WNDCLASSW klass{}; klass.hInstance = GetModuleHandleW(nullptr);
        klass.lpfnWndProc = DefWindowProcW; klass.lpszClassName = L"ArionKeyboardCIPreflight";
        require(RegisterClassW(&klass), "RegisterClass");
        window = CreateWindowExW(WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE, klass.lpszClassName, L"", WS_POPUP,
            0, 0, 16, 16, nullptr, nullptr, klass.hInstance, nullptr);
        require(window != nullptr, "CreateWindowEx on inherited desktop");
        HDC dc = GetDC(window); require(dc != nullptr, "GetDC");
        BOOL drawn = PatBlt(dc, 0, 0, 16, 16, BLACKNESS); ReleaseDC(window, dc);
        require(drawn, "GDI draw");
        check(CoCreateInstance(CLSID_TF_ThreadMgr, nullptr, CLSCTX_INPROC_SERVER, IID_ITfThreadMgr, reinterpret_cast<void**>(&manager)), "TSF ThreadMgr");
        TfClientId client = 0;
        check(manager->Activate(&client), "TSF Activate"); activated = true;
        check(manager->CreateDocumentMgr(&document), "TSF CreateDocumentMgr");
        TfEditCookie cookie = 0;
        check(document->CreateContext(client, 0, nullptr, &context, &cookie), "TSF CreateContext");
        check(document->Push(context), "TSF Push"); pushed = true;
        check(manager->AssociateFocus(window, document, &previous), "TSF AssociateFocus");
        check(manager->SetFocus(document), "TSF SetFocus");
        check(manager->GetFocus(&focused), "TSF GetFocus");
        require(focused == document, "TSF focused document mismatch");
        BYTE synthetic[] = {0x41, 0x72, 0x69, 0x6f, 0x6e};
        DATA_BLOB input{sizeof(synthetic), synthetic};
        require(CryptProtectData(&input, L"Arion keyboard CI profile probe", nullptr, nullptr, nullptr, CRYPTPROTECT_UI_FORBIDDEN, &encrypted), "current-user DPAPI protect");
        require(CryptUnprotectData(&encrypted, nullptr, nullptr, nullptr, nullptr, CRYPTPROTECT_UI_FORBIDDEN, &decrypted), "current-user DPAPI unprotect");
        require(decrypted.cbData == sizeof(synthetic) && memcmp(decrypted.pbData, synthetic, sizeof(synthetic)) == 0, "DPAPI roundtrip mismatch");
        std::printf("{\"sid\":%s,\"elevated\":false,\"integrity\":%lu,\"session\":%lu,\"station\":%s,\"desktop\":%s,\"gui\":true,\"tsf\":true,\"dpapi\":true}\n",
            quoted(sidText).c_str(), integrity, session, quoted(station).c_str(), quoted(desktop).c_str());
        result = 0;
    } catch (const std::exception &error) {
        std::fprintf(stderr, "Keyboard CI preflight failed: %s (Win32 %lu)\n", error.what(), GetLastError());
    }
    if (decrypted.pbData) { SecureZeroMemory(decrypted.pbData, decrypted.cbData); LocalFree(decrypted.pbData); }
    if (encrypted.pbData) LocalFree(encrypted.pbData);
    if (focused) focused->Release();
    if (manager && window) { ITfDocumentMgr *old = nullptr; manager->AssociateFocus(window, previous, &old); if (old) old->Release(); }
    if (previous) previous->Release();
    if (manager) manager->SetFocus(nullptr);
    if (pushed) document->Pop(TF_POPF_ALL);
    if (context) context->Release();
    if (document) document->Release();
    if (activated) manager->Deactivate();
    if (manager) manager->Release();
    if (window) DestroyWindow(window);
    if (initialized) CoUninitialize();
    if (sidText) LocalFree(sidText);
    if (token) CloseHandle(token);
    return result;
}
