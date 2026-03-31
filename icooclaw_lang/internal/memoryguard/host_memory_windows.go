//go:build windows

package memoryguard

import (
	"fmt"
	"syscall"
	"unsafe"
)

type memoryStatusEx struct {
	length               uint32
	memoryLoad           uint32
	totalPhys            uint64
	availPhys            uint64
	totalPageFile        uint64
	availPageFile        uint64
	totalVirtual         uint64
	availVirtual         uint64
	availExtendedVirtual uint64
}

func HostMemoryTotalBytes() (uint64, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GlobalMemoryStatusEx")

	status := memoryStatusEx{
		length: uint32(unsafe.Sizeof(memoryStatusEx{})),
	}

	ret, _, callErr := proc.Call(uintptr(unsafe.Pointer(&status)))
	if ret == 0 {
		if callErr != syscall.Errno(0) {
			return 0, callErr
		}
		return 0, fmt.Errorf("GlobalMemoryStatusEx failed")
	}

	return status.totalPhys, nil
}
