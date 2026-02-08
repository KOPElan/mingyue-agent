package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigPathFallsBackToLocal(t *testing.T) {
	if _, err := os.Stat(defaultConfigPath); err == nil {
		t.Skip("default config path exists in environment")
	}

	tempDir := t.TempDir()
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(workingDir)
	})

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	localConfig := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(localConfig, []byte("server:\n  http_port: 8081\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	resolved := resolveConfigPath(defaultConfigPath)
	if resolved != "config.yaml" {
		t.Fatalf("expected local config fallback, got %q", resolved)
	}
}

func TestResolveConfigPathUsesDefaultWhenMissing(t *testing.T) {
	if _, err := os.Stat(defaultConfigPath); err == nil {
		t.Skip("default config path exists in environment")
	}

	tempDir := t.TempDir()
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(workingDir)
	})

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	resolved := resolveConfigPath(defaultConfigPath)
	if resolved != defaultConfigPath {
		t.Fatalf("expected default config path, got %q", resolved)
	}
}

func TestResolveConfigPathKeepsCustomPath(t *testing.T) {
	customPath := "/tmp/mingyue-agent-custom.yaml"
	resolved := resolveConfigPath(customPath)
	if resolved != customPath {
		t.Fatalf("expected custom path unchanged, got %q", resolved)
	}
}
