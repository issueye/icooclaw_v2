// Package script provides JavaScript scripting engine for icooclaw.
package script

const (
	errFileReadNotAllowed   = "file reading is not allowed"
	errFileWriteNotAllowed  = "file writing is not allowed"
	errFileDeleteNotAllowed = "file deletion is not allowed"
	errFileOpsNotAllowed    = "file operations not allowed"
	errNetworkNotAllowed    = "network access is not allowed"
	errShellNotAllowed      = "shell execution is not allowed"
	errStorageNotConfigured = "storage is not configured"

	defaultExecTimeoutSeconds = 30
	defaultHTTPTimeoutSeconds = 30
	defaultMaxCallStackSize   = 100
	defaultMaxMemoryBytes     = 10 * 1024 * 1024
)
