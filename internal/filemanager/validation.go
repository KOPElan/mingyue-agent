package filemanager

import (
	"fmt"
	"path/filepath"
	"strings"
)

type PathValidator struct {
	allowedPaths []string
}

func NewPathValidator(allowedPaths []string) *PathValidator {
	normalized := make([]string, len(allowedPaths))
	for i, p := range allowedPaths {
		normalized[i] = filepath.Clean(p)
	}
	return &PathValidator{
		allowedPaths: normalized,
	}
}

func (v *PathValidator) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	cleanPath := filepath.Clean(path)

	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal detected")
	}

	if !filepath.IsAbs(cleanPath) {
		return fmt.Errorf("path must be absolute")
	}

	if strings.ContainsAny(cleanPath, "\x00") {
		return fmt.Errorf("null byte in path")
	}

	allowed := false
	for _, allowedPath := range v.allowedPaths {
		rel, err := filepath.Rel(allowedPath, cleanPath)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(rel, "..") {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("path not in allowed directories")
	}

	return nil
}

func (v *PathValidator) ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if strings.ContainsAny(name, "/\x00") {
		return fmt.Errorf("invalid characters in name")
	}

	if name == "." || name == ".." {
		return fmt.Errorf("invalid name")
	}

	return nil
}
