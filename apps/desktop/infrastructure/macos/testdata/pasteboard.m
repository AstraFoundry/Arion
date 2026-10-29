#import <ApplicationServices/ApplicationServices.h>
#import <Carbon/Carbon.h>
#import <Cocoa/Cocoa.h>

static int focus_ok = 1, selection_ok = 1, secure = 0, posts = 0;
int arion_focus_matches(void *element, int pid) { return focus_ok; }
int arion_selection_matches(void *element, void *selection) { return selection_ok; }
int arion_secure_input(void) { return secure; }

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

int main(void) {
    @autoreleasepool {
        NSPasteboard *board = [NSPasteboard pasteboardWithUniqueName];
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
        [board releaseGlobally];
        puts("native pasteboard transaction checks passed");
    }
    return 0;
}
