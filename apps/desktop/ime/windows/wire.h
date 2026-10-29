#pragma once
#include "json.h"

struct WireTerm { std::string id,text,reading; };
struct WireMessage {
    std::wstring op,activation,session,text;
    uint64_t revision=0,request=0,vocabularyRevision=0;
    std::vector<WireTerm> terms;
};

// Use the same strict bounded decoder as settings and manifests. Build a fresh
// message so a rejected frame cannot partially mutate a previous request.
inline bool parseWire(const std::string &input,WireMessage &message) {
    if (input.size()>(1u<<20)) return false;
    Json root;
    if (!JsonReader(input).read(root) || root.type!=Json::Object) return false;
    WireMessage next;
    auto text=[&](const wchar_t *key,std::wstring &out) {
        const auto value=root.get(key);
        if (!value) return true;
        if (value->type!=Json::String) return false;
        out=value->string; return true;
    };
    auto number=[&](const wchar_t *key,uint64_t &out) {
        const auto value=root.get(key);
        if (!value) return true;
        if (value->type!=Json::Number) return false;
        out=value->number; return true;
    };
    if (!text(L"op",next.op) || next.op.empty() || !text(L"activation",next.activation) ||
        !text(L"session",next.session) || !text(L"text",next.text) || !number(L"revision",next.revision) ||
        !number(L"request",next.request) || !number(L"vocabularyRevision",next.vocabularyRevision)) return false;
    if (next.op==L"vocabulary_query" || next.op==L"vocabulary_publish") {
        if (!next.request || next.activation.empty()) return false;
        if (next.op==L"vocabulary_publish") {
            const auto terms=root.get(L"terms");
            if (!next.vocabularyRevision || !terms || terms->type!=Json::Array || terms->array.size()>2000) return false;
            for (const auto &item:terms->array) {
                WireTerm term;
                auto field=[&](const wchar_t *key,std::string &out,size_t limit) {
                    const auto value=item.get(key);
                    if (!value || value->type!=Json::String) return false;
                    out=utf8(value->string);
                    return !out.empty() && out.size()<=limit && out.find('\0')==std::string::npos;
                };
                if (!field(L"id",term.id,128) || !field(L"text",term.text,256) || !field(L"reading",term.reading,448)) return false;
                next.terms.push_back(std::move(term));
            }
        }
    }
    message=std::move(next); return true;
}
