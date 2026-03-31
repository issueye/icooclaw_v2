package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type packageManifest struct {
	Path  string
	Name  string
	Entry string
}

type noManifestError struct {
	message string
}

func (e *noManifestError) Error() string {
	return e.message
}

func isNoManifestError(err error) bool {
	_, ok := err.(*noManifestError)
	return ok
}

func resolveBuildInput(inputPath string) (string, string, string, error) {
	manifest, err := loadBuildManifest(inputPath)
	if err == nil {
		scriptPath := manifest.Entry
		if !filepath.IsAbs(scriptPath) {
			scriptPath = filepath.Join(filepath.Dir(manifest.Path), scriptPath)
		}
		outputName := manifest.Name
		if outputName == "" {
			outputName = strings.TrimSuffix(filepath.Base(scriptPath), filepath.Ext(scriptPath))
		}
		return scriptPath, outputName, filepath.Dir(manifest.Path), nil
	}
	if err != nil && !os.IsNotExist(err) && !isNoManifestError(err) {
		return "", "", "", err
	}

	scriptPath, absErr := filepath.Abs(inputPath)
	if absErr != nil {
		return "", "", "", fmt.Errorf("resolve script path: %w", absErr)
	}
	base := strings.TrimSuffix(filepath.Base(scriptPath), filepath.Ext(scriptPath))
	return scriptPath, base, "", nil
}

func loadBuildManifest(inputPath string) (*packageManifest, error) {
	absInput, err := filepath.Abs(inputPath)
	if err != nil {
		return nil, fmt.Errorf("resolve build input: %w", err)
	}

	stat, err := os.Stat(absInput)
	if err != nil {
		return nil, err
	}

	manifestPath := absInput
	if stat.IsDir() {
		manifestPath = filepath.Join(absInput, "pkg.toml")
	} else if filepath.Base(absInput) != "pkg.toml" {
		return nil, &noManifestError{message: "build input is not a manifest path"}
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	manifest, err := parsePackageManifest(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse manifest %s: %w", manifestPath, err)
	}
	manifest.Path = manifestPath

	if manifest.Entry == "" {
		return nil, fmt.Errorf("manifest %s missing entry", manifestPath)
	}
	return manifest, nil
}

func parsePackageManifest(input string) (*packageManifest, error) {
	manifest := &packageManifest{}
	currentTable := ""

	for lineNo, raw := range strings.Split(input, "\n") {
		line := strings.TrimSpace(stripManifestComment(raw))
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return nil, fmt.Errorf("line %d: invalid table header", lineNo+1)
			}
			currentTable = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}

		key, valueText, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected key = value", lineNo+1)
		}
		key = strings.TrimSpace(key)
		valueText = strings.TrimSpace(valueText)

		value, err := parseManifestStringValue(valueText)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo+1, err)
		}

		if currentTable != "" {
			key = currentTable + "." + key
		}

		switch key {
		case "name":
			manifest.Name = value
		case "entry":
			manifest.Entry = value
		}
	}

	return manifest, nil
}

func parseManifestStringValue(text string) (string, error) {
	if strings.HasPrefix(text, "\"") {
		value, err := strconv.Unquote(text)
		if err != nil {
			return "", fmt.Errorf("invalid string: %w", err)
		}
		return value, nil
	}
	return text, nil
}

func stripManifestComment(line string) string {
	var out strings.Builder
	inString := false
	escaped := false

	for _, r := range line {
		switch {
		case escaped:
			out.WriteRune(r)
			escaped = false
		case r == '\\' && inString:
			out.WriteRune(r)
			escaped = true
		case r == '"':
			out.WriteRune(r)
			inString = !inString
		case r == '#' && !inString:
			return out.String()
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}
