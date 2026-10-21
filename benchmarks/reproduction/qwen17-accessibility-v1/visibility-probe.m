#import <Cocoa/Cocoa.h>
#import <ApplicationServices/ApplicationServices.h>
#include "../../../apps/desktop/infrastructure/macos/context.h"
#include "../../../apps/desktop/infrastructure/macos/native.h"

int main(int argc, char **argv) {
    @autoreleasepool {
        if(argc != 2) return 2;
        FILE *output = fopen(argv[1], "wx");
        if(!output) return 2;
        [NSApplication sharedApplication];
        [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
        NSWindow *window = [[NSWindow alloc] initWithContentRect:NSMakeRect(100,100,700,400) styleMask:NSWindowStyleMaskTitled backing:NSBackingStoreBuffered defer:NO];
        window.title = @"Arion visible text controls";
        NSScrollView *scroll = [[NSScrollView alloc] initWithFrame:NSMakeRect(20,100,650,260)];
        NSTextView *editor = [[NSTextView alloc] initWithFrame:NSMakeRect(0,0,630,5000)];
        NSMutableString *content = [NSMutableString stringWithString:@"VISIBLE_TOP\n"];
        for(int i=0;i<200;i++) [content appendFormat:@"Visible-range fixture line %d\n",i];
        [content appendString:@"OFFSCREEN_END_CANARY"];
        editor.string = content;
        scroll.documentView = editor;
        [window.contentView addSubview:scroll];
        NSTextField *hidden = [NSTextField labelWithString:@"HIDDEN_CANARY_NEVER_READ"];
        hidden.hidden=YES; [window.contentView addSubview:hidden];
        NSSecureTextField *password = [[NSSecureTextField alloc] initWithFrame:NSMakeRect(20,30,300,30)];
        password.stringValue=@"SECRET_CANARY_NEVER_READ"; [window.contentView addSubview:password];
        NSTextField *other = [[NSTextField alloc] initWithFrame:NSMakeRect(350,30,300,30)];
        [window.contentView addSubview:other];
        [window makeKeyAndOrderFront:nil]; [window makeFirstResponder:editor];
        [editor setSelectedRange:NSMakeRange(0,0)]; [editor scrollRangeToVisible:NSMakeRange(0,0)];
        [NSApp activateIgnoringOtherApps:YES];
        dispatch_async(dispatch_get_global_queue(QOS_CLASS_USER_INITIATED,0), ^{
            [NSThread sleepForTimeInterval:.5];
            arion_target target={0};
            int capture=arion_capture_target(&target,0);
            char *text=NULL;
            double ms=0;
            int status=arion_accessibility_window(target.element,target.pid,1.5,&text,&ms);
            NSString *value=text ? @(text) : @"";
            BOOL visible = status==0 && [value containsString:@"VISIBLE_TOP"] && ![value containsString:@"OFFSCREEN_END_CANARY"] && ![value containsString:@"CANARY_NEVER_READ"];
            free(text); text=NULL;
            dispatch_sync(dispatch_get_main_queue(), ^{ [window makeFirstResponder:other]; });
            int stale=arion_accessibility_window(target.element,target.pid,1.5,&text,&ms);
            BOOL rejectedStale=stale!=0 && text==NULL;
            free(text); text=NULL;
            if(target.element) arion_release(target.element);
            if(target.selection) arion_release(target.selection);
            dispatch_sync(dispatch_get_main_queue(), ^{
                for(int i=0;i<520;i++) {
                    NSTextField *label=[NSTextField labelWithString:@"bounded node"];
                    label.frame=NSMakeRect(30,100,200,20); [window.contentView addSubview:label];
                }
            });
            target=(arion_target){0}; arion_capture_target(&target,0);
            int bounded=arion_accessibility_window(target.element,target.pid,1.5,&text,&ms);
            BOOL rejectedLimit=bounded!=0 && text==NULL;
            free(text);
            if(target.element) arion_release(target.element);
            if(target.selection) arion_release(target.selection);
            NSDictionary *result=@{@"captureStatus":@(capture),@"visibleRangeStatus":@(status),@"visibleRangeAndCanariesPass":@(visible),@"staleStatus":@(stale),@"staleRejected":@(rejectedStale),@"limitStatus":@(bounded),@"limitRejectedWithoutPartialText":@(rejectedLimit),@"text":value};
            NSData *data=[NSJSONSerialization dataWithJSONObject:result options:NSJSONWritingSortedKeys error:nil];
            fwrite(data.bytes,1,data.length,output); fclose(output);
            dispatch_async(dispatch_get_main_queue(), ^{ [NSApp terminate:nil]; });
        });
        [NSApp run];
    }
    return 0;
}
