package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// SwiftCLI wraps Swift command-line operations.
type SwiftCLI struct {
	workDir string
}

// NewSwiftCLI creates a new SwiftCLI instance.
func NewSwiftCLI(workDir string) *SwiftCLI {
	return &SwiftCLI{workDir: workDir}
}

// Validate validates the package manifest.
func (s *SwiftCLI) Validate(ctx context.Context) error {
	// Check manifest syntax by dumping package
	if err := s.run(ctx, "package", "dump-package"); err != nil {
		return fmt.Errorf("invalid Package.swift: %w", err)
	}

	// Resolve dependencies to validate they're accessible
	if err := s.run(ctx, "package", "resolve"); err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	return nil
}

// Build builds the package.
func (s *SwiftCLI) Build(ctx context.Context, configuration string) error {
	args := []string{"build"}
	if configuration != "" {
		args = append(args, "-c", configuration)
	}
	return s.run(ctx, args...)
}

// Test runs package tests.
func (s *SwiftCLI) Test(ctx context.Context, cfg TestConfig) error {
	args := []string{"test"}
	if cfg.Configuration != "" {
		args = append(args, "-c", cfg.Configuration)
	}
	if cfg.Coverage {
		args = append(args, "--enable-code-coverage")
	}
	if cfg.Parallel {
		args = append(args, "--parallel")
	}
	return s.run(ctx, args...)
}

// Clean cleans build artifacts.
func (s *SwiftCLI) Clean(ctx context.Context) error {
	return s.run(ctx, "package", "clean")
}

// GetVersion returns the Swift version.
func (s *SwiftCLI) GetVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "swift", "--version")
	cmd.Dir = s.workDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Swift version: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// run executes a swift command.
func (s *SwiftCLI) run(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "swift", args...)
	cmd.Dir = s.workDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errOutput := strings.TrimSpace(stderr.String())
		if errOutput != "" {
			return fmt.Errorf("%s: %w", errOutput, err)
		}
		return err
	}

	return nil
}
