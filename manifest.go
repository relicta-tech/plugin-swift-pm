package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// PackageManifest represents parsed Package.swift content.
type PackageManifest struct {
	Name         string       `json:"name"`
	Platforms    []Platform   `json:"platforms"`
	Products     []Product    `json:"products"`
	Dependencies []Dependency `json:"dependencies"`
	Targets      []Target     `json:"targets"`
	SwiftVersion string       `json:"swift_tools_version"`
}

// Platform represents a supported platform.
type Platform struct {
	Name    string `json:"platformName"`
	Version string `json:"version"`
}

// Product represents a package product.
type Product struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Targets []string `json:"targets"`
}

// Dependency represents a package dependency.
type Dependency struct {
	URL      string `json:"url"`
	Name     string `json:"name"`
	Versions string `json:"version"`
}

// Target represents a package target.
type Target struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Dependencies []string `json:"dependencies"`
}

// ParseManifest extracts package info from Package.swift using Swift CLI.
func ParseManifest(ctx context.Context, workDir string) (*PackageManifest, error) {
	cmd := exec.CommandContext(ctx, "swift", "package", "dump-package")
	cmd.Dir = workDir

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("failed to parse manifest: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	var manifest PackageManifest
	if err := json.Unmarshal(output, &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	return &manifest, nil
}

// UpdateVersionConstant updates the version constant in Package.swift.
func UpdateVersionConstant(path, constantName, version string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	// Pattern: let packageVersion = "1.0.0" or let packageVersion: String = "1.0.0"
	pattern := regexp.MustCompile(
		fmt.Sprintf(`(let\s+%s\s*(?::\s*String\s*)?=\s*")([^"]+)(")`, regexp.QuoteMeta(constantName)),
	)

	if !pattern.Match(content) {
		return fmt.Errorf("version constant '%s' not found in %s", constantName, path)
	}

	newContent := pattern.ReplaceAll(content, []byte(fmt.Sprintf("${1}%s${3}", version)))

	if err := os.WriteFile(path, newContent, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	return nil
}

// ExtractSwiftToolsVersion extracts the swift-tools-version from Package.swift.
func ExtractSwiftToolsVersion(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", path, err)
	}

	// Pattern: // swift-tools-version:5.7 or // swift-tools-version: 5.7
	pattern := regexp.MustCompile(`//\s*swift-tools-version:\s*(\d+\.\d+(?:\.\d+)?)`)
	matches := pattern.FindSubmatch(content)
	if len(matches) < 2 {
		return "", fmt.Errorf("swift-tools-version not found in %s", path)
	}

	return string(matches[1]), nil
}
