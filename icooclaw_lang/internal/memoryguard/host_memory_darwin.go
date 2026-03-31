//go:build darwin

package memoryguard

import "golang.org/x/sys/unix"

func HostMemoryTotalBytes() (uint64, error) {
	return unix.SysctlUint64("hw.memsize")
}
