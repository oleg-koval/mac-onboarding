package brew

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/oleg-koval/mac-onboarding/internal/archive"
	"github.com/oleg-koval/mac-onboarding/internal/config"
	"github.com/oleg-koval/mac-onboarding/internal/runner"
)

func init() {
	runner.Register(&Module{})
}

type Module struct{}

func (m *Module) Name() string { return "brew" }

func (m *Module) Export(cfg *config.Config, opts runner.Options, dst string) error {
	brewfilePath := brewfileSrc(cfg)

	fmt.Printf("  brew: dumping Brewfile from %s\n", brewfilePath)
	if opts.DryRun {
		fmt.Printf("  dry-run: would run: brew bundle dump --force --file=%s\n", brewfilePath)
		return nil
	}

	// Dump current state to Brewfile.
	cmd := exec.Command("brew", "bundle", "dump", "--force", "--file="+brewfilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew bundle dump: %w", err)
	}

	// Add Brewfile to archive.
	return archive.AddFile(dst, brewfilePath, "brew/Brewfile")
}

func (m *Module) Install(cfg *config.Config, opts runner.Options, src string) error {
	// Extract Brewfile from archive to a temp location.
	tmp, err := archive.ExtractFile(src, "brew/Brewfile")
	if err != nil {
		return fmt.Errorf("brew: extract Brewfile: %w", err)
	}
	defer os.Remove(tmp)

	if opts.Verbose {
		data, _ := os.ReadFile(tmp)
		fmt.Printf("  brew: Brewfile contents:\n%s\n", string(data))
	}

	fmt.Println("  brew: running brew bundle install (failures are non-fatal)...")
	if opts.DryRun {
		fmt.Printf("  dry-run: would run: brew bundle install --no-lock --file=%s\n", tmp)
		return nil
	}

	// --no-lock avoids writing Brewfile.lock.json to unexpected locations.
	// We capture stderr so we can surface individual failures without aborting.
	cmd := exec.Command("brew", "bundle", "install", "--no-lock", "--file="+tmp)
	cmd.Stdout = os.Stdout
	// brew bundle install exits non-zero if any single package fails.
	// Capture stderr, print it, but treat overall as a warning rather than fatal.
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Extract failed entries for a clean summary.
		lines := strings.Split(string(out), "\n")
		var failed []string
		for _, l := range lines {
			if strings.Contains(l, "failed") || strings.Contains(l, "Error") {
				failed = append(failed, strings.TrimSpace(l))
			}
		}
		if len(failed) > 0 {
			fmt.Printf("  brew: %d package(s) failed (non-fatal):\n", len(failed))
			for _, f := range failed {
				fmt.Printf("    ! %s\n", f)
			}
		}
		// Not returning the error — partial installs are acceptable.
	} else {
		fmt.Println("  brew: all packages installed")
	}
	return nil
}

func brewfileSrc(cfg *config.Config) string {
	if p, ok := cfg.Modules["brew"].Options["brewfile_path"]; ok && p != "" {
		return expandHome(p)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".Brewfile")
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}
