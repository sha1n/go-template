package main

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// copyTree copies src into dst, skipping the same directories the engine skips
// for traversal plus the VCS metadata that need not be copied.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	skip := map[string]bool{".git": true, "bin": true, "build": true, "generated": true}
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		top := strings.SplitN(filepath.ToSlash(rel), "/", 2)[0]
		if d.IsDir() {
			if skip[top] {
				return fs.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatalf("copyTree: %v", err)
	}
}

func assertNoLeftoverTokens(t *testing.T, root string) {
	t.Helper()
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := filepath.ToSlash(mustRel(t, root, p))
		if d.IsDir() {
			if skipDirs[rel] {
				return fs.SkipDir
			}
			return nil
		}
		if !shouldProcess(rel) {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		s := string(data)
		if strings.Contains(s, "sha1n") || strings.Contains(s, "go-template") {
			t.Errorf("leftover template token in %s", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func mustRel(t *testing.T, base, p string) string {
	t.Helper()
	r, err := filepath.Rel(base, p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestScaffoldingOnRepoCopy(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	copyTree(t, repoRoot, dst)

	changed, err := Run(dst, Config{Owner: "acme", Repo: "widget", GoVersion: "1.22"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(changed) == 0 {
		t.Fatal("expected at least one file to change")
	}

	// (3) go.mod correct
	assertContains(t, dst, "go.mod", "module github.com/acme/widget")
	assertContains(t, dst, "go.mod", "go 1.22")

	// (2) no leftover tokens in processed files
	assertNoLeftoverTokens(t, dst)

	// (1) renamed copy still compiles
	build := exec.Command("go", "build", "-buildvcs=false", "./...")
	build.Dir = dst
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("renamed copy failed to build: %v\n%s", err, out)
	}

	// (4) hooks deployed + idempotent
	if _, err := os.Stat(filepath.Join(dst, ".git", "hooks", "pre-commit")); err != nil {
		t.Fatalf("hook not deployed: %v", err)
	}
	again, err := Run(dst, Config{Owner: "acme", Repo: "widget", GoVersion: "1.22"})
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("second run changed %v, want none (idempotent)", again)
	}
}
