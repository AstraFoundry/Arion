#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>
#import "../typing.h"
#import "../candidate_panel.h"
#import "../learning.h"
#include <rime/dict/db.h>
#include <rime_api.h>
#include "../../shared/input.h"
#include "../../shared/emoji.h"
#include "../../shared/english_admission_cases.h"
#include "../../shared/personal_vocabulary_bridge.h"
#include "../../shared/personal_vocabulary_store.h"
#include "../../shared/learning_rime.h"
#include <rime/dict/prism.h>
#include <map>
#include <memory>
#include <dlfcn.h>

#include "typing_fixture.h"

static NSMutableArray *trace;
static NSDictionary *capture(ArionTyping *typing, TypingEditor *editor) {
    return @{@"text": editor.view.string.copy, @"marked": @(editor.view.hasMarkedText),
        @"insertions": @(editor.insertions), @"preedit": typing.snapshot.preedit ?: @"",
        @"candidates": typing.snapshot.candidates ?: @[], @"page": @(typing.snapshot.page),
        @"revision": @(typing.revision), @"caret": @(typing.snapshot.selection.location), @"ascii": @(typing.ascii),
        @"expanded": @(typing.snapshot.expanded), @"rows": typing.snapshot.rows ?: @[],
        @"currentRow": @(typing.snapshot.currentRow), @"highlighted": @(typing.snapshot.highlighted),
        @"available": @(typing.available), @"recoveryText": typing.recoveryText ?: @""};
}

static void record(NSString *step, ArionTyping *typing, TypingEditor *editor) {
    [trace addObject:@{@"step": step, @"state": capture(typing, editor)}];
}

static void savePanel(ArionCandidatePanel *panel, NSString *report, NSString *name) {
    [NSRunLoop.currentRunLoop runUntilDate:[NSDate dateWithTimeIntervalSinceNow:0.4]];
    NSWindow *window = [panel valueForKey:@"window"];
    NSView *view = window.contentView;
    NSBitmapImageRep *bitmap = [view bitmapImageRepForCachingDisplayInRect:view.bounds];
    require(window.visible && bitmap != nil, @"native candidate panel is visible and drawable");
    [view cacheDisplayInRect:view.bounds toBitmapImageRep:bitmap];
    NSData *png = [bitmap representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
    require([png writeToFile:[report.stringByDeletingLastPathComponent stringByAppendingPathComponent:name]
        options:NSDataWritingAtomic error:nil], @"save actual AppKit candidate rendering");
}

static void saveEditor(TypingEditor *editor, NSString *report, NSString *name) {
    [editor.view layoutSubtreeIfNeeded];
    NSBitmapImageRep *bitmap = [editor.view bitmapImageRepForCachingDisplayInRect:editor.view.bounds];
    require(bitmap != nil, @"native editor is drawable");
    [editor.view cacheDisplayInRect:editor.view.bounds toBitmapImageRep:bitmap];
    NSData *png = [bitmap representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
    require([png writeToFile:[report.stringByDeletingLastPathComponent stringByAppendingPathComponent:name]
        options:NSDataWritingAtomic error:nil], @"save actual AppKit editor rendering");
}

// Resolve the current page each time; never reuse an index after navigation.
static void selectText(ArionTyping *typing, NSString *text) {
    NSUInteger index=[typing.snapshot.candidates indexOfObject:text];
    for (NSUInteger page=0;index==NSNotFound && page<20 && !typing.snapshot.lastPage;page++) {
        require(press(typing,@"",121),@"advance actual native candidate page");
        index=[typing.snapshot.candidates indexOfObject:text];
    }
    require(index!=NSNotFound && [typing select:index revision:typing.revision],
            [@"select actual candidate " stringByAppendingString:text]);
}

static std::map<std::string, std::string> learningRecords(NSString *file, NSData *keyBytes) {
    std::unique_ptr<rime::Db> database(ArionLearningTestDatabase(file, @"arion_pinyin", keyBytes));
    require(database != nullptr && database->OpenReadOnly(), @"read actual encrypted engine dictionary with synthetic key");
    auto cursor = database->QueryAll();
    require(cursor != nullptr, @"enumerate persisted dictionary records");
    std::map<std::string, std::string> records;
    std::string code, value;
    while (cursor->GetNextRecord(&code, &value)) records.emplace(code, value);
    require(database->Close(), @"close read-only dictionary before real engine restart");
    return records;
}

int main(int argc, const char *argv[]) {
    @autoreleasepool {
        if (argc != 7) { fprintf(stderr, "usage: typing-test DATA USER REPORT DEVELOPMENT VALIDATION HOLDOUT\n"); return 2; }
        Dl_info loadedLibrary{};
        if (!dladdr((const void*)rime_get_api,&loadedLibrary) || !loadedLibrary.dli_fname) return 2;
        NSString *nativeLibraryPath=[@(loadedLibrary.dli_fname) stringByResolvingSymlinksInPath];
        [NSApplication sharedApplication];
        NSString *data = @(argv[1]), *user = @(argv[2]), *report = @(argv[3]);
        NSMutableData *testKey = [NSMutableData dataWithLength:32];
        memset(testKey.mutableBytes, 0x71, testKey.length);
        NSError *error = nil;
        ArionKeyboardEngine *engine = ArionKeyboardEngine.shared;
        NSMutableArray *results = [NSMutableArray array];
        NSMutableDictionary *integrity = [@{@"case": @"corrupt immutable data rejects startup before storage", @"acceptance": @"KBD-A01 KBD-A08"} mutableCopy];
        @try {
            NSString *corruptRoot = [user.stringByDeletingLastPathComponent stringByAppendingPathComponent:@"corrupt-runtime"];
            NSString *corruptData = [corruptRoot stringByAppendingPathComponent:@"data"];
            NSString *invalidUser = [user.stringByDeletingLastPathComponent stringByAppendingPathComponent:@"invalid-user"];
            require([NSFileManager.defaultManager createDirectoryAtPath:corruptRoot withIntermediateDirectories:YES attributes:nil error:&error], @"create isolated artifact fixture");
            require([NSFileManager.defaultManager copyItemAtPath:data toPath:corruptData error:&error], @"copy immutable schema closure");
            require([NSFileManager.defaultManager copyItemAtPath:[data.stringByDeletingLastPathComponent stringByAppendingPathComponent:@"runtime.json"]
                toPath:[corruptRoot stringByAppendingPathComponent:@"runtime.json"] error:&error], @"copy manifest without changing hashes");
            NSString *modified = [corruptData stringByAppendingPathComponent:@"build/default.yaml"];
            NSMutableData *bytes = [[NSData dataWithContentsOfFile:modified] mutableCopy];
            require(bytes != nil, @"read real prebuilt artifact");
            [bytes appendData:[@"\n# isolated integrity corruption\n" dataUsingEncoding:NSUTF8StringEncoding]];
            require([bytes writeToFile:modified options:NSDataWritingAtomic error:&error], @"change actual artifact bytes");
            NSError *integrityError = nil;
            require(![engine startWithData:corruptData userDirectory:invalidUser error:&integrityError], @"production startup rejects mismatched manifest hash");
            require([integrityError.domain isEqual:@"ArionKeyboard"] && integrityError.code == 2, @"corruption produces explicit data-unavailable error");
            require(![NSFileManager.defaultManager fileExistsAtPath:invalidUser] && [engine createSession] == 0,
                    @"invalid data cannot create user storage or a conversion session");
            integrity[@"passed"] = @YES;
        } @catch (NSException *exception) {
            integrity[@"passed"] = @NO; integrity[@"failure"] = exception.reason ?: exception.name;
            [engine stop];
        }
        [results addObject:integrity];
        NSLog(@"%@: %@", [integrity[@"passed"] boolValue] ? @"PASS" : @"FAIL", integrity[@"case"]);
        error = nil;
        if (!ArionRegisterTestEncryptedLearning([user stringByAppendingPathComponent:@"encrypted"], testKey, &error)) {
            NSLog(@"Encrypted learning registration failed: %@", error); return 2;
        }
        if (![engine startWithData:data userDirectory:user error:&error]) {
            NSLog(@"Real engine startup failed: %@", error); return 2;
        }
        __block ArionTyping *typing;
        __block TypingEditor *editor;
        void (^run)(NSString *, NSString *, void (^)(void)) = ^(NSString *name, NSString *acceptance, void (^body)(void)) {
            typing = [[ArionTyping alloc] initWithEngine:engine];
            editor = [TypingEditor new];
            trace = [NSMutableArray array];
            NSMutableDictionary *result = [@{@"case": name, @"acceptance": acceptance} mutableCopy];
            @try {
                NSError *activationError = nil;
                require([typing activate:editor error:&activationError], @"create actual engine session");
                if (typing.ascii) [typing toggleASCII];
                body();
                result[@"passed"] = @YES;
            } @catch (NSException *exception) {
                result[@"passed"] = @NO;
                result[@"failure"] = exception.reason ?: exception.name;
                NSLog(@"Assertion failed in %@: %@", name, result[@"failure"]);
            }
            result[@"trace"] = trace;
            result[@"observed"] = capture(typing, editor);
            [typing deactivate];
            [results addObject:result];
            NSLog(@"%@: %@", [result[@"passed"] boolValue] ? @"PASS" : @"FAIL", name);
        };
        run(@"local neural promotion preserves commit and rejects stale replies", @"RNK-A02 RNK-A03 RNK-A07", ^{
            require(!typing.neuralRanking, @"neural ranking defaults off");
            __block NSDictionary *request=nil;
            __block NSUInteger epoch=0, frozenEpoch=0;
            typing.rankingChanged=^(NSDictionary *snapshot,NSUInteger revision) {
                if (!snapshot) frozenEpoch=revision;
                else require(revision>frozenEpoch, @"request epoch must follow freeze epoch");
                request=snapshot; epoch=revision;
            };
            typing.neuralRanking=YES;
            type(typing,@"nihao"); press(typing,@" ",49);
            type(typing,@"xian");
            require(request!=nil && [request[@"candidates"] count]>=2, @"native snapshot contains actual comparable candidates");
            NSArray *original=request[@"candidates"];
            NSMutableArray *order=[original mutableCopy];
            NSDictionary *selected=order[1]; [order removeObjectAtIndex:1]; [order insertObject:selected atIndex:0];
            NSDictionary *reply=@{@"activation":@"test",@"revision":@(epoch),@"status":@"ok",
                @"rankResult":@{@"session_id":@"test",@"generation":@1,@"revision":@(epoch),@"candidates":order}};
            [typing applyRankingMessage:reply activation:@"test"];
            require([typing.snapshot.candidates[0] isEqual:selected[@"text"]], @"existing candidate is promoted");
            NSDictionary *settled=@{@"activation":@"test",@"revision":@(epoch),@"status":@"ok",
                @"rankResult":@{@"session_id":@"test",@"generation":@1,@"revision":@(epoch),@"candidates":original}};
            [typing applyRankingMessage:settled activation:@"test"];
            require([typing.snapshot.candidates[0] isEqual:original[0][@"text"]], @"settled pass uses original candidate identities");
            [typing applyRankingMessage:reply activation:@"test"];
            require([typing.snapshot.candidates[0] isEqual:selected[@"text"]], @"same revision remains eligible before navigation");

            press(typing,@" ",49);
            require([editor.view.string hasSuffix:selected[@"text"]] && editor.insertions==2, @"Space commits promoted engine candidate once");
            type(typing,@"xian");
            NSArray *before=typing.snapshot.candidates.copy;
            [typing applyRankingMessage:reply activation:@"test"];
            require([typing.snapshot.candidates isEqual:before], @"previous revision cannot reorder new composition");
            press(typing,@"\x1b",53);
            [typing applyRankingMessage:reply activation:@"test"];
            require(!typing.composing && editor.insertions==2, @"canceled reply cannot insert text");
        });
        run(@"generated whole-input candidate commits through the engine", @"RNK-A09", ^{
            __block NSDictionary *request=nil;
            __block NSUInteger epoch=0;
            typing.rankingChanged=^(NSDictionary *snapshot,NSUInteger revision) { request=snapshot;epoch=revision; };
            typing.neuralRanking=YES;
            typing.contextRanking=NO;
            type(typing,@"chonggyxia");
            require(request!=nil && [request[@"allowCompletion"] boolValue], @"single full-range candidate can request completion");
            NSDictionary *reply=@{@"activation":@"test",@"revision":@(epoch),@"status":@"ok",
                @"rankResult":@{@"session_id":@"test",@"generation":@1,@"revision":@(epoch),
                    @"completion":@{@"input":@"chonggyxia",@"text":@"重构一下",@"reading":@"chong gou yi xia"}}};
            ArionCandidatePanel *panel=[ArionCandidatePanel new];
            for (NSUInteger stage=0;stage<2;stage++) {
                if (stage) [typing applyRankingMessage:reply activation:@"test"];
                for (NSUInteger dark=0;dark<2;dark++) {
                    panel.appearance=dark?ArionCandidateAppearanceDark:ArionCandidateAppearanceLight;
                    [panel showCandidates:typing.snapshot.candidates highlighted:typing.snapshot.highlighted lastPage:typing.snapshot.lastPage
                        revision:typing.revision caretRect:NSMakeRect(80,300,1,20)];
                    savePanel(panel,report,[NSString stringWithFormat:@"completion-macos-%@-%@.png",stage?@"generated":@"original",dark?@"dark":@"light"]);
                }
            }
            [panel hide];
            require([typing.snapshot.candidates[0] isEqual:@"重构一下"] && editor.insertions==0,
                    @"generated candidate becomes provisional first choice without inserting");
            record(@"generated candidate",typing,editor);
            press(typing,@" ",49);
            require([editor.view.string isEqual:@"重构一下"] && editor.insertions==1, @"Space commits new engine candidate exactly once");
            type(typing,@"chonggyxia");
            NSArray *original=typing.snapshot.candidates.copy;
            [typing applyRankingMessage:reply activation:@"test"];
            require([original isEqual:typing.snapshot.candidates], @"old completion cannot affect new composition");
            press(typing,@"\x1b",53);
            [typing applyRankingMessage:reply activation:@"test"];
            require(!typing.composing && editor.insertions==1, @"cancelled completion cannot insert");
        });
        for (NSString *action in @[@"edit",@"navigate",@"deactivate",@"punctuation",@"original",@"cancel"]) {
            run([@"completion ownership " stringByAppendingString:action],@"RNK-A09",^{
                __block NSDictionary *request=nil;__block NSUInteger epoch=0;
                typing.neuralRanking=YES;typing.contextRanking=NO;
                typing.rankingChanged=^(NSDictionary *snapshot,NSUInteger revision){request=snapshot;epoch=revision;};
                type(typing,@"chonggyxia");
                NSString *original=typing.snapshot.candidates[0];
                require(request!=nil,@"completion request available");
                NSDictionary *reply=@{@"activation":@"test",@"revision":@(epoch),@"status":@"ok",
                    @"rankResult":@{@"session_id":@"test",@"generation":@1,@"revision":@(epoch),
                        @"completion":@{@"input":@"chonggyxia",@"text":@"重构一下",@"reading":@"chong gou yi xia"}}};
                if ([action isEqual:@"edit"]) press(typing,@"\x7f",51);
                if ([action isEqual:@"navigate"]) [typing highlight:1 revision:typing.revision];
                if ([action isEqual:@"deactivate"]) [typing deactivate];
                NSArray *before=typing.snapshot.candidates.copy;
                [typing applyRankingMessage:reply activation:@"test"];
                if ([action isEqual:@"edit"] || [action isEqual:@"navigate"] || [action isEqual:@"deactivate"]) {
                    require(editor.insertions==0 && (!before || [before isEqual:typing.snapshot.candidates]),@"ownership change rejects late completion");
                } else if ([action isEqual:@"punctuation"]) {
                    press(typing,@",",43);
                    require([editor.view.string isEqual:@"重构一下，"],@"punctuation commits the displayed generated engine candidate");
                } else if ([action isEqual:@"original"]) {
                    selectText(typing,original);
                    require([editor.view.string isEqual:original] && editor.insertions==1,@"original candidate retains its engine selection mapping");
                } else {
                    press(typing,@"\x1b",53);
                    require(editor.view.string.length==0 && editor.insertions==0,@"cancel removes provisional generated candidate without commit");
                }
            });
        }
        run(@"real full-Pinyin commit", @"KBD-A01", ^{
            type(typing, @"nihao");
            require(typing.composing && editor.view.hasMarkedText, @"engine preedit becomes AppKit marked text");
            require([typing.snapshot.candidates containsObject:@"你好"], @"real dictionary supplies 你好");
            press(typing, @" ", 49);
            require([editor.view.string isEqual:@"你好"] && editor.insertions == 1, @"Space inserts exact engine commit once");
            require(!typing.composing && !editor.view.hasMarkedText, @"commit clears provisional range");
        });
        run(@"English candidates in Chinese mode", @"KBD-A13", ^{
            typing.pageSize = 5;
            NSMutableString *expected = [NSMutableString string];
            for (NSString *word in @[@"artificial", @"hello", @"computer", @"software", @"can", @"an", @"you", @"hi", @"bailing", @"jingling"]) {
                type(typing, word);
                record(word, typing, editor);
                require([typing.snapshot.candidates containsObject:word], @"exact English is visible even on a five-candidate page");
                NSUInteger index = [typing.snapshot.candidates indexOfObject:word];
                require([typing select:index revision:typing.revision], @"select actual English engine candidate");
                [expected appendString:word];
                require([editor.view.string isEqual:expected] && !typing.ascii && !typing.composing,
                        @"English selection commits without changing language mode");
            }
        });
        run(@"English case selection, punctuation, edits and stale cancellation", @"KBD-A13", ^{
            type(typing, @"artificia");
            require([typing.snapshot.candidates containsObject:@"artificial"], @"incomplete English offers the full word");
            type(typing, @"l");
            require([typing.snapshot.candidates[0] isEqual:@"artificial"], @"full English word leads generated Chinese sentences");
            press(typing, @"\x7f", 51);
            require([typing.snapshot.candidates containsObject:@"artificial"], @"Backspace refreshes matching completions");
            type(typing, @"l"); press(typing, @" ", 49);
            require([editor.view.string isEqual:@"artificial"] && editor.insertions == 1, @"Space commits the English engine candidate once");
            type(typing, @"nihao"); press(typing, @" ", 49);
            type(typing, @"hello");
            NSUInteger title = [typing.snapshot.candidates indexOfObject:@"Hello"];
            while (title==NSNotFound && !typing.snapshot.lastPage) {
                require(press(typing,@"",121),@"page to deferred exact case alternative");
                title=[typing.snapshot.candidates indexOfObject:@"Hello"];
            }
            require(title < typing.pageSize, @"sentence-initial case variant remains selectable");
            press(typing, [NSString stringWithFormat:@"%lu", (unsigned long)title+1], 0);
            press(typing, @",", 43);
            require([editor.view.string isEqual:@"artificial你好Hello，"] && editor.insertions == 4,
                    @"number-selected English mixes with Chinese and Chinese punctuation");
            type(typing, @"software");
            NSUInteger stale = typing.revision;
            press(typing, @"\x1b", 53);
            require(![typing select:0 revision:stale] && [editor.view.string isEqual:@"artificial你好Hello，"] && editor.insertions == 4,
                    @"canceled English cannot commit through an old callback");
        });
        run(@"English prefix completion uses bounded actual engine candidates", @"KBD-A13", ^{
            typing.pageSize = 5;
            for (NSArray<NSString *> *sample in @[@[@"artifi", @"artificial"], @[@"compu", @"computer"],
                                                  @[@"softw", @"software"], @[@"hel", @"hello"]]) {
                type(typing, sample[0]);
                record(sample[0], typing, editor);
                NSUInteger index = [typing.snapshot.candidates indexOfObject:sample[1]];
                for (NSUInteger page = 0; index == NSNotFound && page < 8; ++page) {
                    press(typing, @"=", 24);
                    index = [typing.snapshot.candidates indexOfObject:sample[1]];
                }
                require(index != NSNotFound, @"prefix offers the intended word through actual candidate paging");
                require([typing select:index revision:typing.revision], @"select actual completion by displayed index");
                require([editor.view.string hasSuffix:sample[1]] && !typing.composing && !typing.ascii,
                        @"completion consumes only typed input and keeps Chinese mode");
            }
            type(typing, @"artifi");
            press(typing, @"\x7f", 51);
            require([typing.snapshot.candidates containsObject:@"artificial"], @"shorter prefix refreshes completion");
            type(typing, @"zz");
            require(![typing.snapshot.candidates containsObject:@"artificial"], @"diverged prefix removes the old word");
            [typing cancel];
            for (NSString *code in @[@"shi", @"nihao"]) {
                type(typing, code);
                require([typing.snapshot.candidates[0] characterAtIndex:0] > 127, @"whole-input Chinese stays first");
                [typing cancel];
            }
        });
        run(@"Expanded English vocabulary, canonical case and lower-ranked completions", @"KBD-A13", ^{
            typing.pageSize = 5;
            for (NSArray<NSString *> *sample in @[@[@"github", @"GitHub"], @[@"javascript", @"JavaScript"],
                    @[@"api", @"API"], @[@"python", @"Python"], @[@"frontend", @"frontend"], @[@"backend", @"backend"], @[@"workflow", @"workflow"]]) {
                type(typing, sample[0]);
                selectText(typing,sample[1]);
                require([editor.view.string hasSuffix:sample[1]], @"canonical source spelling survives commit");
            }
            type(typing, @"artifi");
            NSMutableSet<NSString *> *words = [NSMutableSet set];
            for (NSUInteger page = 0; page < 10; ++page) {
                for (NSString *candidate in typing.snapshot.candidates)
                    if ([candidate.lowercaseString hasPrefix:@"artifi"]) [words addObject:candidate.lowercaseString];
                press(typing, @"=", 24);
            }
            require(words.count > 3 && [words containsObject:@"artifice"], @"lower-ranked words survive beyond the old three-word cutoff");
            [typing cancel];
        });
        for (int fixture=4; fixture<argc; ++fixture) {
            const auto samples=EnglishAdmissionCases(std::filesystem::u8path(argv[fixture]));
            NSString *split=[@(argv[fixture]).lastPathComponent stringByDeletingPathExtension];
            run([NSString stringWithFormat:@"English admission %@ on native menus",split], @"KBD-A13", ^{
                for (NSNumber *size in @[@5,@9]) {
                    NSUInteger pageSize=size.unsignedIntegerValue;
                    typing.pageSize=pageSize;
                    for (const auto &sample:samples) {
                        NSString *code=@(sample.input.c_str());
                        type(typing,code);
                        NSString *step=[NSString stringWithFormat:@"%s page %lu",sample.id.c_str(),(unsigned long)pageSize];
                        record(step,typing,editor);
                        NSUInteger englishCount=0;
                        for (NSString *candidate in typing.snapshot.candidates)
                            englishCount+=EnglishAdmissionWord(candidate.UTF8String);
                        require(englishCount<=sample.maximumEnglish,[step stringByAppendingString:@" unexpected English suggestions"]);
                        if (sample.chineseFirst) require(typing.snapshot.candidates.count &&
                            [typing.snapshot.candidates[0] characterAtIndex:0]>127,[step stringByAppendingString:@" Chinese priority"]);
                        if (sample.expected!="-") {
                            NSString *expected=@(sample.expected.c_str());
                            NSUInteger index=[typing.snapshot.candidates indexOfObject:expected];
                            require(index!=NSNotFound,[step stringByAppendingString:@" expected English on first page"]);
                            NSUInteger insertions=editor.insertions;
                            require([typing select:index revision:typing.revision] && editor.insertions==insertions+1 &&
                                [editor.view.string hasSuffix:expected],[step stringByAppendingString:@" actual commit once"]);
                        } else [typing cancel];
                    }
                }
            });
        }
        run(@"English commits do not enter Chinese word learning", @"KBD-A07 KBD-A13", ^{
            [typing deactivate]; [engine stop];
            NSString *vault = [user stringByAppendingPathComponent:@"encrypted"];
            const auto records = learningRecords([vault stringByAppendingPathComponent:@"arion_pinyin.arion-userdb"], testKey);
            for (const auto &[code, value] : records) {
                require(code.find("artificial") == std::string::npos && value.find("artificial") == std::string::npos &&
                        code.find("Hello") == std::string::npos && value.find("Hello") == std::string::npos,
                        @"English table phrases stay out of the Chinese encrypted dictionary");
            }
            for (NSString *member in [NSFileManager.defaultManager enumeratorAtPath:user])
                require([member rangeOfString:@"arion_english"].location == NSNotFound, @"English selection creates no mutable or plaintext dictionary");
            NSError *restartError = nil;
            require(ArionRegisterTestEncryptedLearning(vault, testKey, &restartError) &&
                    [engine startWithData:data userDirectory:user error:&restartError], @"restart real engine after English learning inspection");
        });
        run(@"mixed admission protects Chinese through both native page sizes", @"KBD-A13", ^{
            for (NSNumber *size in @[@5,@9]) {
                typing.pageSize=size.unsignedIntegerValue;
                for (NSString *code in @[@"hel",@"hell",@"compu",@"softw",@"women",@"change",@"dj",@"gdp",@"danci",@"womenqugongyuan"]) {
                    type(typing,code);record(code,typing,editor);
                    require([typing.snapshot.candidates.firstObject characterAtIndex:0]>127,
                            @"short exact English, completion, abbreviation and whole Pinyin retain Chinese first");
                    NSUInteger english=0;
                    for (NSString *candidate in typing.snapshot.candidates) english+=EnglishAdmissionWord(candidate.UTF8String);
                    require(english<=1,@"English variants and completions cannot crowd either first page");
                    [typing cancel];
                }
            }
        });
        run(@"English candidate panel rendering", @"KBD-A13", ^{
            ArionCandidatePanel *panel = [ArionCandidatePanel new];
            @try {
                for (NSNumber *size in @[@5,@9]) {
                    typing.pageSize=size.unsignedIntegerValue;
                    for (NSString *word in @[@"artificial", @"can", @"artifi", @"hel", @"github", @"frontend", @"xin", @"shi"]) {
                        type(typing, word);
                        record([NSString stringWithFormat:@"%@ page %@",word,size], typing, editor);
                        [panel showCandidates:typing.snapshot.candidates highlighted:typing.snapshot.highlighted lastPage:typing.snapshot.lastPage
                            revision:typing.revision caretRect:NSMakeRect(80, 300, 1, 20)];
                        savePanel(panel, report, [NSString stringWithFormat:@"english-macos-%@-%@.png", word,size]);
                        [typing cancel];
                    }
                }
            } @finally { [panel hide]; }
        });
        __block NSArray<NSString *> *emojiOnTextOrder = nil;
        run(@"Emoji candidate native probe", @"KBD-A10", ^{
            type(typing, @"xiaolian");
            record(@"xiaolian candidates", typing, editor);
            require([typing.snapshot.candidates containsObject:@"😊"], @"native Rime filter offers system-rendered Unicode Emoji");
            NSMutableArray<NSString *> *textOrder = typing.snapshot.candidates.mutableCopy;
            [textOrder removeObject:@"😊"];
            emojiOnTextOrder = textOrder.copy;
            NSUInteger index = [typing.snapshot.candidates indexOfObject:@"😊"];
            require(index >= 3 && index <= 4, @"first three Chinese candidates remain protected");
            require([typing select:index revision:typing.revision], @"select actual Emoji engine candidate");
            require([editor.view.string isEqual:@"😊"] && editor.insertions == 1, @"Emoji commits exactly once through engine");
            [typing deactivate]; [engine stop];
            NSString *vault = [user stringByAppendingPathComponent:@"encrypted"];
            NSString *file = [vault stringByAppendingPathComponent:@"arion_pinyin.arion-userdb"];
            const auto records = learningRecords(file, testKey);
            for (const auto &[code, value] : records) {
                (void)value;
                require(code.find("笑脸") == std::string::npos && code.find("😊") == std::string::npos,
                        @"Emoji selection does not enter Rime Chinese learning records");
            }
            NSError *restartError = nil;
            require(ArionRegisterTestEncryptedLearning(vault, testKey, &restartError) &&
                    [engine startWithData:data userDirectory:user error:&restartError], @"restart engine after learning inspection");
        });
        run(@"Emoji preference off preserves Chinese candidates", @"KBD-A10", ^{
            [typing deactivate];
            typing.emojiSuggestions = NO;
            require([typing activate:editor error:nil], @"activate with Emoji suggestions off");
            type(typing, @"xiaolian");
            require(![typing.snapshot.candidates containsObject:@"😊"], @"disabled preference hides Emoji alternatives");
            require([typing.snapshot.candidates containsObject:@"笑脸"], @"Chinese candidate remains available");
            require(typing.snapshot.candidates.count >= emojiOnTextOrder.count &&
                    [[typing.snapshot.candidates subarrayWithRange:NSMakeRange(0, emojiOnTextOrder.count)] isEqual:emojiOnTextOrder],
                    @"Emoji insertion preserves relative text order");
        });
        run(@"Emoji keyboard selection keeps engine ownership", @"KBD-A10", ^{
            type(typing, @"xiaolian");
            NSUInteger index = [typing.snapshot.candidates indexOfObject:@"😊"];
            require(index != NSNotFound && index < 9, @"Emoji is on the first candidate page");
            require(press(typing, [NSString stringWithFormat:@"%lu", (unsigned long)index+1], 0), @"number key selects Emoji");
            require([editor.view.string isEqual:@"😊"] && editor.insertions == 1, @"number key commits one engine Emoji");
            type(typing, @"xiaolian");
            index = [typing.snapshot.candidates indexOfObject:@"😊"];
            require([typing highlight:index revision:typing.revision], @"highlight typed Emoji candidate");
            require(press(typing, @" ", 49), @"Space selects highlighted Emoji");
            require([editor.view.string isEqual:@"😊😊"] && editor.insertions == 2, @"Space commits the same engine Emoji");
        });
        run(@"Emoji punctuation commit stays out of context learning", @"KBD-A10", ^{
            type(typing, @"xiaolian");
            NSUInteger index = [typing.snapshot.candidates indexOfObject:@"😊"];
            require(index != NSNotFound && [typing highlight:index revision:typing.revision], @"highlight Emoji before punctuation");
            require(press(typing, @",", 43), @"punctuation selects the engine highlight");
            require([editor.view.string isEqual:@"😊，"], @"punctuation preserves Emoji candidate and Chinese comma");
        });
        run(@"Emoji eligibility follows the composing caret", @"KBD-A10", ^{
            type(typing, @"xiaolian");
            require([typing.snapshot.candidates containsObject:@"😊"], @"exact complete word offers Emoji");
            require(press(typing, @"", 115), @"Home moves the composing caret");
            record(@"Emoji after Home", typing, editor);
            require(![typing.snapshot.candidates containsObject:@"😊"], @"mid-input caret suppresses Emoji candidates");
            require(press(typing, @"", 119), @"End returns to the suffix end");
            record(@"Emoji after End", typing, editor);
            require([typing.snapshot.candidates containsObject:@"😊"], @"Emoji returns when the caret reaches the suffix end");
            [typing cancel];
            type(typing, @"gongsi");
            require(![typing.snapshot.candidates containsObject:@"😊"], @"unrelated phrase cannot reuse the prior Emoji candidate");
        });
        run(@"reviewed and CLDR Emoji triggers reach actual Pinyin candidates", @"KBD-A10", ^{
            const struct { const char *pinyin; const char *emoji; const char *source; } cases[] = {
                {"yinggelan", "\U0001f3f4\U000e0067\U000e0062\U000e0065\U000e006e\U000e0067\U000e007f", "英格兰"},
                {"xiaolian","😊","笑脸"}, {"kaixin","😄","开心"}, {"daxiao","😂","大笑"},
                {"ku","😭","哭"}, {"aixin","❤️","爱心"}, {"xihuan","❤️","喜欢"},
                {"guzhang","👏","鼓掌"}, {"xiexie","🙏","谢谢"},
                {"jiayou","💪","加油"}, {"shengri","🎂","生日"}, {"dangao","🎂","蛋糕"},
                {"liwu","🎁","礼物"}, {"kafei","☕","咖啡"}, {"taiyang","☀️","太阳"},
                {"yueliang","🌙","月亮"}, {"xingxing","⭐","星星"}, {"xiayu","🌧️","下雨"},
                {"xuehua","❄️","雪花"}, {"hua","🌸","花"}, {"mao","🐱","猫"},
                {"gou","🐶","狗"}, {"pingguo","🍎","苹果"}, {"huo","🔥","火"},
                {"yinyue","🎵","音乐"}, {"diannao","💻","电脑"},
                {"xiongmao","🐼","熊猫"}, {"shouji","📱","手机"},
                {"xigua","🍉","西瓜"}, {"feiji","✈️","飞机"},
                {"laoshi","🧑‍🏫","老师"}, {"zhongguo","🇨🇳","中国"},
                {"nihao","👋","你好"}, {"zaijian","👋","再见"},
                {"paizhao","📸","拍照"}, {"baobao","👶","宝宝"},
            };
            NSMutableArray<NSString *> *unreachable = [NSMutableArray array];
            for (const auto &item : cases) {
                uintptr_t session = [engine createSession];
                require(session != 0, @"create independent reachability session");
                [engine setEmojiSuggestions:YES session:session];
                for (const char *letter=item.pinyin; *letter; letter++)
                    require([engine processKey:*letter modifiers:0 session:session], @"compose reviewed trigger");
                char **texts = nullptr; size_t count = 0;
                require(arion_pinyin_candidates(engine.api, session, 0, 81, &texts, &count), @"read bounded engine window");
                BOOL found = NO;
                for (size_t index=0; index<count; index++) if (strcmp(texts[index],item.emoji)==0) {
                    ArionCandidateMetadata metadata;
                    found = arion_candidate_metadata(session,index,&metadata) && metadata.emoji &&
                        strcmp(metadata.source,item.source)==0 && metadata.end==strlen(item.pinyin);
                    break;
                }
                arion_pinyin_candidates_dispose(texts,count);
                [engine destroySession:session];
                if (!found) [unreachable addObject:@(item.pinyin)];
            }
            require(unreachable.count == 0, [@"unreachable reviewed triggers: " stringByAppendingString:[unreachable componentsJoinedByString:@", "]]);
        });
        run(@"shifted English letters retain case through native editor commit", @"KBD-A15", ^{
            require([typing handleEvent:key(@"H",4,NSEventModifierFlagShift)], @"shifted Latin letter enters composition");
            type(typing,@"ello");
            NSUInteger index=[typing.snapshot.candidates indexOfObject:@"Hello"];
            require(index!=NSNotFound, @"native menu contains cased English candidate");
            require([typing select:index revision:typing.revision], @"select English through native menu identity");
            require([editor.view.string isEqual:@"Hello"] && editor.insertions==1, @"native editor receives exact cased commit once");
        });
        run(@"English dictionary and mixed partial conversions keep engine ownership", @"KBD-A15", ^{
            const struct { const char *input; const char *first; const char *second; const char *expected; } cases[] = {
                {"hello", "hello", "", "hello"},
                {"Hello", "Hello", "", "Hello"},
                {"HELLO", "HELLO", "", "HELLO"},
                {"hElLo", "hElLo", "", "hElLo"},
                {"hell", "hello", "", "hello"},
                {"nihaohello", "你好", "hello", "你好hello"},
                {"hellonihao", "hello", "你好", "hello你好"},
                {"nihaoHELLO", "你好", "HELLO", "你好HELLO"},
                {"HELLOnihao", "HELLO", "你好", "HELLO你好"},
            };
            for (const auto &item : cases) {
                uintptr_t session=[engine createSession];
                require(session!=0, @"create independent bilingual session");
                for (const char *letter=item.input; *letter; ++letter)
                    require([engine processKey:*letter modifiers:0 session:session], @"compose bilingual input");
                for (const char *expected : {item.first,item.second}) {
                    if (!*expected) continue;
                    char **texts=nullptr; size_t count=0, chosen=SIZE_MAX;
                    require(arion_pinyin_candidates(engine.api,session,0,189,&texts,&count), @"enumerate actual bilingual candidates");
                    for (size_t i=0;i<count;++i) if (strcmp(texts[i],expected)==0) { chosen=i; break; }
                    arion_pinyin_candidates_dispose(texts,count);
                    require(chosen!=SIZE_MAX, [NSString stringWithFormat:@"missing bilingual candidate %s for %s",expected,item.input]);
                    require(arion_pinyin_select(engine.api,session,chosen,0), @"select bilingual engine candidate");
                }
                ArionPinyinSnapshot snapshot{};
                require(arion_pinyin_snapshot(engine.api,session,&snapshot)==0, @"read bilingual engine commit");
                require(strcmp(snapshot.commit,item.expected)==0 && !snapshot.preedit[0], @"mixed partial selections commit exactly the chosen text");
                arion_pinyin_snapshot_dispose(&snapshot);
                [engine destroySession:session];
            }
        });
        for (BOOL spellingEnabled : {NO, YES}) {
            NSString *mode=spellingEnabled?@"all spelling options":@"spelling options off";
            run([@"English mid-buffer edit with " stringByAppendingString:mode], @"KBD-A04 KBD-A15", ^{
                typing.typoCorrection=spellingEnabled;
                typing.fuzzyPinyin=spellingEnabled?2047:0;
                type(typing,@"hElXLo");
                press(typing,@"",123);
                require(typing.snapshot.selection.location==5,@"Left moves one raw English character");
                press(typing,@"",123);
                require(typing.snapshot.selection.location==4,@"second Left reaches the inserted middle letter");
                press(typing,@"\x7f",51);
                record(@"delete inserted middle letter",typing,editor);
                press(typing,@"",119);
                NSUInteger index=[typing.snapshot.candidates indexOfObject:@"hElLo"];
                require(index!=NSNotFound,@"edited English candidate preserves explicit mixed case");
                require([typing select:index revision:typing.revision],@"select edited English identity");
                require([editor.view.string isEqual:@"hElLo"] && editor.insertions==1 && !typing.composing,
                        @"middle edit commits exact English once");
            });
            run([@"Mixed English prefix cancellation with " stringByAppendingString:mode], @"KBD-A03 KBD-A05 KBD-A15", ^{
                typing.typoCorrection=spellingEnabled;
                typing.fuzzyPinyin=spellingEnabled?2047:0;
                type(typing,@"HELLOnihao");
                selectText(typing,@"HELLO");
                record(@"English prefix remains provisional",typing,editor);
                require(typing.composing && editor.insertions==0,@"partial English choice does not prematurely commit");
                NSUInteger stale=typing.revision;
                press(typing,@"\x1b",53);
                require(!typing.composing && !editor.view.hasMarkedText && editor.view.string.length==0,
                        @"Escape discards selected English prefix and Chinese suffix");
                require(![typing select:0 revision:stale] && editor.insertions==0,@"late mixed candidate cannot insert");
                type(typing,@"Hello");
                stale=typing.revision;
                [typing toggleASCII];
                require(!typing.composing && editor.view.string.length==0 && ![typing select:0 revision:stale],
                        @"mode change discards English composition and stale callback");
            });
            for (BOOL englishFirst : {NO, YES}) {
                NSString *input=englishFirst?@"HELLOnihaX":@"nihaohElLX";
                NSString *prefix=englishFirst?@"HELLO":@"你好";
                NSString *suffix=englishFirst?@"你好":@"hElLo";
                NSString *expected=[prefix stringByAppendingString:suffix];
                run([NSString stringWithFormat:@"Selected %@ prefix suffix editing and focus loss with %@",prefix,mode],
                    @"KBD-A03 KBD-A04 KBD-A05 KBD-A15", ^{
                    typing.typoCorrection=spellingEnabled;
                    typing.fuzzyPinyin=spellingEnabled?2047:0;
                    type(typing,input);
                    selectText(typing,prefix);
                    record(@"selected prefix before suffix edit",typing,editor);
                    require(typing.composing && editor.insertions==0,@"prefix selection remains provisional");
                    NSUInteger stale=typing.revision;
                    require(press(typing,@"\x7f",51),@"roll back the previous segment selection");
                    record(@"Backspace reopens selected prefix without deleting raw input",typing,editor);
                    // Pinyin preedit inserts a syllable separator; it is not raw input.
                    NSString *reopened=englishFirst?input:@"ni haohElLX";
                    require([typing.snapshot.preedit isEqual:reopened] &&
                            typing.composing && editor.insertions==0,
                            @"first Backspace restores the complete marked spelling without committing");
                    require(press(typing,@"\x7f",51),@"delete wrong trailing suffix letter after rollback");
                    type(typing,@"o");
                    record(@"corrected suffix after selected prefix",typing,editor);
                    require(![typing select:0 revision:stale],@"suffix edit invalidates prior candidate callback");
                    selectText(typing,prefix);
                    selectText(typing,suffix);
                    require([editor.view.string isEqual:expected] && editor.insertions==1 &&
                            !typing.composing && !editor.view.hasMarkedText,@"mixed edited composition commits once with selected prefix");
                    type(typing,input);
                    selectText(typing,prefix);
                    stale=typing.revision;
                    [typing deactivate];
                    record(@"focus lost with selected prefix",typing,editor);
                    require([editor.view.string isEqual:expected] && editor.insertions==1 && !editor.view.hasMarkedText,
                            @"focus loss removes only pending mixed composition");
                    require(![typing select:0 revision:stale],@"old owner cannot commit selected prefix after focus loss");
                    require([typing activate:editor error:nil],@"reactivate same editor with fresh composition");
                    type(typing,@"Hello");selectText(typing,@"Hello");
                    require([editor.view.string isEqual:[expected stringByAppendingString:@"Hello"]] && editor.insertions==2,
                            @"new owner cannot resurrect the cancelled prefix or suffix");
                });
            }
            run([@"Unknown technical spelling raw commit with " stringByAppendingString:mode], @"KBD-A02 KBD-A15", ^{
                typing.typoCorrection=spellingEnabled;
                typing.fuzzyPinyin=spellingEnabled?2047:0;
                type(typing,@"ArionZqxBridge");
                record(@"unknown technical spelling before Return",typing,editor);
                require(press(typing,@"\r",36),@"Return commits unknown raw spelling");
                require([editor.view.string isEqual:@"ArionZqxBridge"] && editor.insertions==1 && !typing.composing,
                        @"raw Return preserves unknown technical case without completion or newline");
                require(press(typing,@".",47),@"punctuation follows raw English commit");
                require([editor.view.string isEqual:@"ArionZqxBridge。"] && editor.insertions==2,
                        @"following punctuation does not reconvert committed English");
            });
        }
        run(@"spelling preference changes revoke composition and survive reactivation", @"KBD-A14", ^{
            type(typing, @"nihao");
            NSUInteger revision=typing.revision;
            typing.typoCorrection=YES;
            require(!typing.composing && editor.insertions==0, @"changing spelling cancels unconfirmed text");
            require(![typing select:0 revision:revision], @"old candidate callback cannot survive a spelling change");
            type(typing, @"zhoongguo");
            NSUInteger index=[typing.snapshot.candidates indexOfObject:@"中国"];
            require(index!=NSNotFound, @"typing applies its correction preference to the active session");
            [typing deactivate];
            require([typing activate:editor error:nil], @"reactivate with retained spelling preferences");
            type(typing, @"zhoongguo");
            require([typing.snapshot.candidates containsObject:@"中国"], @"new session retains correction preference");
            typing.fuzzyPinyin=1u;
            require(!typing.composing, @"changing a fuzzy pair cancels unconfirmed text");
            typing.typoCorrection=NO;
            type(typing, @"zongguo");
            require([typing.snapshot.candidates containsObject:@"中国"], @"fuzzy option works independently of typo correction");
        });
        run(@"bounded typo correction produces real complete candidates", @"KBD-A14", ^{
            const struct { const char *input; const char *expected; const char *option; bool exact; } cases[] = {
                {"zhngguo", "中国", "arion_typo", false},
                {"zhoongguo", "中国", "arion_typo", false},
                {"zhognguo", "中国", "arion_typo", false},
                {"zhomgguo", "中国", "arion_typo", false},
                {"zongguo", "中国", "arion_fuzzy_z_zh", false},
                {"ban", "帮", "arion_fuzzy_an_ang", false},
                {"bang", "班", "arion_fuzzy_an_ang", false},
                {"ben", "崩", "arion_fuzzy_en_eng", false},
                {"beng", "本", "arion_fuzzy_en_eng", false},
                {"bin", "冰", "arion_fuzzy_in_ing", false},
                {"bing", "宾", "arion_fuzzy_in_ing", false},
                {"lian", "两", "arion_fuzzy_ian_iang", false},
                {"liang", "连", "arion_fuzzy_ian_iang", false},
                {"guan", "光", "arion_fuzzy_uan_uang", false},
                {"guang", "关", "arion_fuzzy_uan_uang", false},
                {"zhongguo", "中国", "arion_typo", true},
                {"nihao", "你好", "arion_typo", true},
                {"nj", "那就", "arion_typo", true},
                {"womenzhoumqugongyuan", "我们周末去公园", "arion_typo", false},
                {"qingbangwodkaidiandeng", "请帮我打开电灯", "arion_typo", false},
            };
            for (const auto &item : cases) {
                uintptr_t session = [engine createSession];
                require(session != 0, @"create independent spelling session");
                NSString *originalFirst=nil;
                if (item.exact) {
                    for (const char *letter=item.input; *letter; ++letter) [engine processKey:*letter modifiers:0 session:session];
                    originalFirst=[[engine candidates:session from:0 limit:1] firstObject];
                    [engine clear:session];
                }
                engine.api->set_option(session,item.option,True);
                for (const char *letter=item.input; *letter; ++letter)
                    require([engine processKey:*letter modifiers:0 session:session], @"compose spelling fixture");
                char **texts = nullptr; size_t count = 0; size_t chosen = SIZE_MAX;
                require(arion_pinyin_candidates(engine.api, session, 0, 189, &texts, &count), @"enumerate corrected candidates");
                for (size_t index=0; index<count; ++index) if (strcmp(texts[index],item.expected)==0) {
                    ArionCandidateMetadata metadata{};
                    require(arion_candidate_metadata(session,index,&metadata), @"corrected candidate has raw range");
                    if (metadata.start==0 && metadata.end==strlen(item.input)) { chosen=index; break; }
                }
                arion_pinyin_candidates_dispose(texts,count);
                require(chosen!=SIZE_MAX, [NSString stringWithFormat:@"full corrected candidate missing: %s",item.input]);
                if (item.exact) require([[[engine candidates:session from:0 limit:1] firstObject] isEqual:originalFirst], @"typo expansion retains the original bilingual first choice");
                require(arion_pinyin_select(engine.api,session,chosen,0), @"select actual corrected engine candidate");
                ArionPinyinSnapshot snapshot{};
                require(arion_pinyin_snapshot(engine.api,session,&snapshot)==0, @"read engine commit");
                require(strcmp(snapshot.commit,item.expected)==0 && !snapshot.preedit[0], @"corrected selection commits exactly once without leftover input");
                arion_pinyin_snapshot_dispose(&snapshot);
                [engine destroySession:session];
            }
        });
        run(@"case families are unique and technical English prefixes retain real spans", @"KBD-A13 KBD-A15", ^{
            type(typing,@"API");
            NSUInteger count=0;
            for (NSString *candidate in typing.snapshot.candidates) if ([candidate isEqual:@"API"]) ++count;
            require(count==1,@"explicit all-capital input does not repeat transformed source forms");
            selectText(typing,@"API");
            type(typing,@"GitHubnihao");
            selectText(typing,@"GitHub");
            require(typing.composing && editor.insertions==1,@"selected technical English prefix stays provisional");
            selectText(typing,@"你好");
            require([editor.view.string isEqual:@"APIGitHub你好"] && editor.insertions==2 && !typing.composing,
                    @"technical English prefix and Chinese suffix commit once through the engine");
        });
        run(@"English prefixes retain typo and independent fuzzy suffix conversion", @"KBD-A14 KBD-A15", ^{
            typing.typoCorrection=YES;
            type(typing,@"hellozhngguo");
            selectText(typing,@"hello");
            require(typing.composing && editor.insertions==0,@"English prefix does not prematurely commit a typo suffix");
            selectText(typing,@"中国");
            require([editor.view.string isEqual:@"hello中国"] && editor.insertions==1,@"typo suffix commits its complete raw coverage");
            typing.typoCorrection=NO;typing.fuzzyPinyin=1u;
            type(typing,@"HELLOzongguo");
            selectText(typing,@"HELLO");selectText(typing,@"中国");
            require([editor.view.string isEqual:@"hello中国HELLO中国"] && editor.insertions==2 && !typing.composing,
                    @"fuzzy suffix works with typo disabled and preserves English case");
        });
        run(@"English prefixes retain approved personal suffix candidates", @"KBD-11 KBD-A15", ^{
            uintptr_t session=[engine createSession];
            ArionPersonalTerm term{"mixed-personal","星槎工坊","xing cha gong fang"};
            require(session && arion_personal_vocabulary_replace(session,1,&term,1),@"bind approved personal term to isolated engine session");
            for (const char *letter="helloxcgf";*letter;++letter) [engine processKey:*letter modifiers:0 session:session];
            NSArray *candidates=[engine candidates:session from:0 limit:189];
            NSUInteger index=[candidates indexOfObject:@"hello"];
            ArionCandidateMetadata metadata{};
            require(index!=NSNotFound && arion_candidate_metadata(session,index,&metadata) && metadata.start==0 && metadata.end==5,
                    @"English prefix consumes only its exact raw bytes");
            require([engine selectCandidate:index session:session],@"select actual English prefix");
            ArionKeyboardSnapshot *partial=[engine snapshot:session error:nil];
            require(partial.commit.length==0 && partial.preedit.length>0,@"personal suffix remains provisional after English choice");
            candidates=[engine candidates:session from:0 limit:189];index=[candidates indexOfObject:@"星槎工坊"];
            require(index!=NSNotFound && arion_candidate_metadata(session,index,&metadata) && metadata.start==5 && metadata.end==9,
                    @"personal suffix retains its original consumed range after English");
            require([engine selectCandidate:index session:session],@"select actual personal suffix");
            ArionKeyboardSnapshot *final=[engine snapshot:session error:nil];
            require([final.commit isEqual:@"hello星槎工坊"] && final.preedit.length==0,@"mixed personal selection commits exact engine text once");
            [engine destroySession:session];
        });
        run(@"personal projection persists encrypted and rejects stale updates", @"MEM-08", ^{
            auto provider=arion::ime::registeredLearningProvider();
            rime::Prism prism{rime::path([[data stringByAppendingPathComponent:@"build/pinyin_simp.prism.bin"] UTF8String])};
            require(provider && prism.Load(),@"load actual cache provider and pronunciation prism");
            std::vector<arion::ime::PersonalTerm> terms{{"evision","星槎工坊","xing cha gong fang"}};
            {
                arion::ime::PersonalVocabularyStore cache(provider);
                require(cache.open() && cache.save(prism,1,terms),@"persist approved personal projection");
                require(!cache.save(prism,1,{}),@"reject stale disk projection");
            }
            NSString* file=[NSString stringWithUTF8String:arion::ime::learningPath(provider->directory(),"arion_personal.arion-userdb").c_str()];
            NSData* sealed=[NSData dataWithContentsOfFile:file];
            require(sealed.length>32,@"personal projection ciphertext exists");
            for (NSString* canary in @[@"星槎工坊",@"xing cha gong fang",@"evision"]) {
                require([sealed rangeOfData:[canary dataUsingEncoding:NSUTF8StringEncoding] options:0 range:NSMakeRange(0,sealed.length)].location==NSNotFound,@"private projection fields absent from ciphertext");
            }
            {
                arion::ime::PersonalVocabularyStore cache(provider);
                uint64_t revision=0;std::vector<arion::ime::PersonalTerm> restored;
                require(cache.open() && cache.load(prism,revision,restored),@"reopen encrypted projection offline");
                require(revision==1 && restored.size()==1 && restored[0].id=="evision" && restored[0].text==terms[0].text && restored[0].reading==terms[0].reading,@"restore exact fields without metadata collisions");
                require(!cache.save(prism,2,{{"invalid","星槎","zzzzzz"}}),@"invalid pronunciation cannot replace disk projection");
                require(cache.load(prism,revision,restored) && revision==1 && restored.size()==1,@"failed update preserves previous snapshot");
                require(cache.save(prism,2,{}),@"persist deletion as newer empty snapshot");
            }
            {
                arion::ime::PersonalVocabularyStore cache(provider);
                uint64_t revision=0;std::vector<arion::ime::PersonalTerm> restored;
                require(cache.open() && cache.load(prism,revision,restored) && revision==2 && restored.empty(),@"deleted projection stays empty after restart");
            }
            {
                arion::ime::PersonalVocabularyStore first(provider),second(provider);
                require(first.open(),@"open cache writer");
                require(!provider->cooperative() && !second.open(),@"macOS provider excludes simultaneous cache writers");
                require(first.save(prism,3,terms),@"publish newer full projection");
            }
            {
                arion::ime::PersonalVocabularyStore next(provider);
                require(next.open() && !next.save(prism,3,{}),@"new writer cannot overwrite the same revision");
                uint64_t revision=0;std::vector<arion::ime::PersonalTerm> restored;
                require(next.load(prism,revision,restored) && revision==3 && restored.size()==1,@"new writer sees current complete projection");
                require(next.save(prism,4,{}),@"new writer publishes newer deletion");
            }
            NSData* intact=[NSData dataWithContentsOfFile:file];
            NSMutableData* corrupt=[intact mutableCopy];
            ((unsigned char*)corrupt.mutableBytes)[corrupt.length-1]^=0x01;
            require([corrupt writeToFile:file atomically:YES],@"tamper isolated synthetic cache");
            {
                arion::ime::PersonalVocabularyStore cache(provider);
                require(!cache.open() && cache.error().code!=0,@"tampered projection fails authentication");
                require([[NSData dataWithContentsOfFile:file] isEqual:corrupt],@"authentication failure does not rewrite original file");
            }
            require([intact writeToFile:file atomically:YES],@"restore synthetic cache fixture for remaining tests");
            require([[NSFileManager defaultManager] removeItemAtPath:file error:nil],@"remove isolated cache fixture after its lifecycle checks");
        });
        run(@"personal terms use actual engine commits and revision boundaries", @"MEM-08", ^{
            uintptr_t session=[engine createSession];
            require(session!=0,@"create personal vocabulary session");
            ArionPersonalTerm term{"personal-1","星槎工坊","xing cha gong fang"};
            require(arion_personal_vocabulary_replace(session,1,&term,1),@"accept approved pronunciation projection");
            require(!arion_personal_vocabulary_replace(session,1,nullptr,0),@"reject stale projection");
            uintptr_t isolated=[engine createSession];
            for (const char* letter="xingchagongfang";*letter;++letter) [engine processKey:*letter modifiers:0 session:isolated];
            require([[engine candidates:isolated from:0 limit:189] indexOfObject:@"星槎工坊"]==NSNotFound,@"projection is scoped to its owning session");
            [engine destroySession:isolated];
            for (const char* input:{"xingchagongfang","xcgf","xingchagongfangnihao"}) {
                for (const char* letter=input;*letter;++letter) require([engine processKey:*letter modifiers:0 session:session],@"type personal pronunciation");
                require(!arion_personal_vocabulary_replace(session,2,nullptr,0),@"do not replace active composition");
                NSArray<NSString*>* candidates=[engine candidates:session from:0 limit:189];
                NSUInteger index=[candidates indexOfObject:@"星槎工坊"];
                require(index!=NSNotFound,@"personal term is a real candidate");
                ArionCandidateMetadata metadata{};
                require(arion_candidate_metadata(session,index,&metadata),@"read personal range");
                require(strcmp(metadata.type,"arion_personal")==0 && metadata.start==0,@"preserve personal provenance");
                const bool partial=strstr(input,"nihao")!=nullptr;
                require(metadata.end==(partial?strlen("xingchagongfang"):strlen(input)),@"personal raw input range");
                require(arion_pinyin_select(engine.api,session,index,0),@"select personal candidate through engine");
                ArionPinyinSnapshot snapshot{};
                require(arion_pinyin_snapshot(engine.api,session,&snapshot)==0,@"snapshot personal selection");
                if (partial) {
                    require(snapshot.preedit[0]!=0 && snapshot.commit[0]==0,@"prefix selection keeps suffix provisional");
                    arion_pinyin_snapshot_dispose(&snapshot);
                    NSArray<NSString*>* suffix=[engine candidates:session from:0 limit:189];
                    NSUInteger next=[suffix indexOfObject:@"你好"];
                    require(next!=NSNotFound && arion_pinyin_select(engine.api,session,next,0),@"select remaining suffix");
                    require(arion_pinyin_snapshot(engine.api,session,&snapshot)==0,@"snapshot final personal sentence");
                }
                require(strcmp(snapshot.commit,partial?"星槎工坊你好":"星槎工坊")==0 && !snapshot.preedit[0],@"engine commits personal text exactly once");
                arion_pinyin_snapshot_dispose(&snapshot);
            }
            require(arion_personal_vocabulary_replace(session,2,nullptr,0),@"remove projection between compositions");
            for (const char* letter="xingchagongfang";*letter;++letter) [engine processKey:*letter modifiers:0 session:session];
            NSArray<NSString*>* removed=[engine candidates:session from:0 limit:189];
            require([removed indexOfObject:@"星槎工坊"]==NSNotFound,@"personal selection does not leak into ordinary learning");
            [engine clear:session];
            ArionPersonalTerm multiple[]={{"one","星槎工坊","xing cha gong fang"},{"two","星槎工坊","xing cha gong fang"},{"three","星茶工坊","xing cha gong fang"}};
            require(arion_personal_vocabulary_replace(session,3,multiple,3),@"accept distinct source records for matching spellings");
            for (const char* letter="xingchagongfang";*letter;++letter) [engine processKey:*letter modifiers:0 session:session];
            NSArray<NSString*>* merged=[engine candidates:session from:0 limit:189];
            require([merged indexOfObject:@"星槎工坊"]==1 && [merged indexOfObject:@"星茶工坊"]==2,@"personal alternatives retain deterministic order after ordinary first");
            NSUInteger occurrences=0;
            for (NSString* text in merged) if ([text isEqual:@"星槎工坊"]) ++occurrences;
            require(occurrences==1,@"same text and input range appear once across source records");
            [engine destroySession:session];
            require(!arion_personal_vocabulary_replace(session,4,nullptr,0),@"destroyed session cannot receive a projection");
        });
        run(@"new keyboard sessions restore personal terms without app connection", @"KBD-11", ^{
            uintptr_t first=[engine createSession];
            ArionPersonalTerm term{"offline-one","星槎工坊","xing cha gong fang"};
            require(first && arion_personal_vocabulary_publish(first,1,&term,1),@"persist before applying personal projection");
            [engine destroySession:first];
            uintptr_t restored=[engine createSession];
            require(restored!=0,@"create independent session with cached projection");
            uint64_t revision=0;
            require(arion_personal_vocabulary_restore(restored,&revision) && revision==1,@"report persisted revision for future connection handshake");
            for (const char* letter="xcgf";*letter;++letter) [engine processKey:*letter modifiers:0 session:restored];
            require([[engine candidates:restored from:0 limit:189] indexOfObject:@"星槎工坊"]!=NSNotFound,@"offline session offers cached personal candidate");
            require(!arion_personal_vocabulary_publish(restored,2,nullptr,0),@"busy publish cannot alter persistent projection");
            [engine clear:restored];
            require(arion_personal_vocabulary_restore(restored,&revision) && revision==1,@"busy publish left disk revision unchanged");
            require(arion_personal_vocabulary_publish(restored,2,nullptr,0),@"publish complete deletion after cancellation");
            [engine destroySession:restored];
            uintptr_t empty=[engine createSession];
            require(empty && arion_personal_vocabulary_restore(empty,&revision) && revision==2,@"new session restores deletion revision");
            for (const char* letter="xcgf";*letter;++letter) [engine processKey:*letter modifiers:0 session:empty];
            require([[engine candidates:empty from:0 limit:189] indexOfObject:@"星槎工坊"]==NSNotFound,@"cached deletion does not resurrect personal term");
            [engine destroySession:empty];
            auto provider=arion::ime::registeredLearningProvider();
            NSString* cache=[NSString stringWithUTF8String:arion::ime::learningPath(provider->directory(),"arion_personal.arion-userdb").c_str()];
            require([[NSFileManager defaultManager] removeItemAtPath:cache error:nil],@"remove synthetic offline-cache fixture");
        });
        run(@"native vocabulary messages persist only valid idle snapshots", @"KBD-11", ^{
            NSDictionary *query=@{@"op":@"vocabulary_query",@"activation":@"editor",@"request":@1};
            NSDictionary *term=@{@"id":@"wire-one",@"text":@"星槎工坊",@"reading":@"xing cha gong fang"};
            NSMutableDictionary *publish=[@{@"op":@"vocabulary_publish",@"activation":@"editor",@"request":@1,
                @"vocabularyRevision":@1,@"terms":@[term]} mutableCopy];
            NSDictionary *(^apply)(NSDictionary *)=^(NSDictionary *message) {
                return [typing applyVocabularyMessage:message activation:@"editor" blocked:NO];
            };
            require([apply(query)[@"vocabularyRevision"] isEqual:@0],@"empty cache reports zero revision");
            for (id malformed in @[@YES,@(-1),@1.5,@"1",NSNull.null]) {
                NSMutableDictionary *bad=[publish mutableCopy]; bad[@"request"]=malformed;
                require(apply(bad)==nil,@"reject non-integer request identity");
            }
            NSMutableDictionary *stale=[publish mutableCopy]; stale[@"activation"]=@"old-editor";
            require(apply(stale)==nil,@"reject stale activation");
            require([[typing applyVocabularyMessage:publish activation:@"editor" blocked:YES][@"status"] isEqual:@"busy"],@"voice ownership blocks publication");
            for (id malformed in @[NSNull.null,@{},@[@"term"],@[@{@"id":@"wire-one",@"text":@"星槎工坊",@"reading":@"zzzzzz"}]]) {
                NSMutableDictionary *bad=[publish mutableCopy]; bad[@"terms"]=malformed;
                require([apply(bad)[@"status"] isEqual:@"rejected"],@"invalid snapshot rejected without partial write");
            }
            unichar nulText[]={0x661f,0,0x69ce};
            NSMutableDictionary *nul=[term mutableCopy]; nul[@"text"]=[NSString stringWithCharacters:nulText length:3];
            NSMutableDictionary *bad=[publish mutableCopy]; bad[@"terms"]=@[nul];
            require([apply(bad)[@"status"] isEqual:@"rejected"],@"embedded null cannot truncate a canonical term");
            require([apply(query)[@"vocabularyRevision"] isEqual:@0],@"invalid messages leave persisted revision unchanged");
            require([apply(publish)[@"status"] isEqual:@"ok"],@"valid message persists and applies");
            require([apply(publish)[@"status"] isEqual:@"rejected"],@"stale publication cannot replace current snapshot");
            type(typing,@"xcgf");
            require([typing.snapshot.candidates containsObject:@"星槎工坊"],@"published personal term enters native menu");
            publish[@"vocabularyRevision"]=@2; publish[@"terms"]=@[];
            require([apply(publish)[@"status"] isEqual:@"busy"],@"active composition rejects deletion");
            [typing cancel];
            require([apply(query)[@"vocabularyRevision"] isEqual:@1],@"busy update left disk unchanged");
            require([apply(publish)[@"status"] isEqual:@"ok"],@"explicit empty snapshot deletes shared terms");
            [typing deactivate]; require([typing activate:editor error:nil],@"new session restores persisted deletion");
            require([apply(query)[@"vocabularyRevision"] isEqual:@2],@"native query reports reopened revision");
            type(typing,@"xcgf");
            require(![typing.snapshot.candidates containsObject:@"星槎工坊"],@"deleted term absent after restoration");
            [typing cancel];
            auto provider=arion::ime::registeredLearningProvider();
            NSString *cache=[NSString stringWithUTF8String:arion::ime::learningPath(provider->directory(),"arion_personal.arion-userdb").c_str()];
            require([[NSFileManager defaultManager] removeItemAtPath:cache error:nil],@"remove isolated wire fixture");
        });
        run(@"real candidate ranges distinguish full conversion and prefixes", @"RNK-A01", ^{
            const struct { const char *input; const char *text; size_t start; size_t end; } cases[] = {
                {"nihao", "你好", 0, 5}, {"nihao", "你", 0, 2},
                {"xian", "现", 0, 4}, {"xian", "西安", 0, 4},
                {"ni'hao", "你好", 0, 6}, {"nh", "你好", 0, 2},
            };
            for (const auto &item : cases) {
                uintptr_t session = [engine createSession];
                require(session != 0, @"create independent range session");
                for (const char *letter=item.input; *letter; ++letter)
                    require([engine processKey:*letter modifiers:0 session:session], @"compose range fixture");
                char **texts = nullptr; size_t count = 0;
                require(arion_pinyin_candidates(engine.api, session, 0, 189, &texts, &count), @"enumerate real range fixtures");
                BOOL found = NO;
                for (size_t index=0; index<count; ++index) if (strcmp(texts[index],item.text)==0) {
                    ArionCandidateMetadata metadata{};
                    require(arion_candidate_metadata(session,index,&metadata), @"actual candidate has metadata");
                    require(metadata.start==item.start && metadata.end==item.end && metadata.type[0] && !metadata.emoji,
                            @"producer range and type match raw input, independently of text length");
                    found = YES;
                    break;
                }
                arion_pinyin_candidates_dispose(texts,count);
                [engine destroySession:session];
                require(found, [NSString stringWithFormat:@"real range fixture missing: %s / %s", item.input, item.text]);
            }
        });
        run(@"system Emoji candidate panel rendering evidence", @"KBD-A10", ^{
            type(typing, @"xiaolian");
            NSUInteger index = [typing.snapshot.candidates indexOfObject:@"😊"];
            require(index != NSNotFound, @"real engine offers Emoji for native drawing");
            ArionCandidatePanel *panel = [ArionCandidatePanel new];
            @try {
                for (NSUInteger dark=0; dark<2; dark++) for (NSUInteger size=0; size<3; size++) {
                    panel.appearance = dark ? ArionCandidateAppearanceDark : ArionCandidateAppearanceLight;
                    panel.size = (ArionCandidateSize)size;
                    [panel showCandidates:typing.snapshot.candidates highlighted:index lastPage:typing.snapshot.lastPage
                        revision:typing.revision caretRect:NSMakeRect(80, 300, 1, 20)];
                    NSString *name = [NSString stringWithFormat:@"emoji-macos-%@-%lu.png", dark ? @"dark" : @"light", (unsigned long)size];
                    savePanel(panel, report, name);
                }
            } @finally { [panel hide]; }
        });
        run(@"cancel preserves emoji neighbors", @"KBD-A05 KBD-A06", ^{
            editor.view.string = @"甲😀乙";
            [editor.view setSelectedRange:NSMakeRange(3, 0)];
            type(typing, @"nihao");
            require([editor.view.string hasPrefix:@"甲😀"] && [editor.view.string hasSuffix:@"乙"], @"preedit preserves UTF-16 neighbors");
            press(typing, @"\x1b", 53);
            require([editor.view.string isEqual:@"甲😀乙"] && editor.insertions == 0 && !editor.view.hasMarkedText,
                    @"Escape removes only owned composition");
        });
        run(@"commit preserves emoji neighbors", @"KBD-A06", ^{
            editor.view.string = @"甲😀乙";
            [editor.view setSelectedRange:NSMakeRange(3, 0)];
            type(typing, @"nihao"); press(typing, @" ", 49);
            require([editor.view.string isEqual:@"甲😀你好乙"], @"actual commit replaces only marked UTF-16 range");
        });
        run(@"Backspace reconverts real input", @"KBD-A04", ^{
            type(typing, @"nihao"); press(typing, @"\x7f", 51); type(typing, @"o"); press(typing, @" ", 49);
            require([editor.view.string isEqual:@"你好"], @"delete and reinsert restore conversion");
        });
        run(@"page and relative number selection", @"KBD-A03", ^{
            type(typing, @"shi");
            require(!typing.snapshot.lastPage, @"real dictionary has another candidate page");
            press(typing, @"", 121);
            require(typing.snapshot.page == 1 && typing.snapshot.candidates.count > 0, @"PageDown advances real candidate page");
            NSString *chosen = typing.snapshot.candidates[0];
            press(typing, @"1", 18);
            require([editor.view.string isEqual:chosen] && editor.insertions == 1, @"number selects current-page identity");
        });
        run(@"candidate bar chevron pages forward", @"KBD-A03", ^{
            type(typing, @"shi");
            NSUInteger stale = typing.revision;
            NSArray *first = typing.snapshot.candidates;
            require([typing nextPage:typing.revision], @"chevron advances current revision");
            require(typing.snapshot.page == 1 && ![typing.snapshot.candidates isEqual:first], @"chevron shows the next real page");
            require(![typing nextPage:stale] && typing.snapshot.page == 1, @"stale chevron click refused");
            require(editor.insertions == 0, @"paging never writes the host");
        });
        run(@"context ranker returns a permutation and never promotes a prefix", @"RNK-A01", ^{
            NSMutableArray *writes = [NSMutableArray array];
            ArionContextRanker *ranker = [[ArionContextRanker alloc] initWithRecords:@{} persist:^(NSDictionary *changes) { [writes addObject:changes]; }];
            NSArray *candidates = @[@"你好", @"拟好", @"你", @"妮好"];
            const unsigned char eligible[]={1,1,0,1};
            require([[ranker order:candidates eligible:eligible after:@"我们"] isEqual:@[@0, @1, @2, @3]], @"no learned pair keeps engine order");
            [ranker learn:@"妮好，" after:@"我们"];
            require(writes.count == 1 && [writes[0][@"c\t我们\t妮好"] hasPrefix:@"1 "], @"trailing punctuation is not part of the learned word");
            require([[ranker order:candidates eligible:eligible after:@"我们"] isEqual:@[@0, @3, @1, @2]], @"one confirmation lifts a word two places, not three");
            [ranker learn:@"妮好" after:@"我们"];
            [ranker learn:@"你" after:@"我们"]; [ranker learn:@"你" after:@"我们"]; [ranker learn:@"你" after:@"我们"];
            NSArray *order = [ranker order:candidates eligible:eligible after:@"我们"];
            require([[order sortedArrayUsingSelector:@selector(compare:)] isEqual:@[@0, @1, @2, @3]], @"order is a permutation");
            require([order[0] isEqual:@3] && ![order[0] isEqual:@2], @"same-length word promoted; shorter prefix stays behind");
            require([[ranker order:candidates eligible:eligible after:@"他们"] isEqual:@[@0, @1, @2, @3]], @"another context keeps engine order");
            require([[ranker order:candidates eligible:eligible after:nil] isEqual:@[@0, @1, @2, @3]], @"no context keeps engine order");
            require([[ArionContextRanker contextOf:@"今天天气很好。 "] isEqual:@"很好"] && [ArionContextRanker contextOf:@"，。"] == nil,
                    @"context is the last two characters before punctuation");
        });
        run(@"context ranking promotes the word chosen after the same previous word", @"RNK-A02 RNK-A04", ^{
            require(engine.ranker != nil, @"encrypted context store opens in the test vault");
            // Learn one homophone after 我们 and a different one after 他们, the
            // latter last, so the engine's own frequency learning prefers it.
            // Chooses word, or the skip-th same-length homophone after the first.
            NSString *(^chooseAfter)(NSString *, NSString *, NSUInteger, NSString *) = ^NSString *(NSString *context, NSString *pinyin, NSUInteger skip, NSString *word) {
                type(typing, context); press(typing, @" ", 49);
                type(typing, pinyin);
                NSUInteger full = typing.snapshot.candidates[0].length, index = NSNotFound, seen = 0;
                for (NSUInteger i = word ? 0 : 1; i < typing.snapshot.candidates.count; i++) {
                    NSString *candidate = typing.snapshot.candidates[i];
                    if (word ? [candidate isEqual:word] : candidate.length == full && seen++ == skip) { index = i; break; }
                }
                require(index != NSNotFound, @"engine offers a same-length homophone");
                NSString *chosen = typing.snapshot.candidates[index];
                require([typing select:index revision:typing.revision], @"choose the homophone");
                return chosen;
            };
            NSString *ours = chooseAfter(@"women", @"shiyan", 0, nil);
            chooseAfter(@"women", @"shiyan", 0, ours);
            NSString *theirs = chooseAfter(@"tamen", @"shiyan", 1, nil);
            require(![ours isEqual:theirs], @"two contexts learned different words");
            chooseAfter(@"tamen", @"shiyan", 0, theirs);
            chooseAfter(@"tamen", @"shiyan", 0, theirs);
            type(typing, @"women"); press(typing, @" ", 49);
            typing.contextRanking = NO;
            type(typing, @"shiyan");
            NSString *engineFirst = typing.snapshot.candidates[0];
            [typing cancel];
            typing.contextRanking = YES;
            type(typing, @"women"); press(typing, @" ", 49);
            type(typing, @"shiyan");
            record(@"ranked after 我们", typing, editor);
            require([typing.snapshot.candidates[0] isEqual:ours], @"word learned after 我们 leads after 我们");
            require(![engineFirst isEqual:ours], @"engine order alone would not have put it first");
            NSUInteger before = editor.insertions;
            press(typing, @",", 43);
            require([editor.view.string hasSuffix:[ours stringByAppendingString:@"，"]] && editor.insertions == before + 1,
                    @"punctuation commits Arion's highlighted candidate");
        });
        run(@"next page opens the expanded grid", @"KBD-A03", ^{
            type(typing, @"shi");
            NSArray *first = typing.snapshot.candidates;
            press(typing, @"=", 24);
            record(@"expanded", typing, editor);
            require(typing.snapshot.expanded && typing.snapshot.rows.count >= 2 && typing.snapshot.currentRow == 1 &&
                    [typing.snapshot.rows[0] isEqual:first] && [typing.snapshot.rows[1] isEqual:typing.snapshot.candidates],
                    @"= expands with the next page as the current row");
            press(typing, @"", 125);
            require(typing.snapshot.currentRow == 2 && typing.snapshot.page == 2, @"Down moves to the next row");
            press(typing, @"", 124);
            require(typing.snapshot.highlighted == 1, @"Right moves within the row");
            NSString *chosen = typing.snapshot.candidates[1];
            NSUInteger revision = typing.revision;
            press(typing, @"-", 27); press(typing, @"-", 27);
            require(typing.snapshot.expanded && typing.snapshot.currentRow == 0, @"- moves up to the first row");
            require(![typing select:1 row:2 revision:revision], @"stale grid click refused");
            require([typing select:1 row:2 revision:typing.revision] && [editor.view.string isEqual:chosen],
                    @"a click on another row chooses that candidate");
            type(typing, @"shi"); press(typing, @"=", 24);
            require(typing.snapshot.expanded, @"expanded again");
            type(typing, @"j");
            require(typing.snapshot.expanded && typing.snapshot.rows.count > 1 && typing.snapshot.currentRow == 0,
                    @"typing refreshes the grid from the first row without collapsing");
            press(typing, @"=", 24); press(typing, @"-", 27); press(typing, @"-", 27);
            require(!typing.snapshot.expanded && typing.composing, @"- above the first row collapses");
            press(typing, @"=", 24); press(typing, @"\x1b", 53);
            require(!typing.composing && editor.insertions == 1, @"Escape cancels from the grid too (KBD-03)");
            type(typing, @"shi");
            require([typing toggleExpanded:typing.revision] && typing.snapshot.expanded && typing.snapshot.currentRow == 0,
                    @"the chevron expands in place");
            require([typing toggleExpanded:typing.revision] && !typing.snapshot.expanded, @"the chevron collapses");
        });
        run(@"page preference preserves the displayed candidate mapping", @"KBD-A03", ^{
            typing.pageSize = 9;
            type(typing, @"shi");
            require(typing.snapshot.candidates.count == 9, @"initial menu has nine candidates");
            typing.pageSize = 5;
            press(typing, @"", 121);
            require(typing.snapshot.page == 1 && typing.snapshot.candidates.count == 9,
                    @"existing menu keeps its page width after preference change");
            NSString *chosen = typing.snapshot.candidates[8];
            press(typing, @"9", 25);
            require([editor.view.string isEqual:chosen], @"number selects the displayed ninth candidate");
            type(typing, @"shi");
            require(typing.snapshot.candidates.count == 5, @"new composition applies the five-candidate preference");
            typing.pageSize = 9;
        });
        run(@"short pages consume numbers past the page", @"KBD-A03", ^{
            typing.pageSize = 5;
            type(typing, @"shi");
            require(typing.snapshot.candidates.count == 5 && !typing.snapshot.lastPage, @"page holds five candidates");
            press(typing, @"7", 26);
            require(typing.composing && editor.insertions == 0, @"number past the page selects nothing");
            NSString *sixth = nil;
            press(typing, @"", 121);
            sixth = typing.snapshot.candidates[0];
            press(typing, @"1", 18);
            require([editor.view.string isEqual:sixth], @"page two starts at the sixth candidate");
            typing.pageSize = 9;
        });
        run(@"partial selection and suffix continuation", @"KBD-A03", ^{
            type(typing, @"nihao");
            NSUInteger index = [typing.snapshot.candidates indexOfObject:@"你"];
            for (NSUInteger page = 0; index == NSNotFound && page < 20 && !typing.snapshot.lastPage; page++) {
                press(typing, @"", 121); index = [typing.snapshot.candidates indexOfObject:@"你"];
            }
            require(index != NSNotFound, @"real candidates expose prefix 你");
            require([typing select:index revision:typing.revision], @"select prefix from current engine page");
            require(typing.composing && editor.insertions == 0, @"prefix remains provisional while suffix exists");
            press(typing, @" ", 49);
            require([editor.view.string isEqual:@"你好"] && editor.insertions == 1, @"suffix conversion commits whole confirmed phrase");
        });
        run(@"expanded grid survives partial selection and suffix edits", @"KBD-A03 KBD-A04", ^{
            ArionCandidatePanel *panel = [ArionCandidatePanel new];
            NSRect screen = NSScreen.mainScreen.visibleFrame;
            NSRect caret = NSMakeRect(NSMidX(screen), NSMidY(screen), 0, 20);
            __weak ArionTyping *owner = typing;
            typing.changed = ^{
                ArionTyping *current = owner;
                ArionKeyboardSnapshot *snapshot = current.snapshot;
                if (!current.composing || !snapshot.candidates.count) { [panel hide]; return; }
                [panel showRows:snapshot.rows currentRow:snapshot.currentRow highlighted:snapshot.highlighted
                      expanded:snapshot.expanded more:snapshot.moreRows revision:current.revision caretRect:caret];
            };
            @try {
                type(typing, @"nihaoma");
                press(typing, @"", 125);
                require(typing.snapshot.expanded, @"Down expands the three-syllable composition");
                NSUInteger prefixRow = NSNotFound, prefixIndex = NSNotFound;
                for (NSUInteger row = 0; row < typing.snapshot.rows.count; row++) {
                    NSUInteger index = [typing.snapshot.rows[row] indexOfObject:@"你好"];
                    if (index != NSNotFound) { prefixRow = row; prefixIndex = index; break; }
                }
                require(prefixRow != NSNotFound, @"real grid offers the two-character prefix");
                NSWindow *window = [panel valueForKey:@"window"];
                NSRect expandedFrame = window.frame;
                NSUInteger stale = typing.revision;
                record(@"expanded three syllables", typing, editor);
                require([typing select:prefixIndex row:prefixRow revision:stale], @"choose two characters from the grid");
                record(@"one syllable remains expanded", typing, editor);
                require(typing.composing && !editor.insertions && typing.snapshot.expanded &&
                        typing.snapshot.page == 0 && typing.snapshot.currentRow == 0 && typing.snapshot.highlighted == 0,
                        @"prefix stays provisional and remaining candidates restart in the expanded grid");
                require([[panel valueForKey:@"expanded"] boolValue] && ![panel valueForKey:@"reshow"] &&
                        NSHeight(window.frame) == NSHeight(expandedFrame) && NSMaxY(window.frame) == NSMaxY(expandedFrame),
                        @"partial selection keeps panel height and anchored edge without a collapse transition");
                require(![typing select:prefixIndex row:prefixRow revision:stale], @"old grid callback cannot select the suffix");
                type(typing, @"n");
                require(typing.snapshot.expanded, @"extending the suffix keeps the grid expanded");
                press(typing, @"\x7f", 51);
                require(typing.snapshot.expanded, @"editing the suffix keeps the grid expanded");
                NSUInteger suffix = [typing.snapshot.candidates indexOfObject:@"吗"];
                require(suffix != NSNotFound && [typing select:suffix revision:typing.revision], @"choose the remaining character");
                require(!typing.composing && [editor.view.string isEqual:@"你好吗"] && editor.insertions == 1,
                        @"only the fully confirmed phrase reaches the client");
                type(typing, @"shi");
                require(!typing.snapshot.expanded, @"the next composition starts collapsed after a commit");
                press(typing, @"", 125); press(typing, @"\x1b", 53);
                type(typing, @"shi");
                require(!typing.snapshot.expanded, @"cancellation clears expansion for the next composition");
                press(typing, @"", 125);
                require([typing toggleExpanded:typing.revision], @"explicitly collapse the grid");
                type(typing, @"j");
                require(!typing.snapshot.expanded, @"explicit collapse remains in effect during later edits");
            } @finally {
                typing.changed = nil;
                [panel hide];
            }
        });
        run(@"partial-selection Backspace rollback", @"KBD-A04", ^{
            type(typing, @"nihao");
            NSUInteger index = [typing.snapshot.candidates indexOfObject:@"你"];
            for (NSUInteger page = 0; index == NSNotFound && page < 20 && !typing.snapshot.lastPage; page++) {
                press(typing, @"", 121); index = [typing.snapshot.candidates indexOfObject:@"你"];
            }
            require(index != NSNotFound, @"prefix available for rollback");
            require([typing select:index revision:typing.revision], @"select real prefix");
            NSString *selected = typing.snapshot.preedit;
            record(@"prefix selected", typing, editor);
            press(typing, @"\x7f", 51);
            record(@"after Backspace", typing, editor);
            require(typing.composing && editor.insertions == 0 && ![typing.snapshot.preedit isEqual:selected], @"Backspace reopens selected segment");
            NSUInteger whole = [typing.snapshot.candidates indexOfObject:@"你好"];
            require(whole != NSNotFound && [typing select:whole revision:typing.revision], @"choose original full candidate after rollback");
            require([editor.view.string isEqual:@"你好"], @"reopened segment reconverts original Pinyin");
        });
        run(@"mid-buffer caret edit and Home End", @"KBD-A04", ^{
            type(typing, @"nihao");
            press(typing, @"", 115);
            require(typing.snapshot.selection.location == 0, @"Home moves engine caret to start");
            press(typing, @"", 119);
            require(typing.snapshot.selection.location == typing.snapshot.preedit.length, @"End moves engine caret to UTF-16 end");
            record(@"after End", typing, editor);
            press(typing, @"", 123); record(@"after Left", typing, editor);
            press(typing, @"\x7f", 51); record(@"after middle Backspace", typing, editor);
            type(typing, @"i"); record(@"after insert", typing, editor);
            require([[typing.snapshot.preedit stringByReplacingOccurrencesOfString:@" " withString:@""] isEqual:@"nihao"], @"edit restores raw Pinyin around syllable cursor");
            press(typing, @"", 119); press(typing, @" ", 49);
            require([editor.view.string isEqual:@"你好"], @"middle edit preserves untouched suffix");
        });
        run(@"forward Delete at Home preserves suffix", @"KBD-A04", ^{
            type(typing, @"nihao"); press(typing, @"", 115); press(typing, @"", 117);
            require([[typing.snapshot.preedit stringByReplacingOccurrencesOfString:@" " withString:@""] isEqual:@"ihao"], @"forward Delete removes character at engine caret");
            type(typing, @"n"); press(typing, @"", 119); press(typing, @" ", 49);
            require([editor.view.string isEqual:@"你好"], @"reinserted prefix preserves suffix conversion");
        });
        run(@"standalone left and right Shift switch on short release", @"KBD-A05", ^{
            __block NSUInteger switches = 0;
            typing.modeChanged = ^(ArionTypingSwitch change) { require(change == ArionTypingSwitchLanguage, @"Shift reports a language switch"); switches++; };
            type(typing, @"nihao");
            NSUInteger stale = typing.revision;
            require(!modifier(typing, 56, NSEventModifierFlagShift, 1) && !typing.ascii && typing.composing,
                    @"Shift press leaves mode and draft unchanged");
            require(modifier(typing, 56, 0, 1.1) && typing.ascii && !typing.composing && [editor.view.string isEqual:@"nihao"] &&
                    !editor.view.hasMarkedText && editor.insertions == 1,
                    @"short release commits raw letters once before switching");
            require(switches == 1 && ![typing select:0 revision:stale], @"one mode notification and no stale candidate write");
            require(!modifier(typing, 56, 0, 1.11) && switches == 1 && editor.insertions == 1, @"duplicate release cannot toggle or insert again");
            modifier(typing, 60, NSEventModifierFlagShift, 2);
            require(modifier(typing, 60, 0, 2.1) && !typing.ascii && switches == 2, @"right Shift switches back persistently");
        });
        run(@"Shift commits raw spelling instead of English completion", @"KBD-A05", ^{
            type(typing, @"arion");
            saveEditor(editor, report, @"shift-macos-pending.png");
            modifier(typing, 60, NSEventModifierFlagShift, 1);
            require(modifier(typing, 60, 0, 1.1) && [editor.view.string isEqual:@"arion"] && typing.ascii,
                    @"right Shift preserves the literal product name");
            saveEditor(editor, report, @"shift-macos-committed.png");
            require(!press(typing, @"x", 7) && editor.insertions == 1, @"English letters pass through after the raw commit");
            modifier(typing, 56, NSEventModifierFlagShift, 2);
            modifier(typing, 56, 0, 2.1);
            type(typing, @"hel");
            modifier(typing, 56, NSEventModifierFlagShift, 3);
            modifier(typing, 56, 0, 3.1);
            require([editor.view.string isEqual:@"arionhel"] && editor.insertions == 2, @"completion candidates never replace the raw prefix");
        });
        run(@"Shift preserves edited input and confirmed segments", @"KBD-A04 KBD-A05", ^{
            type(typing, @"arionx");
            press(typing, @"", 51);
            press(typing, @"", 115);
            modifier(typing, 56, NSEventModifierFlagShift, 1);
            modifier(typing, 56, 0, 1.1);
            require([editor.view.string isEqual:@"arion"] && !editor.view.hasMarkedText, @"raw commit includes the complete edited buffer at Home");
            [typing toggleASCII];
            type(typing, @"nihao");
            NSUInteger prefix = [typing.snapshot.candidates indexOfObject:@"你"];
            require(prefix != NSNotFound && [typing select:prefix revision:typing.revision] && typing.composing,
                    @"partial Chinese selection retains the suffix");
            modifier(typing, 60, NSEventModifierFlagShift, 2);
            modifier(typing, 60, 0, 2.1);
            require([editor.view.string isEqual:@"arion你hao"] && editor.insertions == 2,
                    @"confirmed Chinese survives while the pending suffix commits literally");
            [typing toggleASCII];
            type(typing, @"ni'hao");
            modifier(typing, 56, NSEventModifierFlagShift, 3);
            modifier(typing, 56, 0, 3.1);
            require([editor.view.string isEqual:@"arion你haoni'hao"] && editor.insertions == 3,
                    @"explicit apostrophes survive without adding preedit spaces");
        });
        run(@"Shift raw commit ignores Chinese full width", @"KBD-A05 KBD-A09", ^{
            @try {
                require([typing toggleFullWidth], @"enable Chinese full width");
                type(typing, @"arion");
                modifier(typing, 56, NSEventModifierFlagShift, 1);
                modifier(typing, 56, 0, 1.1);
                require([editor.view.string isEqual:@"arion"] && typing.ascii && typing.fullWidth,
                        @"raw letters stay half width without changing the saved Chinese preference");
                require(!press(typing, @"a", 0), @"English mode still passes keys through");
            } @finally {
                if (typing.ascii) [typing toggleASCII];
                if (typing.fullWidth) [typing toggleFullWidth];
            }
        });
        run(@"duplicated host modifier notifications preserve one Shift tap", @"KBD-A05", ^{
            __block NSUInteger switches = 0;
            typing.modeChanged = ^(ArionTypingSwitch change) { require(change == ArionTypingSwitchLanguage, @"Shift reports a language switch"); switches++; };
            for (NSNumber *code in @[@56, @60, @0, @255]) {
                BOOL before = typing.ascii;
                modifier(typing, code.unsignedShortValue, NSEventModifierFlagShift, 1);
                require(!modifier(typing, code.unsignedShortValue, NSEventModifierFlagShift, 1.014), @"duplicate press does not switch");
                require(modifier(typing, code.unsignedShortValue, 0, 1.176) && typing.ascii != before,
                        @"captured duplicate-press sequence switches once on release");
                require(!modifier(typing, code.unsignedShortValue, 0, 1.178), @"duplicate release does not switch twice");
            }
            require(switches == 4 && !typing.ascii, @"known and anonymous taps each notify exactly once");
            modifier(typing, 56, NSEventModifierFlagShift, 2);
            modifier(typing, 56, NSEventModifierFlagShift, 2.29);
            require(!modifier(typing, 56, 0, 2.31) && !typing.ascii, @"duplicate press never extends the short-press deadline");
            modifier(typing, 56, NSEventModifierFlagShift, 3);
            [typing handleEvent:key(@"A", 0, NSEventModifierFlagShift)];
            modifier(typing, 56, NSEventModifierFlagShift, 3.05);
            require(!modifier(typing, 56, 0, 3.1) && !typing.ascii, @"duplicate after typing cannot restore a cancelled tap");
            typing.modeChanged = nil;
        });
        run(@"anonymous IMK Shift events switch only on an unambiguous transition", @"KBD-A05", ^{
            for (NSNumber *code in @[@0, @255]) {
                modifier(typing, code.unsignedShortValue, NSEventModifierFlagShift, 1);
                require(modifier(typing, code.unsignedShortValue, 0, 1.1) && typing.ascii,
                        @"missing keycode is identified from the Shift transition");
                modifier(typing, 60, NSEventModifierFlagShift, 2);
                require(modifier(typing, code.unsignedShortValue, 0, 2.1) && !typing.ascii,
                        @"anonymous release pairs with known right Shift");
            }
            modifier(typing, 0, NSEventModifierFlagShift | NSEventModifierFlagCommand, 3);
            require(!modifier(typing, 0, 0, 3.1) && !typing.ascii, @"ambiguous modifier changes never switch");
            modifier(typing, 0, NSEventModifierFlagShift, 4);
            [typing handleEvent:key(@"A", 0, NSEventModifierFlagShift)];
            require(!modifier(typing, 0, 0, 4.1) && !typing.ascii, @"anonymous Shift chords never switch");
            modifier(typing, 255, NSEventModifierFlagShift, 5);
            require(!modifier(typing, 255, 0, 5.4) && !typing.ascii, @"anonymous long holds never switch");
            modifier(typing, 0, NSEventModifierFlagShift, 6);
            [typing cancel];
            require(!modifier(typing, 0, 0, 6.1) && !typing.ascii, @"cancel invalidates anonymous Shift");
            modifier(typing, 0, NSEventModifierFlagShift, 7);
            require(modifier(typing, 60, 0, 7.1) && typing.ascii, @"anonymous press pairs with known right release");
            [typing toggleASCII];
        });
        run(@"Shift chords long holds and lifecycle changes cannot switch", @"KBD-A05", ^{
            modifier(typing, 56, NSEventModifierFlagShift, 1);
            [typing handleEvent:key(@"A", 0, NSEventModifierFlagShift)];
            require(!modifier(typing, 56, 0, 1.1) && !typing.ascii, @"uppercase chord does not switch");
            [typing cancel];
            modifier(typing, 56, NSEventModifierFlagShift, 2);
            require(!modifier(typing, 56, 0, 2.4) && !typing.ascii, @"long hold does not switch");
            modifier(typing, 56, NSEventModifierFlagShift, 3);
            modifier(typing, 55, NSEventModifierFlagShift | NSEventModifierFlagCommand, 3.02);
            modifier(typing, 55, NSEventModifierFlagShift, 3.04);
            require(!modifier(typing, 56, 0, 3.1) && !typing.ascii, @"modifier chord does not switch");
            modifier(typing, 56, NSEventModifierFlagShift, 4);
            modifier(typing, 60, NSEventModifierFlagShift, 4.02);
            modifier(typing, 56, NSEventModifierFlagShift, 4.04);
            require(!modifier(typing, 60, 0, 4.1) && !typing.ascii, @"overlapping Shift keys do not switch");
            modifier(typing, 56, NSEventModifierFlagShift, 5);
            [typing cancel];
            require(!modifier(typing, 56, 0, 5.1), @"cancel invalidates pending Shift");
            modifier(typing, 56, NSEventModifierFlagShift, 6);
            [typing deactivate];
            require([typing activate:editor error:nil] && !modifier(typing, 56, 0, 6.1) && !typing.ascii,
                    @"activation cannot inherit pending Shift from another focus");
        });
        run(@"Shift Space release does not switch twice", @"KBD-A05", ^{
            modifier(typing, 56, NSEventModifierFlagShift, 1);
            require([typing handleEvent:key(@" ", 49, NSEventModifierFlagShift)] && typing.ascii, @"legacy chord switches");
            require(!modifier(typing, 56, 0, 1.1) && typing.ascii, @"chord release keeps the selected mode");
        });
        run(@"symbols commit directly without candidate composition", @"KBD-A02", ^{
            NSArray *inputs = @[@"/", @"\\", @"%", @"$", @"*", @"~", @"<", @">", @"[", @"]", @"{", @"}", @"|", @"^", @"_", @"(", @")", @"!", @"?", @";", @":", @"'", @"'", @"\"", @"\""];
            NSArray *outputs = @[@"、", @"、", @"%", @"￥", @"*", @"～", @"《", @"》", @"【", @"】", @"｛", @"｝", @"|", @"……", @"——", @"（", @"）", @"！", @"？", @"；", @"：", @"‘", @"’", @"“", @"”"];
            NSMutableString *expected = [NSMutableString string];
            for (NSUInteger i = 0; i < inputs.count; i++) {
                require(press(typing, inputs[i], 0), @"symbol handled by engine");
                [expected appendString:outputs[i]];
                require([editor.view.string isEqual:expected] && !typing.composing && !editor.view.hasMarkedText &&
                        typing.snapshot.candidates.count == 0, @"each symbol commits immediately without a menu");
            }
        });
        run(@"shifted punctuation preserves language and ASCII passes through", @"KBD-A02 KBD-A05", ^{
            modifier(typing, 56, NSEventModifierFlagShift, 1);
            require([typing handleEvent:key(@"!", 18, NSEventModifierFlagShift)] && [editor.view.string isEqual:@"！"] &&
                    !typing.composing, @"actual Shift symbol chord commits directly");
            require(!modifier(typing, 56, 0, 1.1) && !typing.ascii, @"symbol chord release does not switch");
            type(typing, @"nihao");
            NSString *candidate = typing.snapshot.candidates[typing.snapshot.highlighted];
            require(press(typing, @"/", 44) && [editor.view.string isEqual:[@"！" stringByAppendingFormat:@"%@、", candidate]] &&
                    !typing.composing, @"slash commits the highlighted phrase and punctuation without symbol candidates");
            [typing toggleASCII];
            require(!press(typing, @"/", 44) && ![typing handleEvent:key(@"!", 18, NSEventModifierFlagShift)],
                    @"English-mode symbols pass to the host unchanged");
        });
        run(@"full width commits wide ASCII in Chinese mode only", @"KBD-A09", ^{
            NSMutableArray *changes = [NSMutableArray array];
            typing.modeChanged = ^(ArionTypingSwitch change) { [changes addObject:@(change)]; };
            @try {
                require(!typing.fullWidth && !typing.englishPunctuation, @"both options default off");
                require(!press(typing, @"1", 18) && editor.view.string.length == 0, @"half-width digit passes to the host");
                type(typing, @"nihao");
                require([typing toggleFullWidth] && typing.fullWidth && !typing.composing && !editor.view.hasMarkedText &&
                        editor.insertions == 0, @"switch discards unconfirmed composition without committing it");
                require([changes isEqual:@[@(ArionTypingSwitchWidth)]], @"switch notifies once for the width badge");
                require(press(typing, @"1", 18) && [editor.view.string isEqual:@"１"], @"digit commits full-width");
                require(press(typing, @"@", 19) && [editor.view.string isEqual:@"１＠"], @"ASCII symbol from the Chinese table commits full-width");
                require(press(typing, @".", 47) && [editor.view.string isEqual:@"１＠。"], @"Chinese punctuation is unchanged");
                type(typing, @"nihao"); press(typing, @" ", 49);
                require([editor.view.string isEqual:@"１＠。你好"], @"Pinyin letters still compose");
                [typing toggleASCII];
                require(typing.ascii && !press(typing, @"1", 18) && !press(typing, @"a", 0) &&
                        [editor.view.string isEqual:@"１＠。你好"], @"English mode passes keys through half-width");
                [typing toggleASCII];
                require(press(typing, @"2", 19) && [editor.view.string isEqual:@"１＠。你好２"], @"returning to Chinese restores full width");
                require([typing toggleFullWidth] && !typing.fullWidth && !press(typing, @"3", 20), @"switching back restores half width");
            } @finally {
                if (typing.fullWidth) [typing toggleFullWidth];
                typing.modeChanged = nil;
            }
        });
        run(@"English punctuation switches by Ctrl period and combines with full width", @"KBD-A09", ^{
            NSMutableArray *changes = [NSMutableArray array];
            typing.modeChanged = ^(ArionTypingSwitch change) { [changes addObject:@(change)]; };
            @try {
                require(press(typing, @",", 43) && [editor.view.string isEqual:@"，"], @"default commits Chinese punctuation");
                editor.view.string = @""; editor.insertions = 0;
                type(typing, @"nihao");
                require([typing handleEvent:key(@".", 47, NSEventModifierFlagControl)] && typing.englishPunctuation &&
                        !typing.composing && editor.insertions == 0, @"Ctrl+. switches and discards unconfirmed composition");
                require([changes isEqual:@[@(ArionTypingSwitchPunctuation)]], @"switch notifies once for the punctuation badge");
                require(!press(typing, @",", 43) && !press(typing, @".", 47) && ![typing handleEvent:key(@"\"", 39, NSEventModifierFlagShift)] &&
                        editor.view.string.length == 0, @"ASCII punctuation passes to the host");
                type(typing, @"nihao");
                NSString *candidate = typing.snapshot.candidates[typing.snapshot.highlighted];
                require(!press(typing, @",", 43) && [editor.view.string isEqual:candidate] && !typing.composing,
                        @"punctuation commits the highlighted phrase, then passes through");
                editor.view.string = @""; editor.insertions = 0;
                require([typing toggleFullWidth], @"enable full width with English punctuation");
                require(press(typing, @".", 47) && press(typing, @",", 43) && [editor.view.string isEqual:@"．，"],
                        @"English punctuation commits full-width ASCII forms");
                [typing toggleFullWidth];
                require(![typing handleEvent:key(@".", 47, NSEventModifierFlagControl | NSEventModifierFlagShift)] &&
                        ![typing handleEvent:key(@".", 47, NSEventModifierFlagCommand)] && typing.englishPunctuation,
                        @"other period chords pass through");
                [typing toggleASCII];
                require(![typing handleEvent:key(@".", 47, NSEventModifierFlagControl)] && typing.englishPunctuation,
                        @"English mode passes Ctrl+. to the host");
                [typing toggleASCII];
                require([typing handleEvent:key(@".", 47, NSEventModifierFlagControl)] && !typing.englishPunctuation &&
                        press(typing, @".", 47) && [editor.view.string hasSuffix:@"。"], @"Ctrl+. switches back to Chinese punctuation");
            } @finally {
                if (typing.fullWidth) [typing toggleFullWidth];
                if (typing.englishPunctuation) [typing toggleEnglishPunctuation];
                typing.modeChanged = nil;
            }
        });
        run(@"width and punctuation survive engine restart", @"KBD-A09", ^{
            NSString *options = [user stringByAppendingPathComponent:@"options.json"];
            NSString *vault = [user stringByAppendingPathComponent:@"encrypted"];
            NSError *restartError = nil;
            @try {
                require([typing toggleFullWidth] && [typing toggleEnglishPunctuation], @"enable both options");
                [typing deactivate]; [engine stop];
                require(ArionRegisterTestEncryptedLearning(vault, testKey, &restartError) &&
                        [engine startWithData:data userDirectory:user error:&restartError], @"restart real engine");
                require(engine.fullWidth && engine.englishPunctuation, @"restart reads both options");
                require([typing activate:editor error:&restartError] && press(typing, @".", 47) && [editor.view.string isEqual:@"．"],
                        @"new session applies persisted options");
                NSDictionary *saved = [NSJSONSerialization JSONObjectWithData:[NSData dataWithContentsOfFile:options] options:0 error:nil];
                require([saved isEqual:@{@"fullWidth": @YES, @"englishPunctuation": @YES}], @"options file holds only the two choices");
                // A choice that cannot be saved leaves the option unchanged.
                [NSFileManager.defaultManager removeItemAtPath:options error:nil];
                require([NSFileManager.defaultManager createDirectoryAtPath:options withIntermediateDirectories:NO attributes:nil error:&restartError],
                        @"block the isolated options path");
                require(![typing toggleFullWidth] && typing.fullWidth, @"unsaved width choice is refused");
                [NSFileManager.defaultManager removeItemAtPath:options error:nil];
            } @finally {
                [NSFileManager.defaultManager removeItemAtPath:options error:nil];
                [typing deactivate]; [engine stop];
                require(ArionRegisterTestEncryptedLearning(vault, testKey, &restartError) &&
                        [engine startWithData:data userDirectory:user error:&restartError], @"restore real engine for remaining tests");
                require(!engine.fullWidth && !engine.englishPunctuation, @"absent options file means defaults");
                require([typing activate:editor error:&restartError], @"reactivate for the shared fixture teardown");
            }
        });
        run(@"language hint stays by caret without taking focus and expires", @"KBD-A05", ^{
            ArionLanguagePanel *hint = [ArionLanguagePanel new];
            NSWindow *window = [hint valueForKey:@"window"];
            NSWindow *keyWindow = NSApp.keyWindow;
            NSRect screen = NSScreen.mainScreen.visibleFrame;
            NSRect caret = NSMakeRect(NSMaxX(screen) - 1, NSMinY(screen) + 1, 0, 20);
            NSDate *firstShownAt = NSDate.date;
            [hint show:NSLocalizedString(@"KeyboardModeEnglish", nil) previous:NSLocalizedString(@"KeyboardModeChinese", nil) caretRect:caret];
            NSDate *firstShowFinishedAt = NSDate.date;
            require(window.visible && !window.canBecomeKeyWindow && window.ignoresMouseEvents && NSApp.keyWindow == keyWindow,
                    @"zero-width caret shows a click-through hint without changing key window");
            require(NSEqualSizes(window.frame.size, NSMakeSize(23, 23)) && !window.hasShadow &&
                    window.animationBehavior == NSWindowAnimationBehaviorNone,
                    @"compact shadowless badge uses immediate window transitions");
            require(NSContainsRect(screen, window.frame) && NSMinY(window.frame) >= NSMaxY(caret),
                    @"hint fits screen and flips above bottom-edge caret");
            CATextLayer *label = [hint valueForKey:@"label"];
            CATextLayer *outgoing = [hint valueForKey:@"outgoing"];
            BOOL reduceMotion = NSWorkspace.sharedWorkspace.accessibilityDisplayShouldReduceMotion;
            require([label.string isEqual:NSLocalizedString(@"KeyboardModeEnglish", nil)] &&
                    [outgoing.string isEqual:NSLocalizedString(@"KeyboardModeChinese", nil)] && outgoing.opacity == 0,
                    @"badge settles on the new mode and keeps the previous one only for the flip");
            require(reduceMotion ? (![label animationForKey:@"flip"] && ![outgoing animationForKey:@"flip"])
                                 : ([label animationForKey:@"flip"] && [outgoing animationForKey:@"flip"]),
                    @"mode change flips the label unless Reduce Motion is on");
            NSTimer *previousDismissal = [hint valueForKey:@"dismissal"];
            require(previousDismissal.valid && [previousDismissal.fireDate timeIntervalSinceDate:firstShownAt] >= 1.0 &&
                    [previousDismissal.fireDate timeIntervalSinceDate:firstShowFinishedAt] <= 1.0,
                    @"hint schedules the documented one-second lifetime");
            NSDate *nextShownAt = NSDate.date;
            [hint show:NSLocalizedString(@"KeyboardModeChinese", nil) previous:NSLocalizedString(@"KeyboardModeEnglish", nil) caretRect:caret];
            NSDate *nextShowFinishedAt = NSDate.date;
            NSTimer *currentDismissal = [hint valueForKey:@"dismissal"];
            require(!previousDismissal.valid && currentDismissal != previousDismissal &&
                    currentDismissal.valid && [currentDismissal.fireDate timeIntervalSinceDate:nextShownAt] >= 1.0 &&
                    [currentDismissal.fireDate timeIntervalSinceDate:nextShowFinishedAt] <= 1.0 && window.visible,
                    @"new mode invalidates the previous timer and starts a full one-second lifetime");
            // Host scheduling may overshoot a fixed sleep. Check the declared
            // interval directly, then wait for actual run-loop delivery.
            NSDate *deadline = [NSDate dateWithTimeIntervalSinceNow:3];
            while (window.visible && deadline.timeIntervalSinceNow > 0) {
                [NSRunLoop.currentRunLoop runMode:NSDefaultRunLoopMode beforeDate:[NSDate dateWithTimeIntervalSinceNow:0.01]];
            }
            require(!window.visible, @"hint automatically disappears after one second");
            require(!currentDismissal.valid && ![hint valueForKey:@"dismissal"],
                    @"delivered dismissal retires its timer");
            [hint show:NSLocalizedString(@"KeyboardModeEnglish", nil) previous:NSLocalizedString(@"KeyboardModeChinese", nil) caretRect:caret];
            [hint hide];
            require(!window.visible, @"typing or focus loss can hide immediately");
            [hint show:NSLocalizedString(@"KeyboardModeEnglish", nil) previous:NSLocalizedString(@"KeyboardModeChinese", nil) caretRect:NSZeroRect];
            require(!window.visible, @"missing caret never produces screen-corner feedback");
        });
        run(@"Shift Space discards draft and passes ASCII", @"KBD-A05", ^{
            type(typing, @"nihao");
            require([typing handleEvent:key(@" ", 49, NSEventModifierFlagShift)], @"Shift Space handled");
            require(typing.ascii && !typing.composing && editor.view.string.length == 0, @"ASCII switch discards unconfirmed text");
            require(!press(typing, @"x", 7) && editor.view.string.length == 0, @"ASCII letter passes to host without insertion");
        });
        run(@"stale candidate callbacks cannot select", @"KBD-A03 KBD-A05", ^{
            type(typing, @"ni"); NSUInteger stale = typing.revision;
            type(typing, @"hao");
            NSString *before = editor.view.string;
            require(![typing select:0 revision:stale] && ![typing highlight:0 revision:stale], @"older selection and highlight rejected");
            require([editor.view.string isEqual:before] && editor.insertions == 0, @"stale callback cannot write host");
            [typing cancel];
            require(![typing select:0 revision:stale] && editor.view.string.length == 0, @"cancelled callback cannot resurrect composition");
        });
        run(@"deactivation discards draft and rejects callback", @"KBD-A05", ^{
            type(typing, @"nihao"); NSUInteger revision = typing.revision;
            [typing deactivate];
            require(editor.view.string.length == 0 && !editor.view.hasMarkedText, @"deactivation clears only own draft");
            require(![typing select:0 revision:revision], @"old activation cannot select");
        });
        run(@"apostrophe syllable boundary converts real phrase", @"KBD-A02 KBD-A13", ^{
            type(typing, @"ni'hao");
            for (NSString *candidate in typing.snapshot.candidates) {
                require([candidate rangeOfCharacterFromSet:[NSCharacterSet characterSetWithCharactersInString:@"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"]].location == NSNotFound,
                        @"normal apostrophe Pinyin retains Chinese and Emoji lanes without English prefixes");
            }
            require([typing.snapshot.candidates containsObject:@"你好"], @"declared apostrophe boundary resolves actual 你好");
            NSUInteger index = [typing.snapshot.candidates indexOfObject:@"你好"];
            require([typing select:index revision:typing.revision] && [editor.view.string isEqual:@"你好"], @"apostrophe selection commits actual phrase");
        });
        run(@"apostrophe normal Pinyin excludes English prefixes", @"KBD-A02 KBD-A13", ^{
            type(typing, @"xi'an");
            require([typing.snapshot.candidates.firstObject isEqual:@"西安"], @"original full Chinese answer remains first");
            for (NSString *candidate in typing.snapshot.candidates) {
                require([candidate rangeOfCharacterFromSet:[NSCharacterSet characterSetWithCharactersInString:@"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"]].location == NSNotFound,
                        @"explicit syllable boundary does not introduce xi or Xi prefixes");
            }
        });
        run(@"abbreviated Pinyin retains real candidate identity", @"KBD-A02", ^{
            type(typing, @"nh");
            record(@"abbreviation first page", typing, editor);
            NSUInteger index = [typing.snapshot.candidates indexOfObject:@"你好"];
            for (NSUInteger page = 0; index == NSNotFound && page < 20 && !typing.snapshot.lastPage; page++) {
                press(typing, @"", 121); index = [typing.snapshot.candidates indexOfObject:@"你好"];
            }
            require(index != NSNotFound, @"actual abbreviation candidates include 你好");
            require([typing select:index revision:typing.revision] && [editor.view.string isEqual:@"你好"], @"abbreviation commits selected engine candidate");
        });
        run(@"v input represents umlaut Pinyin", @"KBD-A02", ^{
            type(typing, @"lvse");
            record(@"umlaut preedit and candidates", typing, editor);
            require([typing.snapshot.preedit containsString:@"ü"], @"declared schema displays umlaut for v input");
            NSUInteger index = [typing.snapshot.candidates indexOfObject:@"绿色"];
            for (NSUInteger page = 0; index == NSNotFound && page < 20 && !typing.snapshot.lastPage; page++) {
                press(typing, @"", 121); index = [typing.snapshot.candidates indexOfObject:@"绿色"];
            }
            require(index != NSNotFound, @"real dictionary resolves lvse to 绿色");
            require([typing select:index revision:typing.revision] && [editor.view.string isEqual:@"绿色"], @"umlaut phrase commits actual engine text");
        });
        run(@"literal umlaut Pinyin uses the declared schema alias", @"KBD-A02", ^{
            type(typing, @"lüse");
            record(@"literal umlaut preedit and candidates", typing, editor);
            require([typing.snapshot.preedit containsString:@"ü"], @"literal umlaut remains in phonetic preedit");
            NSUInteger index = [typing.snapshot.candidates indexOfObject:@"绿色"];
            for (NSUInteger page = 0; index == NSNotFound && page < 20 && !typing.snapshot.lastPage; page++) {
                press(typing, @"", 121); index = [typing.snapshot.candidates indexOfObject:@"绿色"];
            }
            require(index != NSNotFound, @"literal umlaut resolves the actual schema candidate");
            require([typing select:index revision:typing.revision] && [editor.view.string isEqual:@"绿色"], @"literal umlaut commits engine text");
            int rawUmlaut = ArionKeyboardKey(key(@"Ü", 0, NSEventModifierFlagShift));
            require(rawUmlaut == 0x00dc && arion_key_normalize(rawUmlaut, 1, 0) == 'V', @"uppercase umlaut uses shared schema alias semantics");
            type(typing, @"l");
            require(![typing handleEvent:key(@"ü", 0, NSEventModifierFlagOption)] && !typing.composing &&
                    [editor.view.string isEqual:@"绿色"], @"Option-modified umlaut relinquishes composition before host pass-through");
            [typing toggleASCII];
            require(typing.ascii && !press(typing, @"ü", 0) && [editor.view.string isEqual:@"绿色"],
                    @"English mode passes the original umlaut event without inserting its schema alias");
        });
        run(@"Chinese punctuation follows confirmed phrase", @"KBD-A02", ^{
            type(typing, @"nihao"); press(typing, @" ", 49);
            require(press(typing, @",", 43), @"schema handles Chinese comma");
            require([editor.view.string isEqual:@"你好，"] && !typing.composing && !editor.view.hasMarkedText,
                    @"mixed phrase and punctuation have no Latin residue or newline");
        });
        run(@"Return commits literal raw input", @"KBD-A02 KBD-A03", ^{
            type(typing, @"arionxyz");
            record(@"literal input before Return", typing, editor);
            require(press(typing, @"\r", 36), @"Return handled by real engine");
            require([editor.view.string isEqual:@"arionxyz"] && editor.insertions == 1 && !editor.view.hasMarkedText,
                    @"Return commits exact raw Latin input without newline or invented Chinese");
        });
        run(@"confirmed learning survives restart and cancellation adds nothing", @"KBD-A07", ^{
            type(typing, @"shishi");
            record(@"before confirmed learning", typing, editor);
            NSUInteger chosenIndex = NSNotFound;
            for (NSUInteger i = 1; i < typing.snapshot.candidates.count; i++) {
                if (typing.snapshot.candidates[i].length == 2) { chosenIndex = i; break; }
            }
            require(chosenIndex != NSNotFound, @"real dictionary supplies a nonfirst complete two-syllable word");
            NSString *chosen = typing.snapshot.candidates[chosenIndex];
            NSUInteger staleRevision = typing.revision;
            require([typing select:chosenIndex revision:staleRevision] && [editor.view.string isEqual:chosen] && editor.insertions == 1,
                    @"actual nonfirst full-word selection commits");
            [typing deactivate]; [engine stop];
            require(![typing select:0 revision:staleRevision], @"engine shutdown invalidates old callback");
            NSString *vault = [user stringByAppendingPathComponent:@"encrypted"];
            NSString *file = [vault stringByAppendingPathComponent:@"arion_pinyin.arion-userdb"];
            NSData *ciphertext = [NSData dataWithContentsOfFile:file];
            require(ciphertext.length > 32, @"real engine persisted encrypted learned dictionary");
            require([ciphertext rangeOfData:[chosen dataUsingEncoding:NSUTF8StringEncoding] options:0 range:NSMakeRange(0, ciphertext.length)].location == NSNotFound,
                    @"selected synthetic word absent from persisted ciphertext bytes");
            for (NSString *canary in @[@"shishi", @"shi shi"]) {
                require([ciphertext rangeOfData:[canary dataUsingEncoding:NSUTF8StringEncoding] options:0 range:NSMakeRange(0, ciphertext.length)].location == NSNotFound,
                        @"synthetic Pinyin absent from persisted ciphertext bytes");
            }
            const auto learned = learningRecords(file, testKey);
            BOOL foundLearned = NO;
            for (const auto &[code, value] : learned) {
                (void)value;
                if (code.find(std::string("\t") + chosen.UTF8String) != std::string::npos) foundLearned = YES;
            }
            require(foundLearned, @"actual Rime record includes confirmed selected word");
            NSError *restartError = nil;
            require(ArionRegisterTestEncryptedLearning(vault, testKey, &restartError) &&
                    [engine startWithData:data userDirectory:user error:&restartError], @"restart real engine with same ciphertext and test key");
            uintptr_t shared=[engine createSession];
            ArionPersonalTerm sameWord{"independent-shared",chosen.UTF8String,"shi shi"};
            require(arion_personal_vocabulary_replace(shared,1,&sameWord,1),@"project independently learned identical spelling");
            require(arion_personal_vocabulary_replace(shared,2,nullptr,0),@"remove shared contribution without unlearning independent phrase");
            [engine destroySession:shared];
            editor.view.string = @""; editor.insertions = 0;
            require([typing activate:editor error:&restartError], @"activate fresh generation after restart");
            type(typing, @"shishi");
            record(@"actual candidates after restart", typing, editor);
            require([typing.snapshot.candidates.firstObject isEqual:chosen], @"confirmed nonfirst word is promoted after engine restart");
            [typing cancel];
            type(typing, @"nihao");
            NSUInteger partial = [typing.snapshot.candidates indexOfObject:@"你"];
            require(partial != NSNotFound && [typing select:partial revision:typing.revision] && typing.composing,
                    @"select real provisional prefix before cancellation");
            [typing cancel]; [typing deactivate]; [engine stop];
            const auto cancelled = learningRecords(file, testKey);
            require(learned == cancelled, @"cancelled raw and partial compositions do not change any learned record or tick");
            require(ArionRegisterTestEncryptedLearning(vault, testKey, &restartError) &&
                    [engine startWithData:data userDirectory:user error:&restartError], @"restore real engine for remaining tests");
        });
        run(@"released owner destroys actual engine session without client callbacks", @"KBD-A05", ^{
            TypingEditor *releasedEditor = [TypingEditor new];
            __block NSUInteger callbacks = 0;
            __weak ArionTyping *releasedOwner;
            RimeSessionId releasedSession = 0;
            NSUInteger markings = 0, callbacksBeforeRelease = 0;
            @autoreleasepool {
                ArionTyping *owner = [[ArionTyping alloc] initWithEngine:engine];
                require([owner activate:releasedEditor error:nil], @"activate separately owned real engine session");
                owner.changed = ^{ callbacks++; };
                type(owner, @"nihao");
                releasedSession = [[owner valueForKey:@"session"] unsignedLongLongValue];
                require(releasedSession && rime_get_api()->find_session(releasedSession), @"real Rime service owns live session before release");
                markings = releasedEditor.markings;
                callbacksBeforeRelease = callbacks;
                releasedOwner = owner;
                owner = nil;
            }
            require(releasedOwner == nil && !rime_get_api()->find_session(releasedSession), @"releasing owner destroys actual engine session");
            require(releasedEditor.markings == markings && releasedEditor.insertions == 0 && releasedEditor.view.hasMarkedText,
                    @"destructor never writes to client text or marked range");
            require(callbacks == callbacksBeforeRelease, @"destruction leaves UI callback count stable");
            [trace addObject:@{@"step": @"released real session", @"sessionFound": @(rime_get_api()->find_session(releasedSession)),
                @"clientMarkCalls": @(releasedEditor.markings), @"uiCallbacks": @(callbacks)}];
        });
        run(@"mouse candidate failures revoke callbacks and preserve pending recovery", @"KBD-A05 KBD-A07", ^{
            for (NSString *operation in @[@"select", @"highlight"]) {
                type(typing, @"nihao");
                NSUInteger prefix = [typing.snapshot.candidates indexOfObject:@"你"];
                require(prefix != NSNotFound, @"actual prefix candidate available for mouse callback");
                ArionTyping *priming = [[ArionTyping alloc] initWithEngine:engine];
                TypingEditor *primingEditor = [TypingEditor new];
                require([priming activate:primingEditor error:nil], @"create shared-userdb fault session");
                type(priming, @"nihao"); press(priming, @" ", 49);
                NSString *vault = [user stringByAppendingPathComponent:@"encrypted"];
                NSString *saved = [user stringByAppendingPathComponent:@"encrypted-mouse-saved"];
                NSError *faultError = nil;
                require([NSFileManager.defaultManager moveItemAtPath:vault toPath:saved error:&faultError], @"move isolated vault before actual flush failure");
                @try {
                    require([[@"blocked test directory" dataUsingEncoding:NSUTF8StringEncoding] writeToFile:vault options:NSDataWritingAtomic error:&faultError], @"fault only isolated test filesystem");
                    press(priming, @"n", 45);
                    require(ArionEncryptedLearningError() != nil && !priming.available, @"real translator query flush causes storage failure");
                    NSUInteger stale = typing.revision;
                    BOOL selected = [operation isEqual:@"select"] ? [typing select:prefix revision:stale] : [typing highlight:prefix revision:stale];
                    require(!selected && !typing.available, @"mouse callback surfaces sticky learning failure");
                    require(typing.snapshot == nil && !typing.composing && !editor.view.hasMarkedText && typing.revision > stale,
                            @"mouse failure clears owned draft, candidate data and callback revision");
                    require(typing.recoveryText.length > 0 && editor.insertions == 0, @"pending preedit retained only for explicit recovery");
                    record([operation stringByAppendingString:@" after actual storage failure"], typing, editor);
                    NSString *recovery = typing.recoveryText;
                    NSUInteger revoked = typing.revision;
                    require(![typing select:0 revision:stale] && ![typing highlight:0 revision:stale] &&
                            ![typing select:0 revision:revoked] && ![typing highlight:0 revision:revoked], @"both stale and current callbacks refused while unavailable");
                    require([typing.recoveryText isEqual:recovery] && editor.insertions == 0, @"rejected callbacks neither lose recovery nor insert text");
                } @finally {
                    [priming deactivate]; [typing deactivate]; [engine stop];
                    [NSFileManager.defaultManager removeItemAtPath:vault error:nil];
                    require([NSFileManager.defaultManager moveItemAtPath:saved toPath:vault error:&faultError], @"restore isolated vault");
                }
                require(ArionRegisterTestEncryptedLearning(vault, testKey, &faultError) &&
                    [engine startWithData:data userDirectory:user error:&faultError], @"restart actual engine after explicit fault recovery");
                editor.view.string = @""; editor.insertions = 0;
                require([typing activate:editor error:&faultError], @"activate new generation for next fixture");
            }
        });
        run(@"learning write failure retains actual engine commit", @"KBD-A07", ^{
            // Query flushes pending learning, so compose this session first.
            // Another real session shares the userdb and creates the pending
            // selection that this session's final commit must flush.
            type(typing, @"nihao");
            ArionTyping *priming = [[ArionTyping alloc] initWithEngine:engine];
            TypingEditor *primingEditor = [TypingEditor new];
            NSError *primingError = nil;
            require([priming activate:primingEditor error:&primingError], @"create second real engine session");
            type(priming, @"nihao"); press(priming, @" ", 49);
            require([primingEditor.view.string isEqual:@"你好"] && priming.available, @"prime shared real learning transaction");
            NSString *vault = [user stringByAppendingPathComponent:@"encrypted"];
            NSString *saved = [user stringByAppendingPathComponent:@"encrypted-saved"];
            NSError *faultError = nil;
            NSArray *files = [NSFileManager.defaultManager contentsOfDirectoryAtPath:vault error:&faultError];
            require(files != nil && [files containsObject:@"arion_pinyin.arion-userdb"], @"real encrypted dictionary initialized");
            require([NSFileManager.defaultManager moveItemAtPath:vault toPath:saved error:&faultError], @"move only isolated test vault");
            @try {
                require([[@"blocked test directory" dataUsingEncoding:NSUTF8StringEncoding] writeToFile:vault options:NSDataWritingAtomic error:&faultError], @"replace vault directory with regular file");
                require(ArionEncryptedLearningError() == nil, @"filesystem fault alone does not fabricate learning error");
                press(typing, @" ", 49);
                record(@"second commit after actual filesystem fault", typing, editor);
                require(ArionEncryptedLearningError() != nil && !typing.available, @"actual flush failure disables typing");
                require([editor.view.string isEqual:@"你好"] && editor.insertions == 1,
                        @"learning failure cannot swallow or duplicate actual second engine commit");
                require(!editor.view.hasMarkedText && !typing.composing, @"failed learning leaves committed host text intact");
                NSUInteger insertions = editor.insertions;
                require(!press(typing, @"x", 7) && editor.insertions == insertions && [editor.view.string isEqual:@"你好"],
                        @"unavailable engine stops accepting new input without rewriting committed text");
            } @finally {
                [priming deactivate];
                [NSFileManager.defaultManager removeItemAtPath:vault error:nil];
                require([NSFileManager.defaultManager moveItemAtPath:saved toPath:vault error:&faultError], @"restore isolated test vault path");
            }
        });
        [engine stop];
        BOOL passed = YES;
        for (NSDictionary *result in results) if (![result[@"passed"] boolValue]) passed = NO;
        NSDictionary *output = @{@"passed": @(passed), @"engine": @"librime 1.17.0", @"schema": @"arion_pinyin", @"nativeLibraryPath": nativeLibraryPath,
            @"learning": @"production encrypted backend with isolated synthetic test key; no Keychain access",
            @"boundary": @"real engine and AppKit text store protocol adapter; no installed IMK event dispatch or input-source changes",
            @"cases": results};
        NSData *json = [NSJSONSerialization dataWithJSONObject:output options:NSJSONWritingPrettyPrinted error:&error];
        if (!json || ![json writeToFile:report options:NSDataWritingAtomic error:&error]) { NSLog(@"Report failed: %@", error); return 2; }
        return passed ? 0 : 1;
    }
}
