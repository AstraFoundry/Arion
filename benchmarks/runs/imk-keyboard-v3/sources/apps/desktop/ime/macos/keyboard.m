#import "keyboard.h"
#import "learning.h"
#import <rime_api.h>
#include <limits.h>

extern int32_t arion_keyboard_sha256(const void *, size_t, unsigned char *);

static NSString *const ArionKeyboardErrorDomain = @"ArionKeyboard";
static NSString *const ArionKeyboardSchema = @"arion_pinyin";

@implementation ArionKeyboardSnapshot
@end

static BOOL keyboardFailure(NSError **error, NSInteger code) {
    if (error) *error = [NSError errorWithDomain:ArionKeyboardErrorDomain code:code userInfo:nil];
    return NO;
}

static NSString *fromUTF8(const char *bytes) {
    return bytes ? [[NSString alloc] initWithUTF8String:bytes] : @"";
}

static BOOL utf16Position(const char *bytes, int offset, NSUInteger *position) {
    if (!bytes || offset < 0 || (size_t)offset > strlen(bytes)) return NO;
    NSString *prefix = [[NSString alloc] initWithBytes:bytes length:(NSUInteger)offset encoding:NSUTF8StringEncoding];
    if (!prefix) return NO;
    *position = prefix.length;
    return YES;
}

// The signed bundle supplies this manifest. Verify immutable data before
// provisioning a key or opening sessions; missing/corrupt data never types Latin.
static BOOL verifyData(NSString *directory) {
    NSString *root = directory.stringByDeletingLastPathComponent;
    NSData *manifestBytes = [NSData dataWithContentsOfFile:[root stringByAppendingPathComponent:@"runtime.json"]];
    if (!manifestBytes) return NO;
    id manifest = [NSJSONSerialization JSONObjectWithData:manifestBytes options:0 error:nil];
    if (![manifest isKindOfClass:NSDictionary.class] || ![manifest[@"schema"] isEqual:ArionKeyboardSchema] ||
        ![manifest[@"version"] isEqual:@"1.17.0"] || ![manifest[@"files"] isKindOfClass:NSDictionary.class]) return NO;
    NSDictionary *files = manifest[@"files"];
    for (NSString *required in @[@"data/build/default.yaml", @"data/build/arion_pinyin.schema.yaml",
         @"data/build/pinyin_simp.table.bin", @"data/build/pinyin_simp.prism.bin", @"data/build/pinyin_simp.reverse.bin"]) {
        if (![files[required] isKindOfClass:NSDictionary.class]) return NO;
    }
    for (id name in files) {
        if (![name isKindOfClass:NSString.class]) return NO;
        if (![name hasPrefix:@"data/"]) continue;
        NSArray *components = [(NSString *)name pathComponents];
        if ([components containsObject:@".."] || [components containsObject:@"."]) return NO;
        NSDictionary *record = files[name];
        if (![record isKindOfClass:NSDictionary.class] || ![record[@"sha256"] isKindOfClass:NSString.class] ||
            ![record[@"sizeBytes"] isKindOfClass:NSNumber.class]) return NO;
        NSString *path = [root stringByAppendingPathComponent:name];
        NSDictionary *attributes = [NSFileManager.defaultManager attributesOfItemAtPath:path error:nil];
        if (![attributes[NSFileType] isEqual:NSFileTypeRegular]) return NO;
        NSData *bytes = [NSData dataWithContentsOfFile:path];
        if (!bytes || bytes.length != [record[@"sizeBytes"] unsignedLongLongValue]) return NO;
        unsigned char digest[32];
        if (!arion_keyboard_sha256(bytes.bytes, bytes.length, digest)) return NO;
        NSMutableString *hex = [NSMutableString stringWithCapacity:64];
        for (NSUInteger i = 0; i < sizeof(digest); i++) [hex appendFormat:@"%02x", digest[i]];
        if (![hex isEqual:record[@"sha256"]]) return NO;
    }
    return YES;
}

@interface ArionKeyboardEngine ()
@property RimeApi *api;
@property(readwrite) NSUInteger generation;
@property(copy) NSString *dataDirectory;
@property(copy) NSString *userDirectory;
@property BOOL ready;
@property BOOL ascii;
@end

@implementation ArionKeyboardEngine
+ (instancetype)shared {
    static ArionKeyboardEngine *engine;
    static dispatch_once_t once;
    dispatch_once(&once, ^{ engine = [ArionKeyboardEngine new]; });
    return engine;
}

- (BOOL)startWithData:(NSString *)data userDirectory:(NSString *)user error:(NSError **)error {
    if (self.ready) {
        return ([data isEqual:self.dataDirectory] && [user isEqual:self.userDirectory]) || keyboardFailure(error, 1);
    }
    if (!verifyData(data)) return keyboardFailure(error, 2);
    if (![NSFileManager.defaultManager createDirectoryAtPath:user withIntermediateDirectories:YES
         attributes:@{NSFilePosixPermissions: @0700} error:error]) return NO;
    self.api = rime_get_api();
    if (!RIME_PROVIDED(self.api, highlight_candidate_on_current_page) || !RIME_PROVIDED(self.api, change_page)) {
        return keyboardFailure(error, 3);
    }
    self.dataDirectory = data;
    self.userDirectory = user;
    RIME_STRUCT(RimeTraits, traits);
    traits.shared_data_dir = self.dataDirectory.fileSystemRepresentation;
    traits.prebuilt_data_dir = [self.dataDirectory stringByAppendingPathComponent:@"build"].fileSystemRepresentation;
    traits.staging_dir = traits.prebuilt_data_dir;
    traits.user_data_dir = self.userDirectory.fileSystemRepresentation;
    traits.app_name = "rime.arion";
    traits.distribution_name = "Arion";
    traits.distribution_code_name = "arion";
    traits.distribution_version = "1";
    traits.min_log_level = 3;
    traits.log_dir = "";
    self.api->setup(&traits);
#ifndef ARION_LEARNING_TESTING
    if (!ArionRegisterEncryptedLearning([user stringByAppendingPathComponent:@"learning"], error)) return NO;
#endif
    // Tests register the same backend with an injected key before this call.
    // Production uses the separate Keychain-backed registration above.
    self.api->initialize(&traits);
    if (ArionEncryptedLearningError()) {
        if (error) *error = ArionEncryptedLearningError();
        self.api->finalize();
        return NO;
    }
    self.ready = YES;
    return YES;
}

- (uintptr_t)createSession {
    if (!self.ready) return 0;
    RimeSessionId session = self.api->create_session();
    if (!session || !self.api->select_schema(session, ArionKeyboardSchema.UTF8String)) {
        if (session) self.api->destroy_session(session);
        return 0;
    }
    self.api->set_option(session, "ascii_mode", self.ascii);
    return session;
}

- (void)stop {
    if (!self.ready) return;
    self.generation++;
    self.api->cleanup_all_sessions();
    self.api->finalize();
    self.ready = NO;
}

- (BOOL)resetLearning:(NSError **)error {
    NSString *data = self.dataDirectory;
    NSString *user = self.userDirectory;
    [self stop];
    if (!ArionResetEncryptedLearning([user stringByAppendingPathComponent:@"learning"], error)) return NO;
    return [self startWithData:data userDirectory:user error:error];
}

- (void)setASCII:(BOOL)ascii session:(uintptr_t)session {
    self.ascii = ascii;
    if (session) self.api->set_option(session, "ascii_mode", ascii);
}

- (void)destroySession:(uintptr_t)session {
    if (session) self.api->destroy_session(session);
}
- (BOOL)processKey:(int)key modifiers:(int)modifiers session:(uintptr_t)session {
    return session && key >= 0 && self.api->process_key(session, key, modifiers);
}
- (BOOL)select:(NSUInteger)index session:(uintptr_t)session {
    return session && self.api->select_candidate_on_current_page(session, index);
}
- (BOOL)highlight:(NSUInteger)index session:(uintptr_t)session {
    return session && self.api->highlight_candidate_on_current_page(session, index);
}
- (void)clear:(uintptr_t)session {
    if (session) self.api->clear_composition(session);
}

- (ArionKeyboardSnapshot *)snapshot:(uintptr_t)session error:(NSError **)error {
    if (!self.ready || !session) { keyboardFailure(error, 4); return nil; }
    // A learning I/O error must not swallow an already produced text commit.
    NSError *learningFailure = ArionEncryptedLearningError();
    RIME_STRUCT(RimeContext, context);
    RIME_STRUCT(RimeStatus, status);
    if (!self.api->get_context(session, &context)) { keyboardFailure(error, 5); return nil; }
    if (!self.api->get_status(session, &status)) {
        self.api->free_context(&context);
        keyboardFailure(error, 5);
        return nil;
    }
    ArionKeyboardSnapshot *result = [ArionKeyboardSnapshot new];
    result.failure = learningFailure;
    result.preedit = fromUTF8(context.composition.preedit);
    result.ascii = status.is_ascii_mode;
    BOOL valid = result.preedit != nil;
    NSUInteger cursor = 0;
    if (result.preedit.length > 0) valid = valid && utf16Position(context.composition.preedit, context.composition.cursor_pos, &cursor);
    result.selection = NSMakeRange(cursor, 0);
    NSMutableArray *candidates = [NSMutableArray array];
    NSMutableArray *comments = [NSMutableArray array];
    if (context.menu.num_candidates < 0 || context.menu.num_candidates > 9 || context.menu.page_no < 0) valid = NO;
    if (valid) {
        for (int i = 0; i < context.menu.num_candidates; i++) {
            NSString *text = fromUTF8(context.menu.candidates[i].text);
            NSString *comment = fromUTF8(context.menu.candidates[i].comment);
            if (!text || !comment) { valid = NO; break; }
            [candidates addObject:text];
            [comments addObject:comment];
        }
    }
    result.candidates = candidates;
    result.comments = comments;
    result.page = (NSUInteger)MAX(0, context.menu.page_no);
    result.lastPage = context.menu.is_last_page;
    result.highlighted = (NSUInteger)MAX(0, context.menu.highlighted_candidate_index);
    self.api->free_context(&context);
    self.api->free_status(&status);
    if (!valid) { keyboardFailure(error, 6); return nil; }
    RIME_STRUCT(RimeCommit, commit);
    result.commit = @"";
    if (self.api->get_commit(session, &commit)) {
        result.commit = fromUTF8(commit.text);
        self.api->free_commit(&commit);
        if (!result.commit) { keyboardFailure(error, 6); return nil; }
    }
    return result;
}
@end

int ArionKeyboardKey(NSEvent *event) {
    switch (event.keyCode) {
        case 36: case 76: return 0xff0d;
        case 48: return 0xff09;
        case 51: return 0xff08;
        case 53: return 0xff1b;
        case 115: return 0xff50;
        case 116: return 0xff55;
        case 117: return 0xffff;
        case 119: return 0xff57;
        case 121: return 0xff56;
        case 123: return 0xff51;
        case 124: return 0xff53;
        case 125: return 0xff54;
        case 126: return 0xff52;
        default: break;
    }
    NSString *characters = event.characters;
    if (characters.length != 1) return -1;
    unichar key = [characters characterAtIndex:0];
    return key < 0x80 ? key : -1;
}

int ArionKeyboardModifiers(NSEvent *event) {
    NSEventModifierFlags flags = event.modifierFlags;
    return ((flags & NSEventModifierFlagShift) ? 1 : 0) |
           ((flags & NSEventModifierFlagCapsLock) ? 2 : 0) |
           ((flags & NSEventModifierFlagControl) ? 4 : 0) |
           ((flags & NSEventModifierFlagOption) ? 8 : 0) |
           ((flags & NSEventModifierFlagCommand) ? (1 << 26) : 0);
}
