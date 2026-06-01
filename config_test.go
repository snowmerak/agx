package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCanonicalizePath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}

	cleaned, err := canonicalizePath(cwd)
	if err != nil {
		t.Fatalf("canonicalizePath failed: %v", err)
	}

	if runtime.GOOS == "windows" {
		if cleaned != strings.ToLower(cleaned) {
			t.Errorf("expected lowercase canonicalized path on Windows, got %q", cleaned)
		}
	}

	// Verify relative path resolution
	relPath := "."
	cleanedRel, err := canonicalizePath(relPath)
	if err != nil {
		t.Fatalf("canonicalizePath with relative path failed: %v", err)
	}

	if cleaned != cleanedRel {
		t.Errorf("expected %q and %q to be equal after canonicalization", cleaned, cleanedRel)
	}
}

func TestLoadSaveConfig(t *testing.T) {
	// Setup temp config file
	tempDir, err := os.MkdirTemp("", "agx-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempFile := filepath.Join(tempDir, "config.json")
	configPathOverride = tempFile
	defer func() { configPathOverride = "" }()

	// Test Load when file does not exist
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load non-existent config: %v", err)
	}

	if cfg.SystemPrompt != defaultSystemPrompt {
		t.Errorf("expected default system prompt %q, got %q", defaultSystemPrompt, cfg.SystemPrompt)
	}
	if len(cfg.Mappings) != 0 {
		t.Errorf("expected empty mappings, got %d items", len(cfg.Mappings))
	}

	// Test Save and Reload
	cfg.SystemPrompt = "Custom System Prompt"
	cfg.Mappings["/some/dir"] = "conv-1234"

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if loaded.SystemPrompt != "Custom System Prompt" {
		t.Errorf("expected 'Custom System Prompt', got %q", loaded.SystemPrompt)
	}
	if loaded.Mappings["/some/dir"] != "conv-1234" {
		t.Errorf("expected 'conv-1234', got %q", loaded.Mappings["/some/dir"])
	}
}
