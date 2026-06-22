package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// goDirective matches the `go X.Y[.Z]` version line in go.mod. It is anchored to
// the line start but deliberately NOT to the line end: a `$` end-anchor fails to
// match a CRLF-terminated line (`go 1.25.0\r`), so Windows checkouts would
// silently skip the rewrite. Replacing only `go X.Y[.Z]` leaves any trailing
// `\r`/`\n` untouched, preserving the file's line endings.
var goDirective = regexp.MustCompile(`(?m)^go \d+\.\d+(\.\d+)?`)

var skipDirs = map[string]bool{
	".git": true, ".idea": true, ".vscode": true,
	"bin": true, "build": true, "generated": true,
	"internal/bootstrap": true,
}

type Config struct {
	Owner     string
	Repo      string
	GoVersion string
	DryRun    bool
}

// rewrite applies the template renames to content. Replacements are
// most-specific-first via a single non-overlapping pass, so already-rewritten
// text is never re-scanned. When isGoMod is true the `go X.Y` directive is also
// set to goVersion.
func rewrite(content, owner, repo, goVersion string, isGoMod bool) string {
	r := strings.NewReplacer(
		"github.com/sha1n/go-template", "github.com/"+owner+"/"+repo,
		"sha1n/go-template", owner+"/"+repo,
		"go-template", repo,
		"sha1n", owner,
	)
	content = r.Replace(content)
	if isGoMod {
		content = goDirective.ReplaceAllString(content, "go "+goVersion)
	}
	return content
}

func shouldProcess(rel string) bool {
	switch path.Base(rel) {
	case "go.mod", "Makefile":
		return true
	}
	switch path.Ext(rel) {
	case ".go", ".md", ".yml", ".yaml":
		return true
	}
	return false
}

// Run applies the template renames under root and deploys git hooks (unless
// DryRun). It returns the slash-form relative paths of files whose content
// changed (or, in dry-run, would change).
func Run(root string, cfg Config) ([]string, error) {
	var changed []string
	walkErr := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if skipDirs[rel] {
				return fs.SkipDir
			}
			return nil
		}
		if !shouldProcess(rel) {
			return nil
		}
		ok, err := processFile(p, cfg)
		if err != nil {
			return err
		}
		if ok {
			changed = append(changed, rel)
		}
		return nil
	})
	if walkErr != nil {
		return changed, fmt.Errorf("processing files: %w", walkErr)
	}
	if !cfg.DryRun {
		if err := deployGitHooks(root); err != nil {
			return changed, fmt.Errorf("deploying git hooks: %w", err)
		}
	}
	return changed, nil
}

func processFile(fp string, cfg Config) (bool, error) {
	info, err := os.Stat(fp)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(fp)
	if err != nil {
		return false, err
	}
	orig := string(data)
	out := rewrite(orig, cfg.Owner, cfg.Repo, cfg.GoVersion, filepath.Base(fp) == "go.mod")
	if out == orig {
		return false, nil
	}
	if cfg.DryRun {
		return true, nil
	}
	if err := os.WriteFile(fp, []byte(out), info.Mode().Perm()); err != nil {
		return false, err
	}
	return true, nil
}

// deployGitHooks copies root/.githooks/* into root/.git/hooks/ (executable).
// It is a no-op when no .githooks directory exists.
func deployGitHooks(root string) error {
	srcDir := filepath.Join(root, ".githooks")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	dstDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(srcDir, e.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dstDir, e.Name()), data, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	var cfg Config
	flag.StringVar(&cfg.Owner, "owner", "", "GitHub owner (username or org) [required]")
	flag.StringVar(&cfg.Repo, "repo", "", "Repository name [required]")
	flag.StringVar(&cfg.GoVersion, "go-version", "", "Go version, e.g. 1.22 [required]")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "Preview changes without writing")
	flag.Parse()

	if cfg.Owner == "" || cfg.Repo == "" || cfg.GoVersion == "" {
		fmt.Fprintln(os.Stderr, "Error: --owner, --repo and --go-version are all required")
		flag.Usage()
		os.Exit(2)
	}

	changed, err := Run(".", cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	verb := "Updated"
	if cfg.DryRun {
		verb = "Would update"
	}
	for _, f := range changed {
		fmt.Printf("%s %s\n", verb, f)
	}
	fmt.Printf("%s %d file(s).\n", verb, len(changed))
}
