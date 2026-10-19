// The InputMethodKit host owns client ranges; recognition stays in Arion.
#import <Cocoa/Cocoa.h>
#import <InputMethodKit/InputMethodKit.h>
#import "composition.h"
#import "typing.h"
#include <sys/socket.h>
#include <sys/un.h>
#include <errno.h>
#include <unistd.h>

@class ArionInputController;

@interface ArionBridge : NSObject
@property(weak) ArionInputController *active;
@property int fd;
@property(copy) NSString *connection;
+ (instancetype)shared;
- (void)send:(NSDictionary *)message;
@end

@interface ArionInputController : IMKInputController
@property(strong) ArionComposition *composition;
@property(copy) NSString *clientBundle;
@property(strong) ArionTyping *typing;
@property(strong) IMKCandidates *candidateWindow;
@property BOOL keyboardReady;
@property BOOL updatingCandidates;
- (NSDictionary *)activationMessage;
- (void)apply:(NSDictionary *)message;
- (void)interrupt;
@end

static NSString *applicationSupport(void) {
    NSString *profile = NSProcessInfo.processInfo.environment[@"ARION_DATA_DIR"];
    if (profile) return profile.isAbsolutePath ? profile.stringByStandardizingPath : nil;
    NSString *base = NSSearchPathForDirectoriesInDomains(NSApplicationSupportDirectory, NSUserDomainMask, YES).firstObject;
    return [base stringByAppendingPathComponent:@"Arion"];
}

static NSString *socketPath(void) {
    return [applicationSupport() stringByAppendingPathComponent:@"ime.sock"];
}

@implementation ArionBridge
+ (instancetype)shared {
    static ArionBridge *bridge;
    static dispatch_once_t once;
    dispatch_once(&once, ^{
        bridge = [ArionBridge new];
        bridge.fd = -1;
        [NSTimer scheduledTimerWithTimeInterval:2 repeats:YES block:^(NSTimer *timer) {
            if (bridge.fd < 0 && bridge.active) [bridge send:bridge.active.activationMessage];
        }];
    });
    return bridge;
}

- (BOOL)connect {
    if (self.fd >= 0) return YES;
    int fd = socket(AF_UNIX, SOCK_STREAM, 0);
    if (fd < 0) return NO;
    struct sockaddr_un address = {0};
    address.sun_family = AF_UNIX;
    NSString *path = socketPath();
    if (path.length == 0 || strlen(path.fileSystemRepresentation) >= sizeof(address.sun_path)) { close(fd); return NO; }
    strlcpy(address.sun_path, path.fileSystemRepresentation, sizeof(address.sun_path));
    int yes = 1;
    setsockopt(fd, SOL_SOCKET, SO_NOSIGPIPE, &yes, sizeof(yes));
    struct timeval timeout = {.tv_sec = 0, .tv_usec = 200000};
    setsockopt(fd, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout));
    if (connect(fd, (struct sockaddr *)&address, sizeof(address)) != 0) { close(fd); return NO; }
    int reader = dup(fd);
    if (reader < 0) { close(fd); return NO; }
    self.fd = fd;
    self.connection = NSUUID.UUID.UUIDString;
    NSString *connection = self.connection;
    dispatch_async(dispatch_get_global_queue(QOS_CLASS_USER_INITIATED, 0), ^{
        [self readLoop:reader connection:connection];
    });
    return YES;
}

- (void)disconnect {
    if (self.fd >= 0) { shutdown(self.fd, SHUT_RDWR); close(self.fd); }
    self.fd = -1;
    self.connection = nil;
    [self.active.composition invalidate];
}

- (void)readLoop:(int)fd connection:(NSString *)connection {
    NSMutableData *pending = [NSMutableData data];
    unsigned char bytes[4096];
    BOOL valid = YES;
    while (valid) {
        ssize_t count = read(fd, bytes, sizeof(bytes));
        if (count < 0 && errno == EINTR) continue;
        if (count <= 0) break;
        for (ssize_t i = 0; i < count; i++) {
            if (bytes[i] != '\n') {
                if (pending.length >= 1024 * 1024) { valid = NO; break; }
                [pending appendBytes:bytes + i length:1];
                continue;
            }
            id message = [NSJSONSerialization JSONObjectWithData:pending options:0 error:nil];
            [pending setLength:0];
            if (![message isKindOfClass:NSDictionary.class]) { valid = NO; break; }
            dispatch_async(dispatch_get_main_queue(), ^{
                if ([self.connection isEqual:connection]) [self.active apply:message];
            });
        }
    }
    close(fd);
    dispatch_async(dispatch_get_main_queue(), ^{
        // Descriptor reuse must never let an old reader close a new connection.
        if ([self.connection isEqual:connection]) [self disconnect];
    });
}

- (void)send:(NSDictionary *)message {
    if (!message) return;
    BOOL reconnecting = self.fd < 0;
    if (![self connect]) return;
    if (reconnecting && ![message[@"op"] isEqual:@"activate"]) {
        NSDictionary *activation = self.active.activationMessage;
        if (activation) [self writeMessage:activation];
    }
    if (self.fd >= 0) [self writeMessage:message];
}

- (void)writeMessage:(NSDictionary *)message {
    NSMutableData *data = [[NSJSONSerialization dataWithJSONObject:message options:0 error:nil] mutableCopy];
    if (!data) return;
    [data appendBytes:"\n" length:1];
    size_t offset = 0;
    while (offset < data.length) {
        ssize_t count = write(self.fd, (const char *)data.bytes + offset, data.length - offset);
        if (count < 0 && errno == EINTR) continue;
        if (count <= 0) { [self disconnect]; return; }
        offset += (size_t)count;
    }
}
@end

@implementation ArionInputController
- (NSDictionary *)activationMessage {
    if (!self.composition.activation) return nil;
    return @{@"op": @"activate", @"activation": self.composition.activation, @"client": self.clientBundle ?: @""};
}

- (void)activateServer:(id)sender {
    [super activateServer:sender];
    if (!self.composition) self.composition = [ArionComposition new];
    [self.composition activate:sender];
    if (!self.typing) {
        self.typing = [[ArionTyping alloc] initWithEngine:ArionKeyboardEngine.shared];
        __weak ArionInputController *weakSelf = self;
        self.typing.changed = ^{ [weakSelf refreshCandidates]; };
    }
    if (!self.candidateWindow) {
        self.candidateWindow = [[IMKCandidates alloc] initWithServer:self.server panelType:kIMKSingleColumnScrollingCandidatePanel];
        [self.candidateWindow setAttributes:@{IMKCandidatesSendServerKeyEventFirst: @YES}];
        [self.candidateWindow setDismissesAutomatically:NO];
    }
    self.keyboardReady = [self.typing activate:sender error:nil];
    self.clientBundle = [sender respondsToSelector:@selector(bundleIdentifier)] ? [sender bundleIdentifier] : @"";
    [ArionBridge shared].active = self;
    [[ArionBridge shared] send:self.activationMessage];
}

- (void)deactivateServer:(id)sender {
    [self.typing deactivate];
    [self.candidateWindow hide];
    [self.composition invalidate];
    if ([ArionBridge shared].active == self) {
        [[ArionBridge shared] send:@{@"op": @"deactivate", @"activation": self.composition.activation}];
        [ArionBridge shared].active = nil;
    }
    [super deactivateServer:sender];
}

- (void)interrupt {
    if (!self.composition.live) return;
    [self.composition invalidate];
    [[ArionBridge shared] send:@{@"op": @"escape", @"activation": self.composition.activation, @"session": self.composition.session}];
}

- (BOOL)handleEvent:(NSEvent *)event client:(id)sender {
    if (event.type != NSEventTypeKeyDown) return NO;
    BOOL hadVoice = self.composition.live;
    [self interrupt];
    if (hadVoice && event.keyCode == 53) return YES;
    if (!self.keyboardReady || !self.typing.available) {
        if (event.modifierFlags & (NSEventModifierFlagCommand | NSEventModifierFlagControl | NSEventModifierFlagOption)) return NO;
        NSBeep();
        return YES;
    }
    return [self.typing handleEvent:event];
}

- (void)commitComposition:(id)sender {
    // A host-requested end never implicitly commits an unconfirmed voice draft.
    [self interrupt];
    [self.typing cancel];
}

- (void)apply:(NSDictionary *)message {
    if ([message[@"op"] isEqual:@"begin"] &&
        [message[@"activation"] isEqual:self.composition.activation] &&
        [message[@"session"] isKindOfClass:NSString.class] && [message[@"session"] length] > 0 &&
        ![message[@"session"] isEqual:self.composition.session]) {
        [self.typing cancel];
    }
    NSDictionary *ack = [self.composition apply:message];
    if (ack) [[ArionBridge shared] send:ack];
}
- (void)refreshCandidates {
    if (self.updatingCandidates) return;
    self.updatingCandidates = YES;
    ArionKeyboardSnapshot *snapshot = self.typing.snapshot;
    NSMutableArray *values = [NSMutableArray array];
    for (NSUInteger index = 0; index < snapshot.candidates.count; index++) {
        [values addObject:[[NSAttributedString alloc] initWithString:snapshot.candidates[index]
            attributes:@{@"ArionIndex": @(index), @"ArionRevision": @(self.typing.revision)}]];
    }
    [self.candidateWindow setCandidateData:values];
    if (values.count > 0 && self.typing.composing) {
        NSInteger identifier = [self.candidateWindow candidateStringIdentifier:values[MIN(snapshot.highlighted, values.count - 1)]];
        [self.candidateWindow selectCandidateWithIdentifier:identifier];
        [self.candidateWindow show:kIMKLocateCandidatesBelowHint];
    } else {
        [self.candidateWindow hide];
    }
    self.updatingCandidates = NO;
}

- (void)candidateSelected:(NSAttributedString *)candidate {
    if (self.updatingCandidates || [ArionBridge shared].active != self || candidate.length == 0) return;
    NSNumber *index = [candidate attribute:@"ArionIndex" atIndex:0 effectiveRange:NULL];
    NSNumber *revision = [candidate attribute:@"ArionRevision" atIndex:0 effectiveRange:NULL];
    if ([index isKindOfClass:NSNumber.class] && [revision isKindOfClass:NSNumber.class])
        [self.typing select:index.unsignedIntegerValue revision:revision.unsignedIntegerValue];
}

- (void)candidateSelectionChanged:(NSAttributedString *)candidate {
    if (self.updatingCandidates || [ArionBridge shared].active != self || candidate.length == 0) return;
    NSNumber *index = [candidate attribute:@"ArionIndex" atIndex:0 effectiveRange:NULL];
    NSNumber *revision = [candidate attribute:@"ArionRevision" atIndex:0 effectiveRange:NULL];
    if ([index isKindOfClass:NSNumber.class] && [revision isKindOfClass:NSNumber.class])
        [self.typing highlight:index.unsignedIntegerValue revision:revision.unsignedIntegerValue];
}

- (NSMenu *)menu {
    NSMenu *menu = [[NSMenu alloc] initWithTitle:@""];
    menu.autoenablesItems = NO;
    NSString *key = (!self.keyboardReady || !self.typing.available) ? @"KeyboardUnavailable" :
        (self.typing.ascii ? @"KeyboardSwitchChinese" : @"KeyboardSwitchEnglish");
    NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:NSLocalizedString(key, nil)
        action:@selector(toggleLanguage:) keyEquivalent:@""];
    item.target = self;
    item.enabled = self.keyboardReady && self.typing.available;
    [menu addItem:item];
    NSMenuItem *reset = [[NSMenuItem alloc] initWithTitle:NSLocalizedString(@"KeyboardResetLearning", nil)
        action:@selector(resetLearning:) keyEquivalent:@""];
    reset.target = self;
    [menu addItem:reset];
    if (self.typing.recoveryText.length > 0) {
        NSMenuItem *copy = [[NSMenuItem alloc] initWithTitle:NSLocalizedString(@"KeyboardCopyPending", nil)
            action:@selector(copyPending:) keyEquivalent:@""];
        copy.target = self;
        [menu addItem:copy];
    }
    return menu;
}

- (void)resetLearning:(id)sender {
    [self interrupt];
    [self.typing deactivate];
    // Reset refuses open dictionaries. Finalize before deleting learned data.
    BOOL ready = [ArionKeyboardEngine.shared resetLearning:nil];
    self.keyboardReady = ready && [self.typing activate:self.client error:nil];
    if (!self.keyboardReady) NSBeep();
}

- (void)copyPending:(id)sender {
    if (self.typing.recoveryText.length == 0) return;
    [NSPasteboard.generalPasteboard clearContents];
    [NSPasteboard.generalPasteboard setString:self.typing.recoveryText forType:NSPasteboardTypeString];
}

- (void)toggleLanguage:(id)sender {
    [self interrupt];
    [self.typing toggleASCII];
}

@end

int main(int argc, const char *argv[]) {
    @autoreleasepool {
        [NSApplication sharedApplication];
        NSString *data = [NSBundle.mainBundle.resourcePath stringByAppendingPathComponent:@"keyboard/data"];
        NSString *profile = applicationSupport();
        if (profile) [ArionKeyboardEngine.shared startWithData:data
            userDirectory:[profile stringByAppendingPathComponent:@"keyboard"] error:nil];
        [NSNotificationCenter.defaultCenter addObserverForName:NSApplicationWillTerminateNotification object:nil queue:nil
            usingBlock:^(NSNotification *notification) { [ArionKeyboardEngine.shared stop]; }];
        NSString *connection = [NSBundle.mainBundle objectForInfoDictionaryKey:@"InputMethodConnectionName"];
        __attribute__((objc_precise_lifetime)) IMKServer *server = [[IMKServer alloc] initWithName:connection bundleIdentifier:NSBundle.mainBundle.bundleIdentifier];
        if (!server) return 1;
        [NSApp run];
        (void)server;
    }
    return 0;
}
