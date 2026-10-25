#pragma once
#include <string>
#include <map>
#include <cstdint>
#include <windows.h>

inline std::wstring wide(const std::string &value) {
    int count = MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, value.data(), static_cast<int>(value.size()), nullptr, 0);
    if (count == 0 && !value.empty()) return {};
    std::wstring result(count, L'\0');
    MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, value.data(), static_cast<int>(value.size()), result.data(), count);
    return result;
}
inline std::string utf8(const std::wstring &value) {
    int count = WideCharToMultiByte(CP_UTF8, WC_ERR_INVALID_CHARS, value.data(), static_cast<int>(value.size()), nullptr, 0, nullptr, nullptr);
    if (count == 0 && !value.empty()) return {};
    std::string result(count, '\0');
    WideCharToMultiByte(CP_UTF8, WC_ERR_INVALID_CHARS, value.data(), static_cast<int>(value.size()), result.data(), count, nullptr, nullptr);
    return result;
}
inline std::string quoted(const std::wstring &value) {
    std::string result = "\"";
    for (unsigned char c : utf8(value)) {
        if (c == '"' || c == '\\') { result += '\\'; result += c; }
        else if (c < 32) { const char *hex = "0123456789abcdef"; result += "\\u00"; result += hex[c >> 4]; result += hex[c & 15]; }
        else result += c;
    }
    return result + '"';
}
struct WireMessage { std::wstring op, activation, session, text; uint64_t revision = 0; };

// The bridge accepts only a bounded flat JSON object. Decode Unicode escapes
// into UTF-16 so emoji and combining sequences reach TSF without conversion loss.
inline bool parseWire(const std::string &input, WireMessage &message) {
    size_t i = 0;
    auto space = [&] { while (i < input.size() && (input[i] == ' ' || input[i] == '\r' || input[i] == '\n' || input[i] == '\t')) ++i; };
    auto string = [&](std::wstring &out) {
        if (i >= input.size() || input[i++] != '"') return false;
        std::string chunk;
        auto flush = [&] { auto decoded = wide(chunk); if (decoded.empty() && !chunk.empty()) return false; out += decoded; chunk.clear(); return true; };
        while (i < input.size()) {
            unsigned char c = input[i++];
            if (c == '"') return flush();
            if (c < 32) return false;
            if (c != '\\') { chunk += c; continue; }
            if (!flush() || i == input.size()) return false;
            c = input[i++];
            if (c == 'u') {
                if (i + 4 > input.size()) return false;
                unsigned value = 0;
                for (int n = 0; n < 4; ++n) {
                    char h = input[i++]; int digit = h >= '0' && h <= '9' ? h - '0' : h >= 'a' && h <= 'f' ? h - 'a' + 10 : h >= 'A' && h <= 'F' ? h - 'A' + 10 : -1;
                    if (digit < 0) return false;
                    value = value * 16 + digit;
                }
                out += static_cast<wchar_t>(value);
            } else {
                switch(c) { case '"': case '\\': case '/': out += c; break; case 'n': out += L'\n'; break; case 'r': out += L'\r'; break; case 't': out += L'\t'; break; case 'b': out += L'\b'; break; case 'f': out += L'\f'; break; default: return false; }
            }
        }
        return false;
    };
    space(); if (i == input.size() || input[i++] != '{') return false;
    std::map<std::wstring, std::wstring> fields;
    for (;;) {
        space(); if (i < input.size() && input[i] == '}') { ++i; break; }
        std::wstring key, value;
        if (!string(key)) return false;
        space(); if (i == input.size() || input[i++] != ':') return false;
        space();
        if (key == L"revision") {
            if (i == input.size() || input[i] < '0' || input[i] > '9') return false;
            uint64_t revision = 0;
            while (i < input.size() && input[i] >= '0' && input[i] <= '9') {
                unsigned digit = input[i++] - '0';
                if (revision > (UINT64_MAX - digit) / 10) return false;
                revision = revision * 10 + digit;
            }
            message.revision = revision;
        } else {
            if (!string(value) || fields.count(key)) return false;
            fields[key] = value;
        }
        space(); if (i == input.size()) return false;
        if (input[i] == '}') { ++i; break; }
        if (input[i++] != ',') return false;
    }
    space(); if (i != input.size()) return false;
    message.op = fields[L"op"]; message.activation = fields[L"activation"]; message.session = fields[L"session"]; message.text = fields[L"text"];
    return !message.op.empty();
}
