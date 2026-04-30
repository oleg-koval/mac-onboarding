package runner

import (
	"fmt"

	"github.com/oleg-koval/mac-onboarding/internal/config"
)

// Options carries flags common to export and install.
type Options struct {
	DryRun  bool
	Only    []string
	Verbose bool
	Output  string // export archive path
	Input   string // install archive path
}

// Module is the interface every module must satisfy.
type Module interface {
	Name() string
	Export(cfg *config.Config, opts Options, dst string) error
	Install(cfg *config.Config, opts Options, src string) error
}

var registry []Module

// Register adds a module to the global registry. Called from each module's init().
func Register(m Module) {
	registry = append(registry, m)
}

func Export(cfg *config.Config, opts Options) error {
	mods := select_(opts.Only)
	fmt.Printf("export: %d module(s) selected\n", len(mods))
	for _, m := range mods {
		if cfg.IsSkipped(m.Name()) {
			fmt.Printf("  skip  %s (disabled in config)\n", m.Name())
			continue
		}
		fmt.Printf("  export %s\n", m.Name())
		if opts.DryRun {
			continue
		}
		if err := m.Export(cfg, opts, opts.Output); err != nil {
			return fmt.Errorf("module %s export: %w", m.Name(), err)
		}
	}
	if !opts.DryRun {
		fmt.Printf("export: done → %s\n", opts.Output)
	} else {
		fmt.Println("dry-run: no changes made")
	}
	return nil
}

func Install(cfg *config.Config, opts Options) error {
	mods := select_(opts.Only)
	fmt.Printf("install: %d module(s) selected\n", len(mods))
	for _, m := range mods {
		if cfg.IsSkipped(m.Name()) {
			fmt.Printf("  skip  %s (disabled in config)\n", m.Name())
			continue
		}
		fmt.Printf("  install %s\n", m.Name())
		if opts.DryRun {
			continue
		}
		if err := m.Install(cfg, opts, opts.Input); err != nil {
			return fmt.Errorf("module %s install: %w", m.Name(), err)
		}
	}
	if !opts.DryRun {
		fmt.Println("install: done")
	} else {
		fmt.Println("dry-run: no changes made")
	}
	return nil
}

func select_(only []string) []Module {
	if len(only) == 0 {
		return registry
	}
	set := make(map[string]bool, len(only))
	for _, n := range only {
		set[n] = true
	}
	var out []Module
	for _, m := range registry {
		if set[m.Name()] {
			out = append(out, m)
		}
	}
	return out
}
