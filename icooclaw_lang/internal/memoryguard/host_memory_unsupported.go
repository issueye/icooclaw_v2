//go:build !windows && !linux && !darwin

package memoryguard

import "fmt"

func HostMemoryTotalBytes() (uint64, error) {
	return 0, fmt.Errorf("unsupported platform")
}
