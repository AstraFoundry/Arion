#pragma once
#include <windows.h>
#include <atomic>
#include <cstdint>
#include <deque>
#include <mutex>
#include <string>
#include <thread>
#include <vector>
#include "wire.h"

inline bool tokenValue(HANDLE token, TOKEN_INFORMATION_CLASS kind, std::vector<BYTE> &value) {
    DWORD size = 0;
    GetTokenInformation(token, kind, nullptr, 0, &size);
    if (!size) return false;
    value.resize(size);
    return GetTokenInformation(token, kind, value.data(), size, &size) != FALSE;
}

// Voice composition is supported only between ordinary desktop processes.
// A per-user pipe must never turn a lower-integrity process into an input
// source for an elevated editor, or cross a Windows logon session.
inline bool ordinaryToken(HANDLE token, DWORD &integrity) {
    TOKEN_ELEVATION elevation{}; DWORD size = 0;
    if (!GetTokenInformation(token, TokenElevation, &elevation, sizeof(elevation), &size) || elevation.TokenIsElevated) return false;
    std::vector<BYTE> label;
    if (!tokenValue(token, TokenIntegrityLevel, label)) return false;
    PSID sid = reinterpret_cast<TOKEN_MANDATORY_LABEL*>(label.data())->Label.Sid;
    if (!IsValidSid(sid) || !*GetSidSubAuthorityCount(sid)) return false;
    integrity = *GetSidSubAuthority(sid, *GetSidSubAuthorityCount(sid) - 1);
    return integrity >= SECURITY_MANDATORY_MEDIUM_RID && integrity < SECURITY_MANDATORY_HIGH_RID;
}

inline bool ordinaryHost() {
    HANDLE token = nullptr; DWORD integrity = 0;
    if (!OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &token)) return false;
    bool allowed = ordinaryToken(token, integrity);
    CloseHandle(token); return allowed;
}

inline bool trustedPipeServer(HANDLE pipe) {
    ULONG pid = 0; DWORD ownSession = 0, serverSession = 0;
    if (!GetNamedPipeServerProcessId(pipe, &pid) || !pid ||
        !ProcessIdToSessionId(GetCurrentProcessId(), &ownSession) ||
        !ProcessIdToSessionId(pid, &serverSession) || ownSession != serverSession) return false;
    HANDLE process = OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, FALSE, pid);
    if (!process) return false;
    HANDLE own = nullptr, server = nullptr;
    bool allowed = OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &own) && OpenProcessToken(process, TOKEN_QUERY, &server);
    DWORD ownIntegrity = 0, serverIntegrity = 0;
    std::vector<BYTE> ownUser, serverUser;
    if (allowed) allowed = ordinaryToken(own, ownIntegrity) && ordinaryToken(server, serverIntegrity) && ownIntegrity == serverIntegrity &&
        tokenValue(own, TokenUser, ownUser) && tokenValue(server, TokenUser, serverUser) &&
        EqualSid(reinterpret_cast<TOKEN_USER*>(ownUser.data())->User.Sid, reinterpret_cast<TOKEN_USER*>(serverUser.data())->User.Sid);
    if (own) CloseHandle(own);
    if (server) CloseHandle(server);
    CloseHandle(process); return allowed;
}

// No I/O or COM call runs in send(). The workers own only bytes and native
// pipe handles. Each pending operation has its own event and remains alive
// until cancellation completes, before disconnect joins either worker.
class PipeTransport final {
    HANDLE pipe = INVALID_HANDLE_VALUE, stop = nullptr, ready = nullptr;
    HWND window = nullptr;
    UINT incoming = 0, disconnected = 0;
    std::thread reader, writer;
    std::atomic<bool> stopping{true};
    std::mutex mutex;
    std::deque<std::string> queue;
    size_t queuedBytes = 0;
    // Only the owning UI thread changes or reads this counter. Workers receive
    // an immutable copy, so disconnect can retire messages before joining I/O.
    uint64_t epoch = 0;

    void failed(uint64_t generation) { if (!stopping.exchange(true)) { SetEvent(stop); PostMessageW(window, disconnected, static_cast<WPARAM>(generation), 0); } }

    bool transfer(bool writing, void *bytes, DWORD size, DWORD &count, HANDLE event, DWORD timeout = INFINITE) {
        OVERLAPPED operation{}; operation.hEvent = event;
        ResetEvent(event); count = 0;
        BOOL complete = writing ? WriteFile(pipe, bytes, size, &count, &operation) : ReadFile(pipe, bytes, size, &count, &operation);
        if (complete) return count != 0;
        if (GetLastError() != ERROR_IO_PENDING) return false;
        HANDLE waits[] = {stop, event};
        DWORD result = WaitForMultipleObjects(2, waits, FALSE, timeout);
        if (result != WAIT_OBJECT_0 + 1) {
            CancelIoEx(pipe, &operation);
            // Named-pipe cancellation completes the operation; retain its
            // event and buffer until the kernel has released both.
            GetOverlappedResult(pipe, &operation, &count, TRUE);
            return false;
        }
        return GetOverlappedResult(pipe, &operation, &count, FALSE) && count != 0;
    }

    void read(uint64_t generation) {
        HANDLE event = CreateEventW(nullptr, TRUE, FALSE, nullptr);
        if (!event) { failed(generation); return; }
        std::string line; char buffer[4096]; DWORD count = 0;
        while (!stopping && transfer(false, buffer, sizeof(buffer), count, event)) {
            for (DWORD i = 0; i < count; ++i) {
                if (buffer[i] == '\n') {
                    auto message = new WireMessage;
                    if (!parseWire(line, *message) || !PostMessageW(window, incoming, static_cast<WPARAM>(generation), reinterpret_cast<LPARAM>(message))) delete message;
                    line.clear();
                } else {
                    line += buffer[i];
                    if (line.size() > (1u << 20)) { CloseHandle(event); failed(generation); return; }
                }
            }
        }
        CloseHandle(event); failed(generation);
    }

    void write(uint64_t generation) {
        HANDLE event = CreateEventW(nullptr, TRUE, FALSE, nullptr);
        if (!event) { failed(generation); return; }
        HANDLE waits[] = {stop, ready};
        while (!stopping && WaitForMultipleObjects(2, waits, FALSE, INFINITE) == WAIT_OBJECT_0 + 1) {
            for (;;) {
                std::string value;
                { std::lock_guard<std::mutex> lock(mutex);
                    if (stopping || queue.empty()) break;
                    value = std::move(queue.front()); queue.pop_front(); queuedBytes -= value.size();
                }
                size_t offset = 0;
                ULONGLONG deadline = GetTickCount64() + 1000;
                while (offset < value.size() && !stopping) {
                    DWORD count = 0;
                    ULONGLONG now = GetTickCount64();
                    if (now >= deadline || !transfer(true, value.data() + offset, static_cast<DWORD>(value.size() - offset), count, event, static_cast<DWORD>(deadline - now))) { failed(generation); break; }
                    offset += count;
                }
                if (stopping) break;
            }
        }
        CloseHandle(event);
    }
public:
    ~PipeTransport() { disconnect(); }
    bool connected() const { return pipe != INVALID_HANDLE_VALUE; }
    uint64_t generation() const { return epoch; }
    bool connect(const std::wstring &path, HWND target, UINT inputMessage, UINT disconnectMessage) {
        if (connected()) return false;
        HANDLE handle = CreateFileW(path.c_str(), GENERIC_READ | GENERIC_WRITE, 0, nullptr, OPEN_EXISTING,
            FILE_FLAG_OVERLAPPED | SECURITY_SQOS_PRESENT | SECURITY_IDENTIFICATION, nullptr);
        if (handle == INVALID_HANDLE_VALUE) return false;
        if (!trustedPipeServer(handle)) { CloseHandle(handle); return false; }
        stop = CreateEventW(nullptr, TRUE, FALSE, nullptr);
        ready = CreateEventW(nullptr, FALSE, FALSE, nullptr);
        if (!stop || !ready) {
            if (stop) CloseHandle(stop);
            if (ready) CloseHandle(ready);
            stop = ready = nullptr; CloseHandle(handle); return false;
        }
        pipe = handle; window = target; incoming = inputMessage; disconnected = disconnectMessage; stopping = false;
        const uint64_t generation = ++epoch;
        try { reader = std::thread([this, generation] { read(generation); }); writer = std::thread([this, generation] { write(generation); }); }
        catch (...) { disconnect(); return false; }
        return true;
    }
    void send(std::string value) {
        if (!connected() || stopping) return;
        { std::lock_guard<std::mutex> lock(mutex);
            if (queue.size() >= 64 || value.size() > (1u << 20) || queuedBytes + value.size() > (1u << 20)) { failed(epoch); return; }
            queuedBytes += value.size(); queue.push_back(std::move(value));
        }
        SetEvent(ready);
    }
    void disconnect() {
        ++epoch;
        stopping = true;
        if (stop) SetEvent(stop);
        if (pipe != INVALID_HANDLE_VALUE) CancelIoEx(pipe, nullptr);
        if (reader.joinable()) reader.join();
        if (writer.joinable()) writer.join();
        if (pipe != INVALID_HANDLE_VALUE) CloseHandle(pipe);
        pipe = INVALID_HANDLE_VALUE;
        if (stop) CloseHandle(stop);
        if (ready) CloseHandle(ready);
        stop = ready = nullptr;
        { std::lock_guard<std::mutex> lock(mutex); queue.clear(); queuedBytes = 0; }
        if (window) {
            MSG pending;
            while (PeekMessageW(&pending, window, incoming, incoming, PM_REMOVE)) delete reinterpret_cast<WireMessage*>(pending.lParam);
            while (PeekMessageW(&pending, window, disconnected, disconnected, PM_REMOVE)) {}
        }
        window = nullptr;
    }
};
