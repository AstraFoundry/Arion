#include "learning_rime.h"
#include <rime/dict/db.h>
#include <cstdlib>
#include <iostream>
#include <memory>
using namespace arion::ime;
static void require(bool ok,const char *message) {if (!ok) {std::cerr<<"Rime learning test failed: "<<message<<'\n';std::exit(1);}}
int main(int argc,char **argv) {
    require(argc==2,"fresh owned vault path required");
#ifdef _WIN32
    LearningError error;auto provider=makeWindowsLearningProvider(argv[1],error);require(provider!=nullptr,"create native provider");
    require(registerEncryptedLearning(provider,error),"register actual Rime Db component");
    auto db=std::unique_ptr<rime::Db>(makeEncryptedRimeDatabase(provider,learningPath(argv[1],"test.arion-userdb"),"test"));
    require(db->Open(),"Rime adapter opens and initializes metadata");
    std::string key,value;require(db->MetaFetch("/db_name",&value) && value=="test","existing metadata format");
    auto metadata=db->QueryMetadata();require(metadata && metadata->GetNextRecord(&key,&value) && key[0]=='/',"metadata cursor strips byte prefix");
    require(db->Update("ni hao \t synthetic","c=1"),"Rime writes encrypted record");
    auto cursor=db->Query("ni ");require(cursor && cursor->GetNextRecord(&key,&value) && key=="ni hao \t synthetic" && cursor->exhausted(),"bounded Rime prefix cursor");
    require(db->Update("ni after","c=2"),"mutate after cursor snapshot");require(cursor->Reset() && cursor->GetNextRecord(&key,&value) && key=="ni hao \t synthetic" && cursor->exhausted(),"cursor retains transaction-era snapshot");
    auto transaction=dynamic_cast<rime::Transactional*>(db.get());require(transaction && transaction->BeginTransaction(),"pinned runtime RTTI transaction interface");
    require(db->Update("pending","c=1") && !db->Fetch("pending",&value),"Rime transaction pending invisible");
    require(transaction->AbortTransaction() && !db->Fetch("pending",&value),"Rime abort");
    require(transaction->BeginTransaction() && db->Update("committed","c=2") && transaction->CommitTransaction(),"Rime commit");
    require(db->Close() && db->Open() && db->Fetch("committed",&value),"Rime reopening persisted transaction");
    require(db->Close(),"close Rime database");
    auto context=openContextLearning(error);require(context!=nullptr,"shared context database opens");
    require(context->fetch("\x01/db_name",&value) && value=="arion_context","context metadata matches previous format");
    require(context->begin() && context->update("@tick","1") && context->update("c\tprevious\tword","1 1") && context->commit(),"context atomic pair/tick update");
    auto peer=openContextLearning(error);require(peer && !encryptedLearningError().code,"peer context opens cooperatively");
    require(peer->begin() && peer->update("@tick","2") && peer->update("c\tprevious\tword","2 2") && peer->commit(),"peer context confirmation committed");
    require(context->fetch("c\tprevious\tword",&value) && value=="2 2","context sees peer learning");
    peer.reset();
    context.reset();
    require(resetLearningVault(provider,error),"reset closes encrypted native databases");
    require(!encryptedLearningError().code,"successful common operations content-free error state");
    std::cout<<"Actual pinned Rime encrypted Db/cursor/transaction/context ABI passed.\n";
#endif
}
