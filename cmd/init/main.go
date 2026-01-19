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

type Config struct {
	Owner     string
	Repo      string
	GoVersion string
	DryRun    bool
}

func main() {
	var config Config
	flag.StringVar(&config.Owner, "owner", "", "The GitHub owner (username or org)")
	flag.StringVar(&config.Repo, "repo", "", "The repository name")
	flag.StringVar(&config.GoVersion, "go-version", "", "The Go version (e.g. 1.21)")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Dry run mode (do not change files)")
	flag.Parse()

	if err := Run(config); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func Run(config Config) error {
	// 1. Detect defaults
	detectedOwner, detectedRepo, err := detectGitRemote()
	if err == nil {
		if config.Owner == "" {
			config.Owner = detectedOwner
		}
		if config.Repo == "" {
			config.Repo = detectedRepo
		}
	} else {
		// Fallback detection
		if config.Repo == "" {
			wd, _ := os.Getwd()
			config.Repo = filepath.Base(wd)
		}
		if config.Owner == "" {
			out, _ := exec.Command("git", "config", "user.name").Output()
			config.Owner = strings.TrimSpace(string(out))
		}
	}

	if config.GoVersion == "" {
		// Detect go version
		out, err := exec.Command("go", "version").Output()
		if err == nil {
			// Output example: go version go1.21.0 darwin/arm64
			re := regexp.MustCompile(`go(\d+\.\d+)`)
			match := re.FindStringSubmatch(string(out))
			if len(match) > 1 {
				config.GoVersion = match[1]
			}
		}
		if config.GoVersion == "" {
			config.GoVersion = "1.21"
		}
	}

	// 2. Interactive prompt
	if (config.Owner == "" || config.Repo == "") && !isFlagPassed("owner") && !isFlagPassed("repo") {
		// Only prompt if stdin is a terminal? For now just prompt if missing.
		// Check if we are in a non-interactive environment would be better, but let's stick to simple logic.
		reader := bufio.NewReader(os.Stdin)

		if config.Owner == "" {
			fmt.Printf("Owner [%s]: ", config.Owner)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "" {
				config.Owner = input
			}
		}

		if config.Repo == "" {
			fmt.Printf("Repo [%s]: ", config.Repo)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "" {
				config.Repo = input
			}
		}

		fmt.Printf("Go Version [%s]: ", config.GoVersion)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			config.GoVersion = input
		}
	}

	if config.Owner == "" || config.Repo == "" {
		return fmt.Errorf("owner and Repo are required")
	}

	fmt.Printf("Initializing with Owner: %s, Repo: %s, Go Version: %s\n", config.Owner, config.Repo, config.GoVersion)
	if config.DryRun {
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

		return processFile(path, info, config)
	})

	if err != nil {
		return fmt.Errorf("error processing files: %v", err)
	}

	// 4. Git hooks
	if !config.DryRun {
		err := deployGitHooks()
		if err != nil {
			fmt.Printf("Warning: Failed to deploy git hooks: %v\n", err)
		} else {
			fmt.Println("Git hooks deployed.")
		}
	}

	// 5. Run Make
	if !config.DryRun {
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
	if !config.DryRun {
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

	return nil
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

func processFile(path string, info fs.FileInfo, config Config) error {
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
	content = strings.ReplaceAll(content, "github.com/sha1n/go-template", fmt.Sprintf("github.com/%s/%s", config.Owner, config.Repo))

	// 2. sha1n/go-template -> owner/repo (for badges, etc)
	content = strings.ReplaceAll(content, "sha1n/go-template", fmt.Sprintf("%s/%s", config.Owner, config.Repo))

	// 3. specific replacement for go-template -> repo in Makefile and maybe elsewhere
	content = strings.ReplaceAll(content, "go-template", config.Repo)

	// 4. sha1n -> owner
	content = strings.ReplaceAll(content, "sha1n", config.Owner)

	// 5. Go Version in go.mod
	if filepath.Base(path) == "go.mod" {
		re := regexp.MustCompile(`go \d+\.\d+`)
		content = re.ReplaceAllString(content, "go "+config.GoVersion)
	}

	if content != originalContent {
		fmt.Printf("Updating %s...\n", path)
		if !config.DryRun {
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
