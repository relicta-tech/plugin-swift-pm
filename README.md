# Swift Package Manager Plugin for Relicta

A Relicta plugin for publishing Swift packages to the Swift Package Registry.

## Features

- Publish Swift packages to registries (GitHub Packages, custom registries)
- Package.swift validation and parsing
- Automatic package building and testing
- Archive creation with checksum verification
- Version constant updates in manifests
- Git tag creation
- Dry-run mode for testing

## Installation

Download the pre-built binary from the [releases page](https://github.com/relicta-tech/plugin-swift-pm/releases) or build from source:

```bash
go build -o swift-pm .
```

## Configuration

Add the plugin to your `relicta.yaml`:

```yaml
plugins:
  - name: swift-pm
    enabled: true
    hooks:
      - PrePublish
      - PostPublish
    config:
      # Registry URL (required)
      registry: "https://swift.pkg.github.com"

      # Package scope/organization (required)
      scope: "myorg"

      # Authentication token (required, use env var)
      token: ${SWIFT_REGISTRY_TOKEN}

      # Package name (auto-detected from Package.swift if not set)
      package_name: ""

      # Path to Package.swift
      manifest_path: "Package.swift"

      # Update version constant in Package.swift
      update_manifest: false
      version_constant: "packageVersion"

      # Git tag creation
      create_tag: true
      tag_prefix: ""

      # Pre-publish validation
      validate: true
      build: true
      test: true

      # Test configuration
      test_config:
        configuration: "debug"
        coverage: false
        parallel: true

      # Archive options
      archive:
        include_docs: true
        exclude:
          - ".git"
          - ".build"
          - "Tests"
          - "*.xcodeproj"
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `SWIFT_REGISTRY_TOKEN` | Authentication token for the registry |
| `SWIFT_REGISTRY_URL` | Registry URL (overrides config) |
| `SWIFT_PACKAGE_SCOPE` | Package scope (overrides config) |
| `GITHUB_TOKEN` | GitHub token (for GitHub Packages) |

## GitHub Packages Setup

To publish to GitHub Packages:

1. Create a personal access token with `write:packages` scope
2. Set the token as `SWIFT_REGISTRY_TOKEN` or `GITHUB_TOKEN`
3. Configure the plugin:

```yaml
plugins:
  - name: swift-pm
    config:
      registry: "https://swift.pkg.github.com"
      scope: "your-org"
      token: ${GITHUB_TOKEN}
```

## Package.swift Requirements

Your Package.swift must include:

```swift
// swift-tools-version:5.7
import PackageDescription

let package = Package(
    name: "MyPackage",
    platforms: [
        .macOS(.v10_15),
        .iOS(.v13)
    ],
    products: [
        .library(name: "MyLibrary", targets: ["MyLibrary"])
    ],
    targets: [
        .target(name: "MyLibrary")
    ]
)
```

### Version Constants

To enable automatic version updates, add a version constant:

```swift
let packageVersion = "1.0.0"
```

Then configure the plugin:

```yaml
config:
  update_manifest: true
  version_constant: "packageVersion"
```

## Hooks

### PrePublish

Executed before the release is published:
- Validates Package.swift syntax
- Builds the package (`swift build`)
- Runs tests (`swift test`)
- Updates version constant (if enabled)

### PostPublish

Executed after successful release:
- Creates package archive
- Calculates SHA256 checksum
- Publishes to registry
- Creates git tag (if enabled)

## Dry Run

Test your configuration without publishing:

```bash
relicta publish --dry-run
```

## Archive Structure

The plugin creates a ZIP archive containing:
- Package.swift
- Sources/
- Documentation (if `include_docs: true`)

Default exclusions:
- `.git/`
- `.build/`
- `Tests/`
- `*.xcodeproj`

## Troubleshooting

### Swift CLI not found

Ensure Swift is installed and in your PATH:

```bash
swift --version
```

### Package validation fails

Check your Package.swift syntax:

```bash
swift package dump-package
```

### Registry authentication fails

Verify your token has the correct permissions:
- For GitHub Packages: `write:packages` scope
- For custom registries: consult registry documentation

### Build fails

Run the build manually to see detailed errors:

```bash
swift build -c release
```

## Development

### Running tests

```bash
go test -v ./...
```

### Building

```bash
go build -o swift-pm .
```

## License

MIT License - see [LICENSE](LICENSE) for details.
