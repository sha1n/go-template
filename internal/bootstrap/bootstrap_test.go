package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteReplacesMostSpecificFirst(t *testing.T) {
	in := `import "github.com/sha1n/go-template/pkg" // sha1n owns go-template`
	got := rewrite(in, "acme", "widget", "1.22", false)
	want := `import "github.com/acme/widget/pkg" // acme owns widget`
	if got != want {
		t.Fatalf("rewrite() = %q, want %q", got, want)
	}
}

func TestRewriteRewritesBadgePath(t *testing.T) {
	in := "https://github.com/sha1n/go-template/actions and sha1n/go-template"
	got := rewrite(in, "acme", "widget", "1.22", false)
	want := "https://github.com/acme/widget/actions and acme/widget"
	if got != want {
		t.Fatalf("rewrite() = %q, want %q", got, want)
	}
}

func TestRewriteSetsGoModVersion(t *testing.T) {
	in := "module github.com/sha1n/go-template\n\ngo 1.21\n\n// toolchain go1.21.1\n"
	got := rewrite(in, "acme", "widget", "1.22", true)
	want := "module github.com/acme/widget\n\ngo 1.22\n\n// toolchain go1.21.1\n"
	if got != want {
		t.Fatalf("rewrite() = %q, want %q", got, want)
	}
}

// Windows checkouts produce CRLF line endings. The go directive must still be
// rewritten, with the CRLF preserved. (Regression: the `$`-anchored regex used
// to fail to match `go 1.25.0\r`, leaving the version unchanged on Windows CI.)
func TestRewriteSetsGoModVersionWithCRLF(t *testing.T) {
	in := "module github.com/sha1n/go-template\r\n\r\ngo 1.25.0\r\n\r\n// toolchain go1.25.0\r\n"
	got := rewrite(in, "acme", "widget", "1.22", true)
	want := "module github.com/acme/widget\r\n\r\ngo 1.22\r\n\r\n// toolchain go1.25.0\r\n"
	if got != want {
		t.Fatalf("rewrite() = %q, want %q", got, want)
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertContains(t *testing.T, root, rel, want string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), want) {
		t.Errorf("%s does not contain %q. Got:\n%s", rel, want, data)
	}
}

func TestRunRewritesTreeAndDeploysHooks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module github.com/sha1n/go-template\n\ngo 1.21\n")
	writeFile(t, root, "README.md", "# go-template by sha1n\nhttps://github.com/sha1n/go-template\n")
	writeFile(t, root, "Makefile", "PROJECTNAME := \"go-template\"\n")
	writeFile(t, root, ".githooks/pre-commit", "#!/bin/sh\necho hi\n")

	changed, err := Run(root, Config{Owner: "acme", Repo: "widget", GoVersion: "1.22"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(changed) != 3 {
		t.Fatalf("changed = %v, want 3 files", changed)
	}
	assertContains(t, root, "go.mod", "module github.com/acme/widget")
	assertContains(t, root, "go.mod", "go 1.22")
	assertContains(t, root, "README.md", "https://github.com/acme/widget")
	assertContains(t, root, "Makefile", `PROJECTNAME := "widget"`)
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit")); err != nil {
		t.Fatalf("hook not deployed: %v", err)
	}

	changed2, err := Run(root, Config{Owner: "acme", Repo: "widget", GoVersion: "1.22"})
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if len(changed2) != 0 {
		t.Fatalf("second run changed %v, want none (idempotent)", changed2)
	}
}

func TestRunDryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module github.com/sha1n/go-template\n\ngo 1.21\n")

	changed, err := Run(root, Config{Owner: "acme", Repo: "widget", GoVersion: "1.22", DryRun: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(changed) != 1 {
		t.Fatalf("changed = %v, want 1 (reported but not written)", changed)
	}
	assertContains(t, root, "go.mod", "module github.com/sha1n/go-template") // unchanged on disk
}
