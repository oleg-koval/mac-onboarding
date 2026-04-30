package shell

import (
	"fmt"
	"os"
	"os/exec"
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

func (m *Module) Name() string { return "shell" }

var rcFiles = []string{
	".zshrc", ".zprofile", ".zshenv", ".zlogin",
	".bashrc", ".bash_profile", ".bash_login",
	".p10k.zsh",
}

func (m *Module) Export(cfg *config.Config, opts runner.Options, dst string) error {
	home, _ := os.UserHomeDir()
	redactPat := redactPattern(cfg)

	for _, name := range rcFiles {
		path := filepath.Join(home, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue // file absent — skip silently
		}

		cleaned, count := redact(data, redactPat)
		if count > 0 {
			fmt.Printf("  shell: redacted %d secret line(s) from %s\n", count, name)
		}

		if opts.DryRun {
			fmt.Printf("  dry-run: would add %s to archive\n", name)
			continue
		}

		tmp, err := os.CreateTemp("", "mac-onboarding-shell-*")
		if err != nil {
			return err
		}
		tmp.Write(cleaned)
		tmp.Close()
		defer os.Remove(tmp.Name())

		if err := archive.AddFile(dst, tmp.Name(), "shell/"+name); err != nil {
			return fmt.Errorf("archive %s: %w", name, err)
		}
	}

	// Export oh-my-zsh custom dir (plugins + themes, not the core).
	omzCustom := filepath.Join(home, ".oh-my-zsh", "custom")
	if info, err := os.Stat(omzCustom); err == nil && info.IsDir() {
		if opts.DryRun {
			fmt.Println("  dry-run: would add .oh-my-zsh/custom to archive")
		} else {
			fmt.Println("  shell: exporting oh-my-zsh custom dir...")
			if err := archive.AddDir(dst, omzCustom, "shell/omz-custom"); err != nil {
				return fmt.Errorf("oh-my-zsh custom: %w", err)
			}
		}
	}

	return nil
}

func (m *Module) Install(cfg *config.Config, opts runner.Options, src string) error {
	home, _ := os.UserHomeDir()

	// Restore rc files.
	entries, _ := archive.ListEntries(src)
	for _, entry := range entries {
		if !strings.HasPrefix(entry, "shell/") {
			continue
		}
		name := strings.TrimPrefix(entry, "shell/")
		if strings.HasPrefix(name, "omz-custom/") {
			continue // handled separately below
		}

		dst := filepath.Join(home, name)
		if opts.DryRun {
			fmt.Printf("  dry-run: would restore %s → %s\n", entry, dst)
			continue
		}
		tmp, err := archive.ExtractFile(src, entry)
		if err != nil {
			fmt.Printf("  shell: skip %s (not in archive)\n", name)
			continue
		}
		data, _ := os.ReadFile(tmp)
		os.Remove(tmp)

		if err := os.WriteFile(dst, data, 0600); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}
		fmt.Printf("  shell: restored %s\n", name)
	}

	// Restore oh-my-zsh custom dir.
	omzCustom := filepath.Join(home, ".oh-my-zsh", "custom")
	hasCustom := false
	for _, e := range entries {
		if strings.HasPrefix(e, "shell/omz-custom/") {
			hasCustom = true
			break
		}
	}
	if hasCustom {
		if opts.DryRun {
			fmt.Printf("  dry-run: would restore oh-my-zsh custom → %s\n", omzCustom)
		} else {
			// Install oh-my-zsh core if absent.
			omzDir := filepath.Join(home, ".oh-my-zsh")
			if _, err := os.Stat(omzDir); err != nil {
				fmt.Println("  shell: installing oh-my-zsh core...")
				installOMZ()
			}
			if err := archive.ExtractDir(src, "shell/omz-custom", omzCustom); err != nil {
				return fmt.Errorf("oh-my-zsh custom restore: %w", err)
			}
			fmt.Println("  shell: oh-my-zsh custom dir restored")
		}
	}

	return nil
}

func installOMZ() {
	cmd := exec.Command("/bin/sh", "-c",
		`sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended`)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func redactPattern(cfg *config.Config) *regexp.Regexp {
	pat := `(?i)export\s+\w*(KEY|TOKEN|SECRET|PASSWORD|API|CREDENTIAL)\w*=.+`
	if cfg != nil {
		if m, ok := cfg.Modules["shell"]; ok {
			if custom, ok := m.Options["redact_pattern"]; ok && custom != "" {
				pat = custom
			}
		}
	}
	return regexp.MustCompile(pat)
}

func redact(data []byte, pat *regexp.Regexp) ([]byte, int) {
	lines := strings.Split(string(data), "\n")
	count := 0
	for i, line := range lines {
		if pat.MatchString(line) {
			lines[i] = "# REDACTED — restore manually"
			count++
		}
	}
	return []byte(strings.Join(lines, "\n")), count
}
