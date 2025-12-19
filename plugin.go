package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/relicta-tech/relicta-plugin-sdk/helpers"
	"github.com/relicta-tech/relicta-plugin-sdk/plugin"
)

// Version is set at build time.
var Version = "0.1.0"

// Config represents Swift PM plugin configuration.
type Config struct {
	Registry        string        `json:"registry"`
	Scope           string        `json:"scope"`
	Token           string        `json:"token"`
	PackageName     string        `json:"package_name"`
	ManifestPath    string        `json:"manifest_path"`
	UpdateManifest  bool          `json:"update_manifest"`
	VersionConstant string        `json:"version_constant"`
	CreateTag       bool          `json:"create_tag"`
	TagPrefix       string        `json:"tag_prefix"`
	Validate        bool          `json:"validate"`
	Build           bool          `json:"build"`
	Test            bool          `json:"test"`
	TestConfig      TestConfig    `json:"test_config"`
	Archive         ArchiveConfig `json:"archive"`
	DryRun          bool          `json:"dry_run"`
}

// TestConfig defines test execution options.
type TestConfig struct {
	Configuration string `json:"configuration"`
	Coverage      bool   `json:"coverage"`
	Parallel      bool   `json:"parallel"`
}

// ArchiveConfig defines archive creation options.
type ArchiveConfig struct {
	IncludeDocs bool     `json:"include_docs"`
	Exclude     []string `json:"exclude"`
}

// SwiftPMPlugin implements the Swift Package Manager plugin.
type SwiftPMPlugin struct{}

// GetInfo returns plugin metadata.
func (p *SwiftPMPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		Name:        "swift-pm",
		Version:     Version,
		Description: "Swift Package Manager registry publishing and package management",
		Hooks: []plugin.Hook{
			plugin.HookPrePublish,
			plugin.HookPostPublish,
		},
	}
}

// Validate validates plugin configuration.
func (p *SwiftPMPlugin) Validate(ctx context.Context, config map[string]any) (*plugin.ValidateResponse, error) {
	cfg := p.parseConfig(config)
	vb := helpers.NewValidationBuilder()

	// Check Swift installation
	if _, err := exec.LookPath("swift"); err != nil {
		vb.AddError("swift", "Swift CLI not found in PATH")
	}

	// Check scope
	if cfg.Scope == "" {
		vb.AddError("scope", "Package scope is required")
	}

	// Check token
	if cfg.Token == "" {
		vb.AddError("token", "Registry token is required")
	}

	// Validate registry URL
	if cfg.Registry != "" {
		if _, err := url.Parse(cfg.Registry); err != nil {
			vb.AddError("registry", "Invalid registry URL")
		}
	}

	// Check manifest exists
	manifestPath := cfg.ManifestPath
	if manifestPath == "" {
		manifestPath = "Package.swift"
	}
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		vb.AddError("manifest_path", fmt.Sprintf("Package.swift not found at: %s", manifestPath))
	}

	return vb.Build(), nil
}

// Execute runs the plugin for a given hook.
func (p *SwiftPMPlugin) Execute(ctx context.Context, req plugin.ExecuteRequest) (*plugin.ExecuteResponse, error) {
	cfg := p.parseConfig(req.Config)
	cfg.DryRun = cfg.DryRun || req.DryRun
	logger := slog.Default().With("plugin", "swift-pm", "hook", req.Hook)

	switch req.Hook {
	case plugin.HookPrePublish:
		return p.executePrePublish(ctx, &req.Context, cfg, logger)
	case plugin.HookPostPublish:
		return p.executePostPublish(ctx, &req.Context, cfg, logger)
	default:
		return &plugin.ExecuteResponse{
			Success: true,
			Message: fmt.Sprintf("Hook %s not handled by swift-pm plugin", req.Hook),
		}, nil
	}
}

func (p *SwiftPMPlugin) executePrePublish(ctx context.Context, releaseCtx *plugin.ReleaseContext, cfg *Config, logger *slog.Logger) (*plugin.ExecuteResponse, error) {
	version := releaseCtx.Version
	logger = logger.With("version", version)

	// Determine manifest path
	manifestPath := cfg.ManifestPath
	if manifestPath == "" {
		manifestPath = "Package.swift"
	}

	workDir := filepath.Dir(manifestPath)
	if workDir == "." {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	swift := NewSwiftCLI(workDir)

	// Validate package
	if cfg.Validate {
		logger.Info("Validating package manifest")
		if cfg.DryRun {
			logger.Info("[DRY-RUN] Would validate Package.swift")
		} else {
			if err := swift.Validate(ctx); err != nil {
				return &plugin.ExecuteResponse{
					Success: false,
					Message: fmt.Sprintf("Package validation failed: %v", err),
				}, nil
			}
		}
	}

	// Build package
	if cfg.Build {
		logger.Info("Building package")
		if cfg.DryRun {
			logger.Info("[DRY-RUN] Would build package")
		} else {
			if err := swift.Build(ctx, "release"); err != nil {
				return &plugin.ExecuteResponse{
					Success: false,
					Message: fmt.Sprintf("Build failed: %v", err),
				}, nil
			}
		}
	}

	// Run tests
	if cfg.Test {
		logger.Info("Running tests")
		if cfg.DryRun {
			logger.Info("[DRY-RUN] Would run tests", "config", cfg.TestConfig)
		} else {
			if err := swift.Test(ctx, cfg.TestConfig); err != nil {
				return &plugin.ExecuteResponse{
					Success: false,
					Message: fmt.Sprintf("Tests failed: %v", err),
				}, nil
			}
		}
	}

	// Update version constant in manifest
	if cfg.UpdateManifest && cfg.VersionConstant != "" {
		logger.Info("Updating version in Package.swift", "constant", cfg.VersionConstant)
		if cfg.DryRun {
			logger.Info("[DRY-RUN] Would update version constant",
				"constant", cfg.VersionConstant,
				"version", version)
		} else {
			if err := UpdateVersionConstant(manifestPath, cfg.VersionConstant, version); err != nil {
				return &plugin.ExecuteResponse{
					Success: false,
					Message: fmt.Sprintf("Failed to update version: %v", err),
				}, nil
			}
		}
	}

	logger.Info("PrePublish completed successfully")
	return &plugin.ExecuteResponse{
		Success: true,
		Message: "Package validated and built successfully",
	}, nil
}

func (p *SwiftPMPlugin) executePostPublish(ctx context.Context, releaseCtx *plugin.ReleaseContext, cfg *Config, logger *slog.Logger) (*plugin.ExecuteResponse, error) {
	version := releaseCtx.Version
	logger = logger.With("version", version)

	// Determine manifest path and work directory
	manifestPath := cfg.ManifestPath
	if manifestPath == "" {
		manifestPath = "Package.swift"
	}

	workDir := filepath.Dir(manifestPath)
	if workDir == "." {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Parse manifest to get package name
	packageName := cfg.PackageName
	if packageName == "" {
		manifest, err := ParseManifest(ctx, workDir)
		if err != nil {
			return &plugin.ExecuteResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to parse Package.swift: %v", err),
			}, nil
		}
		packageName = manifest.Name
	}

	logger = logger.With("package", packageName, "scope", cfg.Scope)

	// Create package archive
	logger.Info("Creating package archive")
	var archivePath, checksum string
	var err error

	if cfg.DryRun {
		logger.Info("[DRY-RUN] Would create archive", "exclude", cfg.Archive.Exclude)
		archivePath = "/tmp/dry-run-archive.zip"
		checksum = "dry-run-checksum"
	} else {
		archivePath, checksum, err = CreateArchive(workDir, version, cfg.Archive)
		if err != nil {
			return &plugin.ExecuteResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to create archive: %v", err),
			}, nil
		}
		defer func() { _ = os.Remove(archivePath) }()
	}

	logger.Info("Archive created", "checksum", checksum)

	// Publish to registry
	if cfg.Registry != "" {
		logger.Info("Publishing to registry", "registry", cfg.Registry)

		if cfg.DryRun {
			logger.Info("[DRY-RUN] Would publish to registry",
				"registry", cfg.Registry,
				"scope", cfg.Scope,
				"package", packageName,
				"version", version)
		} else {
			client := NewRegistryClient(cfg.Registry, cfg.Token)
			if err := client.Publish(ctx, cfg.Scope, packageName, version, archivePath, checksum); err != nil {
				return &plugin.ExecuteResponse{
					Success: false,
					Message: fmt.Sprintf("Failed to publish to registry: %v", err),
				}, nil
			}
		}
	}

	// Create git tag
	if cfg.CreateTag {
		tag := cfg.TagPrefix + version
		logger.Info("Creating git tag", "tag", tag)

		if cfg.DryRun {
			logger.Info("[DRY-RUN] Would create git tag", "tag", tag)
		} else {
			if err := createGitTag(ctx, tag); err != nil {
				return &plugin.ExecuteResponse{
					Success: false,
					Message: fmt.Sprintf("Failed to create git tag: %v", err),
				}, nil
			}
		}
	}

	var msg string
	if cfg.DryRun {
		msg = fmt.Sprintf("[DRY-RUN] Would publish %s@%s to registry", packageName, version)
	} else {
		msg = fmt.Sprintf("Published %s@%s to registry", packageName, version)
	}

	logger.Info("PostPublish completed successfully")
	return &plugin.ExecuteResponse{
		Success: true,
		Message: msg,
	}, nil
}

func (p *SwiftPMPlugin) parseConfig(raw map[string]any) *Config {
	parser := helpers.NewConfigParser(raw)

	// Parse exclude list
	var exclude []string
	if excludeRaw, ok := raw["archive"].(map[string]any); ok {
		if excludeList, ok := excludeRaw["exclude"].([]any); ok {
			for _, e := range excludeList {
				if s, ok := e.(string); ok {
					exclude = append(exclude, s)
				}
			}
		}
	}

	// If no exclude patterns provided, use defaults
	if len(exclude) == 0 {
		exclude = []string{".git", ".build", "Tests", "*.xcodeproj"}
	}

	// Parse test config
	testConfig := TestConfig{
		Configuration: "debug",
		Coverage:      false,
		Parallel:      true,
	}
	if testRaw, ok := raw["test_config"].(map[string]any); ok {
		if cfg, ok := testRaw["configuration"].(string); ok {
			testConfig.Configuration = cfg
		}
		if cov, ok := testRaw["coverage"].(bool); ok {
			testConfig.Coverage = cov
		}
		if par, ok := testRaw["parallel"].(bool); ok {
			testConfig.Parallel = par
		}
	}

	// Parse archive config
	archiveConfig := ArchiveConfig{
		IncludeDocs: true,
		Exclude:     exclude,
	}
	if archiveRaw, ok := raw["archive"].(map[string]any); ok {
		if inc, ok := archiveRaw["include_docs"].(bool); ok {
			archiveConfig.IncludeDocs = inc
		}
	}

	return &Config{
		Registry:        parser.GetString("registry", "SWIFT_REGISTRY_URL", "https://swift.pkg.github.com"),
		Scope:           parser.GetString("scope", "SWIFT_PACKAGE_SCOPE", ""),
		Token:           parser.GetString("token", "SWIFT_REGISTRY_TOKEN", ""),
		PackageName:     parser.GetString("package_name", "", ""),
		ManifestPath:    parser.GetString("manifest_path", "", "Package.swift"),
		UpdateManifest:  parser.GetBool("update_manifest", false),
		VersionConstant: parser.GetString("version_constant", "", "packageVersion"),
		CreateTag:       parser.GetBool("create_tag", true),
		TagPrefix:       parser.GetString("tag_prefix", "", ""),
		Validate:        parser.GetBool("validate", true),
		Build:           parser.GetBool("build", true),
		Test:            parser.GetBool("test", true),
		TestConfig:      testConfig,
		Archive:         archiveConfig,
		DryRun:          parser.GetBool("dry_run", false),
	}
}

func createGitTag(ctx context.Context, tag string) error {
	cmd := exec.CommandContext(ctx, "git", "tag", tag)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git tag failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}
