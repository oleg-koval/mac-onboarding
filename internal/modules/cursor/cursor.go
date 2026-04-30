package cursor

import (
	"bufio"
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

func (m *Module) Name() string { return "cursor" }

func (m *Module) Export(cfg *config.Config, opts runner.Options, dst string) error {
	userDir := settingsDir(cfg)

	for _, name := range []string{"settings.json", "keybindings.json"} {
		src := filepath.Join(userDir, name)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		if opts.DryRun {
			fmt.Printf("  dry-run: would add cursor/%s\n", name)
			continue
		}
		if err := archive.AddFile(dst, src, "cursor/"+name); err != nil {
			return fmt.Errorf("cursor %s: %w", name, err)
		}
	}

	// Snippets dir.
	snippetsDir := filepath.Join(userDir, "snippets")
	if info, err := os.Stat(snippetsDir); err == nil && info.IsDir() {
		if opts.DryRun {
			fmt.Println("  dry-run: would add cursor/snippets/")
		} else if err := archive.AddDir(dst, snippetsDir, "cursor/snippets"); err != nil {
			return fmt.Errorf("cursor snippets: %w", err)
		}
	}

	// Extension list.
	exts, err := listExtensions()
	if err != nil {
		fmt.Printf("  cursor: could not list extensions (%v) — skipping\n", err)
	} else if !opts.DryRun {
		tmp, _ := os.CreateTemp("", "cursor-extensions-*")
		tmp.WriteString(strings.Join(exts, "\n"))
		tmp.Close()
		defer os.Remove(tmp.Name())
		if err := archive.AddFile(dst, tmp.Name(), "cursor/extensions.txt"); err != nil {
			return fmt.Errorf("cursor extensions: %w", err)
		}
		fmt.Printf("  cursor: exported %d extension(s)\n", len(exts))
	} else {
		fmt.Printf("  dry-run: would export %d extension(s)\n", len(exts))
	}

	return nil
}

func (m *Module) Install(cfg *config.Config, opts runner.Options, src string) error {
	userDir := settingsDir(cfg)

	for _, name := range []string{"settings.json", "keybindings.json"} {
		if opts.DryRun {
			fmt.Printf("  dry-run: would restore cursor/%s\n", name)
			continue
		}
		tmp, err := archive.ExtractFile(src, "cursor/"+name)
		if err != nil {
			fmt.Printf("  cursor: %s not in archive — skipping\n", name)
			continue
		}
		data, _ := os.ReadFile(tmp)
		os.Remove(tmp)

		os.MkdirAll(userDir, 0755)
		if err := os.WriteFile(filepath.Join(userDir, name), data, 0600); err != nil {
			return fmt.Errorf("cursor write %s: %w", name, err)
		}
		fmt.Printf("  cursor: restored %s\n", name)
	}

	// Restore snippets.
	if !opts.DryRun {
		snippetsDir := filepath.Join(userDir, "snippets")
		os.MkdirAll(snippetsDir, 0755)
		_ = archive.ExtractDir(src, "cursor/snippets", snippetsDir)
	}

	// Install extensions.
	tmp, err := archive.ExtractFile(src, "cursor/extensions.txt")
	if err != nil {
		return nil // no extensions in archive
	}
	defer os.Remove(tmp)

	f, _ := os.Open(tmp)
	defer f.Close()

	cursor, err := exec.LookPath("cursor")
	if err != nil {
		fmt.Println("  cursor: binary not found — install Cursor first, then re-run")
		return nil
	}

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		ext := strings.TrimSpace(sc.Text())
		if ext == "" {
			continue
		}
		if opts.DryRun {
			fmt.Printf("  dry-run: would install extension %s\n", ext)
			continue
		}
		out, err := exec.Command(cursor, "--install-extension", ext).CombinedOutput()
		if err != nil {
			fmt.Printf("  cursor: extension %s failed: %s\n", ext, strings.TrimSpace(string(out)))
		} else {
			fmt.Printf("  cursor: installed %s\n", ext)
		}
	}
	return nil
}

func listExtensions() ([]string, error) {
	cursor, err := exec.LookPath("cursor")
	if err != nil {
		return nil, fmt.Errorf("cursor not in PATH")
	}
	out, err := exec.Command(cursor, "--list-extensions").Output()
	if err != nil {
		return nil, err
	}
	var exts []string
	for _, line := range strings.Split(string(out), "\n") {
		if e := strings.TrimSpace(line); e != "" {
			exts = append(exts, e)
		}
	}
	return exts, nil
}

func settingsDir(cfg *config.Config) string {
	if cfg != nil {
		if m, ok := cfg.Modules["cursor"]; ok {
			if p, ok := m.Options["settings_dir"]; ok && p != "" {
				return expandHome(p)
			}
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "Cursor", "User")
}

func expandHome(p string) string {
	if len(p) >= 2 && p[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}
