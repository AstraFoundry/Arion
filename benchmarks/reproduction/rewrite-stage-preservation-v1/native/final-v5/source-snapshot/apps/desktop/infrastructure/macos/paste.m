//go:build darwin

#import <ApplicationServices/ApplicationServices.h>
#import <Carbon/Carbon.h>
#import <Cocoa/Cocoa.h>
#include "native.h"

static NSString *const pasteMarkerType = @"si.astrafoundry.arion.commit-token";

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
        ![tx.item setString:NSUUID.UUID.UUIDString forType:pasteMarkerType]) return nil;
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
        if (![tx.board writeObjects:@[tx.item]]) return ARION_PASTE_FAILED;
        tx.markerWritten = YES;
        tx.ownedChange = tx.board.changeCount;
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

int arion_paste_finish(void *transaction) {
    @autoreleasepool {
        ArionPasteTransaction *tx = (__bridge_transfer ArionPasteTransaction *)transaction;
        if (!tx || !owns_pasteboard(tx)) return 0;
        [tx.board clearContents];
        if (tx.saved.count && ![tx.board writeObjects:tx.saved]) return -1;
        return 1;
    }
}
