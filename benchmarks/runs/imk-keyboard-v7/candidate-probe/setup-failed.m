#import <Cocoa/Cocoa.h>
#import <InputMethodKit/InputMethodKit.h>

int main(int argc, const char *argv[]) {
    @autoreleasepool {
        if (argc != 2) return 2;
        NSMutableDictionary *report = [@{@"boundary": @"isolated real IMKServer/IMKCandidates without active IMK client, TIS changes, Keychain, key synthesis or candidate-window presentation", @"passed": @NO} mutableCopy];
        NSString *stage = @"application";
        @try {
            [NSApplication sharedApplication];
            [NSApp setActivationPolicy:NSApplicationActivationPolicyProhibited];
            stage = @"server";
            NSString *name = [@"ArionOwnedCandidatesProbe-" stringByAppendingString:NSUUID.UUID.UUIDString];
            IMKServer *server = [[IMKServer alloc] initWithName:name controllerClass:IMKInputController.class delegateClass:nil];
            report[@"serverCreated"] = @(server != nil);
            if (!server) @throw [NSException exceptionWithName:@"ProbeBlocked" reason:@"IMKServer initializer returned nil" userInfo:nil];
            stage = @"candidates";
            IMKCandidates *candidates = [[IMKCandidates alloc] initWithServer:server panelType:kIMKSingleColumnScrollingCandidatePanel];
            report[@"candidatesCreated"] = @(candidates != nil);
            if (!candidates) @throw [NSException exceptionWithName:@"ProbeBlocked" reason:@"IMKCandidates initializer returned nil" userInfo:nil];
            NSArray *values = @[
                [[NSAttributedString alloc] initWithString:@"相同" attributes:@{@"ArionIndex": @0, @"ArionRevision": @7}],
                [[NSAttributedString alloc] initWithString:@"相同" attributes:@{@"ArionIndex": @1, @"ArionRevision": @7}],
                [[NSAttributedString alloc] initWithString:@"不同" attributes:@{@"ArionIndex": @2, @"ArionRevision": @7}]];
            stage = @"setCandidateData";
            [candidates setCandidateData:values];
            report[@"dataSet"] = @YES;
            NSMutableArray *observations = [NSMutableArray array]; report[@"observations"] = observations;
            BOOL valid = YES;
            for (NSUInteger i = 0; i < values.count; i++) {
                stage = @"candidateStringIdentifier";
                NSInteger identifier = [candidates candidateStringIdentifier:values[i]];
                stage = @"selectCandidateWithIdentifier";
                BOOL selected = [candidates selectCandidateWithIdentifier:identifier];
                stage = @"selectedCandidateString";
                NSAttributedString *value = [candidates selectedCandidateString];
                NSDictionary *attributes = value.length ? [value attributesAtIndex:0 effectiveRange:NULL] : @{};
                BOOL preserved = selected && [value.string isEqual:[(NSAttributedString *)values[i] string]] &&
                    [attributes[@"ArionIndex"] isEqual:@(i)] && [attributes[@"ArionRevision"] isEqual:@7];
                [observations addObject:@{@"index": @(i), @"identifier": @(identifier), @"selectionSucceeded": @(selected),
                    @"selectedIdentifier": @([candidates selectedCandidate]), @"selectedText": value.string ?: NSNull.null,
                    @"indexAttribute": attributes[@"ArionIndex"] ?: NSNull.null,
                    @"revisionAttribute": attributes[@"ArionRevision"] ?: NSNull.null,
                    @"preserved": @(preserved)}];
                valid = valid && preserved;
            }
            report[@"duplicateTextIdentifiersDistinct"] = @(![observations[0][@"identifier"] isEqual:observations[1][@"identifier"]]);
            valid = valid && [report[@"duplicateTextIdentifiersDistinct"] boolValue];
            report[@"passed"] = @(valid);
            [candidates hide];
        } @catch (NSException *exception) {
            report[@"exception"] = exception.name; report[@"reason"] = exception.reason ?: @"";
        }
        report[@"lastStage"] = stage;
        NSData *output = [NSJSONSerialization dataWithJSONObject:report options:NSJSONWritingPrettyPrinted error:nil];
        if (![output writeToFile:@(argv[1]) atomically:YES]) return 3;
        return [report[@"passed"] boolValue] ? 0 : 1;
    }
}
