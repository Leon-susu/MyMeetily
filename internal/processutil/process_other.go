//go:build !windows

package processutil

import (
	"os/exec"
	"runtime"
)

// HideWindow is a no-op on non-Windows platforms.
func HideWindow(_ *exec.Cmd) {}

func OpenPath(path string) error {
	return openWithSystem(path)
}

func OpenFolder(path string) error {
	return openWithSystem(path)
}

func openWithSystem(path string) error {
	command := "xdg-open"
	if runtime.GOOS == "darwin" {
		command = "open"
	}
	return exec.Command(command, path).Start()
}
