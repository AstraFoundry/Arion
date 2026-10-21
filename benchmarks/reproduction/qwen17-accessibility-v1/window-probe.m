#import <Cocoa/Cocoa.h>
#import <ApplicationServices/ApplicationServices.h>
#include "../../../apps/desktop/infrastructure/macos/context.h"
#include "../../../apps/desktop/infrastructure/macos/native.h"

int main(int argc, char **argv) {
    @autoreleasepool {
        if (argc != 3) return 2;
        NSArray *cases = [NSJSONSerialization JSONObjectWithData:[NSData dataWithContentsOfFile:@(argv[1])] options:0 error:nil];
        FILE *output = fopen(argv[2], "wx");
        if (!cases || !output) return 2;
        [NSApplication sharedApplication];
        [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
        NSWindow *window = [[NSWindow alloc] initWithContentRect:NSMakeRect(100,100,900,640) styleMask:NSWindowStyleMaskTitled backing:NSBackingStoreBuffered defer:NO];
        window.title = @"Arion owned accessibility evaluation";
        NSTextField *text = [[NSTextField alloc] initWithFrame:NSMakeRect(20,100,860,500)];
        text.editable = NO; text.selectable = YES; text.bordered = NO;
        text.font = [NSFont systemFontOfSize:18];
        [window.contentView addSubview:text];
        NSTextField *focus = [[NSTextField alloc] initWithFrame:NSMakeRect(20,30,300,30)];
        [window.contentView addSubview:focus];
        NSSecureTextField *password = [[NSSecureTextField alloc] initWithFrame:NSMakeRect(400,30,300,30)];
        password.stringValue = @"SECRET_CANARY_NEVER_READ";
        [window.contentView addSubview:password];
        NSTextField *hidden = [NSTextField labelWithString:@"HIDDEN_CANARY_NEVER_READ"];
        hidden.hidden = YES; [window.contentView addSubview:hidden];
        [window makeKeyAndOrderFront:nil]; [window makeFirstResponder:focus];
        [NSApp activateIgnoringOtherApps:YES];
        dispatch_async(dispatch_get_global_queue(QOS_CLASS_USER_INITIATED,0), ^{
            [NSThread sleepForTimeInterval:0.5];
            for (NSDictionary *item in cases) {
                dispatch_sync(dispatch_get_main_queue(), ^{ text.stringValue = item[@"text"]; [window displayIfNeeded]; });
                [NSThread sleepForTimeInterval:0.15];
                arion_target target = {0};
                int capture = arion_capture_target(&target,0);
                for (int repeat=0; repeat<3; repeat++) {
                    char *ax = NULL, *ocr = NULL;
                    double axMs=0, captureMs=0, ocrMs=0;
                    int axStatus = capture == ARION_TARGET_OK ? arion_accessibility_window(target.element,target.pid,1.5,&ax,&axMs) : -capture;
                    int ocrStatus = arion_ocr_window(getpid(),ARION_OCR_ACCURATE,1.5,&ocr,&captureMs,&ocrMs);
                    NSDictionary *row = @{ @"id":item[@"id"], @"repeat":@(repeat), @"axStatus":@(axStatus), @"axText":ax ? @(ax) : @"", @"axMs":@(axMs), @"ocrStatus":@(ocrStatus), @"ocrText":ocr ? @(ocr) : @"", @"captureMs":@(captureMs), @"ocrMs":@(ocrMs), @"axTrusted":@(AXIsProcessTrusted()), @"captureStatus":@(capture)};
                    NSData *data=[NSJSONSerialization dataWithJSONObject:row options:NSJSONWritingSortedKeys error:nil];
                    fwrite(data.bytes,1,data.length,output); fputc('\n',output); fflush(output);
                    free(ax); free(ocr);
                }
                if(target.element) arion_release(target.element);
                if(target.selection) arion_release(target.selection);
            }
            fclose(output);
            dispatch_async(dispatch_get_main_queue(), ^{ [NSApp terminate:nil]; });
        });
        [NSApp run];
    }
    return 0;
}
