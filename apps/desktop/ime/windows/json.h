#pragma once
#include "wire.h"
#include <cstring>
#include <map>
#include <vector>

// Bounded settings/manifest reader. No execution, YAML coercion or permissive
// duplicate keys; offsets/lengths remain bounded before allocation or reads.
struct Json {
    enum Type { Null,Boolean,Number,String,Object,Array } type=Null;
    bool boolean=false;
    uint64_t number=0;
    std::wstring string;
    std::map<std::wstring,Json> object;
    std::vector<Json> array;
    const Json *get(const std::wstring &name) const { auto found=object.find(name); return type==Object && found!=object.end() ? &found->second : nullptr; }
};
class JsonReader {
    const std::string &input;
    size_t position=0,nodes=0;
    void space() { while (position<input.size() && (input[position]==' ' || input[position]=='\t' || input[position]=='\r' || input[position]=='\n')) ++position; }
    bool literal(const char *text) { size_t count=strlen(text); if (input.compare(position,count,text)) return false; position+=count; return true; }
    bool unit(unsigned &value) {
        if (input.size()-position<4) return false;
        value=0;
        for (int n=0;n<4;++n) { char c=input[position++]; int digit=c>='0'&&c<='9'?c-'0':c>='a'&&c<='f'?c-'a'+10:c>='A'&&c<='F'?c-'A'+10:-1; if (digit<0) return false; value=value*16+digit; }
        return true;
    }
    bool text(std::wstring &output) {
        if (position==input.size() || input[position++]!='"') return false;
        std::string chunk;
        auto flush=[&] { auto value=wide(chunk); if (!chunk.empty() && value.empty()) return false; output+=value; chunk.clear(); return true; };
        while (position<input.size()) {
            unsigned char c=input[position++];
            if (c=='"') return flush();
            if (c<32) return false;
            if (c!='\\') { chunk+=c; continue; }
            if (!flush() || position==input.size()) return false;
            c=input[position++];
            if (c=='u') {
                unsigned value=0; if (!unit(value) || (value>=0xdc00 && value<=0xdfff)) return false;
                output+=static_cast<wchar_t>(value);
                if (value>=0xd800 && value<=0xdbff) {
                    if (input.size()-position<6 || input[position++]!='\\' || input[position++]!='u') return false;
                    if (!unit(value) || value<0xdc00 || value>0xdfff) return false;
                    output+=static_cast<wchar_t>(value);
                }
            } else switch(c) { case '"': case '\\': case '/': output+=c; break; case 'n': output+=L'\n'; break; case 'r': output+=L'\r'; break; case 't': output+=L'\t'; break; case 'b': output+=L'\b'; break; case 'f': output+=L'\f'; break; default: return false; }
        }
        return false;
    }
    bool value(Json &output,size_t depth) {
        space(); if (position==input.size() || depth>16 || ++nodes>20000) return false;
        char c=input[position];
        if (c=='"') { output.type=Json::String; return text(output.string); }
        if (c=='t' || c=='f') { output.type=Json::Boolean; output.boolean=c=='t'; return literal(c=='t' ? "true":"false"); }
        if (c=='n') return literal("null");
        if (c>='0' && c<='9') {
            output.type=Json::Number; size_t start=position;
            while (position<input.size() && input[position]>='0' && input[position]<='9') { unsigned digit=input[position++]-'0'; if (output.number>(UINT64_MAX-digit)/10) return false; output.number=output.number*10+digit; }
            return position==start+1 || input[start]!='0';
        }
        if (c=='{' || c=='[') {
            bool object=c=='{'; output.type=object?Json::Object:Json::Array; ++position; space();
            if (position<input.size() && input[position]==(object?'}':']')) { ++position; return true; }
            for (;;) {
                std::wstring key;
                if (object) { space(); if (!text(key) || output.object.count(key)) return false; space(); if (position==input.size() || input[position++]!=':') return false; }
                Json item; if (!value(item,depth+1)) return false;
                if (object) output.object.emplace(std::move(key),std::move(item)); else output.array.push_back(std::move(item));
                space(); if (position==input.size()) return false;
                if (input[position]==(object?'}':']')) { ++position; return true; }
                if (input[position++]!=',') return false;
            }
        }
        return false;
    }
public:
    explicit JsonReader(const std::string &text):input(text) {}
    bool read(Json &output) { if (input.size()>4*1024*1024) return false; Json next; if (!value(next,0)) return false; space(); if (position!=input.size()) return false; output=std::move(next); return true; }
};
inline bool readJsonFile(const std::wstring &path,Json &value) {
    HANDLE file=CreateFileW(path.c_str(),GENERIC_READ,FILE_SHARE_READ|FILE_SHARE_DELETE,nullptr,OPEN_EXISTING,FILE_FLAG_OPEN_REPARSE_POINT,nullptr);
    if (file==INVALID_HANDLE_VALUE) return false;
    BY_HANDLE_FILE_INFORMATION info{}; LARGE_INTEGER size{};
    bool ok=GetFileInformationByHandle(file,&info) && !(info.dwFileAttributes&(FILE_ATTRIBUTE_REPARSE_POINT|FILE_ATTRIBUTE_DIRECTORY)) && GetFileSizeEx(file,&size) && size.QuadPart>=0 && size.QuadPart<=4*1024*1024;
    std::string contents; if (ok) contents.resize(static_cast<size_t>(size.QuadPart));
    DWORD count=0; if (ok) ok=ReadFile(file,contents.data(),static_cast<DWORD>(contents.size()),&count,nullptr) && count==contents.size();
    CloseHandle(file); return ok && JsonReader(contents).read(value);
}
