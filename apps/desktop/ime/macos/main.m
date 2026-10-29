// The InputMethodKit host owns client ranges; recognition stays in Arion.
#import <Cocoa/Cocoa.h>
#import <InputMethodKit/InputMethodKit.h>
#import <CommonCrypto/CommonDigest.h>
#import "candidate_panel.h"
#import "composition.h"
#import "typing.h"
#include "../shared/spelling_options.h"
#import "source_maintenance.h"
#include "../../infrastructure/imebridge/peer_darwin.h"
#include <sys/socket.h>
#include <sys/un.h>
#include <errno.h>
#include <unistd.h>

static NSString *hostBuildIdentity;

@class ArionInputController;

@interface ArionBridge : NSObject
@property(weak) ArionInputController *active;
@property int fd;
@property(copy) NSString *connection;
@property(assign) SecCodeRef peerCode;
@property(assign) SecRequirementRef peerRequirement;
+ (instancetype)shared;
- (void)send:(NSDictionary *)message;
- (BOOL)connect;
@end

@interface ArionInputController : IMKInputController
@property(strong) ArionComposition *composition;
@property(copy) NSString *clientBundle;
@property(strong) ArionTyping *typing;
@property(strong) ArionCandidatePanel *candidateWindow;
@property(strong) ArionLanguagePanel *languageWindow;
@property BOOL keyboardReady;
@property(copy) NSString *dictationKey;
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

// A #rrggbb color of the app-resolved candidate highlight, or nil.
static NSColor *highlightColor(NSDictionary *highlight, NSString *appearance, NSString *part) {
    NSDictionary *colors = highlight[appearance];
    NSString *hex = [colors isKindOfClass:NSDictionary.class] ? colors[part] : nil;
    if (![hex isKindOfClass:NSString.class] || hex.length != 7 || ![hex hasPrefix:@"#"]) return nil;
    unsigned int value = 0;
    NSScanner *scanner = [NSScanner scannerWithString:[hex substringFromIndex:1]];
    if (![scanner scanHexInt:&value] || !scanner.isAtEnd) return nil;
    return [NSColor colorWithSRGBRed:((value >> 16) & 0xff) / 255.0 green:((value >> 8) & 0xff) / 255.0
                                blue:(value & 0xff) / 255.0 alpha:1];
}

// Keyboard preferences the app writes (application.KeyboardPreferences). The
// input method works without the app, so an absent file means the defaults;
// unknown values keep the defaults for that field. Returns the app's
// dictation key (application.HoldKey), or nil.
static NSString *applyKeyboardSettings(ArionTyping *typing, ArionCandidatePanel *panel) {
    NSUInteger pageSize = 9;
    ArionCandidateSize size = ArionCandidateSizeMedium;
    ArionCandidateAppearance appearance = ArionCandidateAppearanceSystem;
    NSColor *lightFill = nil, *lightText = nil, *darkFill = nil, *darkText = nil;
    BOOL emojiSuggestions = YES, typoCorrection = NO;
    uint32_t fuzzyPinyin = 0;
    NSString *dictationKey = nil;
    NSData *data = [NSData dataWithContentsOfFile:[applicationSupport() stringByAppendingPathComponent:@"keyboard.json"]];
    NSDictionary *settings = data ? [NSJSONSerialization JSONObjectWithData:data options:0 error:nil] : nil;
    if ([settings isKindOfClass:NSDictionary.class]) {
        NSNumber *page = settings[@"pageSize"];
        if ([page isKindOfClass:NSNumber.class] && page.integerValue >= 1 && page.integerValue <= 9) pageSize = page.unsignedIntegerValue;
        NSDictionary *sizes = @{@"small": @(ArionCandidateSizeSmall), @"medium": @(ArionCandidateSizeMedium), @"large": @(ArionCandidateSizeLarge)};
        NSDictionary *appearances = @{@"system": @(ArionCandidateAppearanceSystem), @"light": @(ArionCandidateAppearanceLight),
                                      @"dark": @(ArionCandidateAppearanceDark)};
        id value = settings[@"candidateSize"];
        if ([value isKindOfClass:NSString.class] && sizes[value]) size = [sizes[value] integerValue];
        value = settings[@"appearance"];
        if ([value isKindOfClass:NSString.class] && appearances[value]) appearance = [appearances[value] integerValue];
        value = settings[@"highlight"];
        if ([value isKindOfClass:NSDictionary.class]) {
            lightFill = highlightColor(value, @"light", @"fill");
            lightText = highlightColor(value, @"light", @"text");
            darkFill = highlightColor(value, @"dark", @"fill");
            darkText = highlightColor(value, @"dark", @"text");
        }
        value = settings[@"emojiSuggestions"];
        if ([value isKindOfClass:NSNumber.class]) emojiSuggestions = [value boolValue];
        value = settings[@"typoCorrection"];
        if ([value isKindOfClass:NSNumber.class]) typoCorrection = [value boolValue];
        value = settings[@"fuzzyPinyin"];
        if ([value isKindOfClass:NSNumber.class] && [value doubleValue] >= 0 &&
            [value doubleValue] <= ARION_FUZZY_MASK && [value doubleValue] == [value unsignedIntValue])
            fuzzyPinyin = [value unsignedIntValue];
        value = settings[@"dictationKey"];
        if ([value isKindOfClass:NSString.class]) dictationKey = value;
    }
    typing.pageSize = pageSize;
    typing.emojiSuggestions = emojiSuggestions;
    typing.typoCorrection = typoCorrection;
    typing.fuzzyPinyin = fuzzyPinyin;
    panel.size = size;
    panel.appearance = appearance;
    panel.lightFill = lightFill;
    panel.lightText = lightText;
    panel.darkFill = darkFill;
    panel.darkText = darkText;
    return dictationKey;
}

// The Voice Input item's title names the dictation key with the app's copy
// for it; an absent or unknown key leaves the key out (IME-06).
static NSString *voiceInputTitle(NSString *dictationKey) {
    NSDictionary *names = @{
        @"fn": @"HoldKeyFn", @"right_option": @"HoldKeyRightOption", @"right_command": @"HoldKeyRightCommand",
        @"left_option": @"HoldKeyLeftOption", @"left_command": @"HoldKeyLeftCommand",
        @"left_ctrl": @"HoldKeyLeftCtrl", @"right_ctrl": @"HoldKeyRightCtrl",
        @"left_shift": @"HoldKeyLeftShift", @"right_shift": @"HoldKeyRightShift",
        @"space": @"HoldKeySpace", @"enter": @"HoldKeyEnter", @"tab": @"HoldKeyTab",
        @"backspace": @"HoldKeyBackspace", @"delete": @"HoldKeyDelete", @"home": @"HoldKeyHome", @"end": @"HoldKeyEnd",
        @"page_up": @"HoldKeyPageUp", @"page_down": @"HoldKeyPageDown",
        @"left": @"HoldKeyLeft", @"right": @"HoldKeyRight", @"up": @"HoldKeyUp", @"down": @"HoldKeyDown"
    };
    NSDictionary *legends = @{
        @"ctrl": @"Ctrl", @"alt": @"Option", @"shift": @"Shift", @"super": @"Command",
        @"minus": @"-", @"equal": @"=", @"comma": @",", @"period": @".", @"slash": @"/", @"semicolon": @";",
        @"quote": @"'", @"bracket_left": @"[", @"bracket_right": @"]", @"backslash": @"\\", @"grave": @"`"
    };
    NSMutableArray<NSString *> *labels = [NSMutableArray array];
    for (NSString *part in [dictationKey componentsSeparatedByString:@"+"]) {
        NSString *name = names[part];
        if (name) [labels addObject:NSLocalizedString(name, nil)];
        else if (legends[part]) [labels addObject:legends[part]];
        else {
            NSRange range = [part rangeOfString:@"^([a-z0-9]|f([1-9]|1[0-9]|20))$" options:NSRegularExpressionSearch];
            if (range.location == NSNotFound) return NSLocalizedString(@"KeyboardVoiceInput", nil);
            [labels addObject:part.uppercaseString];
        }
    }
    if (!labels.count) return NSLocalizedString(@"KeyboardVoiceInput", nil);
    return [NSLocalizedString(@"KeyboardVoiceInputKey", nil) stringByReplacingOccurrencesOfString:@"{key}"
        withString:[labels componentsJoinedByString:@"+"]];
}

@implementation ArionBridge
+ (instancetype)shared {
    static ArionBridge *bridge;
    static dispatch_once_t once;
    dispatch_once(&once, ^{
        bridge = [ArionBridge new];
        bridge.fd = -1;
        [NSTimer scheduledTimerWithTimeInterval:2 repeats:YES block:^(NSTimer *timer) {
            (void)timer;
            if (bridge.fd < 0) {
                if (bridge.active) [bridge send:bridge.active.activationMessage];
                else [bridge connect];
            }
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
    id appIdentifier=NSBundle.mainBundle.infoDictionary[@"ArionAppIdentifier"];
    SecRequirementRef requirement=[appIdentifier isKindOfClass:NSString.class] ?
        arion_peer_publisher_requirement([appIdentifier UTF8String]) : NULL;
    uint32_t peerPID=0,peerUID=0;
    SecCodeRef peer=requirement ? arion_socket_peer(fd,&peerPID,&peerUID) : NULL;
    if (!arion_peer_matches(peer,requirement)) {
        if (peer) CFRelease(peer);
        if (requirement) CFRelease(requirement);
        close(fd); return NO;
    }
    int reader = dup(fd);
    if (reader < 0) { CFRelease(peer); CFRelease(requirement); close(fd); return NO; }
    self.peerCode=peer;
    self.peerRequirement=requirement;
    self.fd = fd;
    self.connection = NSUUID.UUID.UUIDString;
    [self writeMessage:@{@"op": @"hello", @"build": hostBuildIdentity, @"protocol":@1,
        @"capabilities":@[@"keyboard-vocabulary-v1"]}];
    if (self.fd < 0) { close(reader); return NO; }
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
    if (self.peerCode) { CFRelease(self.peerCode); self.peerCode=NULL; }
    if (self.peerRequirement) { CFRelease(self.peerRequirement); self.peerRequirement=NULL; }
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
                if ([self.connection isEqual:connection]) {
                    if (arion_peer_matches(self.peerCode,self.peerRequirement)) [self.active apply:message];
                    else [self disconnect];
                }
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
    if (!arion_peer_matches(self.peerCode,self.peerRequirement)) { [self disconnect]; return; }
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
        self.typing.modeChanged = ^(ArionTypingSwitch change) {
            ArionInputController *controller = weakSelf;
            if ([ArionBridge shared].active != controller) return;
            [controller interrupt];
            ArionTyping *typing = controller.typing;
            BOOL on = change == ArionTypingSwitchLanguage ? typing.ascii :
                change == ArionTypingSwitchWidth ? typing.fullWidth : typing.englishPunctuation;
            NSString *key = change == ArionTypingSwitchLanguage ? @"KeyboardModeEnglish" :
                change == ArionTypingSwitchWidth ? @"KeyboardModeFullWidth" : @"KeyboardModeEnglishPunctuation";
            NSString *other = change == ArionTypingSwitchLanguage ? @"KeyboardModeChinese" :
                change == ArionTypingSwitchWidth ? @"KeyboardModeHalfWidth" : @"KeyboardModeChinesePunctuation";
            NSRect caret = NSZeroRect;
            [controller.client attributesForCharacterIndex:0 lineHeightRectangle:&caret];
            [controller.languageWindow show:NSLocalizedString(on ? key : other, nil)
                                   previous:NSLocalizedString(on ? other : key, nil) caretRect:caret];
        };
    }
    if (!self.candidateWindow) {
        self.candidateWindow = [ArionCandidatePanel new];
        __weak ArionInputController *weakSelf = self;
        self.candidateWindow.selected = ^(NSUInteger index, NSUInteger row, NSUInteger revision) {
            ArionInputController *controller = weakSelf;
            if ([ArionBridge shared].active == controller) [controller.typing select:index row:row revision:revision];
        };
        self.candidateWindow.toggled = ^(NSUInteger revision) {
            ArionInputController *controller = weakSelf;
            if ([ArionBridge shared].active == controller) [controller.typing toggleExpanded:revision];
        };
    }
    if (!self.languageWindow) self.languageWindow = [ArionLanguagePanel new];
    self.dictationKey = applyKeyboardSettings(self.typing, self.candidateWindow);
    self.languageWindow.appearance = self.candidateWindow.appearance;
    self.keyboardReady = [self.typing activate:sender error:nil];
    self.clientBundle = [sender respondsToSelector:@selector(bundleIdentifier)] ? [sender bundleIdentifier] : @"";
    [ArionBridge shared].active = self;
    [[ArionBridge shared] send:self.activationMessage];
}

- (void)deactivateServer:(id)sender {
    [self.typing deactivate];
    [self.candidateWindow hide];
    [self.languageWindow hide];
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

- (NSUInteger)recognizedEvents:(id)sender {
    (void)sender;
    // Requesting modifier events disables IMK's default mouse termination.
    // Relinquish provisional text before a host click changes its owner.
    return NSEventMaskKeyDown | NSEventMaskFlagsChanged |
        NSEventMaskLeftMouseDown | NSEventMaskRightMouseDown | NSEventMaskOtherMouseDown;
}

- (BOOL)handleEvent:(NSEvent *)event client:(id)sender {
    if (event.type == NSEventTypeLeftMouseDown || event.type == NSEventTypeRightMouseDown ||
        event.type == NSEventTypeOtherMouseDown) {
        [self commitComposition:sender];
        [self.languageWindow hide];
        return NO;
    }
    if (event.type == NSEventTypeFlagsChanged) {
        return self.keyboardReady && self.typing.available && [self.typing handleEvent:event];
    }
    if (event.type != NSEventTypeKeyDown) return NO;
    [self.languageWindow hide];
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
    if ([message[@"op"] isEqual:@"vocabulary_query"] || [message[@"op"] isEqual:@"vocabulary_publish"]) {
        NSDictionary *ack=[self.typing applyVocabularyMessage:message activation:self.composition.activation blocked:self.composition.live];
        if (ack) [[ArionBridge shared] send:ack];
        return;
    }
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
    [self.languageWindow hide];
    ArionKeyboardSnapshot *snapshot = self.typing.snapshot;
    if (!self.typing.composing || snapshot.candidates.count == 0) {
        [self.candidateWindow hide];
        return;
    }
    // The marked text starts at index 0; its line rectangle is in screen space.
    NSRect caret = NSZeroRect;
    [self.client attributesForCharacterIndex:0 lineHeightRectangle:&caret];
    [self.candidateWindow showRows:snapshot.rows
                        currentRow:snapshot.currentRow
                       highlighted:MIN(snapshot.highlighted, snapshot.candidates.count - 1)
                          expanded:snapshot.expanded
                              more:!snapshot.lastPage
                          revision:self.typing.revision
                         caretRect:caret];
}

// The input menu follows the system input sources: the mode switch with its
// shortcut, keyboard maintenance, then the app that owns settings.
- (NSMenu *)menu {
    NSMenu *menu = [[NSMenu alloc] initWithTitle:@""];
    menu.autoenablesItems = NO;
    // Voice input needs the app; reconnect now so the item reflects it.
    ArionBridge *bridge = [ArionBridge shared];
    if (bridge.fd < 0 && bridge.active == self) [bridge send:self.activationMessage];
    NSMenuItem *voice = [[NSMenuItem alloc] initWithTitle:voiceInputTitle(self.dictationKey)
        action:@selector(dictate:) keyEquivalent:@""];
    voice.target = self;
    voice.enabled = bridge.fd >= 0 && bridge.active == self;
    [menu addItem:voice];
    [menu addItem:NSMenuItem.separatorItem];
    NSString *key = (!self.keyboardReady || !self.typing.available) ? @"KeyboardUnavailable" :
        (self.typing.ascii ? @"KeyboardSwitchChinese" : @"KeyboardSwitchEnglish");
    NSMenuItem *item = [self menuItem:key action:@selector(toggleLanguage:)];
    item.enabled = self.keyboardReady && self.typing.available;
    [menu addItem:item];
    NSMenuItem *width = [self menuItem:self.typing.fullWidth ? @"KeyboardUseHalfWidth" : @"KeyboardUseFullWidth"
                                action:@selector(toggleWidth:)];
    width.enabled = item.enabled;
    [menu addItem:width];
    NSMenuItem *punctuation = [self menuItem:self.typing.englishPunctuation ? @"KeyboardUseChinesePunctuation" : @"KeyboardUseEnglishPunctuation"
                                      action:@selector(togglePunctuation:)];
    punctuation.enabled = item.enabled;
    [menu addItem:punctuation];
    [menu addItem:NSMenuItem.separatorItem];
    if (self.typing.recoveryText.length > 0) [menu addItem:[self menuItem:@"KeyboardCopyPending" action:@selector(copyPending:)]];
    [menu addItem:[self menuItem:@"KeyboardResetLearning" action:@selector(resetLearning:)]];
    NSString *app = [NSBundle.mainBundle objectForInfoDictionaryKey:@"ArionAppIdentifier"];
    if ([app isKindOfClass:NSString.class] && [NSWorkspace.sharedWorkspace URLForApplicationWithBundleIdentifier:app]) {
        [menu addItem:NSMenuItem.separatorItem];
        [menu addItem:[self menuItem:@"KeyboardOpenApp" action:@selector(openApp:)]];
    }
    return menu;
}

- (NSMenuItem *)menuItem:(NSString *)key action:(SEL)action {
    NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:NSLocalizedString(key, nil) action:action keyEquivalent:@""];
    item.target = self;
    return item;
}

- (void)openApp:(id)sender {
    NSString *app = [NSBundle.mainBundle objectForInfoDictionaryKey:@"ArionAppIdentifier"];
    NSURL *url = [app isKindOfClass:NSString.class] ? [NSWorkspace.sharedWorkspace URLForApplicationWithBundleIdentifier:app] : nil;
    if (!url) { NSBeep(); return; }
    NSWorkspaceOpenConfiguration *configuration = [NSWorkspaceOpenConfiguration configuration];
    configuration.activates = YES;
    [NSWorkspace.sharedWorkspace openApplicationAtURL:url configuration:configuration completionHandler:nil];
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

// The app toggles dictation as its tray item does; the session then binds
// to this client through begin (IME-06).
- (void)dictate:(id)sender {
    if (!self.composition.activation || [ArionBridge shared].active != self) return;
    [[ArionBridge shared] send:@{@"op": @"dictate", @"activation": self.composition.activation}];
}

- (void)toggleLanguage:(id)sender {
    [self interrupt];
    [self.typing toggleASCII];
}

- (void)toggleWidth:(id)sender {
    [self interrupt];
    if (![self.typing toggleFullWidth]) NSBeep();
}

- (void)togglePunctuation:(id)sender {
    [self interrupt];
    if (![self.typing toggleEnglishPunctuation]) NSBeep();
}

@end

int main(int argc, const char *argv[]) {
    @autoreleasepool {
        if (argc == 3 && strcmp(argv[1], "--arion-source-maintenance") == 0) {
            NSString *operation = [NSString stringWithUTF8String:argv[2]];
            if (![operation isEqual:@"observe"] && ![operation isEqual:@"disable"]) return 2;
            NSArray *statuses = [operation isEqual:@"disable"] ? ArionDisableOwnSources() : @[];
            NSDictionary *result = @{@"version": @1, @"pid": @(getpid()), @"operation": operation,
                @"statuses": statuses, @"observation": ArionSourceObservation()};
            NSData *json = [NSJSONSerialization dataWithJSONObject:result options:0 error:nil];
            if (!json) return 3;
            fwrite(json.bytes, 1, json.length, stdout);
            fputc('\n', stdout);
            return 0;
        }
        // Cache the executable identity at process startup, before an updater can
        // replace the file at this path. Every reconnection uses this identity.
        NSData *executable = [NSData dataWithContentsOfFile:NSBundle.mainBundle.executablePath];
        if (!executable || executable.length > UINT32_MAX) return 1;
        unsigned char digest[CC_SHA256_DIGEST_LENGTH];
        CC_SHA256(executable.bytes, (CC_LONG)executable.length, digest);
        NSMutableString *identity = [NSMutableString stringWithCapacity:CC_SHA256_DIGEST_LENGTH * 2];
        for (NSUInteger i = 0; i < CC_SHA256_DIGEST_LENGTH; i++) [identity appendFormat:@"%02x", digest[i]];
        hostBuildIdentity = identity;
        [NSApplication sharedApplication];
        NSString *data = [NSBundle.mainBundle.resourcePath stringByAppendingPathComponent:@"keyboard/data"];
        NSString *profile = applicationSupport();
        if (profile) [ArionKeyboardEngine.shared startWithData:data
            userDirectory:[profile stringByAppendingPathComponent:@"keyboard"] error:nil];
        [NSNotificationCenter.defaultCenter addObserverForName:NSApplicationWillTerminateNotification object:nil queue:nil
            usingBlock:^(NSNotification *notification) { (void)notification; [ArionKeyboardEngine.shared stop]; }];
        NSString *connection = [NSBundle.mainBundle objectForInfoDictionaryKey:@"InputMethodConnectionName"];
        __attribute__((objc_precise_lifetime)) IMKServer *server = [[IMKServer alloc] initWithName:connection bundleIdentifier:NSBundle.mainBundle.bundleIdentifier];
        if (!server) return 1;
        [[ArionBridge shared] connect];
        [NSApp run];
        (void)server;
    }
    return 0;
}
