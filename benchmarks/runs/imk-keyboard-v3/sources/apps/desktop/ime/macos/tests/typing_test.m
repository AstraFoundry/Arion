#import <Cocoa/Cocoa.h>
#import "../typing.h"
#import "../learning.h"
#include <rime/dict/db.h>
#include <rime_api.h>
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
        @"revision": @(typing.revision), @"caret": @(typing.snapshot.selection.location), @"ascii": @(typing.ascii), @"available": @(typing.available), @"recoveryText": typing.recoveryText ?: @""};
}

static void record(NSString *step, ArionTyping *typing, TypingEditor *editor) {
    [trace addObject:@{@"step": step, @"state": capture(typing, editor)}];
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
