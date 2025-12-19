package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CreateArchive creates a package archive for publishing.
// Returns the path to the archive file, its SHA256 checksum, and any error.
func CreateArchive(sourceDir, version string, cfg ArchiveConfig) (string, string, error) {
	// Create temp file for archive
	archiveFile, err := os.CreateTemp("", "swift-package-*.zip")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp file: %w", err)
	}
	archivePath := archiveFile.Name()

	zipWriter := zip.NewWriter(archiveFile)

	// Walk source directory
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Check exclusions
		if shouldExclude(relPath, cfg.Exclude) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories (they'll be created implicitly)
		if info.IsDir() {
			return nil
		}

		// Add file to archive
		return addFileToZip(zipWriter, path, relPath)
	})

	if err != nil {
		_ = zipWriter.Close()
		_ = archiveFile.Close()
		_ = os.Remove(archivePath)
		return "", "", fmt.Errorf("failed to create archive: %w", err)
	}

	if err := zipWriter.Close(); err != nil {
		_ = archiveFile.Close()
		_ = os.Remove(archivePath)
		return "", "", fmt.Errorf("failed to close zip writer: %w", err)
	}

	// Calculate checksum
	if _, err := archiveFile.Seek(0, 0); err != nil {
		_ = archiveFile.Close()
		_ = os.Remove(archivePath)
		return "", "", fmt.Errorf("failed to seek archive: %w", err)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, archiveFile); err != nil {
		_ = archiveFile.Close()
		_ = os.Remove(archivePath)
		return "", "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	checksum := hex.EncodeToString(hash.Sum(nil))

	if err := archiveFile.Close(); err != nil {
		_ = os.Remove(archivePath)
		return "", "", fmt.Errorf("failed to close archive: %w", err)
	}

	return archivePath, checksum, nil
}

// shouldExclude checks if a path matches any exclusion pattern.
func shouldExclude(path string, patterns []string) bool {
	for _, pattern := range patterns {
		// Handle directory patterns
		if strings.HasSuffix(pattern, "/") {
			dirPattern := strings.TrimSuffix(pattern, "/")
			if path == dirPattern || strings.HasPrefix(path, dirPattern+string(filepath.Separator)) {
				return true
			}
		}

		// Check direct match
		if path == pattern {
			return true
		}

		// Check if any path component matches
		parts := strings.Split(path, string(filepath.Separator))
		for _, part := range parts {
			// Handle glob patterns
			if matched, _ := filepath.Match(pattern, part); matched {
				return true
			}
		}

		// Check basename match for file patterns
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}
	return false
}

// addFileToZip adds a file to the zip archive.
func addFileToZip(zipWriter *zip.Writer, filePath, relativePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	// Use forward slashes for zip paths
	header.Name = strings.ReplaceAll(relativePath, string(filepath.Separator), "/")
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

// GetArchiveSize returns the size of a file in bytes.
func GetArchiveSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
