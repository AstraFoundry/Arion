#import <ApplicationServices/ApplicationServices.h>
#import <Carbon/Carbon.h>
#import <Cocoa/Cocoa.h>

static int focus_ok = 1, selection_ok = 1, secure = 0, posts = 0;
int arion_focus_matches(void *element, int pid) { return focus_ok; }
int arion_selection_matches(void *element, void *selection) { return selection_ok; }
int arion_secure_input(void) { return secure; }
int arion_frontmost_pid(void) { return 1; }
// paste.m calls this before reading focus. The real body lives in native.m,
// which this pasteboard unit does not link. The check never drives a web view.
void arion_enable_web_accessibility(int pid) { (void)pid; }

// Keep real pasteboard and CGEvent construction, but intercept delivery:
// this isolated check must never type into a user's foreground application.
static void test_post(CGEventTapLocation tap, CGEventRef event) {
    if (CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode) != kVK_ANSI_V ||
        CGEventGetFlags(event) != kCGEventFlagMaskCommand) abort();
    if (CGEventGetType(event) != (posts % 2 ? kCGEventKeyUp : kCGEventKeyDown)) abort();
    posts++;
}
#define CGEventPost test_post
#include "../paste.m"
#undef CGEventPost

#define CHECK(x) do { if (!(x)) { fprintf(stderr, "failed at line %d: %s\n", __LINE__, #x); return 1; } } while (0)

static void *retain_tx(ArionPasteTransaction *tx) { return (__bridge_retained void *)tx; }
static void replace_board(NSPasteboard *board, NSString *text) {
    [board clearContents];
    [board setString:text forType:NSPasteboardTypeString];
}

// Qt applications such as WeChat and QQ publish these flavors through the
// Carbon pasteboard. Their names are not valid UTIs, so AppKit lists them but
// returns no data for them.
static NSString *const weChatFlavor = @"com.trolltech.anymime.WeChat_RichEdit_Format";
static NSString *const qqFlavor = @"com.trolltech.anymime.QQ_Unicode_RichEdit_Format";
static char carbonTextItem, carbonOpaqueItem;

static BOOL put_carbon_flavor(PasteboardRef pasteboard, void *item, NSString *flavor, const char *bytes) {
    CFDataRef data = CFDataCreate(kCFAllocatorDefault, (const UInt8 *)bytes, (CFIndex)strlen(bytes));
    OSStatus status = PasteboardPutItemFlavor(pasteboard, (PasteboardItemID)item, (__bridge CFStringRef)flavor, data, kPasteboardFlavorNoFlags);
    CFRelease(data);
    return status == noErr;
}

static BOOL write_carbon_board(NSPasteboard *board, BOOL withText) {
    PasteboardRef pasteboard = NULL;
    if (PasteboardCreate((__bridge CFStringRef)board.name, &pasteboard) != noErr) return NO;
    BOOL ok = PasteboardClear(pasteboard) == noErr;
    PasteboardSynchronize(pasteboard);
    if (withText) {
        ok = ok && put_carbon_flavor(pasteboard, &carbonTextItem, NSPasteboardTypeString, "carbon text");
        ok = ok && put_carbon_flavor(pasteboard, &carbonTextItem, weChatFlavor, "rich");
        ok = ok && put_carbon_flavor(pasteboard, &carbonTextItem, qqFlavor, "rich");
    }
    ok = ok && put_carbon_flavor(pasteboard, &carbonOpaqueItem, weChatFlavor, "rich");
    PasteboardSynchronize(pasteboard);
    CFRelease(pasteboard);
    return ok;
}

int main(void) {
    @autoreleasepool {
        NSPasteboard *board = [NSPasteboard pasteboardWithUniqueName];
        replace_board(board,@"same user text");
        CHECK(!history_excluded(board));
        NSPasteboardItem *privateItem = [NSPasteboardItem new];
        [privateItem setString:@"same user text" forType:NSPasteboardTypeString];
        [privateItem setData:[NSData data] forType:@"org.nspasteboard.ConcealedType"];
        [board clearContents]; [board writeObjects:@[privateItem]];
        CHECK(history_excluded(board));
        replace_board(board,@"same user text");
        CHECK(!history_excluded(board));
        [board clearContents];
        NSPasteboardItem *first = [NSPasteboardItem new], *second = [NSPasteboardItem new];
        NSData *bytes = [NSData dataWithBytes:"\0\1\377" length:3];
        [first setString:@"original" forType:NSPasteboardTypeString];
        [first setData:bytes forType:@"example.binary"];
        [second setString:@"second item" forType:NSPasteboardTypeString];
        [board writeObjects:@[first, second]];

        NSString *text = @"中文🙂e\u0301\nsecond line\r\nthird";
        ArionPasteTransaction *tx = begin_paste(text, board);
        CHECK(tx);
        CHECK([[board.pasteboardItems[0] stringForType:NSPasteboardTypeString] isEqual:@"original"]);
        CHECK(arion_paste_stage((__bridge void *)tx) == ARION_PASTE_OK);
        CHECK(history_excluded(board));
        CHECK([[board stringForType:NSPasteboardTypeString] isEqual:text]);
        CHECK([board.pasteboardItems[0].types containsObject:transientPasteType]);
        CHECK(arion_paste_stage((__bridge void *)tx) == ARION_PASTE_FAILED);
        CHECK(arion_paste_post((__bridge void *)tx, NULL, NULL, 1) == ARION_PASTE_OK);
        CHECK(posts == 2);
        CHECK(arion_paste_post((__bridge void *)tx, NULL, NULL, 1) == ARION_PASTE_FAILED);
        CHECK(posts == 2);
        CHECK(arion_paste_finish(retain_tx(tx)) == 1);
        CHECK(board.pasteboardItems.count == 2);
        CHECK([[board.pasteboardItems[0] dataForType:@"example.binary"] isEqual:bytes]);
        CHECK([[board.pasteboardItems[1] stringForType:NSPasteboardTypeString] isEqual:@"second item"]);

        tx = begin_paste(@"before stage", board);
        replace_board(board, @"new owner before stage");
        CHECK(arion_paste_stage((__bridge void *)tx) == ARION_PASTE_CHANGED);
        CHECK(arion_paste_finish(retain_tx(tx)) == 0);
        CHECK([[board stringForType:NSPasteboardTypeString] isEqual:@"new owner before stage"]);

        tx = begin_paste(@"after stage", board);
        CHECK(arion_paste_stage((__bridge void *)tx) == ARION_PASTE_OK);
        replace_board(board, @"new owner before post");
        CHECK(arion_paste_post((__bridge void *)tx, NULL, NULL, 1) == ARION_PASTE_CHANGED);
        CHECK(arion_paste_finish(retain_tx(tx)) == 0);
        CHECK([[board stringForType:NSPasteboardTypeString] isEqual:@"new owner before post"]);
        CHECK(posts == 2);

        for (int failure = 0; failure < 3; failure++) {
            tx = begin_paste(@"must not post", board);
            CHECK(arion_paste_stage((__bridge void *)tx) == ARION_PASTE_OK);
            focus_ok = failure != 0; selection_ok = failure != 1; secure = failure == 2;
            CHECK(arion_paste_post((__bridge void *)tx, NULL, NULL, 1) ==
                  (failure == 2 ? ARION_PASTE_SECURE : ARION_PASTE_TARGET_CHANGED));
            CHECK(arion_paste_finish(retain_tx(tx)) == 1);
            CHECK(posts == 2);
        }
        focus_ok = selection_ok = 1; secure = 0;

        tx = begin_paste(@"posted before user copies", board);
        CHECK(arion_paste_stage((__bridge void *)tx) == ARION_PASTE_OK);
        CHECK(arion_paste_post((__bridge void *)tx, NULL, NULL, 1) == ARION_PASTE_OK);
        CHECK(posts == 4);
        replace_board(board, @"new owner after post");
        CHECK(arion_paste_finish(retain_tx(tx)) == 0);
        CHECK([[board stringForType:NSPasteboardTypeString] isEqual:@"new owner after post"]);

        [board clearContents];
        tx = begin_paste(@"empty original clipboard", board);
        CHECK(arion_paste_stage((__bridge void *)tx) == ARION_PASTE_OK);
        CHECK(arion_paste_finish(retain_tx(tx)) == 1);
        CHECK(board.pasteboardItems.count == 0);
        CHECK(begin_paste(@"", board) == nil);

        CHECK(!valid_type_identifier(weChatFlavor) && !valid_type_identifier(qqFlavor) && !valid_type_identifier(@""));
        CHECK(valid_type_identifier(NSPasteboardTypeString) && valid_type_identifier(@"dyn.ah62d4rv4gu8y"));
        CHECK(write_carbon_board(board, YES));
        CHECK(board.pasteboardItems.count == 2);
        CHECK([board.pasteboardItems[0].types containsObject:weChatFlavor]);
        CHECK([board.pasteboardItems[0].types containsObject:qqFlavor]);
        NSInteger carbonChange = board.changeCount;
        tx = begin_paste(@"text over a Qt rich clipboard", board);
        CHECK(tx);
        CHECK(tx.saved.count == 1);
        CHECK([tx.saved[0].types isEqual:@[NSPasteboardTypeString]]);
        CHECK(board.changeCount == carbonChange);
        CHECK(arion_paste_stage((__bridge void *)tx) == ARION_PASTE_OK);
        CHECK(arion_paste_finish(retain_tx(tx)) == 1);
        CHECK(board.pasteboardItems.count == 1);
        CHECK([[board stringForType:NSPasteboardTypeString] isEqual:@"carbon text"]);
        CHECK(write_carbon_board(board, NO));
        CHECK(board.pasteboardItems.count == 1);
        CHECK(begin_paste(@"nothing readable to restore", board) == nil);
        [board clearContents];

        NSURL *file = [NSURL fileURLWithPath:[NSTemporaryDirectory() stringByAppendingPathComponent:NSUUID.UUID.UUIDString]];
        CHECK([NSData.data writeToURL:file atomically:YES]);
        CHECK([board writeObjects:@[file]]);
        tx = begin_paste(@"text over a file clipboard", board);
        CHECK(tx);
        CHECK(arion_paste_stage((__bridge void *)tx) == ARION_PASTE_OK);
        CHECK(arion_paste_finish(retain_tx(tx)) == 1);
        CHECK(tx.fileAccess.count == 0);
        CHECK([board.pasteboardItems[0].types containsObject:NSPasteboardTypeFileURL]);
        NSArray<NSURL *> *restored = [board readObjectsForClasses:@[NSURL.class]
            options:@{NSPasteboardURLReadingFileURLsOnlyKey:@YES}];
        CHECK(restored.count == 1);
        CHECK([restored[0].URLByResolvingSymlinksInPath.path isEqual:file.URLByResolvingSymlinksInPath.path]);
        tx = begin_paste(@"file snapshot loses ownership", board);
        CHECK(tx);
        CHECK(arion_paste_stage((__bridge void *)tx) == ARION_PASTE_OK);
        replace_board(board, @"new user copy replaces the file");
        CHECK(arion_paste_finish(retain_tx(tx)) == 0);
        CHECK(tx.fileAccess.count == 0);
        CHECK([[board stringForType:NSPasteboardTypeString] isEqual:@"new user copy replaces the file"]);
        CHECK([NSFileManager.defaultManager removeItemAtURL:file error:nil]);
        [board releaseGlobally];
        puts("native pasteboard transaction checks passed");
    }
    return 0;
}
