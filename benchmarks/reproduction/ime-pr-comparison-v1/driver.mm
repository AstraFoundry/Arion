// The controller owns references. This driver accepts raw input and menu indices only.
#import <Cocoa/Cocoa.h>
#import "typing.h"
#import "learning.h"
#include <mach/mach_time.h>
#include <sys/resource.h>
#include <unistd.h>

@interface ComparisonEditor : NSObject
@property(strong) NSTextView *view;
@property NSUInteger insertions;
@property NSUInteger markings;
@end
@implementation ComparisonEditor
- (instancetype)init {
    if ((self = [super init])) self.view = [[NSTextView alloc] initWithFrame:NSMakeRect(0, 0, 600, 200)];
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
static mach_timebase_info_data_t timebase;
static double elapsed(uint64_t start) {
    return (double)(mach_absolute_time() - start) * timebase.numer / timebase.denom / 1000000.0;
}
static void require(BOOL condition, NSString *message) {
    if (!condition) @throw [NSException exceptionWithName:@"ComparisonFailure" reason:message userInfo:nil];
}
static void emit(NSDictionary *value) {
    NSData *data = [NSJSONSerialization dataWithJSONObject:value options:0 error:nil];
    require(data != nil, @"Serialize driver output");
    fwrite(data.bytes, 1, data.length, stdout); fputc('\n', stdout); fflush(stdout);
}
static NSDictionary *capture(ArionTyping *typing, ComparisonEditor *editor) {
    return @{@"preedit": typing.snapshot.preedit ?: @"", @"candidates": typing.snapshot.candidates ?: @[],
        @"page": @(typing.snapshot.page), @"lastPage": @(typing.snapshot.lastPage),
        @"composing": @(typing.composing), @"available": @(typing.available), @"ascii": @(typing.ascii),
        @"revision": @(typing.revision), @"text": editor.view.string.copy,
        @"marked": @(editor.view.hasMarkedText), @"insertions": @(editor.insertions), @"markings": @(editor.markings),
        @"recoveryText": typing.recoveryText ?: @""};
}
static NSDictionary *letter(ArionTyping *typing, ComparisonEditor *editor, unichar ch) {
    NSString *character = [NSString stringWithCharacters:&ch length:1];
    NSEventModifierFlags flags = ch >= 'A' && ch <= 'Z' ? NSEventModifierFlagShift : 0;
    NSEvent *event = [NSEvent keyEventWithType:NSEventTypeKeyDown location:NSZeroPoint modifierFlags:flags
        timestamp:NSProcessInfo.processInfo.systemUptime windowNumber:0 context:nil characters:character
        charactersIgnoringModifiers:character isARepeat:NO keyCode:0];
    uint64_t start = mach_absolute_time();
    BOOL handled = [typing handleEvent:event];
    if (!handled) [editor insertText:character replacementRange:NSMakeRange(NSNotFound, 0)];
    double milliseconds = elapsed(start);
    require(typing.available && typing.recoveryText.length == 0, @"Native input or storage became unavailable");
    return @{@"key": @(ch), @"handled": @(handled), @"nativeStateMs": @(milliseconds)};
}
static NSArray *type(ArionTyping *typing, ComparisonEditor *editor, NSString *input, NSUInteger pace) {
    NSMutableArray *steps = [NSMutableArray array];
    for (NSUInteger i = 0; i < input.length; ++i) {
        if (pace && i) usleep((useconds_t)(pace * 1000));
        NSMutableDictionary *record = [letter(typing, editor, [input characterAtIndex:i]) mutableCopy];
        record[@"state"] = capture(typing, editor);
        [steps addObject:record];
    }
    return steps;
}
int main(int argc, const char *argv[]) {
    @autoreleasepool {
        if (argc != 4) { fprintf(stderr, "usage: driver DATA USER REQUEST_JSON\n"); return 2; }
        mach_timebase_info(&timebase);
        [NSApplication sharedApplication];
        ArionKeyboardEngine *engine = ArionKeyboardEngine.shared;
        ArionTyping *typing = nil;
        int status = 2;
        @try {
            NSError *error = nil;
            NSData *bytes = [NSData dataWithContentsOfFile:@(argv[3])];
            NSDictionary *request = bytes ? [NSJSONSerialization JSONObjectWithData:bytes options:0 error:&error] : nil;
            require([request isKindOfClass:NSDictionary.class] && error == nil, @"Read reference-free request");
            require([[NSSet setWithArray:request.allKeys] isEqual:[NSSet setWithArray:@[@"input", @"warmups", @"pageSize", @"paceMs"]]], @"Reject references and unknown request fields");
            NSString *input = request[@"input"];
            require([input isKindOfClass:NSString.class] && input.length > 0 &&
                [input rangeOfCharacterFromSet:[[NSCharacterSet characterSetWithCharactersInString:@"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ'"] invertedSet]].location == NSNotFound, @"Validate raw ASCII input");
            NSUInteger page = [request[@"pageSize"] unsignedIntegerValue], pace = [request[@"paceMs"] unsignedIntegerValue];
            require((page == 5 || page == 9) && pace <= 1000 && [request[@"warmups"] isKindOfClass:NSArray.class], @"Validate run controls");
            uint64_t startup = mach_absolute_time();
            NSMutableData *key = [NSMutableData dataWithLength:32]; memset(key.mutableBytes, 0x36, key.length);
            NSString *user = @(argv[2]);
            require(ArionRegisterTestEncryptedLearning([user stringByAppendingPathComponent:@"learning"], key, &error), @"Register isolated encrypted storage");
            require([engine startWithData:@(argv[1]) userDirectory:user error:&error] && error == nil, @"Initialize production engine and verify packaged data");
            ComparisonEditor *editor = [ComparisonEditor new];
            typing = [[ArionTyping alloc] initWithEngine:engine];
            typing.pageSize = page; typing.contextRanking = YES; typing.emojiSuggestions = YES;
            require([typing activate:editor error:&error] && error == nil, @"Activate actual AppKit text store");
            double startupMs = elapsed(startup);
            NSMutableArray *warmups = [NSMutableArray array];
            for (NSString *warmup in request[@"warmups"]) {
                [warmups addObject:@{@"input": warmup, @"steps": type(typing, editor, warmup, 0)}];
                [typing cancel]; require(editor.insertions == 0, @"Warmup must not commit or learn");
            }
            uint64_t typingStart = mach_absolute_time();
            NSArray *steps = type(typing, editor, input, pace);
            double typingWallMs = elapsed(typingStart);
            emit(@{@"status": @"ready", @"startupMs": @(startupMs), @"warmups": warmups, @"steps": steps,
                @"typingWallMs": @(typingWallMs), @"state": capture(typing, editor)});
            char *line = NULL; size_t capacity = 0;
            while (getline(&line, &capacity, stdin) > 0) {
                @autoreleasepool {
                    NSData *commandBytes = [[NSString stringWithUTF8String:line] dataUsingEncoding:NSUTF8StringEncoding];
                    NSDictionary *command = [NSJSONSerialization JSONObjectWithData:commandBytes options:0 error:&error];
                    require([command isKindOfClass:NSDictionary.class], @"Read command");
                    NSString *operation = command[@"op"];
                    BOOL handled = NO;
                    uint64_t start = mach_absolute_time();
                    if ([operation isEqual:@"next"]) handled = [typing nextPage:typing.revision];
                    else if ([operation isEqual:@"select"]) handled = [typing select:[command[@"index"] unsignedIntegerValue] revision:typing.revision];
                    else if ([operation isEqual:@"cancel"]) { [typing cancel]; handled = YES; }
                    else if ([operation isEqual:@"profile"]) {
                        NSUInteger count = [command[@"iterations"] unsignedIntegerValue];
                        require(count > 0 && count <= 100000, @"Bound profile workload");
                        for (NSUInteger i = 0; i < count; ++i) { [typing cancel]; type(typing, editor, input, 0); }
                        handled = YES;
                    } else if ([operation isEqual:@"quit"]) break;
                    else require(NO, @"Unknown operation");
                    double milliseconds = elapsed(start);
                    emit(@{@"status": @"step", @"op": operation, @"handled": @(handled), @"nativeStateMs": @(milliseconds), @"state": capture(typing, editor)});
                }
            }
            free(line);
            [typing deactivate]; [engine stop];
            require(ArionEncryptedLearningError() == nil, @"Storage teardown succeeded");
            struct rusage usage; require(getrusage(RUSAGE_SELF, &usage) == 0, @"Read memory usage");
            emit(@{@"status": @"completed", @"peakRSSBytes": @(usage.ru_maxrss)});
            status = 0;
        } @catch (NSException *exception) {
            emit(@{@"status": @"failed", @"failure": exception.reason ?: @"Unknown native exception"});
            [typing deactivate]; [engine stop];
        }
        return status;
    }
}
