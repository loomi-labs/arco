//go:build linux

package platform

// ShowDockIcon is a no-op on Linux.
// On macOS, this would show the application icon in the dock.
func ShowDockIcon() {
	// No-op on Linux
}

// HideDockIcon is a no-op on Linux.
// On macOS, this would hide the application icon from the dock.
func HideDockIcon() {
	// No-op on Linux
}

// HasFullDiskAccess always returns true on Linux.
// Full Disk Access is a macOS-specific permission.
func HasFullDiskAccess() bool {
	return true
}
