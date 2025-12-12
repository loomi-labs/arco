//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

void showDockIcon() {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
        [NSApp activateIgnoringOtherApps:YES];
    });
}

void hideDockIcon() {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    });
}
*/
import "C"

// ShowDockIcon shows the application icon in the macOS dock.
// This should be called when a window is opened.
func ShowDockIcon() {
	C.showDockIcon()
}

// HideDockIcon hides the application icon from the macOS dock.
// This should be called when all windows are closed.
func HideDockIcon() {
	C.hideDockIcon()
}
