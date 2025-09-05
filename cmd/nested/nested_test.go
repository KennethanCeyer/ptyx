package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetProjectRoot(t *testing.T) {
	root, err := getProjectRoot()
	if err != nil {
		t.Fatalf("getProjectRoot() failed: %v", err)
	}

	if root == "" {
		t.Error("getProjectRoot() returned an empty string")
	}

	goModPath := filepath.Join(root, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		t.Errorf("go.mod not found in the determined project root: %s", root)
	}
}
