#include "learning.h"
#include <algorithm>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <iterator>
using namespace arion::ime;
static void require(bool ok,const char *message) {if (!ok) {std::cerr<<"Learning test failed: "<<message<<'\n';std::exit(1);}}
static LearningBytes read(const char *path) {std::ifstream file(path,std::ios::binary);require(file.good(),"open codec fixture");return LearningBytes(std::istreambuf_iterator<char>(file),{});}
class FailingProvider final : public LearningProvider {
    std::shared_ptr<LearningProvider> base_;
public:
    explicit FailingProvider(std::shared_ptr<LearningProvider> base):base_(std::move(base)) {}
    bool refuse=false;
    const std::string &directory() const override {return base_->directory();}
    bool cooperative() const override {return base_->cooperative();}
    bool lease(const std::string &p,bool r,std::unique_ptr<LearningLock> &h,LearningError &e) override {return base_->lease(p,r,h,e);}
    bool revision(const std::string &p,std::string &v,LearningError &e) override {return base_->revision(p,v,e);}
    bool lock(const std::string &p,bool r,bool x,std::unique_ptr<LearningLock> &h,LearningError &e) override {return base_->lock(p,r,x,h,e);}
    bool read(const std::string &p,LearningBytes &b,bool &m,LearningError &e) override {return base_->read(p,b,m,e);}
    bool writeAtomic(const std::string &p,const LearningBytes &b,LearningError &e) override {if (refuse) {e={6,"injected write failure"};return false;}return base_->writeAtomic(p,b,e);}
    bool remove(const std::string &p,LearningError &e) override {return base_->remove(p,e);}
    bool files(std::vector<std::string> &n,LearningError &e) override {return base_->files(n,e);}
    bool seal(const std::string &n,const LearningBytes &p,LearningBytes &s,LearningError &e) override {return base_->seal(n,p,s,e);}
    bool unseal(const std::string &n,const LearningBytes &s,LearningBytes &p,LearningError &e) override {return base_->unseal(n,s,p,e);}
};
int main(int argc,char **argv) {
    require(argc==4,"vault, old plist fixture and emitted plist paths required");
    LearningError error;LearningRecords legacy;
    require(decodeLearningRecords(read(argv[2]),legacy,error),"decode original NSData-pair binary plist");
    LearningRecords expected{{std::string("\0binary",7),std::string("\0\1\xff",3)},{"repeat","repeat"},{u8"ni hao \t 合成词","c=5 d=1 t=7"}};
    require(legacy==expected,"legacy values and embedded NUL preserved");
    LearningBytes encoded;require(encodeLearningRecords(legacy,encoded,error),"encode shared binary plist");
    std::ofstream emitted(argv[3],std::ios::binary);emitted.write(reinterpret_cast<const char*>(encoded.data()),encoded.size());emitted.close();
    LearningRecords decoded;require(decodeLearningRecords(encoded,decoded,error) && decoded==expected,"codec roundtrip");
    for (size_t size:{size_t{0},size_t{8},encoded.size()-1}) {auto bad=encoded;bad.resize(size);require(!decodeLearningRecords(bad,decoded,error),"truncated plist rejected");}
    auto malformed=encoded;malformed[8]=0xdf;require(!decodeLearningRecords(malformed,decoded,error),"wrong root type rejected");
    LearningRecords tooLarge{{"key",std::string(maximumLearningField+1,'x')}};require(!encodeLearningRecords(tooLarge,encoded,error),"per-field bound");
#ifdef _WIN32
    auto base=makeWindowsLearningProvider(argv[1],error);require(base!=nullptr,"create actual DPAPI provider");
    const auto file=learningPath(argv[1],"test.arion-userdb"),backup=learningPath(argv[1],"export.userdb.txt");
    auto provider=std::make_shared<FailingProvider>(base);
    LearningDatabase db(provider,file,"test");require(db.open(),"open new database");
    require(db.update("synthetic-secret-phrase","c=5"),"persist real ciphertext");
    auto ciphertext=read(file.c_str());const std::string canary="synthetic-secret-phrase";
    require(std::search(ciphertext.begin(),ciphertext.end(),canary.begin(),canary.end())==ciphertext.end(),"snapshot contains no plaintext term");
    require(std::string(ciphertext.begin(),ciphertext.begin()+8)=="ARIONWD1","Windows DPAPI envelope");
    require(db.update("synthetic-secret-phrase","c=5") && read(file.c_str())!=ciphertext,"fresh random encryption per write");
    std::string value;
    require(db.begin() && db.update("pending","value") && !db.fetch("pending",&value),"transaction writes invisible");
    auto beforeAbort=read(file.c_str());require(db.abort() && read(file.c_str())==beforeAbort,"abort leaves ciphertext unchanged");
    require(db.begin() && db.update("second","c=3") && db.commit() && db.fetch("second",&value),"commit transaction");
    require(db.backup(backup),"arbitrary-extension backup encrypted");
    auto exported=read(backup.c_str());require(std::search(exported.begin(),exported.end(),canary.begin(),canary.end())==exported.end(),"backup contains no plaintext term");
    require(db.erase("second") && db.restore(backup) && db.fetch("second",&value),"authenticated backup restore");
    LearningDatabase competing(base,file,"test");require(competing.open(),"cooperative writer opens beside active writer");
    require(competing.update("peer","value") && db.fetch("peer",&value) && value=="value","peer committed changes become visible");
    require(db.update("conflict","first") && !competing.update("conflict","second") && competing.error().code==10,"unknown concurrent same-key overwrite explicitly refused");
    require(db.fetch("conflict",&value) && value=="first","unsupported conflict preserves peer ciphertext");
    require(db.close() && !db.remove() && db.error().code==8,"remove refuses another process lifetime lease");
    require(competing.close(),"close cooperative peer");
    require(db.open(),"reopen after cooperative lease check");
    require(!resetLearningVault(base,error) && error.code==8,"reset refuses live database");
    require(db.close() && db.open() && db.fetch("second",&value) && value=="c=3","learning survives reopen");
    require(db.close(),"close before readonly access");
    LearningDatabase reader(base,file,"test");require(reader.open(true),"open readonly snapshot");
    LearningDatabase secondReader(base,file,"test");require(secondReader.open(true),"multiple readers permitted");
    require(!reader.update("forbidden","value"),"readonly write refused");require(reader.close() && secondReader.close(),"close readers");
    auto beforeAuthentication=read(file.c_str());LearningDatabase renamed(base,file,"wrong-dictionary");
    require(!renamed.open() && read(file.c_str())==beforeAuthentication,"DPAPI entropy binds dictionary identity");
    LearningBytes authenticated;require(!base->unseal("test",LearningBytes{'p','l','a','i','n'},authenticated,error),"plaintext import refused");
    auto tampered=beforeAuthentication;tampered.back()^=1;require(base->writeAtomic(file,tampered,error),"write isolated tamper fixture");
    require(!db.open() && read(file.c_str())==tampered,"authentication failure never replaces data");
    require(!validateLearningVault(base,error),"corrupt vault registration refused");
    require(base->writeAtomic(file,beforeAuthentication,error),"restore fixture ciphertext");
    require(validateLearningVault(base,error),"valid encrypted vault accepted");
    require(db.open() && db.fetch("second",&value),"reopen before injected persistence failure");
    provider->refuse=true;auto previous=read(file.c_str());
    require(!db.update("must-not-persist","c=1") && db.disabled() && !db.fetch("must-not-persist",&value),"failed persistence not accepted into memory");
    require(!db.update("next","c=1") && read(file.c_str())==previous,"later writes disabled after failure");
    require(db.close(),"close failed writer");provider->refuse=false;
    require(resetLearningVault(base,error),"explicit reset closed databases");
    require(!std::filesystem::exists(std::filesystem::u8path(file)),"reset removes ciphertext");
    require(std::filesystem::exists(std::filesystem::u8path(backup)),"explicit arbitrary export retained");
    std::cout<<"Shared codec, transactions and actual Windows DPAPI persistence passed.\n";
#else
    std::cout<<"Shared codec compatibility passed.\n";
#endif
}
