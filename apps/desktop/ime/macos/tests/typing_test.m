#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>
#import "../typing.h"
#import "../candidate_panel.h"
#import "../learning.h"
#include <rime/dict/db.h>
#include <rime_api.h>
#include "../../shared/input.h"
#include "../../shared/emoji.h"
#include <map>
#include <memory>

// The client protocol adapter owns a real AppKit text store; conversion is real librime.
@interface TypingEditor : NSObject
@property(strong) NSTextView *view;
@property NSUInteger insertions;
@property NSUInteger markings;
@end
@implementation TypingEditor
- (instancetype)init {
    if ((self = [super init])) self.view = [[NSTextView alloc] initWithFrame:NSMakeRect(0, 0, 500, 200)];
    return self;
}
- (void)setMarkedText:(id)text selectionRange:(NSRange)selection replacementRange:(NSRange)replacement {
    self.markings++;
    [self.view setMarkedText:text selectedRange:selection replacementRange:replacement];
}
- (void)insertText:(id)text replacementRange:(NSRange)replacement {
    self.insertions++;
    [self.view insertText:text replacementRange:replacement];
}
@end

static void require(BOOL condition, NSString *description) {
    if (!condition) @throw [NSException exceptionWithName:@"AssertionFailure" reason:description userInfo:nil];
}
static NSEvent *key(NSString *characters, unsigned short code, NSEventModifierFlags flags) {
    return [NSEvent keyEventWithType:NSEventTypeKeyDown location:NSZeroPoint modifierFlags:flags
        timestamp:0 windowNumber:0 context:nil characters:characters charactersIgnoringModifiers:characters
        isARepeat:NO keyCode:code];
}
static BOOL modifier(ArionTyping *typing, unsigned short code, NSEventModifierFlags flags, NSTimeInterval timestamp) {
    NSEvent *event = [NSEvent keyEventWithType:NSEventTypeFlagsChanged location:NSZeroPoint modifierFlags:flags
        timestamp:timestamp windowNumber:0 context:nil characters:@"" charactersIgnoringModifiers:@""
        isARepeat:NO keyCode:code];
    return [typing handleEvent:event];
}
static BOOL press(ArionTyping *typing, NSString *characters, unsigned short code) {
    return [typing handleEvent:key(characters, code, 0)];
}
static void type(ArionTyping *typing, NSString *input) {
    for (NSUInteger i = 0; i < input.length; i++) {
        NSString *character = [input substringWithRange:NSMakeRange(i, 1)];
        require(press(typing, character, 0), [@"engine handles synthetic letter " stringByAppendingString:character]);
    }
}
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
        if (argc != 4) { fprintf(stderr, "usage: typing-test DATA USER REPORT\n"); return 2; }
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
            require(![typing.snapshot.candidates containsObject:@"artificial"], @"English prefixes do not flood Pinyin with completions");
            type(typing, @"l");
            require([typing.snapshot.candidates[0] isEqual:@"artificial"], @"full English word leads generated Chinese sentences");
            press(typing, @"\x7f", 51);
            require(![typing.snapshot.candidates containsObject:@"artificial"], @"Backspace removes a stale exact match");
            type(typing, @"l"); press(typing, @" ", 49);
            require([editor.view.string isEqual:@"artificial"] && editor.insertions == 1, @"Space commits the English engine candidate once");
            type(typing, @"nihao"); press(typing, @" ", 49);
            type(typing, @"hello");
            NSUInteger title = [typing.snapshot.candidates indexOfObject:@"Hello"];
            require(title < 9, @"sentence-initial case variant is visible");
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
        run(@"English candidate panel rendering", @"KBD-A13", ^{
            ArionCandidatePanel *panel = [ArionCandidatePanel new];
            @try {
                for (NSString *word in @[@"artificial", @"can"]) {
                    type(typing, word);
                    record(word, typing, editor);
                    [panel showCandidates:typing.snapshot.candidates highlighted:typing.snapshot.highlighted lastPage:typing.snapshot.lastPage
                        revision:typing.revision caretRect:NSMakeRect(80, 300, 1, 20)];
                    savePanel(panel, report, [NSString stringWithFormat:@"english-macos-%@.png", word]);
                    [typing cancel];
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
                    ArionEmojiCandidateMetadata metadata;
                    found = arion_emoji_candidate_metadata(session,index,&metadata) && metadata.emoji &&
                        strcmp(metadata.source,item.source)==0 && metadata.end==strlen(item.pinyin);
                    break;
                }
                arion_pinyin_candidates_dispose(texts,count);
                [engine destroySession:session];
                if (!found) [unreachable addObject:@(item.pinyin)];
            }
            require(unreachable.count == 0, [@"unreachable reviewed triggers: " stringByAppendingString:[unreachable componentsJoinedByString:@", "]]);
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
            require([[ranker order:candidates after:@"我们"] isEqual:@[@0, @1, @2, @3]], @"no learned pair keeps engine order");
            [ranker learn:@"妮好，" after:@"我们"];
            require(writes.count == 1 && [writes[0][@"c\t我们\t妮好"] hasPrefix:@"1 "], @"trailing punctuation is not part of the learned word");
            require([[ranker order:candidates after:@"我们"] isEqual:@[@0, @3, @1, @2]], @"one confirmation lifts a word two places, not three");
            [ranker learn:@"妮好" after:@"我们"];
            [ranker learn:@"你" after:@"我们"]; [ranker learn:@"你" after:@"我们"]; [ranker learn:@"你" after:@"我们"];
            NSArray *order = [ranker order:candidates after:@"我们"];
            require([[order sortedArrayUsingSelector:@selector(compare:)] isEqual:@[@0, @1, @2, @3]], @"order is a permutation");
            require([order[0] isEqual:@3] && ![order[0] isEqual:@2], @"same-length word promoted; shorter prefix stays behind");
            require([[ranker order:candidates after:@"他们"] isEqual:@[@0, @1, @2, @3]], @"another context keeps engine order");
            require([[ranker order:candidates after:nil] isEqual:@[@0, @1, @2, @3]], @"no context keeps engine order");
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
            require(modifier(typing, 56, 0, 1.1) && typing.ascii && !typing.composing && editor.view.string.length == 0,
                    @"short release switches and clears unconfirmed draft");
            require(switches == 1 && ![typing select:0 revision:stale], @"one mode notification and no stale candidate write");
            require(!modifier(typing, 56, 0, 1.11) && switches == 1, @"duplicate release cannot toggle");
            modifier(typing, 60, NSEventModifierFlagShift, 2);
            require(modifier(typing, 60, 0, 2.1) && !typing.ascii && switches == 2, @"right Shift switches back persistently");
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
        run(@"apostrophe syllable boundary converts real phrase", @"KBD-A02", ^{
            type(typing, @"ni'hao");
            require([typing.snapshot.candidates containsObject:@"你好"], @"declared apostrophe boundary resolves actual 你好");
            NSUInteger index = [typing.snapshot.candidates indexOfObject:@"你好"];
            require([typing select:index revision:typing.revision] && [editor.view.string isEqual:@"你好"], @"apostrophe selection commits actual phrase");
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
        NSDictionary *output = @{@"passed": @(passed), @"engine": @"librime 1.17.0", @"schema": @"arion_pinyin",
            @"learning": @"production encrypted backend with isolated synthetic test key; no Keychain access",
            @"boundary": @"real engine and AppKit text store protocol adapter; no installed IMK event dispatch or input-source changes",
            @"cases": results};
        NSData *json = [NSJSONSerialization dataWithJSONObject:output options:NSJSONWritingPrettyPrinted error:&error];
        if (!json || ![json writeToFile:report options:NSDataWritingAtomic error:&error]) { NSLog(@"Report failed: %@", error); return 2; }
        return passed ? 0 : 1;
    }
}
