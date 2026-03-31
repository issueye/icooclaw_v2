package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var projectNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func initProject(targetDir, projectName string) (string, error) {
	if strings.TrimSpace(targetDir) == "" {
		return "", fmt.Errorf("project directory is required")
	}

	projectDir, err := filepath.Abs(targetDir)
	if err != nil {
		return "", fmt.Errorf("resolve project directory: %w", err)
	}

	if strings.TrimSpace(projectName) == "" {
		projectName = filepath.Base(projectDir)
	}
	projectName = sanitizeProjectName(projectName)
	if projectName == "" {
		return "", fmt.Errorf("project name is empty")
	}

	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return "", fmt.Errorf("create project directory: %w", err)
	}

	projectFiles := map[string]string{
		"pkg.toml":  defaultManifestTemplate(projectName),
		"main.is":   defaultMainTemplate(projectName),
		"README.md": defaultReadmeTemplate(projectName),
	}
	for relPath, content := range projectFiles {
		if err := writeProjectFile(projectDir, relPath, content); err != nil {
			return "", err
		}
	}

	modulesDir := filepath.Join(projectDir, "modules")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		return "", fmt.Errorf("create modules directory: %w", err)
	}

	return projectDir, nil
}

func writeProjectFile(projectDir, relPath, content string) error {
	fullPath := filepath.Join(projectDir, relPath)
	if _, err := os.Stat(fullPath); err == nil {
		return fmt.Errorf("%s already exists", relPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check %s: %w", relPath, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", relPath, err)
	}
	return nil
}

func sanitizeProjectName(name string) string {
	name = strings.TrimSpace(name)
	name = projectNameSanitizer.ReplaceAllString(name, "_")
	name = strings.Trim(name, "._-")
	return name
}

func defaultManifestTemplate(projectName string) string {
	return fmt.Sprintf(`name = %q
version = "0.1.0"
entry = "./main.is"
description = %q
`, projectName, projectName+" project")
}

func defaultMainTemplate(projectName string) string {
	return fmt.Sprintf(`print("Hello from %s")
`, projectName)
}

func defaultReadmeTemplate(projectName string) string {
	return fmt.Sprintf(`# %s

## Run

`+"```bash"+`
iclang run main.is
`+"```"+`

## Build

`+"```bash"+`
iclang build .
`+"```"+`
`, projectName)
}
