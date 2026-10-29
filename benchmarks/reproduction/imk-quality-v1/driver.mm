// Receives only raw input and run controls; evaluator references are never read.
#import <Cocoa/Cocoa.h>
#import "keyboard.h"
#import "learning.h"
#include "../../../apps/desktop/ime/shared/spelling_options.h"
#include "../../../apps/desktop/ime/shared/emoji.h"
#include <mach/mach_time.h>
#include <sys/resource.h>

static mach_timebase_info_data_t timebase;
static double elapsed(uint64_t start) {
    return (double)(mach_absolute_time() - start) * timebase.numer / timebase.denom / 1000000.0;
}
static void require(BOOL condition, NSString *message) {
    if (!condition) @throw [NSException exceptionWithName:@"DriverFailure" reason:message userInfo:nil];
}
static NSDictionary *state(ArionKeyboardSnapshot *snapshot) {
    return @{@"preedit": snapshot.preedit, @"candidates": snapshot.candidates,
        @"highlighted": @(snapshot.highlighted), @"page": @(snapshot.page),
        @"lastPage": @(snapshot.lastPage), @"commit": snapshot.commit ?: NSNull.null,
        @"selection": @[@(snapshot.selection.location), @(snapshot.selection.length)], @"ascii": @(snapshot.ascii)};
}
static NSDictionary *step(ArionKeyboardEngine *engine, uintptr_t session, int key) {
    uint64_t start = mach_absolute_time();
    BOOL handled = [engine processKey:key modifiers:0 session:session];
    NSError *error = nil;
    ArionKeyboardSnapshot *snapshot = [engine snapshot:session error:&error];
    double milliseconds = elapsed(start);
    require(snapshot != nil && error == nil && snapshot.failure == nil, @"Engine snapshot or encrypted storage failed");
    return @{@"key": @(key), @"handled": @(handled), @"keyToSnapshotMs": @(milliseconds), @"snapshot": state(snapshot)};
}

int main(int argc, const char *argv[]) {
    @autoreleasepool {
        if (argc != 5) { fprintf(stderr, "usage: driver DATA USER INPUT_JSON OUTPUT_JSON\n"); return 2; }
        mach_timebase_info(&timebase);
        NSString *data = @(argv[1]), *user = @(argv[2]), *inputFile = @(argv[3]), *output = @(argv[4]);
        ArionKeyboardEngine *engine = ArionKeyboardEngine.shared;
        NSMutableDictionary *result = [@{@"status": @"failed", @"timer": @"mach_absolute_time", @"boundary": @"production engine key-processing plus synchronous snapshot copying; no OS events or rendered preedit"} mutableCopy];
        uintptr_t session = 0;
        int status = 2;
        @try {
            NSError *error = nil;
            NSData *bytes = [NSData dataWithContentsOfFile:inputFile];
            NSDictionary *request = bytes ? [NSJSONSerialization JSONObjectWithData:bytes options:0 error:&error] : nil;
            require([request isKindOfClass:NSDictionary.class] && error == nil, @"Read reference-free request");
            NSMutableArray *fields = [NSMutableArray arrayWithArray:@[@"id", @"input", @"warmups"]];
            if (request[@"spelling"]) [fields addObject:@"spelling"];
            require([[NSSet setWithArray:request.allKeys] isEqual:[NSSet setWithArray:fields]], @"Reject extra driver request fields");
            require([request[@"id"] isKindOfClass:NSString.class] && [request[@"input"] isKindOfClass:NSString.class] && [request[@"warmups"] isKindOfClass:NSArray.class], @"Validate request fields");
            NSString *input = request[@"input"];
            require(input.length > 0 && [input rangeOfCharacterFromSet:[[NSCharacterSet characterSetWithCharactersInString:@"abcdefghijklmnopqrstuvwxyz'"] invertedSet]].location == NSNotFound, @"Input must contain raw ASCII Pinyin only");
            result[@"id"] = request[@"id"];
            result[@"input"] = input;
            uint64_t startup = mach_absolute_time();
            NSMutableData *key = [NSMutableData dataWithLength:32];
            memset(key.mutableBytes, 0x58, key.length);
            require(ArionRegisterTestEncryptedLearning([user stringByAppendingPathComponent:@"learning"], key, &error), @"Register isolated synthetic encrypted backend");
            require([engine startWithData:data userDirectory:user error:&error] && error == nil, @"Initialize actual pinned production engine");
            session = [engine createSession];
            require(session != 0, @"Create engine session");
            if (request[@"spelling"]) {
                NSDictionary *options = request[@"spelling"];
                require([options isKindOfClass:NSDictionary.class] && [[NSSet setWithArray:options.allKeys] isEqual:[NSSet setWithArray:@[@"typo", @"fuzzy"]]], @"Validate spelling controls");
                NSNumber *typo = options[@"typo"], *fuzzy = options[@"fuzzy"];
                require([typo isKindOfClass:NSNumber.class] && CFGetTypeID((__bridge CFTypeRef)typo) == CFBooleanGetTypeID(), @"Typo control must be Boolean");
                require([fuzzy isKindOfClass:NSNumber.class] && CFGetTypeID((__bridge CFTypeRef)fuzzy) != CFBooleanGetTypeID() && fuzzy.doubleValue == fuzzy.unsignedIntValue && fuzzy.unsignedIntValue <= ARION_FUZZY_MASK, @"Validate fuzzy mask");
                arion_spelling_apply(engine.api, session, typo.boolValue, fuzzy.unsignedIntValue);
                require(!!engine.api->get_option(session,"arion_typo") == typo.boolValue, @"Verify actual typo option");
                for (unsigned i=0; i<ARION_FUZZY_COUNT; ++i)
                    require(!!engine.api->get_option(session,arion_fuzzy_options[i]) == !!(fuzzy.unsignedIntValue & (1u<<i)), @"Verify actual fuzzy options");
                result[@"spelling"] = options;
            }
            result[@"startupToSessionMs"] = @(elapsed(startup));
            uint64_t warmupStart = mach_absolute_time();
            NSMutableArray *warmups = [NSMutableArray array];
            for (NSString *warmup in request[@"warmups"]) {
                require([warmup isKindOfClass:NSString.class] && warmup.length > 0, @"Validate warmup input");
                NSMutableArray *steps = [NSMutableArray array];
                for (NSUInteger i = 0; i < warmup.length; i++) [steps addObject:step(engine, session, [warmup characterAtIndex:i])];
                [engine clear:session];
                [warmups addObject:@{@"input": warmup, @"steps": steps, @"completion": @"cancelled-without-commit"}];
            }
            result[@"warmups"] = warmups;
            result[@"warmupMs"] = @(elapsed(warmupStart));
            NSMutableArray *steps = [NSMutableArray array];
            uint64_t typing = mach_absolute_time();
            for (NSUInteger i = 0; i < input.length; i++) [steps addObject:step(engine, session, [input characterAtIndex:i])];
            result[@"inputKeysToFinalSnapshotMs"] = @(elapsed(typing));
            result[@"steps"] = steps;
            NSDictionary *before = steps.lastObject[@"snapshot"];
            result[@"beforeConfirmation"] = before;
            // Inspect only candidates already materialized by the final snapshot.
            // Keep this observation outside the key-processing timer.
            uint64_t rangeStart = mach_absolute_time();
            require([before[@"page"] unsignedIntegerValue] == 0, @"Final typing snapshot must be the first page");
            NSMutableArray *ranges = [NSMutableArray array];
            NSArray<NSString *> *page = before[@"candidates"];
            for (NSUInteger index = 0; index < page.count; ++index) {
                ArionCandidateMetadata metadata{};
                require(arion_candidate_metadata(session,index,&metadata), @"Read first-page consumed-input ranges");
                [ranges addObject:@{@"index":@(index), @"text":page[index], @"start":@(metadata.start), @"end":@(metadata.end), @"type":@(metadata.type)}];
            }
            result[@"candidateRangeVersion"] = @1;
            result[@"candidateRanges"] = ranges;
            result[@"candidateRangeInspectionMs"] = @(elapsed(rangeStart));
            NSMutableArray *confirmations = [NSMutableArray array];
            NSMutableString *committed = [NSMutableString string];
            for (NSDictionary *record in steps) {
                id text = record[@"snapshot"][@"commit"];
                if ([text isKindOfClass:NSString.class]) [committed appendString:text];
            }
            NSString *terminal = @"confirmation-step-limit";
            for (NSUInteger i = 0; i < 16; i++) {
                NSDictionary *record = step(engine, session, ' ');
                [confirmations addObject:record];
                NSDictionary *snapshot = record[@"snapshot"];
                id text = snapshot[@"commit"];
                if ([text isKindOfClass:NSString.class]) [committed appendString:text];
                if ([snapshot[@"preedit"] length] == 0) { terminal = @"empty-preedit"; break; }
                if ([before[@"preedit"] isEqual:snapshot[@"preedit"]] && [before[@"candidates"] isEqual:snapshot[@"candidates"]]) { terminal = @"confirmation-stalled"; break; }
                before = snapshot;
            }
            result[@"confirmations"] = confirmations;
            result[@"committed"] = committed;
            result[@"terminal"] = terminal;
            uint64_t teardown = mach_absolute_time();
            [engine destroySession:session]; session = 0;
            [engine stop];
            require(ArionEncryptedLearningError() == nil, @"Encrypted learning teardown must succeed");
            result[@"teardownMs"] = @(elapsed(teardown));
            struct rusage usage;
            require(getrusage(RUSAGE_SELF, &usage) == 0, @"Read process RSS");
            result[@"peakRSSBytes"] = @(usage.ru_maxrss);
            result[@"status"] = @"completed";
            status = 0;
        } @catch (NSException *exception) {
            result[@"failure"] = exception.reason;
            if (session) [engine destroySession:session];
            [engine stop];
        }
        NSError *error = nil;
        NSData *encoded = [NSJSONSerialization dataWithJSONObject:result options:NSJSONWritingPrettyPrinted error:&error];
        if (!encoded || ![encoded writeToFile:output options:0 error:&error]) { fprintf(stderr, "Cannot preserve driver result\n"); return 2; }
        return status;
    }
}
