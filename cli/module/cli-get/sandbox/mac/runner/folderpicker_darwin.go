//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>
#import <stdlib.h>

static char* runnerChooseFolderNative(const char *initialPath, double timeoutSeconds, char **errOut) {
	@autoreleasepool {
		[NSApplication sharedApplication];
		[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
		[NSApp finishLaunching];
		[NSApp activateIgnoringOtherApps:YES];

		NSOpenPanel *panel = [NSOpenPanel openPanel];
		[panel setCanChooseFiles:NO];
		[panel setCanChooseDirectories:YES];
		[panel setAllowsMultipleSelection:NO];
		[panel setCanCreateDirectories:NO];
		[panel setResolvesAliases:YES];
		[panel setPrompt:@"允许"];
		[panel setMessage:@"CLI_SANDBOX 请选择允许访问的目录"];
		if (initialPath != NULL && initialPath[0] != '\0') {
			NSString *initialDir = [NSString stringWithUTF8String:initialPath];
			if (initialDir != nil && [initialDir length] > 0) {
				NSURL *initialURL = [NSURL fileURLWithPath:initialDir isDirectory:YES];
				if (initialURL != nil) {
					[panel setDirectoryURL:initialURL];
				}
			}
		}

		__block BOOL timedOut = NO;
		if (timeoutSeconds > 0) {
			dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(timeoutSeconds * NSEC_PER_SEC)), dispatch_get_main_queue(), ^{
				if ([panel isVisible]) {
					timedOut = YES;
					[NSApp abortModal];
					[panel orderOut:nil];
				}
			});
		}

		NSInteger response = [panel runModal];
		if (timedOut) {
			*errOut = strdup("目录授权弹窗超时，请切回桌面确认选择窗口后重试");
			return NULL;
		}
		if (response == NSModalResponseCancel) {
			*errOut = strdup("已取消目录授权");
			return NULL;
		}
		if (response != NSModalResponseOK) {
			NSString *message = [NSString stringWithFormat:@"目录授权失败: %ld", (long)response];
			*errOut = strdup([message UTF8String]);
			return NULL;
		}

		NSURL *selectedURL = [[panel URLs] firstObject];
		if (selectedURL == nil) {
			*errOut = strdup("目录授权失败: 未选择目录");
			return NULL;
		}
		NSString *path = [selectedURL path];
		if (path == nil || [path length] == 0) {
			*errOut = strdup("目录授权失败: 目录路径为空");
			return NULL;
		}
		return strdup([path UTF8String]);
	}
}
*/
import "C"

import (
	"errors"
	"runtime"
	"strings"
	"time"
	"unsafe"
)

func sandboxChooseFolderWithTimeout(timeout time.Duration) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	initialPath := defaultMacPickerDirectory()
	var initialPathText *C.char
	if initialPath != "" {
		initialPathText = C.CString(initialPath)
		defer C.free(unsafe.Pointer(initialPathText))
	}
	var errText *C.char
	result := C.runnerChooseFolderNative(initialPathText, C.double(timeout.Seconds()), &errText)
	if errText != nil {
		defer C.free(unsafe.Pointer(errText))
		return "", errors.New(strings.TrimSpace(C.GoString(errText)))
	}
	if result == nil {
		return "", errors.New("目录授权失败")
	}
	defer C.free(unsafe.Pointer(result))
	return strings.TrimSpace(C.GoString(result)), nil
}
