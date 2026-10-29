#import "typing_fixture.h"
#import "../learning.h"
#include "../../../infrastructure/imebridge/peer_darwin.h"
#include <dlfcn.h>
#include <signal.h>
#include <sys/select.h>

static void sendFrame(int fd, NSDictionary *frame) {
    NSError *error=nil;
    NSMutableData *bytes=[[NSJSONSerialization dataWithJSONObject:frame options:0 error:&error] mutableCopy];
    require(bytes!=nil && error==nil,@"encode native frame");
    [bytes appendBytes:"\n" length:1];
    size_t sent=0;
    while(sent<bytes.length) {
        ssize_t count=write(fd,(const char*)bytes.bytes+sent,bytes.length-sent);
        require(count>0,@"write complete authenticated frame");
        sent+=(size_t)count;
    }
}

static NSDictionary *probe(ArionTyping *typing, TypingEditor *editor, NSString *selection) {
    type(typing,@"xcgf");
    NSArray *candidates=typing.snapshot.candidates.copy;
    NSString *commit=@"";
    if(selection.length) {
        NSUInteger index=[candidates indexOfObject:selection];
        require(index!=NSNotFound,@"published term occurs in actual native candidates");
        require([typing select:index revision:typing.revision],@"select actual published candidate");
        commit=editor.view.string.copy;
        require(!typing.composing && !editor.view.hasMarkedText,@"selected full term leaves no draft");
    }
    [typing cancel];
    editor.view.string=@"";
    return @{@"input":@"xcgf",@"candidates":candidates,@"committed":commit};
}

int main(int argc,const char *argv[]) {
    @autoreleasepool {
        if(argc!=6) {fprintf(stderr,"usage: vocabulary-peer DATA USER SOCKET APP_IDENTIFIER REPORT\n");return 2;}
        signal(SIGPIPE,SIG_IGN);
        [NSApplication sharedApplication];
        NSMutableDictionary *report=[@{@"passed":@NO,@"boundary":@"signed native handler and actual AppKit/Rime fixture; no installed IMK dispatch; synthetic encrypted key"} mutableCopy];
        NSMutableArray *exchanges=[NSMutableArray array];report[@"exchanges"]=exchanges;
        ArionKeyboardEngine *engine=ArionKeyboardEngine.shared;
        ArionTyping *typing=nil;
        SecCodeRef peer=NULL;SecRequirementRef requirement=NULL;int fd=-1;int status=1;
        @try {
            Dl_info image{};require(dladdr((const void*)rime_get_api,&image)&&image.dli_fname,@"observe actual Rime image");
            report[@"nativeLibraryPath"]=[@(image.dli_fname) stringByResolvingSymlinksInPath];
            NSString *user=@(argv[2]);NSMutableData *key=[NSMutableData dataWithLength:32];memset(key.mutableBytes,0x71,key.length);
            NSError *error=nil;
            require(ArionRegisterTestEncryptedLearning([user stringByAppendingPathComponent:@"encrypted"],key,&error),@"register isolated encrypted provider");
            require([engine startWithData:@(argv[1]) userDirectory:user error:&error],@"start actual pinned engine");
            TypingEditor *editor=[TypingEditor new];typing=[[ArionTyping alloc] initWithEngine:engine];typing.contextRanking=NO;typing.emojiSuggestions=NO;
            require([typing activate:editor error:&error],@"activate native handler");if(typing.ascii)[typing toggleASCII];
            NSString *activation=@"signed-native-fixture";
            NSDictionary *initial=[typing applyVocabularyMessage:@{@"op":@"vocabulary_query",@"activation":activation,@"request":@1} activation:activation blocked:NO];
            require([initial[@"status"] isEqual:@"ok"],@"read native restored revision");
            report[@"initialRevision"]=initial[@"vocabularyRevision"];
            report[@"initialProbe"]=probe(typing,editor,nil);
            fd=socket(AF_UNIX,SOCK_STREAM,0);require(fd>=0,@"create owned socket");
            sockaddr_un address{};address.sun_family=AF_UNIX;
            require(strlen(argv[3])<sizeof(address.sun_path),@"bound socket path");strcpy(address.sun_path,argv[3]);
            require(connect(fd,(sockaddr*)&address,sizeof(address))==0,@"connect to signed application fixture");
            requirement=arion_peer_publisher_requirement(argv[4]);uint32_t pid=0,uid=0;
            peer=arion_socket_peer(fd,&pid,&uid);require(arion_peer_matches(peer,requirement),@"authenticate actual application publisher");
            report[@"authenticatedParentPID"]=@(pid);
            sendFrame(fd,@{@"op":@"hello",@"build":@"signed-native-vocabulary-fixture",@"protocol":@1,@"capabilities":@[@"keyboard-vocabulary-v1"]});
            sendFrame(fd,@{@"op":@"activate",@"activation":activation});
            NSMutableData *pending=[NSMutableData data];BOOL done=NO;
            while(!done) {
                fd_set ready;FD_ZERO(&ready);FD_SET(fd,&ready);FD_SET(STDIN_FILENO,&ready);timeval timeout{20,0};
                require(select(fd+1,&ready,NULL,NULL,&timeout)>0,@"receive owned fixture traffic before timeout");
                if(FD_ISSET(STDIN_FILENO,&ready)){char stop;require(read(STDIN_FILENO,&stop,1)==0,@"fixture shutdown uses stdin EOF");done=YES;continue;}
                char buffer[8192];ssize_t count=read(fd,buffer,sizeof(buffer));require(count>0,@"application connection stays live until fixture shutdown");
                [pending appendBytes:buffer length:(NSUInteger)count];require(pending.length<1024*1024,@"bound fixture frame");
                while(YES) {
                    const char *newline=(const char*)memchr(pending.bytes,'\n',pending.length);if(!newline)break;
                    NSUInteger length=(NSUInteger)(newline-(const char*)pending.bytes);
                    NSData *line=[pending subdataWithRange:NSMakeRange(0,length)];[pending replaceBytesInRange:NSMakeRange(0,length+1) withBytes:NULL length:0];
                    require(arion_peer_matches(peer,requirement),@"retain live publisher proof before every native operation");
                    NSDictionary *message=[NSJSONSerialization JSONObjectWithData:line options:0 error:&error];require([message isKindOfClass:NSDictionary.class],@"parse native request");
                    NSDictionary *reply=[typing applyVocabularyMessage:message activation:activation blocked:NO];require(reply!=nil,@"production handler recognizes current request");
                    NSMutableDictionary *exchange=[@{@"request":message,@"reply":reply} mutableCopy];[exchanges addObject:exchange];
                    sendFrame(fd,reply);
                    if([message[@"op"] isEqual:@"vocabulary_publish"] && [reply[@"status"] isEqual:@"ok"]) {
                        NSArray *terms=message[@"terms"];NSString *selection=terms.count?terms[0][@"text"]:nil;
                        exchange[@"probe"]=probe(typing,editor,selection);
                    }
                }
            }
            report[@"passed"]=@YES;status=0;
        } @catch(NSException *exception) {report[@"error"]=exception.reason?:exception.name;}
        [typing deactivate];[engine stop];if(peer)CFRelease(peer);if(requirement)CFRelease(requirement);if(fd>=0)close(fd);
        NSError *error=nil;NSData *json=[NSJSONSerialization dataWithJSONObject:report options:NSJSONWritingPrettyPrinted error:&error];
        if(!json||![json writeToFile:@(argv[5]) options:NSDataWritingAtomic error:&error])return 2;
        return status;
    }
}
