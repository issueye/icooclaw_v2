package memoryguard

import (
	"os"
	"strings"
	"testing"
)

func withHostMemoryTotalStub(total uint64, fn func()) {
	original := hostMemoryTotalFunc
	hostMemoryTotalFunc = func() (uint64, error) {
		return total, nil
	}
	defer func() {
		hostMemoryTotalFunc = original
	}()
	fn()
}

func TestResolveLimitBytesPrefersCLI(t *testing.T) {
	t.Setenv(maxMemoryEnv, "64")

	got, err := ResolveLimitBytes(32, 0)
	if err != nil {
		t.Fatalf("ResolveLimitBytes() error = %v", err)
	}
	want := int64(32 * 1024 * 1024)
	if got != want {
		t.Fatalf("ResolveLimitBytes() = %d, want %d", got, want)
	}
}

func TestResolveLimitBytesFallsBackToEnv(t *testing.T) {
	t.Setenv(maxMemoryEnv, "48")

	got, err := ResolveLimitBytes(0, 0)
	if err != nil {
		t.Fatalf("ResolveLimitBytes() error = %v", err)
	}
	want := int64(48 * 1024 * 1024)
	if got != want {
		t.Fatalf("ResolveLimitBytes() = %d, want %d", got, want)
	}
}

func TestResolveLimitBytesSupportsPercentThreshold(t *testing.T) {
	withHostMemoryTotalStub(1000, func() {
		got, err := ResolveLimitBytes(0, 80)
		if err != nil {
			t.Fatalf("ResolveLimitBytes() error = %v", err)
		}
		if got != 800 {
			t.Fatalf("ResolveLimitBytes() = %d, want 800", got)
		}
	})
}

func TestResolveLimitBytesDefaultsToNinetyPercentOfHostMemory(t *testing.T) {
	t.Setenv(maxMemoryEnv, "")
	t.Setenv(maxMemoryPercentEnv, "")

	withHostMemoryTotalStub(1000, func() {
		got, err := ResolveLimitBytes(0, 0)
		if err != nil {
			t.Fatalf("ResolveLimitBytes() error = %v", err)
		}
		if got != 900 {
			t.Fatalf("ResolveLimitBytes() = %d, want 900", got)
		}
	})
}

func TestCheckNowReturnsErrorWhenLimitIsTooLow(t *testing.T) {
	restoreEnv := os.Getenv(maxMemoryEnv)
	defer os.Setenv(maxMemoryEnv, restoreEnv)

	restore := Activate(1)
	defer restore()

	err := CheckNow()
	if err == nil {
		t.Fatal("expected CheckNow() to fail when limit is below current process usage")
	}
	if !strings.Contains(err.Error(), "memory limit exceeded") {
		t.Fatalf("unexpected error: %v", err)
	}
}
