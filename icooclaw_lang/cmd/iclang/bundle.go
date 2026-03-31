package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var errBundlePayloadNotFound = errors.New("bundle payload not found")

const bundleMagic = "ICLANG_BUNDLE_V1"

type bundlePayload struct {
	ScriptPath   string `json:"script_path"`
	ScriptSource string `json:"script_source"`
}

func defaultBundleOutputPath(scriptPath string) string {
	base := strings.TrimSuffix(filepath.Base(scriptPath), filepath.Ext(scriptPath))
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func buildBundle(scriptPath, outputPath string) error {
	runtimePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve runtime executable: %w", err)
	}
	return buildBundleWithRuntime(scriptPath, outputPath, runtimePath)
}

func buildBundleWithRuntime(scriptPath, outputPath, runtimePath string) error {
	scriptAbs, err := filepath.Abs(scriptPath)
	if err != nil {
		return fmt.Errorf("resolve script path: %w", err)
	}

	scriptBytes, err := os.ReadFile(scriptAbs)
	if err != nil {
		return fmt.Errorf("read script: %w", err)
	}

	payloadBytes, err := json.Marshal(bundlePayload{
		ScriptPath:   filepath.Base(scriptAbs),
		ScriptSource: string(scriptBytes),
	})
	if err != nil {
		return fmt.Errorf("encode bundle payload: %w", err)
	}

	outputAbs, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputAbs), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	if err := copyFile(runtimePath, outputAbs); err != nil {
		return fmt.Errorf("copy runtime executable: %w", err)
	}
	if err := appendBundlePayload(outputAbs, payloadBytes); err != nil {
		return fmt.Errorf("append bundle payload: %w", err)
	}
	return nil
}

func tryRunBundledExecutable(scriptArgs []string) (bool, error) {
	exePath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("resolve executable path: %w", err)
	}

	payload, err := readBundlePayload(exePath)
	if err != nil {
		if errors.Is(err, errBundlePayloadNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, executeScriptSource(payload.ScriptPath, payload.ScriptSource, scriptArgs)
}

func readBundlePayload(executablePath string) (*bundlePayload, error) {
	file, err := os.Open(executablePath)
	if err != nil {
		return nil, fmt.Errorf("open executable: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat executable: %w", err)
	}

	trailerSize := int64(len(bundleMagic) + 8)
	if stat.Size() < trailerSize {
		return nil, errBundlePayloadNotFound
	}

	trailer := make([]byte, trailerSize)
	if _, err := file.ReadAt(trailer, stat.Size()-trailerSize); err != nil {
		return nil, fmt.Errorf("read bundle trailer: %w", err)
	}

	if string(trailer[8:]) != bundleMagic {
		return nil, errBundlePayloadNotFound
	}

	payloadSize := int64(binary.LittleEndian.Uint64(trailer[:8]))
	if payloadSize <= 0 || payloadSize > stat.Size()-trailerSize {
		return nil, fmt.Errorf("invalid bundle payload size: %d", payloadSize)
	}

	payloadBytes := make([]byte, payloadSize)
	if _, err := file.ReadAt(payloadBytes, stat.Size()-trailerSize-payloadSize); err != nil {
		return nil, fmt.Errorf("read bundle payload: %w", err)
	}

	var payload bundlePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("decode bundle payload: %w", err)
	}
	return &payload, nil
}

func appendBundlePayload(executablePath string, payload []byte) error {
	file, err := os.OpenFile(executablePath, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	defer file.Close()

	trailer := make([]byte, 8+len(bundleMagic))
	binary.LittleEndian.PutUint64(trailer[:8], uint64(len(payload)))
	copy(trailer[8:], []byte(bundleMagic))

	if _, err := file.Write(payload); err != nil {
		return err
	}
	if _, err := file.Write(trailer); err != nil {
		return err
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
