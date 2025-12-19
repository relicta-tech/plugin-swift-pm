package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateVersionConstant(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		constantName string
		newVersion   string
		expected     string
		wantErr      bool
	}{
		{
			name: "simple version constant",
			content: `// swift-tools-version:5.7
let packageVersion = "1.0.0"

import PackageDescription
let package = Package(name: "Test")`,
			constantName: "packageVersion",
			newVersion:   "2.0.0",
			expected: `// swift-tools-version:5.7
let packageVersion = "2.0.0"

import PackageDescription
let package = Package(name: "Test")`,
			wantErr: false,
		},
		{
			name: "version constant with type annotation",
			content: `// swift-tools-version:5.7
let version: String = "1.0.0"

import PackageDescription
let package = Package(name: "Test")`,
			constantName: "version",
			newVersion:   "3.0.0",
			expected: `// swift-tools-version:5.7
let version: String = "3.0.0"

import PackageDescription
let package = Package(name: "Test")`,
			wantErr: false,
		},
		{
			name: "constant not found",
			content: `// swift-tools-version:5.7
import PackageDescription
let package = Package(name: "Test")`,
			constantName: "nonexistent",
			newVersion:   "1.0.0",
			expected:     "",
			wantErr:      true,
		},
		{
			name: "multiple constants only updates specified one",
			content: `// swift-tools-version:5.7
let appVersion = "1.0.0"
let packageVersion = "2.0.0"

import PackageDescription`,
			constantName: "packageVersion",
			newVersion:   "3.0.0",
			expected: `// swift-tools-version:5.7
let appVersion = "1.0.0"
let packageVersion = "3.0.0"

import PackageDescription`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			path := filepath.Join(tempDir, "Package.swift")

			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}

			err := UpdateVersionConstant(path, tt.constantName, tt.newVersion)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			result, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("expected:\n%s\n\ngot:\n%s", tt.expected, string(result))
			}
		})
	}
}

func TestExtractSwiftToolsVersion(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
		wantErr  bool
	}{
		{
			name: "standard format",
			content: `// swift-tools-version:5.7
import PackageDescription`,
			expected: "5.7",
			wantErr:  false,
		},
		{
			name: "with space after colon",
			content: `// swift-tools-version: 5.9
import PackageDescription`,
			expected: "5.9",
			wantErr:  false,
		},
		{
			name: "patch version",
			content: `// swift-tools-version:5.7.1
import PackageDescription`,
			expected: "5.7.1",
			wantErr:  false,
		},
		{
			name: "missing version",
			content: `import PackageDescription
let package = Package(name: "Test")`,
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			path := filepath.Join(tempDir, "Package.swift")

			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}

			result, err := ExtractSwiftToolsVersion(path)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
