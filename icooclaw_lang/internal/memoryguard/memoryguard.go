package memoryguard

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync/atomic"
)

const (
	maxMemoryEnv                   = "ICLANG_MAX_MEMORY_MB"
	maxMemoryPercentEnv            = "ICLANG_MAX_MEMORY_PERCENT"
	defaultMaxMemoryPercent        = 90
	checkInterval           uint64 = 256
)

var (
	limitBytes          atomic.Int64
	checkCounter        atomic.Uint64
	hostMemoryTotalFunc = HostMemoryTotalBytes
	limitPercentValue   atomic.Int64
)

func ResolveLimitBytes(cliMaxMemoryMB, cliMaxMemoryPercent int) (int64, error) {
	if cliMaxMemoryMB > 0 {
		return int64(cliMaxMemoryMB) * 1024 * 1024, nil
	}

	if cliMaxMemoryPercent > 0 {
		return resolveLimitBytesFromPercent(cliMaxMemoryPercent)
	}

	if value := os.Getenv(maxMemoryEnv); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			return int64(parsed) * 1024 * 1024, nil
		}
	}

	if value := os.Getenv(maxMemoryPercentEnv); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			return resolveLimitBytesFromPercent(parsed)
		}
	}

	return resolveLimitBytesFromPercent(defaultMaxMemoryPercent)
}

func resolveLimitBytesFromPercent(percent int) (int64, error) {
	if percent <= 0 || percent > 100 {
		return 0, fmt.Errorf("invalid memory percent threshold: %d", percent)
	}

	total, err := hostMemoryTotalFunc()
	if err != nil {
		return 0, fmt.Errorf("could not determine host memory: %w", err)
	}
	if total == 0 {
		return 0, fmt.Errorf("could not determine host memory: total memory is 0")
	}

	limit := int64((total * uint64(percent)) / 100)
	if limit <= 0 {
		return 0, fmt.Errorf("invalid memory limit derived from host memory")
	}
	return limit, nil
}

func Activate(limit int64) func() {
	if limit <= 0 {
		limitBytes.Store(0)
		limitPercentValue.Store(0)
		checkCounter.Store(0)
		return func() {}
	}

	previous := debug.SetMemoryLimit(limit)
	limitBytes.Store(limit)
	checkCounter.Store(0)

	return func() {
		limitBytes.Store(0)
		limitPercentValue.Store(0)
		checkCounter.Store(0)
		debug.SetMemoryLimit(previous)
	}
}

func SetActivePercent(percent int) {
	if percent <= 0 {
		limitPercentValue.Store(0)
		return
	}
	limitPercentValue.Store(int64(percent))
}

func Checkpoint() error {
	limit := limitBytes.Load()
	if limit <= 0 {
		return nil
	}

	count := checkCounter.Add(1)
	if count%checkInterval != 0 {
		return nil
	}

	return CheckNow()
}

func CheckNow() error {
	limit := limitBytes.Load()
	if limit <= 0 {
		return nil
	}

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	current := stats.Alloc
	if stats.HeapAlloc > current {
		current = stats.HeapAlloc
	}
	if stats.Sys > current {
		current = stats.Sys
	}

	if current < uint64(limit) {
		return nil
	}

	return fmt.Errorf(
		"memory limit exceeded: current=%dMB alloc=%dMB sys=%dMB limit=%dMB",
		bytesToMB(current),
		bytesToMB(stats.Alloc),
		bytesToMB(stats.Sys),
		bytesToMB(uint64(limit)),
	)
}

func bytesToMB(v uint64) uint64 {
	return v / 1024 / 1024
}

type Stats struct {
	LimitBytes       int64
	LimitPercent     int64
	AllocBytes       uint64
	HeapAllocBytes   uint64
	SysBytes         uint64
	HostTotalBytes   uint64
	HostUsagePercent int64
}

func CurrentStats() Stats {
	var runtimeStats runtime.MemStats
	runtime.ReadMemStats(&runtimeStats)

	stats := Stats{
		LimitBytes:     limitBytes.Load(),
		LimitPercent:   limitPercentValue.Load(),
		AllocBytes:     runtimeStats.Alloc,
		HeapAllocBytes: runtimeStats.HeapAlloc,
		SysBytes:       runtimeStats.Sys,
	}

	if total, err := hostMemoryTotalFunc(); err == nil && total > 0 {
		stats.HostTotalBytes = total
		if stats.SysBytes > 0 {
			stats.HostUsagePercent = int64((stats.SysBytes * 100) / total)
		}
	}

	return stats
}
