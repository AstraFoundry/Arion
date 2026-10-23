#import <Foundation/Foundation.h>
#include <sys/mount.h>
#include <errno.h>
int main(int argc, const char **argv) {
    @autoreleasepool {
        if (argc != 2) return 2;
        NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:argv[1]] isDirectory:YES];
        NSNumber *supported = nil;
        NSError *error = nil;
        BOOL obtained = [url getResourceValue:&supported forKey:NSURLVolumeSupportsSwapRenamingKey error:&error];
        struct statfs volume;
        int status = statfs(argv[1], &volume);
        NSDictionary *result = @{@"resourceQuerySucceeded": @(obtained),
            @"swapSupported": supported ?: NSNull.null,
            @"resourceErrorCode": error ? @(error.code) : @0,
            @"filesystem": status == 0 ? [NSString stringWithUTF8String:volume.f_fstypename] : @"unknown",
            @"statfsErrno": status == 0 ? @0 : @(errno)};
        NSData *data = [NSJSONSerialization dataWithJSONObject:result options:NSJSONWritingPrettyPrinted error:nil];
        puts([[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding].UTF8String);
        return obtained && status == 0 ? 0 : 1;
    }
}
