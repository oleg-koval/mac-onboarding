package hotkeys

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/oleg-koval/mac-onboarding/internal/archive"
	"github.com/oleg-koval/mac-onboarding/internal/config"
	"github.com/oleg-koval/mac-onboarding/internal/runner"
)

func init() {
	runner.Register(&Module{})
}

type Module struct{}

func (m *Module) Name() string { return "hotkeys" }

func (m *Module) Export(cfg *config.Config, opts runner.Options, dst string) error {
	home, _ := os.UserHomeDir()
	plistPath := filepath.Join(home, "Library", "Preferences", "com.apple.symbolichotkeys.plist")

	if _, err := os.Stat(plistPath); err != nil {
		fmt.Println("  hotkeys: symbolichotkeys.plist not found — skipping")
		return nil
	}

	if opts.DryRun {
		fmt.Println("  dry-run: would export macOS hotkeys plist")
		return nil
	}

	data, _ := os.ReadFile(plistPath)
	tmp, _ := os.CreateTemp("", "mac-onboarding-hotkeys-*")
	tmp.Write(data)
	tmp.Close()
	defer os.Remove(tmp.Name())

	if err := archive.AddFile(dst, tmp.Name(), "hotkeys/symbolichotkeys.plist"); err != nil {
		return fmt.Errorf("hotkeys: %w", err)
	}

	fmt.Println("  hotkeys: exported")
	return nil
}

func (m *Module) Install(cfg *config.Config, opts runner.Options, src string) error {
	if opts.DryRun {
		fmt.Println("  dry-run: would restore hotkeys plist")
		return nil
	}

	tmp, err := archive.ExtractFile(src, "hotkeys/symbolichotkeys.plist")
	if err != nil {
		fmt.Println("  hotkeys: not in archive — skipping")
		return nil
	}
	defer os.Remove(tmp)

	home, _ := os.UserHomeDir()
	dstPath := filepath.Join(home, "Library", "Preferences", "com.apple.symbolichotkeys.plist")

	data, _ := os.ReadFile(tmp)
	os.MkdirAll(filepath.Dir(dstPath), 0755)
	if err := os.WriteFile(dstPath, data, 0600); err != nil {
		return fmt.Errorf("hotkeys write: %w", err)
	}

	fmt.Println("  hotkeys: restored")
	fmt.Println("  hotkeys: restart System Preferences for changes to take effect")

	// Restart the pbs daemon (pasteboard server) to pick up hotkey changes.
	exec.Command("/System/Library/CoreServices/pbs", "-flush").Run()
	return nil
}
