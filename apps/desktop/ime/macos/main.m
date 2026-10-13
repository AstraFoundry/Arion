// ArionInput is Arion's InputMethodKit server. It owns no recognition: it
// shows the voice draft Arion streams as marked text in the focused client,
// commits it through IMK when Arion finishes, and passes every key through.
// It talks to the running Arion app over a per-user local socket.
#import <Cocoa/Cocoa.h>
#import <InputMethodKit/InputMethodKit.h>
#include <sys/socket.h>
#include <sys/un.h>

@class ArionInputController;

// Bridge keeps one connection to Arion and the controller that is active.
@interface ArionBridge : NSObject
@property(weak) ArionInputController *active;
@property(assign) int fd;
+ (instancetype)shared;
- (void)send:(NSDictionary *)message;
@end

@interface ArionInputController : IMKInputController
@property(assign) BOOL composing;
@property(copy) NSString *session;
- (void)apply:(NSDictionary *)message;
@end

static NSString *socketPath(void) {
    NSString *base = NSSearchPathForDirectoriesInDomains(NSApplicationSupportDirectory, NSUserDomainMask, YES).firstObject;
    return [base stringByAppendingPathComponent:@"Arion/ime.sock"];
}

@implementation ArionBridge
+ (instancetype)shared {
    static ArionBridge *bridge;
    static dispatch_once_t once;
    dispatch_once(&once, ^{
        bridge = [ArionBridge new];
        bridge.fd = -1;
        // Arion may start after this source was selected: reconnect and
        // announce the active client until the app answers.
        [NSTimer scheduledTimerWithTimeInterval:2 repeats:YES block:^(NSTimer *timer) {
            if (bridge.fd < 0 && bridge.active) {
                [bridge send:@{@"op": @"activate", @"client": @""}];
            }
        }];
    });
    return bridge;
}

- (BOOL)connect {
    if (self.fd >= 0) return YES;
    int fd = socket(AF_UNIX, SOCK_STREAM, 0);
    struct sockaddr_un address = {0};
    address.sun_family = AF_UNIX;
    strlcpy(address.sun_path, socketPath().fileSystemRepresentation, sizeof(address.sun_path));
    if (connect(fd, (struct sockaddr *)&address, sizeof(address)) != 0) {
        close(fd);
        return NO;
    }
    self.fd = fd;
    dispatch_async(dispatch_get_global_queue(QOS_CLASS_USER_INITIATED, 0), ^{ [self readLoop:fd]; });
    return YES;
}

- (void)readLoop:(int)fd {
    FILE *stream = fdopen(dup(fd), "r");
    char *line = NULL;
    size_t capacity = 0;
    while (getline(&line, &capacity, stream) > 0) {
        NSData *data = [NSData dataWithBytes:line length:strlen(line)];
        NSDictionary *message = [NSJSONSerialization JSONObjectWithData:data options:0 error:nil];
        if (![message isKindOfClass:NSDictionary.class]) continue;
        dispatch_async(dispatch_get_main_queue(), ^{ [self.active apply:message]; });
    }
    free(line);
    fclose(stream);
    dispatch_async(dispatch_get_main_queue(), ^{
        if (self.fd == fd) { close(fd); self.fd = -1; }
        [self.active apply:@{@"op": @"cancel"}];
    });
}

- (void)send:(NSDictionary *)message {
    if (![self connect]) return;
    NSMutableData *data = [[NSJSONSerialization dataWithJSONObject:message options:0 error:nil] mutableCopy];
    [data appendBytes:"\n" length:1];
    if (write(self.fd, data.bytes, data.length) < 0) { close(self.fd); self.fd = -1; }
}
@end

@implementation ArionInputController
- (void)activateServer:(id)sender {
    [super activateServer:sender];
    [ArionBridge shared].active = self;
    NSString *bundle = [sender respondsToSelector:@selector(bundleIdentifier)] ? [sender bundleIdentifier] : @"";
    [[ArionBridge shared] send:@{@"op": @"activate", @"client": bundle ?: @""}];
}

- (void)deactivateServer:(id)sender {
    // IME-03: deactivation clears owned composition; Arion invalidates the
    // session so late results never reach another client.
    [self clearMarked:sender];
    [[ArionBridge shared] send:@{@"op": @"deactivate"}];
    if ([ArionBridge shared].active == self) [ArionBridge shared].active = nil;
    [super deactivateServer:sender];
}

- (BOOL)handleEvent:(NSEvent *)event client:(id)sender {
    if (self.composing && event.type == NSEventTypeKeyDown && event.keyCode == 53) { // Escape
        [self clearMarked:sender];
        [[ArionBridge shared] send:@{@"op": @"escape"}];
        return YES;
    }
    // Typing passes through untouched (IME-04).
    return NO;
}

- (void)commitComposition:(id)sender {
    // The host asks to end composition (focus change, click elsewhere): an
    // unconfirmed voice draft is discarded rather than committed.
    if (self.composing) {
        [self clearMarked:sender];
        [[ArionBridge shared] send:@{@"op": @"deactivate"}];
    }
}

- (void)clearMarked:(id)sender {
    if (!self.composing) return;
    [sender setMarkedText:@"" selectionRange:NSMakeRange(0, 0) replacementRange:NSMakeRange(NSNotFound, NSNotFound)];
    self.composing = NO;
}

- (void)apply:(NSDictionary *)message {
    id client = self.client;
    NSString *op = message[@"op"];
    if (!client) return;
    if ([op isEqual:@"mark"]) {
        NSString *text = message[@"text"] ?: @"";
        NSDictionary *attributes = [self markForStyle:kTSMHiliteSelectedRawText atRange:NSMakeRange(0, text.length)];
        NSAttributedString *marked = [[NSAttributedString alloc] initWithString:text attributes:attributes];
        [client setMarkedText:marked selectionRange:NSMakeRange(text.length, 0) replacementRange:NSMakeRange(NSNotFound, NSNotFound)];
        self.composing = text.length > 0;
        self.session = message[@"session"];
    } else if ([op isEqual:@"commit"]) {
        NSString *text = message[@"text"] ?: @"";
        [client insertText:text replacementRange:NSMakeRange(NSNotFound, NSNotFound)];
        self.composing = NO;
        [[ArionBridge shared] send:@{@"op": @"committed", @"session": message[@"session"] ?: @""}];
    } else if ([op isEqual:@"cancel"]) {
        [self clearMarked:client];
    }
}
@end

int main(int argc, const char *argv[]) {
    @autoreleasepool {
        NSString *connection = [NSBundle.mainBundle objectForInfoDictionaryKey:@"InputMethodConnectionName"];
        IMKServer *server = [[IMKServer alloc] initWithName:connection bundleIdentifier:NSBundle.mainBundle.bundleIdentifier];
        (void)server;
        [NSApplication sharedApplication];
        [NSApp run];
    }
    return 0;
}
