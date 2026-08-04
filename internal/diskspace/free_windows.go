//go:build windows

package diskspace

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func FreeBytes(path string) (uint64, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return 0, err
	}
	probe := absolute
	for {
		if _, statErr := os.Stat(probe); statErr == nil {
			break
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			break
		}
		probe = parent
	}
	pointer, err := windows.UTF16PtrFromString(probe)
	if err != nil {
		return 0, err
	}
	var available uint64
	if err := windows.GetDiskFreeSpaceEx(pointer, &available, nil, nil); err != nil {
		return 0, err
	}
	return available, nil
}
