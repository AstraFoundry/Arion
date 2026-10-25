#include <iostream>
#include <cstring>
#include "json.h"
int main() {
    Json value;
    if (!JsonReader("{\"pageSize\":9,\"contextRanking\":true,\"highlight\":{\"light\":{\"fill\":\"#ffcc00\"}},\"items\":[\"\\ud83d\\ude42\"]}").read(value)) return 1;
    if (!value.get(L"pageSize") || value.get(L"pageSize")->number!=9 || value.get(L"items")->array[0].string!=L"\U0001f642") return 2;
    for (const char *text:{"{\"a\":1,\"a\":2}","[01]","[18446744073709551616]","[\"\\ud800\"]","[\"\\udc00\"]","[true,]","{} trailing","{\"x\":Infinity}"}) if (JsonReader(text).read(value)) return 3;
    std::string deep(18,'['); deep+="0"; deep+=std::string(18,']'); if (JsonReader(deep).read(value)) return 4;
    std::cout<<"PASS: strict nested settings/manifest parsing, UTF16 surrogate handling, duplicate/overflow/depth/trailing rejection.\n";
}
