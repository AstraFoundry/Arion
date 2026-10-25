#include "ranking_unicode.h"
#include <algorithm>
#include <cstdint>
namespace arion::ime {
namespace {
struct UnicodeRange { uint32_t first, last; uint16_t flags; };
#include "ranking_unicode_data.h"
uint16_t properties(uint32_t cp) {
    auto found=std::lower_bound(std::begin(unicodeRanges),std::end(unicodeRanges),cp,
        [](const UnicodeRange &range,uint32_t value){ return range.last<value; });
    return found!=std::end(unicodeRanges) && found->first<=cp ? found->flags : 0;
}
struct Point { uint32_t cp; size_t offset; uint16_t flags; };
bool decode(const std::string &text,std::vector<Point> &points) {
    for (size_t i=0;i<text.size();) {
        size_t start=i; uint8_t first=static_cast<uint8_t>(text[i++]); uint32_t cp=first; size_t extra=0;
        if (first>=0xc2 && first<=0xdf) {cp=first&31;extra=1;}
        else if (first>=0xe0 && first<=0xef) {cp=first&15;extra=2;}
        else if (first>=0xf0 && first<=0xf4) {cp=first&7;extra=3;}
        else if (first>=0x80) return false;
        if (extra>text.size()-i) return false;
        for (size_t n=0;n<extra;++n) {uint8_t next=static_cast<uint8_t>(text[i++]);if ((next&0xc0)!=0x80) return false;cp=(cp<<6)|(next&63);}
        if ((extra==1 && cp<0x80)||(extra==2 && cp<0x800)||(extra==3 && cp<0x10000)||cp>0x10ffff||(cp>=0xd800 && cp<=0xdfff)) return false;
        points.push_back({cp,start,properties(cp)});
    }
    return true;
}
bool boundary(const std::vector<Point> &points,size_t i) {
    const auto a=points[i-1].flags, b=points[i].flags; auto x=a&15, y=b&15;
    if (x==1 && y==2) return false;
    if (x==1 || x==2 || x==3 || y==1 || y==2 || y==3) return true;
    if (x==9 && (y==9 || y==10 || y==12 || y==13)) return false;
    if ((x==12 || x==10) && (y==10 || y==11)) return false;
    if ((x==13 || x==11) && y==11) return false;
    if (y==4 || y==5 || y==8 || x==7) return false;
    if ((b&96)==32) {
        size_t j=i; bool linker=false;
        while (j && ((points[j-1].flags&96)==64 || (points[j-1].flags&96)==96)) {linker=linker || (points[j-1].flags&96)==64;--j;}
        if (j && linker && (points[j-1].flags&96)==32) return false;
    }
    if ((b&16) && x==5) {
        size_t j=i-1; while (j && (points[j-1].flags&15)==4) --j;
        if (j && (points[j-1].flags&16)) return false;
    }
    if (x==6 && y==6) {size_t n=0,j=i;while (j && (points[j-1].flags&15)==6) {--j;++n;}if (n%2) return false;}
    return true;
}
}
bool rankingGraphemes(const std::string &text,std::vector<size_t> &result) {
    std::vector<Point> points; if (!decode(text,points)) return false;
    std::vector<size_t> next{0};
    for (size_t i=1;i<points.size();++i) if (boundary(points,i)) next.push_back(points[i].offset);
    if (!text.empty()) next.push_back(text.size());
    result=std::move(next); return true;
}
int rankingContainsHan(const std::string &text) {
    std::vector<Point> points;
    if (!decode(text,points)) return -1;
    return std::any_of(points.begin(),points.end(),[](const Point &point) { return (point.flags&256)!=0; })?1:0;
}
std::string rankingTrimmed(const std::string &text) {
    std::vector<Point> points;if (!decode(text,points)) return {};
    size_t end=text.size();while (!points.empty() && (points.back().flags&128)) {end=points.back().offset;points.pop_back();}
    return text.substr(0,end);
}
}
