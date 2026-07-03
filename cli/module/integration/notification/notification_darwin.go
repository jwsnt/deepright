//go:build darwin && cgo

package notification

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework UserNotifications
#include <stdlib.h>
#include <string.h>
#include <dispatch/dispatch.h>
#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>

static char* deeprightCopyCString(NSString* value) {
	if (value == nil) {
		return NULL;
	}
	const char* utf8 = [value UTF8String];
	if (utf8 == NULL) {
		return NULL;
	}
	size_t len = strlen(utf8);
	char* buffer = (char*)malloc(len + 1);
	if (buffer == NULL) {
		return NULL;
	}
	memcpy(buffer, utf8, len + 1);
	return buffer;
}

static int deeprightDeliverNotification(const char* title, const char* message, char** errOut) {
	@autoreleasepool {
		NSString* titleText = title ? [NSString stringWithUTF8String:title] : @"";
		NSString* messageText = message ? [NSString stringWithUTF8String:message] : @"";

		UNUserNotificationCenter* center = [UNUserNotificationCenter currentNotificationCenter];
		dispatch_semaphore_t authSem = dispatch_semaphore_create(0);
		__block BOOL granted = NO;
		__block NSError* authError = nil;
		[center requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
		                      completionHandler:^(BOOL allowed, NSError* error) {
			granted = allowed;
			authError = error;
			dispatch_semaphore_signal(authSem);
		}];
		if (dispatch_semaphore_wait(authSem, dispatch_time(DISPATCH_TIME_NOW, 5 * NSEC_PER_SEC)) != 0) {
			if (errOut != NULL) {
				*errOut = deeprightCopyCString(@"request notification authorization timed out");
			}
			return 1;
		}
		if (authError != nil) {
			if (errOut != NULL) {
				*errOut = deeprightCopyCString([authError localizedDescription]);
			}
			return 1;
		}
		if (!granted) {
			if (errOut != NULL) {
				*errOut = deeprightCopyCString(@"notification permission not granted");
			}
			return 1;
		}

		UNMutableNotificationContent* content = [[UNMutableNotificationContent alloc] init];
		content.title = titleText;
		if ([messageText length] > 0) {
			content.body = messageText;
		}

		NSString* identifier = [[NSUUID UUID] UUIDString];
		UNTimeIntervalNotificationTrigger* trigger =
			[UNTimeIntervalNotificationTrigger triggerWithTimeInterval:1 repeats:NO];
		UNNotificationRequest* request =
			[UNNotificationRequest requestWithIdentifier:identifier content:content trigger:trigger];

		dispatch_semaphore_t addSem = dispatch_semaphore_create(0);
		__block NSError* addError = nil;
		[center addNotificationRequest:request withCompletionHandler:^(NSError* error) {
			addError = error;
			dispatch_semaphore_signal(addSem);
		}];
		if (dispatch_semaphore_wait(addSem, dispatch_time(DISPATCH_TIME_NOW, 5 * NSEC_PER_SEC)) != 0) {
			if (errOut != NULL) {
				*errOut = deeprightCopyCString(@"schedule notification timed out");
			}
			return 1;
		}
		if (addError != nil) {
			if (errOut != NULL) {
				*errOut = deeprightCopyCString([addError localizedDescription]);
			}
			return 1;
		}
		[[NSRunLoop currentRunLoop] runUntilDate:[NSDate dateWithTimeIntervalSinceNow:1.5]];
		return 0;
	}
}
*/
import "C"
import (
	"errors"
	"io"
	"os/exec"
	"strings"
	"unsafe"
)

var errDarwinNotificationFailed = errors.New("darwin notification failed")
var notificationDarwinOpenSettingsURLFn = openDarwinNotificationSettingsURL

func supported() bool {
	return true
}

func settingsSupported() bool {
	return true
}

func notify(opts Options) error {
	title := C.CString(opts.Title)
	defer C.free(unsafe.Pointer(title))

	message := C.CString(opts.Message)
	defer C.free(unsafe.Pointer(message))

	var errOut *C.char
	if C.deeprightDeliverNotification(title, message, &errOut) != 0 {
		defer C.free(unsafe.Pointer(errOut))
		if errOut != nil {
			errMessage := strings.TrimSpace(C.GoString(errOut))
			if errMessage == ErrNotificationsNotAllowed.Error() {
				return ErrNotificationsNotAllowed
			}
			return errors.New(errMessage)
		}
		return errDarwinNotificationFailed
	}
	return nil
}

func openSettings() error {
	var lastErr error
	for _, target := range []string{
		"x-apple.systempreferences:com.apple.Notifications-Settings.extension",
		"x-apple.systempreferences:com.apple.preference.notifications",
	} {
		if err := notificationDarwinOpenSettingsURLFn(target); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return lastErr
}

func openDarwinNotificationSettingsURL(target string) error {
	cmd := exec.Command("open", target)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}
