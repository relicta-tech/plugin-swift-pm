package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// RegistryClient wraps the Swift Package Registry API.
type RegistryClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewRegistryClient creates a new RegistryClient.
func NewRegistryClient(baseURL, token string) *RegistryClient {
	return &RegistryClient{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			},
		},
	}
}

// Release represents a package release.
type Release struct {
	Version   string            `json:"version"`
	Checksum  string            `json:"checksum"`
	Signature string            `json:"signature,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ListReleases lists all releases for a package.
func (c *RegistryClient) ListReleases(ctx context.Context, scope, name string) ([]Release, error) {
	endpoint := fmt.Sprintf("/%s/%s", scope, name)

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.swift.registry.v1+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return []Release{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list releases: status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response - simplified for now
	return []Release{}, nil
}

// GetRelease gets metadata for a specific release.
func (c *RegistryClient) GetRelease(ctx context.Context, scope, name, version string) (*Release, error) {
	endpoint := fmt.Sprintf("/%s/%s/%s", scope, name, version)

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.swift.registry.v1+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get release: status %d: %s", resp.StatusCode, string(body))
	}

	return &Release{Version: version}, nil
}

// Publish publishes a package version to the registry.
func (c *RegistryClient) Publish(ctx context.Context, scope, name, version, archivePath, checksum string) error {
	endpoint := fmt.Sprintf("/%s/%s/%s", scope, name, version)

	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat archive: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", c.baseURL+endpoint, file)
	if err != nil {
		return err
	}

	req.ContentLength = stat.Size()
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/zip")
	req.Header.Set("Accept", "application/vnd.swift.registry.v1+json")
	req.Header.Set("Digest", "sha-256="+checksum)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("publish request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("publish failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// VersionExists checks if a version already exists.
func (c *RegistryClient) VersionExists(ctx context.Context, scope, name, version string) (bool, error) {
	release, err := c.GetRelease(ctx, scope, name, version)
	if err != nil {
		return false, err
	}
	return release != nil, nil
}

// GetManifest retrieves the Package.swift for a specific version.
func (c *RegistryClient) GetManifest(ctx context.Context, scope, name, version string) (string, error) {
	endpoint := fmt.Sprintf("/%s/%s/%s/Package.swift", scope, name, version)

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "text/x-swift")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get manifest: status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
