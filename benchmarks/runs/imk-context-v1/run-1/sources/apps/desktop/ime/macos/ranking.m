#import "ranking.h"

// One confirmation in context outweighs a two-place producer gap, two outweigh
// four places and three outweigh eight: ln(1 + n) * weight against ln(1 + rank).
static const double kWeight = 1.6;
// Counts lose half their weight after this many later confirmations.
static const double kHalfLife = 2000;
// Bounded store: the oldest pairs go first once it grows past the limit.
static const NSUInteger kMaximumPairs = 20000;
static const NSUInteger kEvictBatch = 2000;
// Longer commits are sentences, too sparse to recur as one unit.
static const NSUInteger kMaximumWordLength = 12;
static NSString *const kTickKey = @"@tick";

static NSUInteger composedLength(NSString *text) {
    __block NSUInteger count = 0;
    [text enumerateSubstringsInRange:NSMakeRange(0, text.length) options:NSStringEnumerationByComposedCharacterSequences |
        NSStringEnumerationSubstringNotRequired usingBlock:^(__unused NSString *s, __unused NSRange r,
        __unused NSRange e, __unused BOOL *stop) { count++; }];
    return count;
}

static NSString *trimmed(NSString *text) {
    static NSCharacterSet *trailing;
    static dispatch_once_t once;
    dispatch_once(&once, ^{
        NSMutableCharacterSet *set = [NSMutableCharacterSet whitespaceAndNewlineCharacterSet];
        [set formUnionWithCharacterSet:NSCharacterSet.punctuationCharacterSet];
        [set formUnionWithCharacterSet:NSCharacterSet.symbolCharacterSet];
        trailing = set;
    });
    NSUInteger end = text.length;
    while (end > 0 && [trailing characterIsMember:[text characterAtIndex:end - 1]]) end--;
    return [text substringToIndex:end];
}

static NSString *pairKey(NSString *context, NSString *word) {
    return [NSString stringWithFormat:@"c\t%@\t%@", context, word];
}

@interface ArionContextRanker ()
@property(strong) NSMutableDictionary<NSString *, NSString *> *records;
@property(copy) void (^persist)(NSDictionary<NSString *, id> *changes);
@property unsigned long long tick;
@end

@implementation ArionContextRanker
- (instancetype)initWithRecords:(NSDictionary<NSString *, NSString *> *)records
                        persist:(void (^)(NSDictionary<NSString *, id> *))persist {
    if (!(self = [super init])) return nil;
    self.records = [records mutableCopy];
    self.persist = persist;
    self.tick = strtoull(records[kTickKey].UTF8String ?: "0", NULL, 10);
    return self;
}

- (NSUInteger)count { return self.records.count; }

+ (NSString *)contextOf:(NSString *)commit {
    NSString *text = trimmed(commit ?: @"");
    if (text.length == 0) return nil;
    // Last two composed characters: one word of context, as Doubao's user bigrams.
    __block NSUInteger start = text.length, taken = 0;
    [text enumerateSubstringsInRange:NSMakeRange(0, text.length)
        options:NSStringEnumerationByComposedCharacterSequences | NSStringEnumerationReverse | NSStringEnumerationSubstringNotRequired
        usingBlock:^(__unused NSString *s, NSRange range, __unused NSRange e, BOOL *stop) {
            start = range.location;
            if (++taken == 2) *stop = YES;
        }];
    return [text substringFromIndex:start];
}

- (double)boostFor:(NSString *)word context:(NSString *)context {
    NSString *value = self.records[pairKey(context, word)];
    if (!value) return 0;
    unsigned long long count = 0, tick = 0;
    if (sscanf(value.UTF8String, "%llu %llu", &count, &tick) != 2 || count == 0) return 0;
    double age = self.tick > tick ? (double)(self.tick - tick) : 0;
    return kWeight * log1p((double)count) * pow(0.5, age / kHalfLife);
}

- (NSArray<NSNumber *> *)order:(NSArray<NSString *> *)candidates after:(NSString *)previous {
    NSMutableArray<NSNumber *> *identity = [NSMutableArray arrayWithCapacity:candidates.count];
    for (NSUInteger i = 0; i < candidates.count; i++) [identity addObject:@(i)];
    NSString *context = [ArionContextRanker contextOf:previous];
    if (!context || candidates.count < 2) return identity;
    NSUInteger full = composedLength(candidates[0]);
    NSMutableArray<NSNumber *> *scores = [NSMutableArray arrayWithCapacity:candidates.count];
    BOOL boosted = NO;
    for (NSUInteger i = 0; i < candidates.count; i++) {
        // Shorter candidates convert only a prefix; promoting one would split
        // the user's input unexpectedly.
        double boost = composedLength(candidates[i]) == full ? [self boostFor:candidates[i] context:context] : 0;
        boosted = boosted || boost > 0;
        [scores addObject:@(boost - log1p((double)i))];
    }
    if (!boosted) return identity;
    return [identity sortedArrayWithOptions:NSSortStable usingComparator:^NSComparisonResult(NSNumber *a, NSNumber *b) {
        double x = scores[a.unsignedIntegerValue].doubleValue, y = scores[b.unsignedIntegerValue].doubleValue;
        return x > y ? NSOrderedAscending : x < y ? NSOrderedDescending : NSOrderedSame;
    }];
}

- (void)learn:(NSString *)commit after:(NSString *)previous {
    // Punctuation that committed the word is not part of it.
    NSString *word = trimmed(commit);
    NSString *context = [ArionContextRanker contextOf:previous];
    NSUInteger length = composedLength(word);
    if (!context || length == 0 || length > kMaximumWordLength) return;
    self.tick++;
    NSString *key = pairKey(context, word);
    unsigned long long count = 0, tick = 0;
    sscanf((self.records[key] ?: @"").UTF8String, "%llu %llu", &count, &tick);
    NSMutableDictionary<NSString *, id> *changes = [NSMutableDictionary dictionary];
    changes[key] = [NSString stringWithFormat:@"%llu %llu", count + 1, self.tick];
    changes[kTickKey] = [NSString stringWithFormat:@"%llu", self.tick];
    [self.records addEntriesFromDictionary:changes];
    if (self.records.count > kMaximumPairs + 1) {
        NSMutableArray<NSString *> *pairs = [NSMutableArray array];
        for (NSString *candidate in self.records) if ([candidate hasPrefix:@"c\t"]) [pairs addObject:candidate];
        [pairs sortUsingComparator:^NSComparisonResult(NSString *a, NSString *b) {
            unsigned long long x = 0, y = 0, unused = 0;
            sscanf(self.records[a].UTF8String, "%llu %llu", &unused, &x);
            sscanf(self.records[b].UTF8String, "%llu %llu", &unused, &y);
            return x < y ? NSOrderedAscending : x > y ? NSOrderedDescending : NSOrderedSame;
        }];
        for (NSString *stale in [pairs subarrayWithRange:NSMakeRange(0, MIN(kEvictBatch, pairs.count))]) {
            [self.records removeObjectForKey:stale];
            changes[stale] = NSNull.null;
        }
    }
    if (self.persist) self.persist(changes);
}
@end
