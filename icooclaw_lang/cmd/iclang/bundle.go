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
	"sort"
	"strings"
)

var errBundlePayloadNotFound = errors.New("bundle payload not found")

const bundleMagic = "ICLANG_BUNDLE_V1"

type bundlePayload struct {
	ScriptPath   string            `json:"script_path"`
	ScriptSource string            `json:"script_source"`
	ProjectRoot  string            `json:"project_root,omitempty"`
	Files        map[string]string `json:"files,omitempty"`
}

func defaultBundleOutputPath(scriptPath string) string {
	_, base, _, err := resolveBuildInput(scriptPath)
	if err != nil {
		base = strings.TrimSuffix(filepath.Base(scriptPath), filepath.Ext(scriptPath))
	}
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func buildBundle(inputPath, outputPath string) error {
	runtimePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve runtime executable: %w", err)
	}
	return buildBundleWithRuntime(inputPath, outputPath, runtimePath)
}

func buildBundleWithRuntime(inputPath, outputPath, runtimePath string) error {
	scriptAbs, _, projectRoot, err := resolveBuildInput(inputPath)
	if err != nil {
		return err
	}

	scriptBytes, err := os.ReadFile(scriptAbs)
	if err != nil {
		return fmt.Errorf("read script: %w", err)
	}

	payload := bundlePayload{
		ScriptPath:   filepath.Base(scriptAbs),
		ScriptSource: string(scriptBytes),
	}
	if projectRoot != "" {
		files, err := collectBundleFiles(projectRoot)
		if err != nil {
			return fmt.Errorf("collect project files: %w", err)
		}
		relScriptPath, err := filepath.Rel(projectRoot, scriptAbs)
		if err != nil {
			return fmt.Errorf("derive entry path: %w", err)
		}
		payload.ScriptPath = filepath.ToSlash(relScriptPath)
		payload.ProjectRoot = filepath.Base(projectRoot)
		payload.Files = files
	}

	payloadBytes, err := json.Marshal(payload)
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

	maxGoroutines, remainingArgs, err := parseRuntimeOptions(scriptArgs)
	if err != nil {
		return true, err
	}

	if len(payload.Files) == 0 {
		return true, executeScriptSource(payload.ScriptPath, payload.ScriptSource, remainingArgs, maxGoroutines)
	}

	projectDir, scriptPath, cleanup, err := materializeBundledProject(payload)
	if err != nil {
		return false, err
	}
	defer cleanup()

	workingRoot := filepath.Dir(projectDir)
	originalWD, err := os.Getwd()
	if err != nil {
		return false, fmt.Errorf("resolve working dir: %w", err)
	}
	if err := os.Chdir(workingRoot); err != nil {
		return false, fmt.Errorf("switch to bundled project dir: %w", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	return true, runFileForBundle(scriptPath, remainingArgs, maxGoroutines)
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

func collectBundleFiles(projectRoot string) (map[string]string, error) {
	files := map[string]string{}

	err := filepath.WalkDir(projectRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") {
				if path == projectRoot {
					return nil
				}
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)
		if shouldSkipBundleFile(relPath) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[relPath] = string(data)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func shouldSkipBundleFile(relPath string) bool {
	base := filepath.Base(relPath)
	ext := strings.ToLower(filepath.Ext(base))

	if strings.HasSuffix(strings.ToLower(base), ".exe") {
		return true
	}
	switch ext {
	case ".out", ".log":
		return true
	}
	return false
}

func materializeBundledProject(payload *bundlePayload) (string, string, func(), error) {
	rootName := payload.ProjectRoot
	if rootName == "" {
		rootName = "iclang_project"
	}

	tempRoot, err := os.MkdirTemp("", "iclang-bundle-*")
	if err != nil {
		return "", "", nil, fmt.Errorf("create temp project dir: %w", err)
	}

	projectDir := filepath.Join(tempRoot, rootName)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		os.RemoveAll(tempRoot)
		return "", "", nil, fmt.Errorf("create bundled project root: %w", err)
	}

	paths := make([]string, 0, len(payload.Files))
	for relPath := range payload.Files {
		paths = append(paths, relPath)
	}
	sort.Strings(paths)

	for _, relPath := range paths {
		targetPath := filepath.Join(projectDir, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			os.RemoveAll(tempRoot)
			return "", "", nil, fmt.Errorf("create bundled file dir: %w", err)
		}
		if err := os.WriteFile(targetPath, []byte(payload.Files[relPath]), 0o644); err != nil {
			os.RemoveAll(tempRoot)
			return "", "", nil, fmt.Errorf("write bundled file: %w", err)
		}
	}

	scriptPath := filepath.Join(projectDir, filepath.FromSlash(payload.ScriptPath))
	cleanup := func() {
		_ = os.RemoveAll(tempRoot)
	}
	return projectDir, scriptPath, cleanup, nil
}

func runFileForBundle(filename string, scriptArgs []string, maxGoroutines int) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Error: could not read file '%s': %s", filename, err)
	}

	if err := executeScriptSource(filename, string(data), scriptArgs, maxGoroutines); err != nil {
		return err
	}
	return nil
}
