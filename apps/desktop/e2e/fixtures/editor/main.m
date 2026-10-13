// ArionEditorFixture is an owned, isolated editor for product E2E: two text
// views and a secure field whose real contents and edit counts it reports
// over a local socket. It never stores or transmits text elsewhere.
#import <Cocoa/Cocoa.h>
#include <sys/socket.h>
#include <sys/un.h>

@interface Fixture : NSObject <NSApplicationDelegate, NSTextViewDelegate>
@property(strong) NSWindow *window;
@property(strong) NSTextView *editorA;
@property(strong) NSTextView *editorB;
@property(strong) NSSecureTextField *secure;
@property(assign) NSInteger editsA, editsB;
@end

static NSTextView *makeEditor(NSRect frame, NSString *name) {
    NSTextView *view = [[NSTextView alloc] initWithFrame:frame];
    view.richText = NO;
    view.automaticQuoteSubstitutionEnabled = NO;
    view.automaticDashSubstitutionEnabled = NO;
    view.automaticTextReplacementEnabled = NO;
    view.automaticSpellingCorrectionEnabled = NO;
    view.font = [NSFont systemFontOfSize:14];
    view.accessibilityLabel = name;
    return view;
}

@implementation Fixture
- (void)applicationDidFinishLaunching:(NSNotification *)note {
    self.window = [[NSWindow alloc] initWithContentRect:NSMakeRect(200, 200, 520, 360)
                                              styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable
                                                backing:NSBackingStoreBuffered defer:NO];
    self.window.title = @"Arion E2E Editor";
    NSView *content = self.window.contentView;
    self.editorA = makeEditor(NSMakeRect(20, 200, 480, 140), @"Editor A");
    self.editorB = makeEditor(NSMakeRect(20, 50, 480, 140), @"Editor B");
    self.editorA.delegate = self;
    self.editorB.delegate = self;
    self.secure = [[NSSecureTextField alloc] initWithFrame:NSMakeRect(20, 15, 480, 24)];
    [content addSubview:self.editorA];
    [content addSubview:self.editorB];
    [content addSubview:self.secure];
    [self.window makeKeyAndOrderFront:nil];
    [NSApp activateIgnoringOtherApps:YES];
    NSString *socketPath = NSProcessInfo.processInfo.environment[@"ARION_FIXTURE_SOCKET"];
    if (socketPath.length) [self serve:socketPath];
}

- (void)textDidChange:(NSNotification *)note {
    if (note.object == self.editorA) self.editsA++;
    if (note.object == self.editorB) self.editsB++;
}

- (NSDictionary *)state {
    NSRange a = self.editorA.selectedRange, b = self.editorB.selectedRange;
    NSResponder *first = self.window.firstResponder;
    NSString *focused = first == self.editorA ? @"A" : first == self.editorB ? @"B" : (first == self.secure || first == self.secure.currentEditor) ? @"secure" : @"";
    return @{
        @"A": @{@"text": self.editorA.string, @"edits": @(self.editsA), @"selection": @[@(a.location), @(a.length)]},
        @"B": @{@"text": self.editorB.string, @"edits": @(self.editsB), @"selection": @[@(b.location), @(b.length)]},
        @"secure": @{@"length": @(self.secure.stringValue.length)},
        @"focused": focused,
        @"active": @(NSApp.isActive),
        @"pid": @(NSProcessInfo.processInfo.processIdentifier),
    };
}

- (NSDictionary *)handle:(NSDictionary *)command {
    NSString *op = command[@"op"];
    if ([op isEqual:@"focus"]) {
        NSString *target = command[@"target"];
        [NSApp activateIgnoringOtherApps:YES];
        [self.window makeKeyAndOrderFront:nil];
        NSResponder *responder = [target isEqual:@"A"] ? self.editorA : [target isEqual:@"B"] ? self.editorB : self.secure;
        [self.window makeFirstResponder:responder];
    } else if ([op isEqual:@"reset"]) {
        self.editorA.string = @"";
        self.editorB.string = @"";
        self.secure.stringValue = @"";
        self.editsA = self.editsB = 0;
    } else if ([op isEqual:@"quit"]) {
        dispatch_async(dispatch_get_main_queue(), ^{ [NSApp terminate:nil]; });
    }
    return [self state];
}

- (void)serve:(NSString *)path {
    unlink(path.fileSystemRepresentation);
    int server = socket(AF_UNIX, SOCK_STREAM, 0);
    struct sockaddr_un address = {0};
    address.sun_family = AF_UNIX;
    strlcpy(address.sun_path, path.fileSystemRepresentation, sizeof(address.sun_path));
    if (bind(server, (struct sockaddr *)&address, sizeof(address)) != 0 || listen(server, 4) != 0) {
        NSLog(@"fixture socket unavailable");
        return;
    }
    dispatch_async(dispatch_get_global_queue(QOS_CLASS_USER_INITIATED, 0), ^{
        for (;;) {
            int client = accept(server, NULL, NULL);
            if (client < 0) continue;
            FILE *stream = fdopen(client, "r+");
            char *line = NULL;
            size_t capacity = 0;
            while (getline(&line, &capacity, stream) > 0) {
                NSData *data = [NSData dataWithBytes:line length:strlen(line)];
                NSDictionary *command = [NSJSONSerialization JSONObjectWithData:data options:0 error:nil];
                __block NSDictionary *reply = nil;
                dispatch_sync(dispatch_get_main_queue(), ^{ reply = [self handle:command ?: @{}]; });
                NSData *out = [NSJSONSerialization dataWithJSONObject:reply options:0 error:nil];
                fwrite(out.bytes, 1, out.length, stream);
                fputc('\n', stream);
                fflush(stream);
            }
            free(line);
            fclose(stream);
        }
    });
}
@end

int main(void) {
    @autoreleasepool {
        NSApplication *app = [NSApplication sharedApplication];
        app.activationPolicy = NSApplicationActivationPolicyRegular;
        Fixture *fixture = [Fixture new];
        app.delegate = fixture;
        [app run];
    }
    return 0;
}
