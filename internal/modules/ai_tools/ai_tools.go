// Package ai_tools handles Claude Code, Codex, and pi.dev config export/install.
package ai_tools

import (
	"bufio"
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
	runner.Register(&claudeModule{})
	runner.Register(&codexModule{})
	runner.Register(&piModule{})
}

// ---- shared helpers --------------------------------------------------------

// tokenPattern matches common JSON/YAML credential fields.
var tokenPattern = regexp.MustCompile(`(?i)"(token|key|secret|password|api_key|auth)":\s*"[^"]+"`)

func redactTokens(data []byte) ([]byte, int) {
	count := 0
	result := tokenPattern.ReplaceAllFunc(data, func(match []byte) []byte {
		count++
		// Keep the key name, replace value.
		parts := tokenPattern.FindSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		return []byte(fmt.Sprintf(`"%s": "REDACTED"`, parts[1]))
	})
	return result, count
}

func expandHome(p string) string {
	if len(p) >= 2 && p[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

// ---- Claude Code -----------------------------------------------------------

type claudeModule struct{}

func (m *claudeModule) Name() string { return "claude" }

// claudeInclude lists files/dirs to export from ~/.claude/.
// Excludes: plugins cache (large, re-downloadable), session data.
var claudeInclude = []string{
	"CLAUDE.md",
	"settings.json",
	"settings.local.json",
	"keybindings.json",
	"RTK.md",
}

func (m *claudeModule) Export(cfg *config.Config, opts runner.Options, dst string) error {
	home, _ := os.UserHomeDir()
	base := expandHome(configPath(cfg, "claude", "config_dir", filepath.Join(home, ".claude")))

	for _, name := range claudeInclude {
		src := filepath.Join(base, name)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		data, _ := os.ReadFile(src)
		cleaned, n := redactTokens(data)
		if n > 0 {
			fmt.Printf("  claude: redacted %d token(s) in %s\n", n, name)
		}
		if opts.DryRun {
			fmt.Printf("  dry-run: would add claude/%s\n", name)
			continue
		}
		tmp, _ := os.CreateTemp("", "mac-onboarding-claude-*")
		tmp.Write(cleaned)
		tmp.Close()
		defer os.Remove(tmp.Name())
		if err := archive.AddFile(dst, tmp.Name(), "claude/"+name); err != nil {
			return fmt.Errorf("claude %s: %w", name, err)
		}
	}

	// Export memory/ dir if present.
	memDir := filepath.Join(base, "memory")
	if info, err := os.Stat(memDir); err == nil && info.IsDir() {
		if opts.DryRun {
			fmt.Println("  dry-run: would add claude/memory/")
		} else if err := archive.AddDir(dst, memDir, "claude/memory"); err != nil {
			return fmt.Errorf("claude memory: %w", err)
		}
	}

	// Export projects/ dir (CLAUDE.md files per project) — small metadata only.
	projDir := filepath.Join(base, "projects")
	if info, err := os.Stat(projDir); err == nil && info.IsDir() {
		if opts.DryRun {
			fmt.Println("  dry-run: would add claude/projects/ (CLAUDE.md files only)")
		} else {
			_ = filepath.Walk(projDir, func(path string, fi os.FileInfo, err error) error {
				if err != nil || fi.IsDir() || fi.Name() != "CLAUDE.md" {
					return nil
				}
				rel, _ := filepath.Rel(projDir, path)
				return archive.AddFile(dst, path, "claude/projects/"+rel)
			})
		}
	}

	return nil
}

func (m *claudeModule) Install(cfg *config.Config, opts runner.Options, src string) error {
	home, _ := os.UserHomeDir()
	base := expandHome(configPath(cfg, "claude", "config_dir", filepath.Join(home, ".claude")))

	if !opts.DryRun {
		// Check for conflicts
		if _, err := os.Stat(base); err == nil {
			// Directory exists
			if opts.Force {
				// Silent overwrite
			} else if opts.Backup {
				// Backup before extracting
				backupPath, err := archive.BackupDir(base)
				if err != nil {
					return fmt.Errorf("claude: backup failed: %w", err)
				}
				fmt.Printf("  claude: backed up %s → %s\n", base, backupPath)
			} else {
				// Prompt user
				fmt.Printf("  claude: %s exists. Overwrite? (y/N) ", base)
				scanner := bufio.NewScanner(os.Stdin)
				if !scanner.Scan() {
					return fmt.Errorf("claude: cancelled")
				}
				response := strings.ToLower(strings.TrimSpace(scanner.Text()))
				if response != "y" && response != "yes" {
					return fmt.Errorf("claude: cancelled")
				}
			}
		}
	}

	entries, _ := archive.ListEntries(src)
	for _, entry := range entries {
		if !strings.HasPrefix(entry, "claude/") {
			continue
		}
		rel := strings.TrimPrefix(entry, "claude/")
		dst := filepath.Join(base, rel)

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
			return fmt.Errorf("claude write %s: %w", rel, err)
		}
	}
	if !opts.DryRun {
		fmt.Printf("  claude: config restored to %s\n", base)
	}
	return nil
}

// ---- Codex -----------------------------------------------------------------

type codexModule struct{}

func (m *codexModule) Name() string { return "codex" }

// codexInclude: files and dirs to export from ~/.codex/
var codexInclude = []string{"config.toml", "AGENTS.md"}
var codexIncludeDirs = []string{"rules", "prompts", "automations"}

func (m *codexModule) Export(cfg *config.Config, opts runner.Options, dst string) error {
	home, _ := os.UserHomeDir()
	base := expandHome(configPath(cfg, "codex", "config_dir", filepath.Join(home, ".codex")))

	for _, name := range codexInclude {
		src := filepath.Join(base, name)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		if opts.DryRun {
			fmt.Printf("  dry-run: would add codex/%s\n", name)
			continue
		}
		if err := archive.AddFile(dst, src, "codex/"+name); err != nil {
			return fmt.Errorf("codex %s: %w", name, err)
		}
	}

	for _, dir := range codexIncludeDirs {
		srcDir := filepath.Join(base, dir)
		if info, err := os.Stat(srcDir); err != nil || !info.IsDir() {
			continue
		}
		if opts.DryRun {
			fmt.Printf("  dry-run: would add codex/%s/\n", dir)
			continue
		}
		if err := archive.AddDir(dst, srcDir, "codex/"+dir); err != nil {
			return fmt.Errorf("codex dir %s: %w", dir, err)
		}
	}

	fmt.Println("  codex: exported (auth.json and sqlite excluded)")
	return nil
}

func (m *codexModule) Install(cfg *config.Config, opts runner.Options, src string) error {
	home, _ := os.UserHomeDir()
	base := expandHome(configPath(cfg, "codex", "config_dir", filepath.Join(home, ".codex")))

	if !opts.DryRun {
		// Check for conflicts
		if _, err := os.Stat(base); err == nil {
			// Directory exists
			if opts.Force {
				// Silent overwrite
			} else if opts.Backup {
				// Backup before extracting
				backupPath, err := archive.BackupDir(base)
				if err != nil {
					return fmt.Errorf("codex: backup failed: %w", err)
				}
				fmt.Printf("  codex: backed up %s → %s\n", base, backupPath)
			} else {
				// Prompt user
				fmt.Printf("  codex: %s exists. Overwrite? (y/N) ", base)
				scanner := bufio.NewScanner(os.Stdin)
				if !scanner.Scan() {
					return fmt.Errorf("codex: cancelled")
				}
				response := strings.ToLower(strings.TrimSpace(scanner.Text()))
				if response != "y" && response != "yes" {
					return fmt.Errorf("codex: cancelled")
				}
			}
		}
	}

	entries, _ := archive.ListEntries(src)
	for _, entry := range entries {
		if !strings.HasPrefix(entry, "codex/") {
			continue
		}
		rel := strings.TrimPrefix(entry, "codex/")
		dst := filepath.Join(base, rel)

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
		os.WriteFile(dst, data, 0600)
	}
	if !opts.DryRun {
		fmt.Printf("  codex: config restored to %s\n", base)
		fmt.Println("  codex: NOTE — re-run auth separately (auth.json not migrated)")
	}
	return nil
}

// ---- pi.dev ----------------------------------------------------------------

type piModule struct{}

func (m *piModule) Name() string { return "pi" }

func (m *piModule) Export(cfg *config.Config, opts runner.Options, dst string) error {
	home, _ := os.UserHomeDir()
	base := expandHome(configPath(cfg, "pi", "config_dir", filepath.Join(home, ".pi")))

	if _, err := os.Stat(base); err != nil {
		fmt.Printf("  pi: config dir not found at %s — skipping\n", base)
		return nil
	}

	if opts.DryRun {
		fmt.Printf("  dry-run: would archive %s → archive:pi/\n", base)
		return nil
	}

	fmt.Printf("  pi: exporting %s\n", base)
	return archive.AddDir(dst, base, "pi")
}

func (m *piModule) Install(cfg *config.Config, opts runner.Options, src string) error {
	home, _ := os.UserHomeDir()
	base := expandHome(configPath(cfg, "pi", "config_dir", filepath.Join(home, ".pi")))

	if opts.DryRun {
		fmt.Printf("  dry-run: would restore archive:pi/ → %s\n", base)
		return nil
	}

	// Check for conflicts
	if _, err := os.Stat(base); err == nil {
		// Directory exists
		if opts.Force {
			// Silent overwrite
		} else if opts.Backup {
			// Backup before extracting
			backupPath, err := archive.BackupDir(base)
			if err != nil {
				return fmt.Errorf("pi: backup failed: %w", err)
			}
			fmt.Printf("  pi: backed up %s → %s\n", base, backupPath)
		} else {
			// Prompt user
			fmt.Printf("  pi: %s exists. Overwrite? (y/N) ", base)
			scanner := bufio.NewScanner(os.Stdin)
			if !scanner.Scan() {
				return fmt.Errorf("pi: cancelled")
			}
			response := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if response != "y" && response != "yes" {
				return fmt.Errorf("pi: cancelled")
			}
		}
	}

	os.MkdirAll(base, 0755)
	if err := archive.ExtractDir(src, "pi", base); err != nil {
		return fmt.Errorf("pi: extract: %w", err)
	}
	fmt.Printf("  pi: config restored to %s\n", base)
	return nil
}

// ---- shared config helper --------------------------------------------------

func configPath(cfg *config.Config, module, key, fallback string) string {
	if cfg == nil {
		return fallback
	}
	if m, ok := cfg.Modules[module]; ok {
		if p, ok := m.Options[key]; ok && p != "" {
			return p
		}
	}
	return fallback
}
