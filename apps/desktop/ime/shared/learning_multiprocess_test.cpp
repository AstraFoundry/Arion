// Owned native processes, real current-user DPAPI and pinned Rime conversion.
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include "learning_rime.h"
#include "ranking.h"
#include "pinyin.h"
#include "english.h"
#include "emoji.h"
#include <rime/dict/db.h>
#include <rime/algo/dynamics.h>
#include <filesystem>
#include <iostream>
#include <memory>
#include <sstream>
#include <stdexcept>
#include <cmath>
using namespace arion::ime;
static void require(bool ok,const char *message) {if (!ok) throw std::runtime_error(message);}
static const std::string wordKey="ni hao \t" u8"你好";
static void unpack(const std::string &text,int &count,double &dee,uint64_t &tick) {
    unsigned long long value=0;require(std::sscanf(text.c_str(),"c=%d d=%lf t=%llu",&count,&dee,&value)==3,"canonical Rime record missing");tick=value;
}
class GateProvider final : public LearningProvider {
    std::shared_ptr<LearningProvider> base_;
public:
    bool holdWrite=false;
    explicit GateProvider(std::shared_ptr<LearningProvider> base):base_(std::move(base)) {}
    const std::string &directory() const override {return base_->directory();}
    bool cooperative() const override {return base_->cooperative();}
    bool lease(const std::string &p,bool r,std::unique_ptr<LearningLock> &h,LearningError &e) override {return base_->lease(p,r,h,e);}
    bool revision(const std::string &p,std::string &v,LearningError &e) override {return base_->revision(p,v,e);}
    bool lock(const std::string &p,bool r,bool x,std::unique_ptr<LearningLock> &h,LearningError &e) override {return base_->lock(p,r,x,h,e);}
    bool read(const std::string &p,LearningBytes &b,bool &m,LearningError &e) override {return base_->read(p,b,m,e);}
    bool writeAtomic(const std::string &p,const LearningBytes &b,LearningError &e) override {
        if (holdWrite) {std::cout<<"WRITE_LOCKED"<<std::endl;Sleep(INFINITE);}
        return base_->writeAtomic(p,b,e);
    }
    bool remove(const std::string &p,LearningError &e) override {return base_->remove(p,e);}
    bool files(std::vector<std::string> &n,LearningError &e) override {return base_->files(n,e);}
    bool seal(const std::string &n,const LearningBytes &p,LearningBytes &s,LearningError &e) override {return base_->seal(n,p,s,e);}
    bool unseal(const std::string &n,const LearningBytes &s,LearningBytes &p,LearningError &e) override {return base_->unseal(n,s,p,e);}
};
static int worker(const std::string &mode,const std::string &vault,const std::string &data,const std::string &user) {
    LearningError error;auto native=makeWindowsLearningProvider(vault,error);require(native!=nullptr,"worker provider");
    auto provider=std::make_shared<GateProvider>(native);require(registerEncryptedLearning(provider,error),"worker registration");
    RimeApi *api=nullptr;uintptr_t session=0,alternateSession=0;
    std::unique_ptr<rime::Db> db;std::unique_ptr<LearningDatabase> context;std::unique_ptr<ContextRanker> ranker;
    uint64_t localTick=0;
    if (mode=="engine") {
        std::filesystem::create_directories(std::filesystem::u8path(user));api=rime_get_api();
        std::string build=learningPath(data,"build");RimeTraits traits{};RIME_STRUCT_INIT(RimeTraits,traits);
        traits.shared_data_dir=data.c_str();traits.prebuilt_data_dir=build.c_str();traits.staging_dir=build.c_str();traits.user_data_dir=user.c_str();traits.app_name="rime.arion";traits.min_log_level=3;traits.log_dir="";
        api->setup(&traits);arion_emoji_register();api->initialize(&traits);
        char registrationError[512]={};
        if (!arion_english_register(data.c_str(),registrationError,sizeof(registrationError)))
            throw std::runtime_error(std::string("worker bilingual registration: ")+registrationError);
        session=arion_pinyin_create_session(api);alternateSession=arion_pinyin_create_session(api);require(session && alternateSession,"native worker sessions");
    } else if (mode=="context") {
        context=openContextLearning(error);require(context!=nullptr,"worker context");ranker=std::make_unique<ContextRanker>(context->records());
    } else {
        db.reset(makeEncryptedRimeDatabase(provider,learningPath(vault,"arion_pinyin.arion-userdb"),"arion_pinyin"));
        require(mode=="reader"?db->OpenReadOnly():db->Open(),"worker Db open");
        std::string tick;if (db->MetaFetch("/tick",&tick)) localTick=std::stoull(tick);
    }
    std::cout<<"READY"<<std::endl;std::string command;
    while (std::getline(std::cin,command)) {
        if (command=="quit") break;
        if (mode=="engine") {
            if (command=="flush") {
                arion_pinyin_process_key(api,session,0xff0d,0);ArionPinyinSnapshot empty{};require(!arion_pinyin_snapshot(api,session,&empty),"flush snapshot");arion_pinyin_snapshot_dispose(&empty);
                require(!encryptedLearningError().code,"flush learning failure");std::cout<<"FLUSHED"<<std::endl;continue;
            }
            const bool other=command=="other";require(command=="hello" || command=="alternate" || command=="prepare" || command=="select" || other,"engine command");
            uintptr_t active=command=="alternate"?alternateSession:session;
            const std::string keys=other?"xiexie":"nihao",expected=other?u8"谢谢":u8"你好";
            if (command!="select") for (unsigned char key:keys) require(arion_pinyin_process_key(api,active,key,0),"native Pinyin key");
            if (command=="prepare") {std::cout<<"PREPARED"<<std::endl;continue;}
            char **texts=nullptr;size_t count=0,index=SIZE_MAX;require(arion_pinyin_candidates(api,active,0,189,&texts,&count),"native candidates");
            for (size_t i=0;i<count;++i) if (texts[i]==expected) {index=i;break;}
            arion_pinyin_candidates_dispose(texts,count);require(index!=SIZE_MAX && arion_pinyin_select(api,active,index,0),"native candidate choice");
            ArionPinyinSnapshot snapshot{};require(!arion_pinyin_snapshot(api,active,&snapshot),"native commit snapshot");
            bool committed=snapshot.commit==expected;arion_pinyin_snapshot_dispose(&snapshot);require(committed,"native target commit");
            require(!encryptedLearningError().code,"native engine learning failure");std::cout<<"COMMITTED"<<std::endl;
        } else if (mode=="context") {
            require(command=="learn","context command");auto changes=ranker->learn(u8"热词","previous");require(context->begin(),"context begin");
            for (const auto &[key,value]:changes) require(value?context->update(key,*value):context->erase(key),"context stage");
            require(context->commit(),"context commit");std::cout<<"LEARNED"<<std::endl;
        } else if (command=="read") {
            std::string value,tick;int count=0;double dee=0;uint64_t entryTick=0;
            if (db->Fetch(wordKey,&value)) unpack(value,count,dee,entryTick);
            db->MetaFetch("/tick",&tick);require(!encryptedLearningError().code,"reader refresh failure");
            std::cout<<"VALUE "<<count<<' '<<(tick.empty()?"0":tick)<<' '<<dee<<' '<<entryTick<<std::endl;
        } else if (command=="abort") {
            require(dynamic_cast<rime::Transactional*>(db.get())->AbortTransaction(),"abort staged confirmation");std::cout<<"ABORTED"<<std::endl;
        } else if (command=="commit") {
            require(dynamic_cast<rime::Transactional*>(db.get())->CommitTransaction(),"commit staged confirmation");std::cout<<"SAVED"<<std::endl;
        } else {
            require(command=="stage" || command=="hold","Db command");
            auto transaction=dynamic_cast<rime::Transactional*>(db.get());require(transaction && transaction->BeginTransaction(),"begin native Db transaction");
            std::string value;int count=0;double dee=0;uint64_t tick=0;
            if (db->Fetch(wordKey,&value)) unpack(value,count,dee,tick);
            tick=std::min(tick,localTick);++localTick;
            require(db->MetaUpdate("/tick",std::to_string(localTick)),"stage local clock");
            std::ostringstream packed;packed<<"c="<<std::abs(count)+1<<" d="<<rime::algo::formula_d(1,static_cast<double>(localTick),dee,static_cast<double>(tick))<<" t="<<localTick;
            require(db->Update(wordKey,packed.str()),"stage native Db word");
            if (command=="hold") {provider->holdWrite=true;require(transaction->CommitTransaction(),"held commit");}
            std::cout<<"STAGED"<<std::endl;
        }
    }
    context.reset();db.reset();if (api) {arion_pinyin_destroy_session(api,session);arion_pinyin_destroy_session(api,alternateSession);api->finalize();}
    require(!encryptedLearningError().code,"worker finalization failure");std::cout<<"BYE"<<std::endl;return 0;
}
class Child {
    HANDLE process_=nullptr,input_=nullptr,output_=nullptr;
    static HANDLE job() {
        struct OwnedJob {
            HANDLE handle=CreateJobObjectW(nullptr,nullptr);
            OwnedJob() {JOBOBJECT_EXTENDED_LIMIT_INFORMATION policy{};policy.BasicLimitInformation.LimitFlags=JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE;require(handle && SetInformationJobObject(handle,JobObjectExtendedLimitInformation,&policy,sizeof(policy)),"owned child process lifetime policy");}
            ~OwnedJob() {CloseHandle(handle);}
        };
        static OwnedJob owned;return owned.handle;
    }
public:
    Child(const std::string &mode,const std::string &vault,const std::string &data,const std::string &user) {
        HANDLE ownedJob=job();
        wchar_t executable[32768]{};require(GetModuleFileNameW(nullptr,executable,32768)!=0,"owned executable identity");
        std::wstring line=L"\""+std::wstring(executable)+L"\" --worker";
        for (const auto &argument:{mode,vault,data,user}) line+=L" \""+std::filesystem::u8path(argument).wstring()+L"\"";
        SECURITY_ATTRIBUTES attributes{sizeof(attributes),nullptr,TRUE};HANDLE read=nullptr,write=nullptr;
        require(CreatePipe(&read,&input_,&attributes,0) && CreatePipe(&output_,&write,&attributes,0),"owned child channels");
        require(SetHandleInformation(input_,HANDLE_FLAG_INHERIT,0) && SetHandleInformation(output_,HANDLE_FLAG_INHERIT,0),"private parent channel ends");
        STARTUPINFOW startup{};startup.cb=sizeof(startup);startup.dwFlags=STARTF_USESTDHANDLES;startup.hStdInput=read;startup.hStdOutput=write;startup.hStdError=GetStdHandle(STD_ERROR_HANDLE);
        PROCESS_INFORMATION child{};bool ok=CreateProcessW(executable,line.data(),nullptr,nullptr,TRUE,CREATE_NO_WINDOW|CREATE_SUSPENDED,nullptr,nullptr,&startup,&child)!=FALSE;
        CloseHandle(read);CloseHandle(write);require(ok,"launch owned native child");process_=child.hProcess;
        if (!AssignProcessToJobObject(ownedJob,process_)) {TerminateProcess(process_,99);CloseHandle(child.hThread);CloseHandle(process_);process_=nullptr;CloseHandle(input_);input_=nullptr;CloseHandle(output_);output_=nullptr;throw std::runtime_error("own native child process job");}
        require(ResumeThread(child.hThread)!=static_cast<DWORD>(-1),"start owned child");CloseHandle(child.hThread);
        expect("READY");
    }
    ~Child() {if (process_) {if (WaitForSingleObject(process_,0)==WAIT_TIMEOUT) TerminateProcess(process_,99);WaitForSingleObject(process_,3000);CloseHandle(process_);}if (input_) CloseHandle(input_);if (output_) CloseHandle(output_);}
    void send(const std::string &command) {std::string line=command+"\n";DWORD count=0;require(WriteFile(input_,line.data(),static_cast<DWORD>(line.size()),&count,nullptr) && count==line.size(),"send child command");}
    std::string line() {
        std::string text;ULONGLONG deadline=GetTickCount64()+15000;
        while (GetTickCount64()<deadline) {
            DWORD available=0;require(PeekNamedPipe(output_,nullptr,0,nullptr,&available,nullptr)!=FALSE,"child output channel");
            if (!available) {require(WaitForSingleObject(process_,0)==WAIT_TIMEOUT,"child exited before reply");Sleep(1);continue;}
            char byte=0;DWORD count=0;require(ReadFile(output_,&byte,1,&count,nullptr) && count==1,"read child reply");
            if (byte=='\n') return text;
            if (byte!='\r') text.push_back(byte);
            require(text.size()<8192,"bounded child reply");
        }
        throw std::runtime_error("native child reply deadline");
    }
    void expect(const char *expected) {auto reply=line();if (reply!=expected) throw std::runtime_error("unexpected native reply: "+reply+" expected "+expected);}
    void finish() {send("quit");expect("BYE");require(WaitForSingleObject(process_,10000)==WAIT_OBJECT_0,"native child finish deadline");DWORD status=1;require(GetExitCodeProcess(process_,&status) && status==0,"native child success");}
    void kill() {require(TerminateProcess(process_,77) && WaitForSingleObject(process_,3000)==WAIT_OBJECT_0,"terminate only owned child");}
};
static void value(Child &reader,int expected,uint64_t tick,double expectedDee=-1) {
    reader.send("read");std::istringstream line(reader.line());std::string label;int count=0;uint64_t actualTick=0,entryTick=0;double dee=0;line>>label>>count>>actualTick>>dee>>entryTick;
    require(label=="VALUE" && count==expected && actualTick==tick,"reader count/global tick differ");
    require(entryTick<=actualTick,"entry clock exceeds global clock");
    if (expectedDee>=0) require(std::abs(dee-expectedDee)<0.0005,"merged Rime dynamic weight differs from sequential confirmation replay");
}
static void pass(const char *name) {std::cout<<"PASS "<<name<<std::endl;}
int main(int argc,char **argv) {
    try {
        if (argc==6 && std::string(argv[1])=="--worker") return worker(argv[2],argv[3],argv[4],argv[5]);
        require(argc==3,"owned parent directory and pinned data path required");std::string root=argv[1],data=argv[2];
        auto path=[&](const char *name){return learningPath(root,name);};
        const auto vault=path("shared");
        {Child seed("db",vault,data,path("seed"));seed.finish();}
        Child first("db",vault,data,path("first")),second("db",vault,data,path("second")),reader("reader",vault,data,path("reader"));
        first.send("stage");second.send("stage");first.expect("STAGED");second.expect("STAGED");value(reader,0,0);
        first.send("commit");second.send("commit");first.expect("SAVED");second.expect("SAVED");value(reader,2,2);
        pass("two-process-pending-private-and-same-word-increments");
        first.send("stage");first.expect("STAGED");second.send("stage");second.expect("STAGED");first.send("abort");first.expect("ABORTED");second.send("commit");second.expect("SAVED");value(reader,3,3);
        pass("abort-does-not-remove-peer-confirmation");first.finish();second.finish();reader.finish();
        {Child pending("db",vault,data,path("pending"));pending.send("stage");pending.expect("STAGED");pending.kill();}
        {Child check("reader",vault,data,path("check"));value(check,3,3);check.finish();}pass("process-kill-discards-private-pending-transaction");
        {Child held("db",vault,data,path("held")),survivor("db",vault,data,path("survivor"));held.send("hold");held.expect("WRITE_LOCKED");survivor.send("stage");Sleep(20);held.kill();survivor.expect("STAGED");survivor.send("commit");survivor.expect("SAVED");survivor.finish();}
        {Child check("reader",vault,data,path("check2"));value(check,4,4);check.finish();}pass("process-kill-releases-active-snapshot-write-lock");
        const auto nativeVault=path("engine");Child a("engine",nativeVault,data,path("engine-a")),b("engine",nativeVault,data,path("engine-b"));
        Child liveReader("reader",nativeVault,data,path("live-reader"));
        for (int i=0;i<12;++i) {a.send(i%2?"alternate":"hello");b.send(i%2?"alternate":"hello");a.expect("COMMITTED");b.expect("COMMITTED");}
        double sequential=0;for (int i=0;i<24;++i) {std::ostringstream packed;packed<<rime::algo::formula_d(1,i+1,sequential,i);sequential=std::stod(packed.str());}
        a.finish();b.finish();value(liveReader,24,24,sequential);liveReader.finish();pass("real-pinned-Rime-parallel-editors-count-clock-and-dynamic-weight");
        {Child c("engine",nativeVault,data,path("engine-c")),d("engine",nativeVault,data,path("engine-d"));c.send("hello");d.send("other");c.expect("COMMITTED");d.expect("COMMITTED");c.finish();d.finish();}
        {Child check("reader",nativeVault,data,path("check3"));value(check,25,26);check.finish();}pass("real-pinned-Rime-different-words-merge-and-fresh-process-reopen");
        const auto paced=path("paced");
        {Child slow("engine",paced,data,path("slow")),fast("engine",paced,data,path("fast"));slow.send("hello");slow.expect("COMMITTED");
            for (int i=0;i<3;++i) {fast.send("hello");fast.expect("COMMITTED");fast.send("flush");fast.expect("FLUSHED");}
            slow.send("prepare");slow.expect("PREPARED");slow.send("select");slow.expect("COMMITTED");slow.send("flush");slow.expect("FLUSHED");slow.finish();fast.finish();}
        {Child check("reader",paced,data,path("paced-check"));value(check,5,5);check.finish();}pass("real-pinned-Rime-unequal-editor-pace-and-pending-read-refresh");
        const auto contextVault=path("context");Child x("context",contextVault,data,path("context-a")),y("context",contextVault,data,path("context-b"));
        for (int i=0;i<12;++i) {x.send("learn");y.send("learn");x.expect("LEARNED");y.expect("LEARNED");}x.finish();y.finish();
        LearningError error;auto provider=makeWindowsLearningProvider(contextVault,error);require(provider && registerEncryptedLearning(provider,error),"reopen context vault");auto context=openContextLearning(error);require(context!=nullptr,"fresh-process context view");auto records=context->records();require(records["@tick"]=="24" && records[u8"c\tus\t热词"]=="24 24","merged context count/tick");
        ContextRanker ranked(records);require(ranked.order({u8"冷词","plain",u8"热词"},"previous")==std::vector<size_t>({2,1,0}),"peer learning changes ranking");
        require(!resetLearningVault(provider,error) && error.code==8,"reset refuses active shared lease");context.reset();require(resetLearningVault(provider,error),"reset after every lease closes");pass("context-pair-clock-ranking-and-active-reset");
        std::cout<<"All owned multi-process encrypted learning checks passed.\n";return 0;
    } catch (const std::exception &error) {std::cerr<<"FAIL "<<error.what()<<'\n';return 1;}
}
