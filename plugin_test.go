package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/relicta-tech/relicta-plugin-sdk/plugin"
)

func TestSwiftPMPlugin_GetInfo(t *testing.T) {
	p := &SwiftPMPlugin{}
	info := p.GetInfo()

	if info.Name != "swift-pm" {
		t.Errorf("expected name 'swift-pm', got %s", info.Name)
	}

	if info.Version != Version {
		t.Errorf("expected version %s, got %s", Version, info.Version)
	}

	if len(info.Hooks) != 2 {
		t.Errorf("expected 2 hooks, got %d", len(info.Hooks))
	}

	hooks := map[plugin.Hook]bool{
		plugin.HookPrePublish:  false,
		plugin.HookPostPublish: false,
	}
	for _, h := range info.Hooks {
		hooks[h] = true
	}

	for hook, found := range hooks {
		if !found {
			t.Errorf("expected hook %v not found", hook)
		}
	}
}

func TestSwiftPMPlugin_ParseConfig(t *testing.T) {
	p := &SwiftPMPlugin{}

	tests := []struct {
		name     string
		config   map[string]any
		expected *Config
	}{
		{
			name:   "default values",
			config: map[string]any{},
			expected: &Config{
				Registry:        "https://swift.pkg.github.com",
				ManifestPath:    "Package.swift",
				VersionConstant: "packageVersion",
				CreateTag:       true,
				Validate:        true,
				Build:           true,
				Test:            true,
				TestConfig: TestConfig{
					Configuration: "debug",
					Coverage:      false,
					Parallel:      true,
				},
				Archive: ArchiveConfig{
					IncludeDocs: true,
					Exclude:     []string{".git", ".build", "Tests", "*.xcodeproj"},
				},
			},
		},
		{
			name: "custom values",
			config: map[string]any{
				"registry":      "https://custom.registry.com",
				"scope":         "myorg",
				"token":         "secret-token",
				"package_name":  "MyPackage",
				"manifest_path": "Sources/Package.swift",
				"create_tag":    false,
				"validate":      false,
				"build":         false,
				"test":          false,
				"dry_run":       true,
			},
			expected: &Config{
				Registry:        "https://custom.registry.com",
				Scope:           "myorg",
				Token:           "secret-token",
				PackageName:     "MyPackage",
				ManifestPath:    "Sources/Package.swift",
				VersionConstant: "packageVersion",
				CreateTag:       false,
				Validate:        false,
				Build:           false,
				Test:            false,
				DryRun:          true,
				TestConfig: TestConfig{
					Configuration: "debug",
					Coverage:      false,
					Parallel:      true,
				},
				Archive: ArchiveConfig{
					IncludeDocs: true,
					Exclude:     []string{".git", ".build", "Tests", "*.xcodeproj"},
				},
			},
		},
		{
			name: "with test config",
			config: map[string]any{
				"test_config": map[string]any{
					"configuration": "release",
					"coverage":      true,
					"parallel":      false,
				},
			},
			expected: &Config{
				Registry:        "https://swift.pkg.github.com",
				ManifestPath:    "Package.swift",
				VersionConstant: "packageVersion",
				CreateTag:       true,
				Validate:        true,
				Build:           true,
				Test:            true,
				TestConfig: TestConfig{
					Configuration: "release",
					Coverage:      true,
					Parallel:      false,
				},
				Archive: ArchiveConfig{
					IncludeDocs: true,
					Exclude:     []string{".git", ".build", "Tests", "*.xcodeproj"},
				},
			},
		},
		{
			name: "with custom archive exclude",
			config: map[string]any{
				"archive": map[string]any{
					"include_docs": false,
					"exclude":      []any{"custom-dir", "*.log"},
				},
			},
			expected: &Config{
				Registry:        "https://swift.pkg.github.com",
				ManifestPath:    "Package.swift",
				VersionConstant: "packageVersion",
				CreateTag:       true,
				Validate:        true,
				Build:           true,
				Test:            true,
				TestConfig: TestConfig{
					Configuration: "debug",
					Coverage:      false,
					Parallel:      true,
				},
				Archive: ArchiveConfig{
					IncludeDocs: false,
					Exclude:     []string{"custom-dir", "*.log"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := p.parseConfig(tt.config)

			if cfg.Registry != tt.expected.Registry {
				t.Errorf("expected registry %s, got %s", tt.expected.Registry, cfg.Registry)
			}
			if cfg.Scope != tt.expected.Scope {
				t.Errorf("expected scope %s, got %s", tt.expected.Scope, cfg.Scope)
			}
			if cfg.Token != tt.expected.Token {
				t.Errorf("expected token %s, got %s", tt.expected.Token, cfg.Token)
			}
			if cfg.PackageName != tt.expected.PackageName {
				t.Errorf("expected package_name %s, got %s", tt.expected.PackageName, cfg.PackageName)
			}
			if cfg.ManifestPath != tt.expected.ManifestPath {
				t.Errorf("expected manifest_path %s, got %s", tt.expected.ManifestPath, cfg.ManifestPath)
			}
			if cfg.CreateTag != tt.expected.CreateTag {
				t.Errorf("expected create_tag %v, got %v", tt.expected.CreateTag, cfg.CreateTag)
			}
			if cfg.Validate != tt.expected.Validate {
				t.Errorf("expected validate %v, got %v", tt.expected.Validate, cfg.Validate)
			}
			if cfg.Build != tt.expected.Build {
				t.Errorf("expected build %v, got %v", tt.expected.Build, cfg.Build)
			}
			if cfg.Test != tt.expected.Test {
				t.Errorf("expected test %v, got %v", tt.expected.Test, cfg.Test)
			}
			if cfg.DryRun != tt.expected.DryRun {
				t.Errorf("expected dry_run %v, got %v", tt.expected.DryRun, cfg.DryRun)
			}
			if cfg.TestConfig.Configuration != tt.expected.TestConfig.Configuration {
				t.Errorf("expected test configuration %s, got %s", tt.expected.TestConfig.Configuration, cfg.TestConfig.Configuration)
			}
			if cfg.TestConfig.Coverage != tt.expected.TestConfig.Coverage {
				t.Errorf("expected test coverage %v, got %v", tt.expected.TestConfig.Coverage, cfg.TestConfig.Coverage)
			}
			if cfg.TestConfig.Parallel != tt.expected.TestConfig.Parallel {
				t.Errorf("expected test parallel %v, got %v", tt.expected.TestConfig.Parallel, cfg.TestConfig.Parallel)
			}
			if cfg.Archive.IncludeDocs != tt.expected.Archive.IncludeDocs {
				t.Errorf("expected archive include_docs %v, got %v", tt.expected.Archive.IncludeDocs, cfg.Archive.IncludeDocs)
			}
			if len(cfg.Archive.Exclude) != len(tt.expected.Archive.Exclude) {
				t.Errorf("expected %d exclude patterns, got %d", len(tt.expected.Archive.Exclude), len(cfg.Archive.Exclude))
			}
		})
	}
}

func TestSwiftPMPlugin_Validate(t *testing.T) {
	p := &SwiftPMPlugin{}

	// Create a temporary Package.swift for testing
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "Package.swift")
	if err := os.WriteFile(manifestPath, []byte(`// swift-tools-version:5.7
import PackageDescription
let package = Package(name: "TestPackage")
`), 0644); err != nil {
		t.Fatalf("failed to create manifest: %v", err)
	}

	tests := []struct {
		name       string
		config     map[string]any
		wantErrors bool
		errorField string
	}{
		{
			name: "valid config with all required fields",
			config: map[string]any{
				"scope":         "myorg",
				"token":         "secret-token",
				"manifest_path": manifestPath,
			},
			wantErrors: false,
		},
		{
			name: "missing scope",
			config: map[string]any{
				"token":         "secret-token",
				"manifest_path": manifestPath,
			},
			wantErrors: true,
			errorField: "scope",
		},
		{
			name: "missing token",
			config: map[string]any{
				"scope":         "myorg",
				"manifest_path": manifestPath,
			},
			wantErrors: true,
			errorField: "token",
		},
		{
			name: "missing manifest",
			config: map[string]any{
				"scope":         "myorg",
				"token":         "secret-token",
				"manifest_path": "/nonexistent/Package.swift",
			},
			wantErrors: true,
			errorField: "manifest_path",
		},
		{
			name: "invalid registry URL",
			config: map[string]any{
				"scope":         "myorg",
				"token":         "secret-token",
				"manifest_path": manifestPath,
				"registry":      "://invalid-url",
			},
			wantErrors: true,
			errorField: "registry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := p.Validate(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Note: Swift CLI validation is skipped if Swift isn't installed
			hasExpectedError := false
			for _, e := range resp.Errors {
				if e.Field == tt.errorField {
					hasExpectedError = true
					break
				}
			}

			if tt.wantErrors && tt.errorField != "" && !hasExpectedError {
				// Check if the error might be about Swift not being installed
				for _, e := range resp.Errors {
					if e.Field == "swift" {
						// Swift not installed, skip field-specific validation
						return
					}
				}
				if len(resp.Errors) == 0 {
					t.Errorf("expected error for field %s, got none", tt.errorField)
				}
			}
		})
	}
}

func TestSwiftPMPlugin_Execute_DryRun(t *testing.T) {
	p := &SwiftPMPlugin{}

	// Create temp directory with Package.swift
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "Package.swift")
	if err := os.WriteFile(manifestPath, []byte(`// swift-tools-version:5.7
import PackageDescription
let package = Package(name: "TestPackage")
`), 0644); err != nil {
		t.Fatalf("failed to create manifest: %v", err)
	}

	config := map[string]any{
		"scope":         "testorg",
		"token":         "test-token",
		"manifest_path": manifestPath,
		"registry":      "https://test.registry.com",
		"dry_run":       true,
		"validate":      false, // Skip validation (requires Swift CLI)
		"build":         false, // Skip build (requires Swift CLI)
		"test":          false, // Skip tests (requires Swift CLI)
		"create_tag":    false, // Skip git operations
	}

	releaseCtx := plugin.ReleaseContext{
		Version: "1.0.0",
	}

	// Test PrePublish
	req := plugin.ExecuteRequest{
		Hook:    plugin.HookPrePublish,
		Context: releaseCtx,
		Config:  config,
		DryRun:  true,
	}
	resp, err := p.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("PrePublish failed: %v", err)
	}
	if !resp.Success {
		t.Errorf("PrePublish should succeed in dry-run mode: %s", resp.Message)
	}

	// Test PostPublish
	req.Hook = plugin.HookPostPublish
	resp, err = p.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("PostPublish failed: %v", err)
	}
	if !resp.Success {
		t.Errorf("PostPublish should succeed in dry-run mode: %s", resp.Message)
	}
}

func TestRegistryClient_Publish(t *testing.T) {
	// Create a test server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}

		// Check authorization
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected 'Bearer test-token', got %s", auth)
		}

		// Check content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/zip" {
			t.Errorf("expected 'application/zip', got %s", contentType)
		}

		// Check digest
		digest := r.Header.Get("Digest")
		if digest == "" {
			t.Error("expected Digest header")
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	// Create a temp archive file
	tempFile, err := os.CreateTemp("", "test-*.zip")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tempFile.Name()) }()
	_, _ = tempFile.Write([]byte("test archive content"))
	_ = tempFile.Close()

	client := &RegistryClient{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

	err = client.Publish(context.Background(), "testorg", "TestPackage", "1.0.0", tempFile.Name(), "abc123checksum")
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}
}

func TestRegistryClient_GetRelease(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		response   interface{}
		wantNil    bool
		wantErr    bool
	}{
		{
			name:    "release exists",
			status:  http.StatusOK,
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "release not found",
			status:  http.StatusNotFound,
			wantNil: true,
			wantErr: false,
		},
		{
			name:    "server error",
			status:  http.StatusInternalServerError,
			wantNil: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				if tt.response != nil {
					_ = json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client := &RegistryClient{
				baseURL:    server.URL,
				token:      "test-token",
				httpClient: server.Client(),
			}

			release, err := client.GetRelease(context.Background(), "testorg", "TestPackage", "1.0.0")

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantNil && release != nil {
				t.Error("expected nil release")
			}
		})
	}
}
