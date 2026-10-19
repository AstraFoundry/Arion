#import "composition.h"
#import <InputMethodKit/InputMethodKit.h>

@interface ArionComposition ()
@property(weak) id client;
@property(readwrite, copy) NSString *activation;
@property(readwrite, copy) NSString *session;
@property(readwrite) BOOL live;
@property(readwrite) BOOL composing;
@property BOOL committed;
@property unsigned long long revision;
@end

@implementation ArionComposition
- (void)activate:(id)client {
    [self invalidate];
    self.client = client;
    self.activation = NSUUID.UUID.UUIDString;
    self.session = nil;
}

- (void)invalidate {
    BOOL clear = self.composing;
    id client = self.client;
    self.composing = NO;
    self.live = NO;
    if (clear) {
        [client setMarkedText:@"" selectionRange:NSMakeRange(0, 0)
            replacementRange:NSMakeRange(NSNotFound, NSNotFound)];
    }
}

- (NSDictionary *)acknowledgement {
    return @{@"op": @"committed", @"activation": self.activation, @"session": self.session};
}

- (NSDictionary *)apply:(NSDictionary *)message {
    if (!self.client || ![message[@"activation"] isEqual:self.activation]) return nil;
    NSString *op = message[@"op"];
    NSString *session = message[@"session"];
    if (![op isKindOfClass:NSString.class] || ![session isKindOfClass:NSString.class] || session.length == 0) return nil;
    if ([op isEqual:@"begin"]) {
        if ([session isEqual:self.session]) return nil;
        [self invalidate];
        self.session = session;
        self.revision = 0;
        self.committed = NO;
        self.live = YES;
        return nil;
    }
    if (![session isEqual:self.session]) return nil;
    if ([op isEqual:@"commit"] && self.committed) return [self acknowledgement];
    if (!self.live) return nil;
    if ([op isEqual:@"cancel"]) {
        [self invalidate];
        return nil;
    }
    NSString *text = message[@"text"];
    if (![text isKindOfClass:NSString.class]) return nil;
    if ([op isEqual:@"mark"]) {
        NSNumber *revision = message[@"revision"];
        if (![revision isKindOfClass:NSNumber.class] || revision.doubleValue < 1 ||
            revision.doubleValue != (double)revision.unsignedLongLongValue ||
            revision.unsignedLongLongValue <= self.revision) return nil;
        NSAttributedString *marked = [[NSAttributedString alloc] initWithString:text
            attributes:@{NSUnderlineStyleAttributeName: @(NSUnderlineStyleSingle)}];
        self.revision = revision.unsignedLongLongValue;
        self.composing = YES;
        [self.client setMarkedText:marked selectionRange:NSMakeRange(text.length, 0)
                  replacementRange:NSMakeRange(NSNotFound, NSNotFound)];
    } else if ([op isEqual:@"commit"]) {
        // Terminal before calling the host, which can reenter the input method.
        self.live = NO;
        self.committed = YES;
        self.composing = NO;
        NSDictionary *ack = [self acknowledgement];
        [self.client insertText:text replacementRange:NSMakeRange(NSNotFound, NSNotFound)];
        return ack;
    }
    return nil;
}
@end
