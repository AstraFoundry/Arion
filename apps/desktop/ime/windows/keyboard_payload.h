#pragma once
#include <algorithm>
#include <set>
#include "json.h"
#include "payload.h"

#ifndef ARION_KEYBOARD_MANIFEST_SHA256
#define ARION_KEYBOARD_MANIFEST_SHA256 ""
#endif
struct KeyboardPayloadFile { std::wstring relative,sha256; uint64_t size=0; };
inline std::wstring keyboardNativePath(const std::wstring &root,const std::wstring &relative);
inline bool keyboardExactMembers(const std::wstring &root,const std::set<std::wstring> &files,const std::set<std::wstring> &directories,const std::wstring &relative=L"",size_t depth=0) {
    if (depth>32) return false;
    auto directory=relative.empty()?root:keyboardNativePath(root,relative);
    WIN32_FIND_DATAW entry{}; HANDLE search=FindFirstFileW((directory+L"\\*").c_str(),&entry);
    if (search==INVALID_HANDLE_VALUE) return false;
    bool valid=true;
    do {
        std::wstring name=entry.cFileName; if (name==L"." || name==L"..") continue;
        auto member=relative.empty()?name:relative+L"/"+name;
        if (entry.dwFileAttributes&(FILE_ATTRIBUTE_REPARSE_POINT|FILE_ATTRIBUTE_DEVICE)) { valid=false; break; }
        if (entry.dwFileAttributes&FILE_ATTRIBUTE_DIRECTORY) {
            if (!directories.count(member) || !keyboardExactMembers(root,files,directories,member,depth+1)) { valid=false; break; }
        } else if (!files.count(member)) { valid=false; break; }
    } while (FindNextFileW(search,&entry));
    if (valid && GetLastError()!=ERROR_NO_MORE_FILES) valid=false;
    FindClose(search); return valid;
}
inline bool keyboardRelativePath(const std::wstring &path) {
    if (path.empty() || path.front()==L'/' || path.back()==L'/') return false;
    size_t start=0;
    for (size_t i=0;i<=path.size();++i) {
        if (i<path.size() && path[i]!=L'/') { if (path[i]<32 || wcschr(L"\\:\"<>|?*",path[i])) return false; continue; }
        auto part=path.substr(start,i-start); if (part.empty() || part==L"." || part==L".." || part.back()==L'.' || part.back()==L' ') return false;
        start=i+1;
    }
    return true;
}
inline std::wstring keyboardNativePath(const std::wstring &root,const std::wstring &relative) {
    std::wstring path=relative; std::replace(path.begin(),path.end(),L'/',L'\\'); return root+L"\\"+path;
}
inline bool keyboardRegularPath(const std::wstring &root,const std::wstring &relative) {
    if (!keyboardRelativePath(relative)) return false;
    DWORD rootFlags=GetFileAttributesW(root.c_str()); if (rootFlags==INVALID_FILE_ATTRIBUTES || !(rootFlags&FILE_ATTRIBUTE_DIRECTORY) || (rootFlags&FILE_ATTRIBUTE_REPARSE_POINT)) return false;
    size_t offset=0;
    for (;;) {
        size_t end=relative.find(L'/',offset);
        auto part=relative.substr(0,end);
        DWORD flags=GetFileAttributesW(keyboardNativePath(root,part).c_str());
        if (flags==INVALID_FILE_ATTRIBUTES || (flags&FILE_ATTRIBUTE_REPARSE_POINT)) return false;
        if (end==std::wstring::npos) return !(flags&FILE_ATTRIBUTE_DIRECTORY);
        if (!(flags&FILE_ATTRIBUTE_DIRECTORY)) return false;
        offset=end+1;
    }
}
inline HRESULT verifyKeyboardPayload(const std::wstring &root,const std::wstring &expected,std::vector<KeyboardPayloadFile> &files) {
    if (!hashName(expected) || !keyboardRegularPath(root,L"runtime.json") || fileHash(root+L"\\runtime.json")!=expected) return TRUST_E_BAD_DIGEST;
    Json manifest; if (!readJsonFile(root+L"\\runtime.json",manifest)) return E_INVALIDARG;
    const Json *entries=manifest.get(L"files"); if (!entries || entries->type!=Json::Object || entries->object.empty() || entries->object.size()>2048) return E_INVALIDARG;
    std::set<std::wstring> unique,exactFiles{L"runtime.json"},directories; std::vector<KeyboardPayloadFile> next;
    for (const auto &[name,value]:entries->object) {
        const Json *hash=value.get(L"sha256"),*size=value.get(L"sizeBytes");
        if (!keyboardRelativePath(name) || name==L"runtime.json" || !hash || hash->type!=Json::String || !hashName(hash->string) || !size || size->type!=Json::Number) return E_INVALIDARG;
        std::wstring folded=name; CharLowerBuffW(folded.data(),static_cast<DWORD>(folded.size())); if (folded==L"runtime.json" || !unique.insert(folded).second) return E_INVALIDARG;
        exactFiles.insert(name);
        for (size_t slash=name.find(L'/');slash!=std::wstring::npos;slash=name.find(L'/',slash+1)) directories.insert(name.substr(0,slash));
        if (!keyboardRegularPath(root,name)) return E_ACCESSDENIED;
        WIN32_FILE_ATTRIBUTE_DATA attributes{};
        auto path=keyboardNativePath(root,name);
        if (!GetFileAttributesExW(path.c_str(),GetFileExInfoStandard,&attributes)) return HRESULT_FROM_WIN32(GetLastError());
        uint64_t bytes=(static_cast<uint64_t>(attributes.nFileSizeHigh)<<32)|attributes.nFileSizeLow;
        if (bytes!=size->number || fileHash(path)!=hash->string) return TRUST_E_BAD_DIGEST;
        next.push_back({name,hash->string,size->number});
    }
    if (!unique.count(L"lib/arionkeyboard.dll") || !unique.count(L"lib/arionrime.dll") || !unique.count(L"data/build/arion_pinyin.schema.yaml")) return E_INVALIDARG;
    if (!keyboardExactMembers(root,exactFiles,directories)) return E_ACCESSDENIED;
    files=std::move(next); return S_OK;
}
