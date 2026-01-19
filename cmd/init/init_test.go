package main

import (
	"os"
	"strings"
	"testing"
)

func TestInitRun(t *testing.T) {
	// Create a temp directory
	tempDir, err := os.MkdirTemp("", "init-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change working directory to temp dir
	originalWd, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}
	defer os.Chdir(originalWd)

	// Setup dummy files
	files := map[string]string{
		"go.mod": `module github.com/sha1n/go-template

go 1.21
`,
		"README.md": `# go-template
https://github.com/sha1n/go-template
`,
		"Makefile": `PROJECTNAME := "go-template"
`,
	}

	for name, content := range files {
		if err := os.WriteFile(name, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", name, err)
		}
	}

	// Create dummy .githooks dir
	if err := os.Mkdir(".githooks", 0755); err != nil {
		t.Fatalf("Failed to create .githooks: %v", err)
	}
	if err := os.WriteFile(".githooks/pre-commit", []byte("#!/bin/sh\necho hook"), 0755); err != nil {
		t.Fatalf("Failed to create hook: %v", err)
	}

	// Initialize git repo to allow hook deployment and remote detection (though we override it)
	// We just need .git directory for hook deployment
	if err := os.Mkdir(".git", 0755); err != nil {
		t.Fatalf("Failed to create .git: %v", err)
	}

	// Mocking exec commands is hard without dependency injection,
	// but since we pass explicit config, we bypass detection logic mostly.
	// However, `Run` calls `make` and `deployGitHooks`.
	// `deployGitHooks` works on files, so it's fine.
	// `Run` calls `make`. We don't have a Makefile that works, or make installed maybe.
	// But we wrote a Makefile. `make` might fail if it's not installed or the Makefile is invalid.
	// Let's create a dummy makefile that actually works or just suppress the error?
	// The `Run` function logs warning if make fails but doesn't return error. So it's fine.

	// Create a dummy cmd/init to test deletion
	if err := os.MkdirAll("cmd/init", 0755); err != nil {
		t.Fatalf("Failed to create cmd/init: %v", err)
	}

	// Create dummy init.sh
	if err := os.WriteFile("init.sh", []byte("echo init"), 0755); err != nil {
		t.Fatalf("Failed to create init.sh: %v", err)
	}


	config := Config{
		Owner:     "newowner",
		Repo:      "newrepo",
		GoVersion: "1.99",
		DryRun:    false,
	}

	// Run Init
	// We expect `make` to fail or print output, which is fine.
	// We expect files to be updated.
	if err := Run(config); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify go.mod
	content, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}
	sContent := string(content)
	if !strings.Contains(sContent, "module github.com/newowner/newrepo") {
		t.Errorf("go.mod module not updated. Got: %s", sContent)
	}
	if !strings.Contains(sContent, "go 1.99") {
		t.Errorf("go.mod version not updated. Got: %s", sContent)
	}

	// Verify README.md
	content, err = os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("Failed to read README.md: %v", err)
	}
	sContent = string(content)
	if !strings.Contains(sContent, "github.com/newowner/newrepo") {
		t.Errorf("README.md link not updated. Got: %s", sContent)
	}

	// Verify Makefile
	content, err = os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("Failed to read Makefile: %v", err)
	}
	sContent = string(content)
	if !strings.Contains(sContent, `PROJECTNAME := "newrepo"`) {
		t.Errorf("Makefile project name not updated. Got: %s", sContent)
	}

	// Verify git hooks
	if _, err := os.Stat(".git/hooks/pre-commit"); os.IsNotExist(err) {
		t.Errorf("Git hook not deployed")
	}

	// Verify cleanup
	if _, err := os.Stat("init.sh"); !os.IsNotExist(err) {
		t.Errorf("init.sh should have been deleted")
	}
	if _, err := os.Stat("cmd/init"); !os.IsNotExist(err) {
		t.Errorf("cmd/init should have been deleted")
	}
}
