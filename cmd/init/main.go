package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	owner     string
	repo      string
	goVersion string
	dryRun    bool
)

func main() {
	flag.StringVar(&owner, "owner", "", "The GitHub owner (username or org)")
	flag.StringVar(&repo, "repo", "", "The repository name")
	flag.StringVar(&goVersion, "go-version", "", "The Go version (e.g. 1.21)")
	flag.BoolVar(&dryRun, "dry-run", false, "Dry run mode (do not change files)")
	flag.Parse()

	// 1. Detect defaults
	detectedOwner, detectedRepo, err := detectGitRemote()
	if err == nil {
		if owner == "" {
			owner = detectedOwner
		}
		if repo == "" {
			repo = detectedRepo
		}
	} else {
		// Fallback detection
		if repo == "" {
			wd, _ := os.Getwd()
			repo = filepath.Base(wd)
		}
		if owner == "" {
			out, _ := exec.Command("git", "config", "user.name").Output()
			owner = strings.TrimSpace(string(out))
		}
	}

	if goVersion == "" {
		// Detect go version
		out, err := exec.Command("go", "version").Output()
		if err == nil {
			// Output example: go version go1.21.0 darwin/arm64
			re := regexp.MustCompile(`go(\d+\.\d+)`)
			match := re.FindStringSubmatch(string(out))
			if len(match) > 1 {
				goVersion = match[1]
			}
		}
		if goVersion == "" {
			goVersion = "1.21"
		}
	}

	// 2. Interactive prompt
	if !isFlagPassed("owner") || !isFlagPassed("repo") {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Owner [%s]: ", owner)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			owner = input
		}

		fmt.Printf("Repo [%s]: ", repo)
		input, _ = reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			repo = input
		}

		fmt.Printf("Go Version [%s]: ", goVersion)
		input, _ = reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			goVersion = input
		}
	}

	if owner == "" || repo == "" {
		fmt.Println("Error: Owner and Repo are required.")
		os.Exit(1)
	}

	fmt.Printf("Initializing with Owner: %s, Repo: %s, Go Version: %s\n", owner, repo, goVersion)
	if dryRun {
		fmt.Println("DRY RUN MODE: No changes will be applied.")
	}

	// 3. Process files
	err = filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip excluded dirs
		if info.IsDir() {
			if path == ".git" || path == ".idea" || path == ".vscode" || path == "bin" || path == "build" || path == "generated" {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip specific files
		if path == "init.sh" || strings.HasPrefix(path, "cmd/init/") {
			return nil
		}

		return processFile(path, info)
	})

	if err != nil {
		fmt.Printf("Error processing files: %v\n", err)
		os.Exit(1)
	}

	// 4. Git hooks
	if !dryRun {
		err := deployGitHooks()
		if err != nil {
			fmt.Printf("Warning: Failed to deploy git hooks: %v\n", err)
		} else {
			fmt.Println("Git hooks deployed.")
		}
	}

	// 5. Run Make
	if !dryRun {
		fmt.Println("Running build (make)...")
		cmd := exec.Command("make")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Warning: make failed: %v\n", err)
		} else {
			fmt.Println("Build successful.")
		}
	}

	// 6. Cleanup
	if !dryRun {
		fmt.Println("Cleaning up initialization scripts...")

		if err := os.Remove("init.sh"); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Warning: failed to delete init.sh: %v\n", err)
		}

		// Delete cmd/init directory
		// Since we are running from cmd/init/main.go (via go run), deleting the source file is usually fine on Unix.
		if err := os.RemoveAll("cmd/init"); err != nil {
			fmt.Printf("Warning: failed to delete cmd/init: %v\n", err)
		} else {
			fmt.Println("Cleanup complete.")
		}
	}
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func detectGitRemote() (string, string, error) {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", err
	}
	url := strings.TrimSpace(string(out))
	// Formats:
	// https://github.com/owner/repo.git
	// git@github.com:owner/repo.git

	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		repo := parts[len(parts)-1]
		ownerPart := parts[len(parts)-2]

		// Handle git@github.com:owner
		if strings.Contains(ownerPart, ":") {
			subParts := strings.Split(ownerPart, ":")
			ownerPart = subParts[len(subParts)-1]
		}
		return ownerPart, repo, nil
	}
	return "", "", fmt.Errorf("could not parse remote url")
}

func processFile(path string, info fs.FileInfo) error {
	// Skip binary files or other irrelevant files?
	ext := filepath.Ext(path)
	// We want to process .go, .md, .yml, .mod, Makefile
	if ext != ".go" && ext != ".md" && ext != ".yml" && ext != ".yaml" && filepath.Base(path) != "Makefile" && filepath.Base(path) != "go.mod" {
		return nil
	}

	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(contentBytes)
	originalContent := content

	// Replacements

	// 1. github.com/sha1n/go-template -> github.com/owner/repo
	content = strings.ReplaceAll(content, "github.com/sha1n/go-template", fmt.Sprintf("github.com/%s/%s", owner, repo))

	// 2. sha1n/go-template -> owner/repo (for badges, etc)
	content = strings.ReplaceAll(content, "sha1n/go-template", fmt.Sprintf("%s/%s", owner, repo))

	// 3. specific replacement for go-template -> repo in Makefile and maybe elsewhere
	content = strings.ReplaceAll(content, "go-template", repo)

	// 4. sha1n -> owner
	content = strings.ReplaceAll(content, "sha1n", owner)

	// 5. Go Version in go.mod
	if filepath.Base(path) == "go.mod" {
		re := regexp.MustCompile(`go \d+\.\d+`)
		content = re.ReplaceAllString(content, "go "+goVersion)
	}

	if content != originalContent {
		fmt.Printf("Updating %s...\n", path)
		if !dryRun {
			err = os.WriteFile(path, []byte(content), info.Mode())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func deployGitHooks() error {
	hooksDir := ".githooks"
	gitHooksDir := filepath.Join(".git", "hooks")

	entries, err := os.ReadDir(hooksDir)
	if err != nil {
		return err
	}

	if _, err := os.Stat(gitHooksDir); os.IsNotExist(err) {
		if err := os.MkdirAll(gitHooksDir, 0755); err != nil {
			return err
		}
	}

	for _, entry := range entries {
		srcPath := filepath.Join(hooksDir, entry.Name())
		dstPath := filepath.Join(gitHooksDir, entry.Name())

		input, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}

		err = os.WriteFile(dstPath, input, 0755)
		if err != nil {
			return err
		}
		fmt.Printf("Deployed hook: %s\n", entry.Name())
	}

	return nil
}
