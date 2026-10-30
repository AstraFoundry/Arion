// ArionEditorFixture is an owned, isolated editor for product E2E: two text
// views and a secure field whose real contents and edit counts it reports
// over a local socket. It never stores or transmits text elsewhere.
#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>
#include <sys/socket.h>
#include <sys/un.h>

@interface Fixture : NSObject <NSApplicationDelegate, NSTextViewDelegate, NSWindowDelegate, WKScriptMessageHandler>
@property(strong) NSWindow *window;
@property(strong) NSTextView *editorA;
@property(strong) NSTextView *editorB;
@property(strong) NSSecureTextField *secure;
@property(assign) NSInteger editsA, editsB;
@property(assign) BOOL transitioning;
@property(strong) WKWebView *web;
@property(copy) NSString *webText;
@property(assign) NSInteger webEdits;
@property(assign) BOOL webFocused;
@property(strong) NSArray<NSPasteboardItem *> *savedClipboard;
@property(assign) NSInteger clipboardChange;

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
    // AppKit dispatches Command-V through the application's Edit menu.
    NSMenu *menu = [NSMenu new];
    NSMenuItem *edit = [[NSMenuItem alloc] initWithTitle:@"Edit" action:nil keyEquivalent:@""];
    NSMenu *commands = [[NSMenu alloc] initWithTitle:@"Edit"];
    [commands addItemWithTitle:@"Paste" action:@selector(paste:) keyEquivalent:@"v"];
    edit.submenu = commands;
    [menu addItem:edit];
    NSApp.mainMenu = menu;
    self.window = [[NSWindow alloc] initWithContentRect:NSMakeRect(200, 200, 520, 360)
                                              styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable | NSWindowStyleMaskResizable
                                                backing:NSBackingStoreBuffered defer:NO];
    self.window.title = @"Arion E2E Editor";
    self.window.delegate = self;
    NSView *content = self.window.contentView;
    self.editorA = makeEditor(NSMakeRect(20, 200, 480, 140), @"Editor A");
    self.editorB = makeEditor(NSMakeRect(20, 50, 480, 140), @"Editor B");
    self.editorA.delegate = self;
    self.editorB.delegate = self;
    self.secure = [[NSSecureTextField alloc] initWithFrame:NSMakeRect(20, 15, 480, 24)];
    [content addSubview:self.editorA];
    [content addSubview:self.editorB];
    [content addSubview:self.secure];
    WKWebViewConfiguration *config = [WKWebViewConfiguration new];
    [config.userContentController addScriptMessageHandler:self name:@"state"];
    self.web = [[WKWebView alloc] initWithFrame:content.bounds configuration:config];
    self.web.hidden = YES;
    self.webText = @"";
    [content addSubview:self.web];
    [self.web loadHTMLString:@"<textarea id='t' style='width:95%;height:90%'></textarea><script>let n=0;function report(){webkit.messageHandlers.state.postMessage({text:t.value,edits:n,focused:document.activeElement===t})}t.oninput=()=>{n++;report()};t.onfocus=report;t.onblur=report;</script>" baseURL:nil];
    [self.window makeKeyAndOrderFront:nil];
    [NSApp activateIgnoringOtherApps:YES];
    NSString *socketPath = NSProcessInfo.processInfo.environment[@"ARION_FIXTURE_SOCKET"];
    if (socketPath.length) [self serve:socketPath];
}

- (void)userContentController:(WKUserContentController *)controller didReceiveScriptMessage:(WKScriptMessage *)message {
    self.webText = message.body[@"text"];
    self.webEdits = [message.body[@"edits"] integerValue];
    self.webFocused = [message.body[@"focused"] boolValue];
}

- (void)textDidChange:(NSNotification *)note {
    if (note.object == self.editorA) self.editsA++;
    if (note.object == self.editorB) self.editsB++;
}

- (void)windowWillEnterFullScreen:(NSNotification *)note { self.transitioning = YES; }
- (void)windowDidEnterFullScreen:(NSNotification *)note { self.transitioning = NO; }
- (void)windowWillExitFullScreen:(NSNotification *)note { self.transitioning = YES; }
- (void)windowDidExitFullScreen:(NSNotification *)note { self.transitioning = NO; }

- (NSDictionary *)state {
    NSRange a = self.editorA.selectedRange, b = self.editorB.selectedRange;
    NSResponder *first = self.window.firstResponder;
    NSString *focused = first == self.editorA ? (self.editorA.editable ? @"A" : @"readonly") : first == self.editorB ? @"B" : (first == self.secure || first == self.secure.currentEditor) ? @"secure" : @"";
    return @{
        @"web": @{@"text": self.webText, @"edits": @(self.webEdits), @"focused": @(self.webFocused)},
        @"clipboardOwned": @((BOOL)([NSPasteboard generalPasteboard].changeCount == self.clipboardChange)),
        @"A": @{@"text": self.editorA.string, @"edits": @(self.editsA), @"selection": @[@(a.location), @(a.length)],
                 @"marked": @(self.editorA.hasMarkedText), @"markedRange": self.editorA.hasMarkedText ? @[@(self.editorA.markedRange.location), @(self.editorA.markedRange.length)] : @[]},
        @"B": @{@"text": self.editorB.string, @"edits": @(self.editsB), @"selection": @[@(b.location), @(b.length)],
                 @"marked": @(self.editorB.hasMarkedText), @"markedRange": self.editorB.hasMarkedText ? @[@(self.editorB.markedRange.location), @(self.editorB.markedRange.length)] : @[]},
        @"secure": @{@"length": @(self.secure.stringValue.length)},
        @"focused": focused,
        @"active": @(NSApp.isActive),
        @"pid": @(NSProcessInfo.processInfo.processIdentifier),
        @"fullscreen": @((BOOL)((self.window.styleMask & NSWindowStyleMaskFullScreen) != 0)),
        @"transitioning": @(self.transitioning),
    };
}

- (NSDictionary *)handle:(NSDictionary *)command {
    NSString *op = command[@"op"];
    if ([op isEqual:@"focus"]) {
        NSString *target = command[@"target"];
        [NSApp activateIgnoringOtherApps:YES];
        [self.window makeKeyAndOrderFront:nil];
        self.web.hidden = ![target isEqual:@"web"];
        if ([target isEqual:@"web"]) {
            [self.window makeFirstResponder:self.web];
            [self.web evaluateJavaScript:@"t.focus();report()" completionHandler:nil];
            return [self state];
        }
        self.editorA.editable = ![target isEqual:@"readonly"];
        NSResponder *responder = [target isEqual:@"none"] ? nil : ([target isEqual:@"A"] || [target isEqual:@"readonly"]) ? self.editorA : [target isEqual:@"B"] ? self.editorB : self.secure;
        [self.window makeFirstResponder:responder];
    } else if ([op isEqual:@"set"]) {
        NSTextView *editor = [command[@"target"] isEqual:@"B"] ? self.editorB : self.editorA;
        editor.string = command[@"text"] ?: @"";
        NSUInteger location = [command[@"location"] unsignedIntegerValue];
        NSUInteger length = [command[@"length"] unsignedIntegerValue];
        if (location <= editor.string.length && length <= editor.string.length - location) {
            editor.selectedRange = NSMakeRange(location, length);
        }
        self.editsA = self.editsB = 0;
    } else if ([op isEqual:@"clipboardSave"]) {
        NSPasteboard *board = NSPasteboard.generalPasteboard;
        NSInteger initialChange = board.changeCount;
        NSMutableArray *saved = [NSMutableArray new];
        for (NSPasteboardItem *source in board.pasteboardItems) {
            NSPasteboardItem *item = [NSPasteboardItem new];
            for (NSPasteboardType type in source.types) {
                NSData *data = [source dataForType:type];
                if (!data || ![item setData:data forType:type]) abort();
            }
            [saved addObject:item];
        }
        if (board.changeCount != initialChange) abort();
        self.savedClipboard = saved;
        self.clipboardChange = board.changeCount;
    } else if ([op isEqual:@"clipboardSet"]) {
        NSPasteboard *board = NSPasteboard.generalPasteboard;
        [board clearContents];
        [board setString:command[@"text"] forType:NSPasteboardTypeString];
        self.clipboardChange = board.changeCount;
    } else if ([op isEqual:@"clipboardCheck"]) {
        NSPasteboard *board = NSPasteboard.generalPasteboard;
        // Only acknowledge the expected test-owned value for cleanup.
        BOOL matches = [[board stringForType:NSPasteboardTypeString] isEqual:command[@"text"]];
        if (matches) self.clipboardChange = board.changeCount;
        NSMutableDictionary *state = [[self state] mutableCopy];
        state[@"clipboardOwned"] = @(matches);
        return state;
    } else if ([op isEqual:@"clipboardRestore"]) {
        NSPasteboard *board = NSPasteboard.generalPasteboard;
        if (self.savedClipboard && board.changeCount == self.clipboardChange) {
            [board clearContents];
            if (self.savedClipboard.count) [board writeObjects:self.savedClipboard];
        }
    } else if ([op isEqual:@"fullscreen"]) {
        BOOL requested = [command[@"enabled"] boolValue];
        BOOL current = (self.window.styleMask & NSWindowStyleMaskFullScreen) != 0;
        if (requested != current) {
            if (requested) {
                // Match the overlay's display under the pointer, including
                // hosts whose pointer is currently on a secondary display.
                NSPoint pointer = NSEvent.mouseLocation;
                for (NSScreen *screen in NSScreen.screens) {
                    if (!NSPointInRect(pointer, screen.frame)) continue;
                    [self.window setFrameOrigin:NSMakePoint(NSMidX(screen.frame) - self.window.frame.size.width / 2,
                                                           NSMidY(screen.frame) - self.window.frame.size.height / 2)];
                    break;
                }
            }
            [self.window toggleFullScreen:nil];
        }
    } else if ([op isEqual:@"replace"]) {
        // Retypes the first occurrence of `from` in an editor as a user
        // would, so the change arrives through the text system.
        NSTextView *editor = [command[@"target"] isEqual:@"B"] ? self.editorB : self.editorA;
        NSRange range = [editor.string rangeOfString:command[@"from"] ?: @""];
        if (range.location != NSNotFound && command[@"to"]) {
            [editor setSelectedRange:range];
            [editor insertText:command[@"to"] replacementRange:range];
        }
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
