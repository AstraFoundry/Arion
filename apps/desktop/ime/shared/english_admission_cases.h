#ifndef ARION_ENGLISH_ADMISSION_CASES_H
#define ARION_ENGLISH_ADMISSION_CASES_H
#include <array>
#include <filesystem>
#include <fstream>
#include <sstream>
#include <stdexcept>
#include <string>
#include <vector>
struct EnglishAdmissionCase {
    std::string id,input,expected;
    size_t maximumEnglish;
    bool chineseFirst;
};
inline std::vector<EnglishAdmissionCase> EnglishAdmissionCases(const std::filesystem::path &file) {
    std::ifstream stream(file);
    if (!stream) throw std::runtime_error("English admission cases unavailable");
    std::vector<EnglishAdmissionCase> cases;
    std::string line;
    while (std::getline(stream,line)) {
        if(line.empty() || line[0]=='#') continue;
        std::istringstream row(line);
        std::array<std::string,5> fields;
        for(auto &field:fields) if(!std::getline(row,field,'\t')) throw std::runtime_error("Invalid English admission case");
        if(row.peek()!=std::char_traits<char>::eof()) throw std::runtime_error("Extra English admission case fields");
        cases.push_back({fields[0],fields[1],fields[2],std::stoul(fields[3]),fields[4]=="1"});
    }
    if(!stream.eof() || cases.empty()) throw std::runtime_error("Incomplete English admission cases");
    return cases;
}
inline bool EnglishAdmissionWord(const std::string &word) {
    if(word.size()<2) return false;
    for(char c:word) if(!(c>='a' && c<='z') && !(c>='A' && c<='Z')) return false;
    return true;
}
#endif
