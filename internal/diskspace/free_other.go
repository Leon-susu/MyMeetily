//go:build !windows

package diskspace

func FreeBytes(string) (uint64, error) {
	return ^uint64(0), nil
}
