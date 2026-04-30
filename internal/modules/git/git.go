package git

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/oleg-koval/mac-onboarding/internal/archive"
	"github.com/oleg-koval/mac-onboarding/internal/config"
	"github.com/oleg-koval/mac-onboarding/internal/runner"
)

func init() {
	runner.Register(&Module{})
}

type Module struct{}

func (m *Module) Name() string { return "git" }

var gitFiles = []string{".gitconfig", ".gitignore_global"}

func (m *Module) Export(cfg *config.Config, opts runner.Options, dst string) error {
	home, _ := os.UserHomeDir()

	for _, name := range gitFiles {
		src := filepath.Join(home, name)
		data, err := os.ReadFile(src)
		if err != nil {
			continue // file absent — skip
		}

		// Redact credential.helper lines with embedded passwords.
		cleaned := redactCredentials(data)

		if opts.DryRun {
			fmt.Printf("  dry-run: would add git/%s\n", name)
			continue
		}

		tmp, _ := os.CreateTemp("", "mac-onboarding-git-*")
		tmp.Write(cleaned)
		tmp.Close()
		defer os.Remove(tmp.Name())

		if err := archive.AddFile(dst, tmp.Name(), "git/"+name); err != nil {
			return fmt.Errorf("git %s: %w", name, err)
		}
	}

	// Also check ~/.config/git/ (Git 2.13+).
	configDir := filepath.Join(home, ".config", "git")
	if info, err := os.Stat(configDir); err == nil && info.IsDir() {
		if opts.DryRun {
			fmt.Println("  dry-run: would add git/.config/ files")
		} else if err := archive.AddDir(dst, configDir, "git/.config"); err != nil {
			fmt.Printf("  git: .config/ failed: %v\n", err)
		}
	}

	fmt.Println("  git: exported (SSH keys not migrated — add to GitHub/GitLab manually)")
	return nil
}

func (m *Module) Install(cfg *config.Config, opts runner.Options, src string) error {
	home, _ := os.UserHomeDir()

	entries, _ := archive.ListEntries(src)
	for _, entry := range entries {
		if !strings.HasPrefix(entry, "git/") {
			continue
		}

		rel := strings.TrimPrefix(entry, "git/")
		dst := filepath.Join(home, rel)

		if opts.DryRun {
			fmt.Printf("  dry-run: would restore %s → %s\n", entry, dst)
			continue
		}

		tmp, err := archive.ExtractFile(src, entry)
		if err != nil {
			continue
		}
		data, _ := os.ReadFile(tmp)
		os.Remove(tmp)

		os.MkdirAll(filepath.Dir(dst), 0755)
		if err := os.WriteFile(dst, data, 0600); err != nil {
			return fmt.Errorf("git write %s: %w", rel, err)
		}
	}

	if !opts.DryRun {
		fmt.Println("  git: config restored")
		fmt.Println("  ⚠  git: add SSH keys to ~/.ssh/ and register with GitHub/GitLab")
	}
	return nil
}

// redactCredentials replaces credential.helper lines with embedded passwords.
func redactCredentials(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	credPat := regexp.MustCompile(`(?i)^\s*helper\s*=.*password.*`)
	for i, line := range lines {
		if credPat.MatchString(line) {
			lines[i] = "\t# REDACTED — restore manually if needed"
		}
	}
	return []byte(strings.Join(lines, "\n"))
}
