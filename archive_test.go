package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateArchive(t *testing.T) {
	// Create a temporary source directory
	tempDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"Package.swift":         "// swift-tools-version:5.7",
		"Sources/main.swift":    "print(\"Hello\")",
		"Sources/lib.swift":     "public func greet() {}",
		"Tests/test.swift":      "import XCTest",
		".git/config":           "[core]",
		".build/debug/lib":      "binary",
		"project.xcodeproj/foo": "xcode file",
	}

	for name, content := range files {
		path := filepath.Join(tempDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	tests := []struct {
		name           string
		config         ArchiveConfig
		expectedFiles  []string
		excludedFiles  []string
	}{
		{
			name: "default exclusions",
			config: ArchiveConfig{
				IncludeDocs: true,
				Exclude:     []string{".git", ".build", "Tests", "*.xcodeproj"},
			},
			expectedFiles: []string{
				"Package.swift",
				"Sources/main.swift",
				"Sources/lib.swift",
			},
			excludedFiles: []string{
				"Tests/test.swift",
				".git/config",
				".build/debug/lib",
				"project.xcodeproj/foo",
			},
		},
		{
			name: "custom exclusions",
			config: ArchiveConfig{
				IncludeDocs: true,
				Exclude:     []string{"Sources"},
			},
			expectedFiles: []string{
				"Package.swift",
				"Tests/test.swift",
			},
			excludedFiles: []string{
				"Sources/main.swift",
				"Sources/lib.swift",
			},
		},
		{
			name: "no exclusions",
			config: ArchiveConfig{
				IncludeDocs: true,
				Exclude:     []string{},
			},
			expectedFiles: []string{
				"Package.swift",
				"Sources/main.swift",
				"Sources/lib.swift",
				"Tests/test.swift",
			},
			excludedFiles: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archivePath, checksum, err := CreateArchive(tempDir, "1.0.0", tt.config)
			if err != nil {
				t.Fatalf("CreateArchive failed: %v", err)
			}
			defer func() { _ = os.Remove(archivePath) }()

			if checksum == "" {
				t.Error("checksum should not be empty")
			}

			// Open the archive and verify contents
			reader, err := zip.OpenReader(archivePath)
			if err != nil {
				t.Fatalf("failed to open archive: %v", err)
			}
			defer func() { _ = reader.Close() }()

			fileSet := make(map[string]bool)
			for _, f := range reader.File {
				fileSet[f.Name] = true
			}

			for _, expected := range tt.expectedFiles {
				if !fileSet[expected] {
					t.Errorf("expected file %s not found in archive", expected)
				}
			}

			for _, excluded := range tt.excludedFiles {
				if fileSet[excluded] {
					t.Errorf("excluded file %s found in archive", excluded)
				}
			}
		})
	}
}

func TestShouldExclude(t *testing.T) {
	tests := []struct {
		path     string
		patterns []string
		expected bool
	}{
		{
			path:     ".git",
			patterns: []string{".git", ".build"},
			expected: true,
		},
		{
			path:     ".git/config",
			patterns: []string{".git"},
			expected: true,
		},
		{
			path:     ".build/debug/binary",
			patterns: []string{".build"},
			expected: true,
		},
		{
			path:     "Sources/main.swift",
			patterns: []string{".git", ".build"},
			expected: false,
		},
		{
			path:     "test.log",
			patterns: []string{"*.log"},
			expected: true,
		},
		{
			path:     "logs/test.log",
			patterns: []string{"*.log"},
			expected: true,
		},
		{
			path:     "project.xcodeproj",
			patterns: []string{"*.xcodeproj"},
			expected: true,
		},
		{
			path:     "project.xcodeproj/project.pbxproj",
			patterns: []string{"*.xcodeproj"},
			expected: true,
		},
		{
			path:     "Tests",
			patterns: []string{"Tests"},
			expected: true,
		},
		{
			path:     "Tests/MyTests.swift",
			patterns: []string{"Tests"},
			expected: true,
		},
		{
			path:     "Sources/Tests.swift",
			patterns: []string{"Tests"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := shouldExclude(tt.path, tt.patterns)
			if result != tt.expected {
				t.Errorf("shouldExclude(%q, %v) = %v, expected %v", tt.path, tt.patterns, result, tt.expected)
			}
		})
	}
}

func TestGetArchiveSize(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-*.zip")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tempFile.Name()) }()

	content := []byte("test content with some data")
	if _, err := tempFile.Write(content); err != nil {
		t.Fatalf("failed to write to file: %v", err)
	}
	_ = tempFile.Close()

	size, err := GetArchiveSize(tempFile.Name())
	if err != nil {
		t.Fatalf("GetArchiveSize failed: %v", err)
	}

	if size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), size)
	}
}
