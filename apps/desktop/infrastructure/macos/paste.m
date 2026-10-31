//go:build darwin

#import <ApplicationServices/ApplicationServices.h>
#import <Carbon/Carbon.h>
#import <Cocoa/Cocoa.h>
#include "native.h"
#include "focus.h"
#include <stdatomic.h>

static NSString *const pasteMarkerType = @"si.astrafoundry.arion.commit-token";
// The nspasteboard.org convention for content that is restored shortly:
// clipboard managers and history tools do not record it.
static NSString *const transientPasteType = @"org.nspasteboard.TransientType";
static _Atomic(long long) historyExcludedRevision = -1;

@interface ArionPasteTransaction : NSObject
@property(strong) NSPasteboard *board;
@property(strong) NSArray<NSPasteboardItem *> *saved;
@property(strong) NSPasteboardItem *item;
@property NSInteger capturedChange;
@property NSInteger ownedChange;
@property BOOL staged;
@property BOOL posted;
@property BOOL markerWritten;
@property CGEventRef down;
@property CGEventRef up;
@end

@implementation ArionPasteTransaction
- (void)dealloc {
    if (_down) CFRelease(_down);
    if (_up) CFRelease(_up);
}
@end

// begin_paste snapshots and allocates everything before mutating the board.
// The board parameter lets isolated tests use an actual named pasteboard.
static ArionPasteTransaction *begin_paste(NSString *text, NSPasteboard *board) {
    if (!text.length) return nil;
    ArionPasteTransaction *tx = [ArionPasteTransaction new];
    tx.board = board;
    tx.capturedChange = board.changeCount;
    tx.ownedChange = -1;
    NSMutableArray<NSPasteboardItem *> *saved = [NSMutableArray array];
    for (NSPasteboardItem *item in board.pasteboardItems) {
        NSPasteboardItem *copy = [NSPasteboardItem new];
        for (NSPasteboardType type in item.types) {
            NSData *data = [item dataForType:type];
            if (!data || ![copy setData:data forType:type]) return nil;
        }
        [saved addObject:copy];
    }
    if (board.changeCount != tx.capturedChange) return nil;
    tx.saved = saved;
    tx.item = [NSPasteboardItem new];
    if (![tx.item setString:text forType:NSPasteboardTypeString] ||
        ![tx.item setString:NSUUID.UUID.UUIDString forType:pasteMarkerType] ||
        ![tx.item setData:[NSData data] forType:transientPasteType]) return nil;
    CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
    if (!source) return nil;
    tx.down = CGEventCreateKeyboardEvent(source, kVK_ANSI_V, true);
    tx.up = CGEventCreateKeyboardEvent(source, kVK_ANSI_V, false);
    CFRelease(source);
    if (!tx.down || !tx.up) return nil;
    CGEventSetFlags(tx.down, kCGEventFlagMaskCommand);
    CGEventSetFlags(tx.up, kCGEventFlagMaskCommand);
    return tx;
}

void *arion_paste_begin(const char *utf8) {
    @autoreleasepool {
        ArionPasteTransaction *tx = begin_paste([NSString stringWithUTF8String:utf8], NSPasteboard.generalPasteboard);
        return tx ? (__bridge_retained void *)tx : NULL;
    }
}

static BOOL owns_pasteboard(ArionPasteTransaction *tx) {
    if (tx.ownedChange < 0 || tx.board.changeCount != tx.ownedChange) return NO;
    return !tx.markerWritten || [[tx.board stringForType:pasteMarkerType] isEqual:[tx.item stringForType:pasteMarkerType]];
}

int arion_paste_stage(void *transaction) {
    @autoreleasepool {
        ArionPasteTransaction *tx = (__bridge ArionPasteTransaction *)transaction;
        if (!tx || tx.staged || tx.ownedChange >= 0) return ARION_PASTE_FAILED;
        if (tx.board.changeCount != tx.capturedChange) return ARION_PASTE_CHANGED;
        tx.ownedChange = [tx.board clearContents];
        if (tx.board == NSPasteboard.generalPasteboard) atomic_store(&historyExcludedRevision, tx.ownedChange);
        if (![tx.board writeObjects:@[tx.item]]) return ARION_PASTE_FAILED;
        tx.markerWritten = YES;
        tx.ownedChange = tx.board.changeCount;
        if (tx.board == NSPasteboard.generalPasteboard) atomic_store(&historyExcludedRevision, tx.ownedChange);
        if (!owns_pasteboard(tx)) return ARION_PASTE_CHANGED;
        tx.staged = YES;
        return ARION_PASTE_OK;
    }
}

int arion_paste_post(void *transaction, void *element, void *selection, int pid) {
    @autoreleasepool {
        ArionPasteTransaction *tx = (__bridge ArionPasteTransaction *)transaction;
        if (!tx || !tx.staged || tx.posted) return ARION_PASTE_FAILED;
        if (!arion_focus_matches(element, pid) || !arion_selection_matches(element, selection)) return ARION_PASTE_TARGET_CHANGED;
        if (arion_secure_input()) return ARION_PASTE_SECURE;
        if (!owns_pasteboard(tx)) return ARION_PASTE_CHANGED;
        // This is the irreversible boundary. Neither this transaction nor
        // its caller retries or switches transport after posting.
        tx.posted = YES;
        CGEventPost(kCGAnnotatedSessionEventTap, tx.down);
        CGEventPost(kCGAnnotatedSessionEventTap, tx.up);
        return ARION_PASTE_OK;
    }
}

int arion_clipboard_publish(const char *utf8) {
    @autoreleasepool {
        if (!utf8) return 0;
        NSString *text = [NSString stringWithUTF8String:utf8];
        if (!text.length) return 0;
        NSPasteboard *board = NSPasteboard.generalPasteboard;
        [board clearContents];
        NSPasteboardItem *item = [NSPasteboardItem new];
        [item setString:text forType:NSPasteboardTypeString];
        [item setString:NSUUID.UUID.UUIDString forType:pasteMarkerType];
        BOOL ok = [board writeObjects:@[item]];
        atomic_store(&historyExcludedRevision, board.changeCount);
        return ok ? 1 : 0;
    }
}

int arion_paste_finish(void *transaction) {
    @autoreleasepool {
        ArionPasteTransaction *tx = (__bridge_transfer ArionPasteTransaction *)transaction;
        if (!tx || !owns_pasteboard(tx)) return 0;
        [tx.board clearContents];
        BOOL ok = !tx.saved.count || [tx.board writeObjects:tx.saved];
        if (tx.board == NSPasteboard.generalPasteboard) atomic_store(&historyExcludedRevision, tx.board.changeCount);
        if (!ok) return -1;
        return 1;
    }
}

long long arion_clipboard_revision(void) {
    @autoreleasepool { return NSPasteboard.generalPasteboard.changeCount; }
}

static int history_excluded(NSPasteboard *board) {
    @autoreleasepool {
        if (board == NSPasteboard.generalPasteboard && board.changeCount == atomic_load(&historyExcludedRevision)) return 1;
        for (NSPasteboardType type in @[pasteMarkerType, @"application/x-arion-internal",
            @"org.nspasteboard.TransientType", @"org.nspasteboard.ConcealedType", @"org.nspasteboard.AutoGeneratedType",
            @"com.agilebits.onepassword", @"net.antelle.keeweb"]) {
            if ([board.types containsObject:type]) return 1;
        }
        return 0;
    }
}

int arion_clipboard_excluded(void) { return history_excluded(NSPasteboard.generalPasteboard); }

int arion_history_activate(int pid) {
    @autoreleasepool {
        if (!AXIsProcessTrusted()) return ARION_TARGET_UNTRUSTED;
        NSRunningApplication *app = [NSRunningApplication runningApplicationWithProcessIdentifier:pid];
        if (!app || pid == NSProcessInfo.processInfo.processIdentifier || ![app activateWithOptions:0]) return ARION_TARGET_NONE;
        return ARION_TARGET_OK;
    }
}

int arion_history_post(int pid, void *element, long long revision) {
    @autoreleasepool {
        if (!AXIsProcessTrusted()) return ARION_PASTE_FAILED;
        if (!arion_history_check(pid,element)) return ARION_PASTE_TARGET_CHANGED;
        if (arion_secure_input()) return ARION_PASTE_SECURE;
        if (NSPasteboard.generalPasteboard.changeCount != revision) return ARION_PASTE_CHANGED;
        CGEventRef down = CGEventCreateKeyboardEvent(NULL,kVK_ANSI_V,true);
        CGEventRef up = CGEventCreateKeyboardEvent(NULL,kVK_ANSI_V,false);
        if (!down || !up) { if (down) CFRelease(down); if (up) CFRelease(up); return ARION_PASTE_FAILED; }
        CGEventSetFlags(down,kCGEventFlagMaskCommand); CGEventSetFlags(up,kCGEventFlagMaskCommand);
        CGEventPost(kCGAnnotatedSessionEventTap,down); CGEventPost(kCGAnnotatedSessionEventTap,up);
        CFRelease(down); CFRelease(up);
        return ARION_PASTE_OK;
    }
}

void *arion_history_focus(int pid, int *secure, int *no_input) {
    @autoreleasepool {
        *secure = arion_secure_input();
        if (no_input) *no_input = 0;
        if (!AXIsProcessTrusted()) return NULL;
        arion_enable_web_accessibility(pid);
        AXUIElementRef app = AXUIElementCreateApplication(pid);
        AXUIElementSetMessagingTimeout(app,0.2);
        AXUIElementRef element = arion_copy_focus(app, *secure, no_input, NULL);
        CFRelease(app);
        if (element) {
            AXUIElementSetMessagingTimeout((AXUIElementRef)element,0.2);
            CFTypeRef subrole = NULL;
            AXUIElementCopyAttributeValue((AXUIElementRef)element,kAXSubroleAttribute,&subrole);
            if (subrole) { if (CFEqual(subrole,kAXSecureTextFieldSubrole)) *secure = 1; CFRelease(subrole); }
        }
        return (void *)element;
    }
}

int arion_history_check(int pid, void *element) {
    if (!AXIsProcessTrusted() || arion_secure_input() || !arion_focus_matches(element,pid)) return 0;
    int secure = 0;
    void *current = arion_history_focus(pid,&secure,NULL);
    if (current) CFRelease(current);
    return !secure;
}
