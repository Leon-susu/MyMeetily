//go:build windows

package processutil

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

const createNoWindow = 0x08000000

// HideWindow prevents command-line helpers from flashing console windows when
// launched by the desktop application.
func HideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}

// OpenPath asks Windows to open a file with its registered default app.
func OpenPath(path string) error {
	return windows.ShellExecute(
		0,
		windows.StringToUTF16Ptr("open"),
		windows.StringToUTF16Ptr(path),
		nil,
		nil,
		windows.SW_SHOWNORMAL,
	)
}

// OpenFolder opens a directory in File Explorer.
func OpenFolder(path string) error {
	return windows.ShellExecute(
		0,
		windows.StringToUTF16Ptr("explore"),
		windows.StringToUTF16Ptr(path),
		nil,
		nil,
		windows.SW_SHOWNORMAL,
	)
}
