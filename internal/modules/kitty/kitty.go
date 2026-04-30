package kitty

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oleg-koval/mac-onboarding/internal/archive"
	"github.com/oleg-koval/mac-onboarding/internal/config"
	"github.com/oleg-koval/mac-onboarding/internal/runner"
)

func init() {
	runner.Register(&Module{})
}

type Module struct{}

func (m *Module) Name() string { return "kitty" }

func (m *Module) Export(cfg *config.Config, opts runner.Options, dst string) error {
	cfgDir := configDir(cfg)

	if _, err := os.Stat(cfgDir); err != nil {
		fmt.Printf("  kitty: config dir not found at %s — skipping\n", cfgDir)
		return nil
	}

	if opts.DryRun {
		fmt.Printf("  dry-run: would archive %s → archive:kitty/\n", cfgDir)
		return nil
	}

	fmt.Printf("  kitty: exporting %s\n", cfgDir)
	return archive.AddDir(dst, cfgDir, "kitty")
}

func (m *Module) Install(cfg *config.Config, opts runner.Options, src string) error {
	cfgDir := configDir(cfg)

	if opts.DryRun {
		fmt.Printf("  dry-run: would restore archive:kitty/ → %s\n", cfgDir)
		return nil
	}

	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		return fmt.Errorf("kitty: mkdir %s: %w", cfgDir, err)
	}

	if err := archive.ExtractDir(src, "kitty", cfgDir); err != nil {
		return fmt.Errorf("kitty: extract: %w", err)
	}

	fmt.Printf("  kitty: config restored to %s\n", cfgDir)
	return nil
}

func configDir(cfg *config.Config) string {
	if cfg != nil {
		if m, ok := cfg.Modules["kitty"]; ok {
			if p, ok := m.Options["config_dir"]; ok && p != "" {
				return expandHome(p)
			}
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "kitty")
}

func expandHome(p string) string {
	if len(p) >= 2 && p[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}
