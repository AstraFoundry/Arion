// Loads the production module exactly as a native host, using synthetic data.
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <cstring>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <thread>
#include <atomic>
#include "keyboard_runtime.h"
#include "emoji_text.h"
#include "../shared/pinyin.h"
static void require(bool condition,const char *message) { if (!condition) throw std::runtime_error(message); }
template<typename Function> static Function symbol(HMODULE module,const char *name) {
    FARPROC address=GetProcAddress(module,name); require(address!=nullptr,name);
    Function result; static_assert(sizeof(result)==sizeof(address)); std::memcpy(&result,&address,sizeof(result)); return result;
}
// A separate editor process holds the same shared lifetime vault lock as the
// production provider. No database bytes are changed by this peer.
static int holdVault(wchar_t **arguments) {
    HANDLE ready=OpenEventW(EVENT_MODIFY_STATE,FALSE,arguments[3]);
    HANDLE done=OpenEventW(SYNCHRONIZE,FALSE,arguments[4]);
    HANDLE file=CreateFileW(arguments[2],GENERIC_READ|GENERIC_WRITE,FILE_SHARE_READ|FILE_SHARE_WRITE,
        nullptr,OPEN_EXISTING,FILE_ATTRIBUTE_NORMAL,nullptr);
    OVERLAPPED position{};
    if (!ready || !done || file==INVALID_HANDLE_VALUE || !LockFileEx(file,LOCKFILE_FAIL_IMMEDIATELY,0,1,0,&position)) return 3;
    SetEvent(ready);
    const DWORD result=WaitForSingleObject(done,30000);
    UnlockFileEx(file,0,1,0,&position); CloseHandle(file); CloseHandle(ready); CloseHandle(done);
    return result==WAIT_OBJECT_0?0:4;
}
class VaultPeer {
    HANDLE job=nullptr,ready=nullptr,done=nullptr,process=nullptr;
public:
    ~VaultPeer() {
        if (done) SetEvent(done);
        if (process) { WaitForSingleObject(process,5000); CloseHandle(process); }
        if (job) CloseHandle(job);
        if (ready) CloseHandle(ready);
        if (done) CloseHandle(done);
    }
    void start(const std::filesystem::path &vault) {
        const auto name=L"Local\\ArionResetTest-"+std::to_wstring(GetCurrentProcessId())+L"-"+std::to_wstring(GetTickCount64());
        const auto readyName=name+L"-ready",doneName=name+L"-done";
        ready=CreateEventW(nullptr,TRUE,FALSE,readyName.c_str());
        done=CreateEventW(nullptr,TRUE,FALSE,doneName.c_str());
        job=CreateJobObjectW(nullptr,nullptr);
        require(ready && done && job,"create isolated peer synchronization");
        JOBOBJECT_EXTENDED_LIMIT_INFORMATION limits{};
        limits.BasicLimitInformation.LimitFlags=JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE;
        require(SetInformationJobObject(job,JobObjectExtendedLimitInformation,&limits,sizeof(limits)),"bound peer lifetime");
        wchar_t executable[32768]{};
        require(GetModuleFileNameW(nullptr,executable,32768)!=0,"locate peer fixture");
        std::wstring command=L"\""+std::wstring(executable)+L"\" --hold-vault \""+vault.wstring()+L"\" \""+readyName+L"\" \""+doneName+L"\"";
        STARTUPINFOW startup{}; startup.cb=sizeof(startup); PROCESS_INFORMATION child{};
        require(CreateProcessW(executable,command.data(),nullptr,nullptr,FALSE,CREATE_NO_WINDOW|CREATE_SUSPENDED,
            nullptr,nullptr,&startup,&child),"launch separate vault peer");
        process=child.hProcess;
        if (!AssignProcessToJobObject(job,process)) { TerminateProcess(process,3); CloseHandle(child.hThread); throw std::runtime_error("assign peer lifetime"); }
        ResumeThread(child.hThread); CloseHandle(child.hThread);
        HANDLE observed[]={ready,process};
        require(WaitForMultipleObjects(2,observed,FALSE,10000)==WAIT_OBJECT_0,"peer acquired shared vault lease");
    }
};
int wmain(int count,wchar_t **arguments) {
    if (count==5 && std::wcscmp(arguments[1],L"--hold-vault")==0) return holdVault(arguments);
    if (count!=4) return 2;
    try {
        HMODULE module=LoadLibraryExW(arguments[1],nullptr,LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR|LOAD_LIBRARY_SEARCH_SYSTEM32);
        require(module!=nullptr,"load absolute module with restricted dependency search");
        auto start=symbol<decltype(&arion_keyboard_start)>(module,"arion_keyboard_start");
        auto stop=symbol<decltype(&arion_keyboard_stop)>(module,"arion_keyboard_stop");
        auto enter=symbol<decltype(&arion_keyboard_enter)>(module,"arion_keyboard_enter");
        auto leave=symbol<decltype(&arion_keyboard_leave)>(module,"arion_keyboard_leave");
        auto error=symbol<decltype(&arion_keyboard_error)>(module,"arion_keyboard_error");
        auto generation=symbol<decltype(&arion_keyboard_generation)>(module,"arion_keyboard_generation");
        auto reset=symbol<decltype(&arion_keyboard_reset_learning)>(module,"arion_keyboard_reset_learning");
        auto order=symbol<decltype(&arion_keyboard_order)>(module,"arion_keyboard_order");
        auto learn=symbol<decltype(&arion_keyboard_learn)>(module,"arion_keyboard_learn");
        auto context=symbol<decltype(&arion_keyboard_context)>(module,"arion_keyboard_context");
        auto metadata=symbol<decltype(&arion_keyboard_candidate_metadata)>(module,"arion_keyboard_candidate_metadata");
        auto refresh=symbol<decltype(&arion_keyboard_refresh_emoji)>(module,"arion_keyboard_refresh_emoji");
        const auto data=std::filesystem::path(arguments[2]).u8string(),user=std::filesystem::path(arguments[3]).u8string();
        char failure[512]{};
        RimeApi *api=start(data.c_str(),user.c_str(),failure,sizeof(failure));
        if (!api) std::cerr<<failure<<"\n";
        require(api!=nullptr,"production module startup");
        require(start(data.c_str(),user.c_str(),failure,sizeof(failure))==api,"second client shares runtime");
        require(start(data.c_str(),(user+"-different").c_str(),failure,sizeof(failure))==nullptr,"different profile refuses reuse");
        stop();
        enter();
        auto emojiSession=arion_pinyin_create_session(api);
        require(emojiSession!=0,"create Emoji metadata session");
        api->set_option(emojiSession,"arion_emoji",1);
        for (char key:std::string("xiaolian")) require(arion_pinyin_process_key(api,emojiSession,key,0),"type exact Emoji trigger");
        auto emojiIndex=[&]() {
            char **texts=nullptr; size_t count=0,index=static_cast<size_t>(-1);
            require(arion_pinyin_candidates(api,emojiSession,0,54,&texts,&count)!=0,"read actual typed engine menu");
            for (size_t i=0;i<count;++i) {
                ArionEmojiCandidateMetadata value{};
                require(metadata(emojiSession,i,&value)!=0,"read metadata across validated module ABI");
                if (value.emoji && std::strcmp(texts[i],u8"😊")==0) {
                    require(value.start==0 && value.end==8 && std::strcmp(value.source,u8"笑脸")==0 && std::strcmp(value.sequence_id,"1F60A")==0,"exact Emoji identity and input range");
                    index=i;
                }
            }
            arion_pinyin_candidates_dispose(texts,count); return index;
        };
        require(emojiIndex()!=static_cast<size_t>(-1),"native font permits smile candidate");
        arion_pinyin_process_key(api,emojiSession,0xff50,0); require(refresh(emojiSession)!=0,"refresh at Home");
        require(emojiIndex()==static_cast<size_t>(-1),"middle caret removes Emoji");
        arion_pinyin_process_key(api,emojiSession,0xff57,0); require(refresh(emojiSession)!=0,"refresh at End");
        auto index=emojiIndex(); require(index!=static_cast<size_t>(-1),"end caret restores Emoji");
        require(arion_pinyin_select(api,emojiSession,index,0)!=0,"select actual Emoji engine candidate");
        ArionPinyinSnapshot emojiCommit{};
        require(arion_pinyin_snapshot(api,emojiSession,&emojiCommit)==0 && emojiCommit.commit && std::strcmp(emojiCommit.commit,u8"😊")==0,"module commits exact Unicode Emoji");
        arion_pinyin_snapshot_dispose(&emojiCommit);
        api->set_option(emojiSession,"arion_emoji",0);
        for (char key:std::string("xiaolian")) arion_pinyin_process_key(api,emojiSession,key,0);
        require(emojiIndex()==static_cast<size_t>(-1),"disabled session emits no Emoji");
        arion_pinyin_destroy_session(api,emojiSession);
        leave();
        auto convert=[&] {
            enter();
            bool ok=true;
            auto session=arion_pinyin_create_session(api);
            ok=session!=0;
            if (ok) {
                arion_pinyin_set_options(api,session,0,0,0);
                for (char key:std::string("nihao")) ok=arion_pinyin_process_key(api,session,key,0)&&ok;
                ok=arion_pinyin_process_key(api,session,' ',0)&&ok;
                ArionPinyinSnapshot snapshot{};
                ok=arion_pinyin_snapshot(api,session,&snapshot)==0&&ok;
                ok=snapshot.commit && std::strcmp(snapshot.commit,"\xe4\xbd\xa0\xe5\xa5\xbd")==0&&ok;
                arion_pinyin_snapshot_dispose(&snapshot);
                arion_pinyin_destroy_session(api,session);
            }
            leave(); return ok;
        };
        require(convert(),"remaining client converts after balanced release");
        std::atomic<bool> parallelOK{true};
        auto worker=[&] { for (int i=0;i<4;++i) if (!convert()) parallelOK=false; };
        std::thread first(worker),second(worker); first.join(); second.join();
        require(parallelOK,"serialized calls from multiple host threads");
        require(error(failure,sizeof(failure))==0,"no learning/runtime failure");
        const char *candidates[]={u8"你好",u8"拟好",u8"你",u8"妮好"};
        auto rankedFirst=[&] {
            size_t indices[4]{};
            require(order(candidates,4,u8"很好",indices,failure,sizeof(failure))!=0,"read encrypted context ranking");
            return indices[0];
        };
        auto teach=[&] {
            for (int i=0;i<3;++i) require(learn(u8"妮好",u8"很好",failure,sizeof(failure))!=0,"persist actual context confirmation");
            require(rankedFirst()==3,"learned full candidate promoted through module ABI");
        };
        char previous[64]{};
        require(context(u8"今天天气很好。 ",previous,sizeof(previous))!=0 && std::strcmp(previous,u8"很好")==0,"module uses shared context extraction");
        require(rankedFirst()==0,"fresh profile retains producer order");
        auto contextBytes=[&] {
            std::ifstream bytes(std::filesystem::path(arguments[3])/L"learning"/L"arion_context.arion-userdb",std::ios::binary);
            require(bytes.good(),"read owned context ciphertext");
            return std::string(std::istreambuf_iterator<char>(bytes),std::istreambuf_iterator<char>());
        };
        const auto unlearned=contextBytes();
        for (const char *word:{"Latin",u8"é🙂かな한글",""}) {
            for (int i=0;i<3;++i) require(learn(word,u8"很好",failure,sizeof(failure))!=0 && !failure[0],"non-Han learning is successful no-op");
        }
        for (const auto &bad:{std::string("\xc0\xaf"),std::string("\xed\xa0\x80"),std::string("\xf4\x90\x80\x80"),std::string(u8"你")+"\xe2\x82"})
            require(learn(bad.c_str(),u8"很好",failure,sizeof(failure))==0 && failure[0],"module rejects malformed UTF-8 even after Han");
        require(contextBytes()==unlearned,"non-Han and invalid commits never rewrite context ciphertext");
        const char *hanCases[][2]={{u8"A你好",u8"A妮好"},{u8"你好",u8"\U00020000好"}};
        for (const auto &mixed:hanCases) {
            const char *word=mixed[1];
            for (int i=0;i<3;++i) require(learn(word,u8"很好",failure,sizeof(failure))!=0,"mixed/supplementary Han is learned");
            size_t indices[2]{};
            require(order(mixed,2,u8"很好",indices,failure,sizeof(failure))!=0 && indices[0]==1,"Han confirmations promote through actual encrypted module ABI");
        }
        teach();
        {
            VaultPeer peer;
            peer.start(std::filesystem::path(arguments[3])/L"learning"/L".learning.lock");
            const auto beforeRefusal=generation();
            require(reset(failure,sizeof(failure))==0 && failure[0],"peer-active reset is explicitly refused");
            require(generation()!=beforeRefusal,"refused reset invalidates finalized sessions");
            require(error(failure,sizeof(failure))==0 && convert(),"refused reset restores usable real conversion");
            require(rankedFirst()==3,"refused reset preserves confirmed context learning");
        }
        enter();
        auto stale=arion_pinyin_create_session(api);
        require(stale!=0,"create active session before reset");
        arion_pinyin_process_key(api,stale,'n',0);
        const auto beforeReset=generation();
        leave();
        require(reset(failure,sizeof(failure))!=0,"reset owned synthetic learning profile");
        require(generation()!=beforeReset,"reset changes session generation");
        enter();
        const bool invalidated=api->find_session(stale)==0;
        leave();
        require(invalidated && convert(),"expired session is gone and replacement converts");
        require(rankedFirst()==0,"reset closes active context database and clears learned ranking");
        teach();
        const auto beforeStop=generation();
        stop();
        require(generation()!=beforeStop,"final release changes session generation");
        require(error(failure,sizeof(failure))!=0,"final release shuts down runtime");
        api=start(data.c_str(),user.c_str(),failure,sizeof(failure));
        require(api!=nullptr && convert(),"reopen encrypted profile after complete shutdown");
        require(rankedFirst()==3,"context ranking survives module lifecycle restart");
        {
            VaultPeer peer;
            peer.start(std::filesystem::path(arguments[3])/L"learning"/L".learning.lock");
            const auto file=std::filesystem::path(arguments[3])/L"learning"/L"arion_context.arion-userdb";
            std::fstream bytes(file,std::ios::in|std::ios::out|std::ios::binary);
            char first=0; bytes.read(&first,1); require(bytes.good(),"read owned encrypted tamper fixture");
            first^=0x40; bytes.seekp(0); bytes.write(&first,1); bytes.close();
            require(reset(failure,sizeof(failure))==0,"busy reset cannot recover a tampered vault");
            require(error(failure,sizeof(failure))!=0,"tampered recovery stays unavailable");
            require(start(data.c_str(),user.c_str(),failure,sizeof(failure))==nullptr,"new client cannot reuse finalized failed runtime");
            size_t indices[4]{};
            require(order(candidates,4,u8"很好",indices,failure,sizeof(failure))==0,"failed runtime refuses context reads");
            require(learn(u8"妮好",u8"很好",failure,sizeof(failure))==0,"failed runtime refuses further learning writes");
            require(learn("Latin",u8"很好",failure,sizeof(failure))==0,"non-Han no-op never bypasses failed runtime readiness");
        }
        stop(); FreeLibrary(module);
        std::cout<<"PASS: restricted DLL load, shared clients, real conversion, thread serialization, Han-only learning/no-op ciphertext, invalid UTF-8 refusal, reset generation, peer-busy recovery, encrypted-profile reopen and tampered recovery refusal.\n";
        return 0;
    } catch (const std::exception &error) { std::cerr<<error.what()<<"\n"; return 1; }
}
