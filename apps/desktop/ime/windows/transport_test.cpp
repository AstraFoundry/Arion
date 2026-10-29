#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <iostream>
#include "transport.h"

static constexpr UINT Input = WM_APP + 71, Lost = WM_APP + 72;
static unsigned disconnected = 0, received = 0, stale = 0;
static PipeTransport *current = nullptr;
static LRESULT CALLBACK dispatch(HWND window, UINT message, WPARAM wp, LPARAM lp) {
    if (message == Input) {
        if (current && wp == current->generation()) ++received; else ++stale;
        delete reinterpret_cast<WireMessage*>(lp); return 0;
    }
    if (message == Lost) {
        if (current && wp == current->generation()) { ++disconnected; current->disconnect(); } else ++stale;
        return 0;
    }
    return DefWindowProcW(window, message, wp, lp);
}
static bool require(bool value, const char *name) { if (!value) std::cerr << "FAIL: " << name << std::endl; return value; }

static HANDLE server(const std::wstring &path) {
    return CreateNamedPipeW(path.c_str(), PIPE_ACCESS_DUPLEX | FILE_FLAG_OVERLAPPED | FILE_FLAG_FIRST_PIPE_INSTANCE,
        PIPE_TYPE_BYTE | PIPE_REJECT_REMOTE_CLIENTS, 1, 1024, 1024, 0, nullptr);
}
static bool writeServer(HANDLE pipe) {
    const std::string bytes = "{\"op\":\"begin\",\"activation\":\"same-context\",\"session\":\"fixture\"}\n";
    OVERLAPPED operation{}; operation.hEvent = CreateEventW(nullptr, TRUE, FALSE, nullptr);
    if (!operation.hEvent) return false;
    DWORD written = 0;
    bool sent = WriteFile(pipe, bytes.data(), static_cast<DWORD>(bytes.size()), &written, &operation) != FALSE;
    if (!sent && GetLastError() == ERROR_IO_PENDING) {
        if (WaitForSingleObject(operation.hEvent, 1000) != WAIT_OBJECT_0) CancelIoEx(pipe, &operation);
        sent = GetOverlappedResult(pipe, &operation, &written, TRUE) != FALSE;
    }
    CloseHandle(operation.hEvent);
    return sent && written == bytes.size();
}
static bool take(HWND window, UINT kind, MSG &message) {
    ULONGLONG deadline = GetTickCount64() + 2000;
    do {
        if (PeekMessageW(&message, window, kind, kind, PM_REMOVE)) return true;
        Sleep(2);
    } while (GetTickCount64() < deadline);
    return false;
}
static bool reconnectRetiresAlreadyRemovedMessages(HWND window) {
    PipeTransport transport; current = &transport;
    const std::wstring base = L"\\\\.\\pipe\\Arion.TSF.Transport.Epoch." + std::to_wstring(GetCurrentProcessId());
    HANDLE first = server(base + L".old"), next = INVALID_HANDLE_VALUE;
    MSG oldInput{}, oldLost{};
    bool heldInput = false, good = false;
    const unsigned lostBefore = disconnected, receivedBefore = received, staleBefore = stale;
    do {
        if (!require(first != INVALID_HANDLE_VALUE && transport.connect(base + L".old", window, Input, Lost), "old generation connects")) break;
        const uint64_t previous = transport.generation();
        if (!require(writeServer(first) && take(window, Input, oldInput), "outer pump removes old Incoming before reconnect")) break;
        heldInput = true;
        if (!require(oldInput.wParam == previous, "reader tags Incoming with its generation")) break;
        CloseHandle(first); first = INVALID_HANDLE_VALUE;
        if (!require(take(window, Lost, oldLost) && oldLost.wParam == previous, "outer pump removes tagged old Disconnected")) break;
        // disconnect already drains queued messages. It cannot recall a MSG
        // held by an outer pump across reentrant focus callbacks.
        transport.disconnect();
        if (!require(transport.generation() != previous, "disconnect retires its generation immediately")) break;
        next = server(base + L".new");
        if (!require(next != INVALID_HANDLE_VALUE && transport.connect(base + L".new", window, Input, Lost), "replacement generation connects")) break;
        if (!require(transport.generation() != previous, "replacement has a distinct generation")) break;
        DispatchMessageW(&oldInput); heldInput = false;
        DispatchMessageW(&oldLost);
        if (!require(transport.connected() && disconnected == lostBefore && received == receivedBefore && stale == staleBefore + 2,
            "held old messages cannot begin or disconnect the replacement")) break;
        MSG fresh{};
        if (!require(writeServer(next) && take(window, Input, fresh), "replacement reader remains live")) break;
        const bool tagged = fresh.wParam == transport.generation();
        DispatchMessageW(&fresh);
        if (!require(tagged && received == receivedBefore + 1, "current-generation Incoming is accepted")) break;
        good = true;
    } while (false);
    if (heldInput) delete reinterpret_cast<WireMessage*>(oldInput.lParam);
    transport.disconnect(); current = nullptr;
    if (first != INVALID_HANDLE_VALUE) CloseHandle(first);
    if (next != INVALID_HANDLE_VALUE) CloseHandle(next);
    return good;
}

static bool vocabularyWireCases() {
    WireMessage message;
    if (!require(parseWire(R"({"op":"vocabulary_query","activation":"a","request":18446744073709551615})",message) && message.request==UINT64_MAX,"lossless vocabulary request")) return false;
    if (!require(parseWire(R"({"op":"vocabulary_publish","activation":"a","request":1,"vocabularyRevision":2,"terms":[{"id":"term","text":"星槎","reading":"xing cha"}]})",message) && message.terms.size()==1 && message.terms[0].text==u8"星槎","decode bounded personal term")) return false;
    if (!require(parseWire(R"({"op":"vocabulary_publish","activation":"a","request":2,"vocabularyRevision":3,"terms":[]})",message) && message.terms.empty(),"explicit empty snapshot clears prior terms")) return false;
    const char *invalid[]={
        R"({"op":"vocabulary_query","activation":"a","request":true})",
        R"({"op":"vocabulary_query","activation":"a","request":1.0})",
        R"({"op":"vocabulary_query","activation":"a","request":-1})",
        R"({"op":"vocabulary_query","activation":"a","request":18446744073709551616})",
        R"({"op":"vocabulary_query","activation":"a","request":1,"request":2})",
        R"({"op":"vocabulary_query","activation":"a","request":01})",
        R"({"op":"vocabulary_query","activation":"a","request":0})",
        R"({"op":"vocabulary_publish","activation":"a","request":1,"vocabularyRevision":1})",
        R"({"op":"vocabulary_publish","activation":"a","request":1,"vocabularyRevision":1,"terms":null})",
        R"({"op":"vocabulary_publish","activation":"a","request":1,"vocabularyRevision":1,"terms":[{"id":"x\u0000y","text":"词","reading":"ci"}]})",
        R"({"op":"mark","revision":1,"revision":2})",
        R"({"op":"mark","text":"\ud800"})",
        R"({"op":"mark",})"
    };
    for (const auto input:invalid) {
        if (!require(!parseWire(input,message),"reject malformed wire fields")) return false;
        if (!require(message.request==2 && message.vocabularyRevision==3 && message.terms.empty(),"invalid frame preserves prior message")) return false;
    }
    if (!require(parseWire(R"({"op":"mark","activation":"a","session":"s","revision":5,"text":"\ud83d\ude00"})",message) && utf8(message.text)==u8"😀","voice Unicode survives shared parser")) return false;
    return true;
}
int main() {
    if (!vocabularyWireCases()) return 1;
    if (!ordinaryHost()) {
        HANDLE diagnostic = nullptr; TOKEN_ELEVATION elevation{}; DWORD size = 0;
        bool opened = OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &diagnostic) != FALSE;
        bool queried = opened && GetTokenInformation(diagnostic, TokenElevation, &elevation, sizeof(elevation), &size);
        std::vector<BYTE> data; DWORD level = 0;
        bool labelRead = opened && tokenValue(diagnostic, TokenIntegrityLevel, data);
        if (labelRead) { PSID sid = reinterpret_cast<TOKEN_MANDATORY_LABEL*>(data.data())->Label.Sid; level = *GetSidSubAuthority(sid, *GetSidSubAuthorityCount(sid) - 1); }
        std::cerr << "Tests require ordinary integrity; tokenOpen=" << opened << " elevationRead=" << queried << " elevated=" << elevation.TokenIsElevated << " labelRead=" << labelRead << " integrity=" << level << " error=" << GetLastError() << std::endl;
        if (diagnostic) CloseHandle(diagnostic);
        return 77;
    }
    if (!require(!trustedPipeServer(INVALID_HANDLE_VALUE), "invalid server identity is refused")) return 1;
    HANDLE token = nullptr, low = nullptr;
    if (!OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY | TOKEN_DUPLICATE, &token) ||
        !DuplicateTokenEx(token, TOKEN_QUERY | TOKEN_ADJUST_DEFAULT, nullptr, SecurityImpersonation, TokenPrimary, &low)) return 2;
    SID_IDENTIFIER_AUTHORITY authority = SECURITY_MANDATORY_LABEL_AUTHORITY; PSID sid = nullptr;
    if (!AllocateAndInitializeSid(&authority, 1, SECURITY_MANDATORY_LOW_RID, 0,0,0,0,0,0,0, &sid)) return 2;
    TOKEN_MANDATORY_LABEL label{}; label.Label.Sid = sid; label.Label.Attributes = SE_GROUP_INTEGRITY;
    bool lowered = SetTokenInformation(low, TokenIntegrityLevel, &label, sizeof(label) + GetLengthSid(sid)) != FALSE;
    DWORD integrity = 0;
    bool rejected = lowered && !ordinaryToken(low, integrity);
    FreeSid(sid); CloseHandle(low); CloseHandle(token);
    if (!require(rejected, "low-integrity token is refused")) return 1;

    HINSTANCE instance = GetModuleHandleW(nullptr);
    WNDCLASSW definition{}; definition.lpfnWndProc = dispatch; definition.hInstance = instance; definition.lpszClassName = L"ArionTSFTransportTests";
    if (!RegisterClassW(&definition)) return 2;
    HWND window = CreateWindowExW(0, definition.lpszClassName, L"", 0, 0,0,0,0, HWND_MESSAGE, nullptr, instance, nullptr);
    if (!window) return 2;
    for (unsigned iteration = 0; iteration < 3; ++iteration) {
        std::wstring path = L"\\\\.\\pipe\\Arion.TSF.Transport.Test." + std::to_wstring(GetCurrentProcessId()) + L"." + std::to_wstring(iteration);
        HANDLE server = CreateNamedPipeW(path.c_str(), PIPE_ACCESS_DUPLEX | FILE_FLAG_OVERLAPPED | FILE_FLAG_FIRST_PIPE_INSTANCE,
            PIPE_TYPE_BYTE | PIPE_REJECT_REMOTE_CLIENTS, 1, 1024, 1024, 0, nullptr);
        if (server == INVALID_HANDLE_VALUE) return 2;
        PipeTransport transport;
        current = &transport;
        if (!require(transport.connect(path, window, Input, Lost), "same-user/session/integrity server connects")) return 1;
        if (iteration == 1) {
            ULONGLONG start = GetTickCount64();
            transport.send(std::string(512 * 1024, 'x'));
            if (!require(GetTickCount64() - start < 250, "sender returns without waiting for server")) return 1;
            // The server deliberately never reads. The writer must expire
            // without blocking the editor's message loop or the reader.
            while (!disconnected && GetTickCount64() - start < 2500) {
                MSG message;
                while (PeekMessageW(&message, nullptr, 0, 0, PM_REMOVE)) DispatchMessageW(&message);
                Sleep(5);
            }
            if (!require(disconnected == 1, "stalled server produces bounded disconnect")) return 1;
        } else if (iteration == 2) {
            transport.send(std::string(512 * 1024, 'x'));
            Sleep(20);
        }
        ULONGLONG start = GetTickCount64();
        transport.disconnect();
        if (!require(GetTickCount64() - start < 500, "pending read/write cancellation joins promptly")) return 1;
        if (!require(!transport.connected(), "disconnected handle is retired")) return 1;
        CloseHandle(server);
        current = nullptr;
    }
    if (!reconnectRetiresAlreadyRemovedMessages(window)) return 1;
    DestroyWindow(window);
    std::cout << "PASS: peer identity, integrity rejection, nonblocking send, stalled server timeout, pending read/write cancellation, held-message generation rejection" << std::endl;
    return 0;
}
