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
