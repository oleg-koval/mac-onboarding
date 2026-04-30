package system

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/oleg-koval/mac-onboarding/internal/archive"
	"github.com/oleg-koval/mac-onboarding/internal/config"
	"github.com/oleg-koval/mac-onboarding/internal/runner"
)

func init() {
	runner.Register(&Module{})
}

type Module struct{}

func (m *Module) Name() string { return "system" }

// defaultsAllowlist: safe domains to export/restore. MDM-controlled domains are excluded.
var defaultsAllowlist = map[string]bool{
	"com.apple.dock":              true,
	"com.apple.finder":            true,
	"NSGlobalDomain":              true, // Key repeat, trackpad
	"com.apple.screensaver":       true,
	"com.apple.desktopservices":   true,
	"com.apple.LaunchServices":    true,
	"com.apple.systemuiserver":    true,
	"com.apple.menuextra.battery": true,
}

func (m *Module) Export(cfg *config.Config, opts runner.Options, dst string) error {
	if opts.DryRun {
		fmt.Println("  dry-run: would capture macOS system defaults")
		return nil
	}

	fmt.Println("  system: capturing defaults...")
	var manifest []string
	manifest = append(manifest, "# macOS defaults export")
	manifest = append(manifest, "# Restore with: defaults write <domain> <key> <value>")
	manifest = append(manifest, "")

	// Dock defaults.
	manifest = append(manifest, "# Dock")
	for _, key := range []string{"orientation", "position-immutable", "autohide", "show-recents"} {
		cmd := exec.Command("defaults", "read", "com.apple.dock", key)
		if out, err := cmd.Output(); err == nil {
			manifest = append(manifest, fmt.Sprintf("defaults write com.apple.dock %s -string \"%s\"", key, strings.TrimSpace(string(out))))
		}
	}

	// Finder defaults.
	manifest = append(manifest, "# Finder")
	cmd := exec.Command("defaults", "read", "com.apple.finder", "AppleShowAllFiles")
	if out, err := cmd.Output(); err == nil {
		manifest = append(manifest, fmt.Sprintf("defaults write com.apple.finder AppleShowAllFiles -bool %s", strings.TrimSpace(string(out))))
	}

	// Global defaults (keyboard repeat, trackpad, etc.)
	manifest = append(manifest, "# Keyboard & Input")
	for _, key := range []string{"KeyRepeat", "InitialKeyRepeat"} {
		cmd := exec.Command("defaults", "read", "NSGlobalDomain", key)
		if out, err := cmd.Output(); err == nil {
			manifest = append(manifest, fmt.Sprintf("defaults write NSGlobalDomain %s -int %s", key, strings.TrimSpace(string(out))))
		}
	}

	// Screenshot location.
	manifest = append(manifest, "# Screenshots")
	cmd = exec.Command("defaults", "read", "com.apple.screencapture", "location")
	if out, err := cmd.Output(); err == nil {
		loc := strings.TrimSpace(string(out))
		manifest = append(manifest, fmt.Sprintf("defaults write com.apple.screencapture location -string \"%s\"", loc))
	}

	// Write manifest to temp file and add to archive.
	content := strings.Join(manifest, "\n")
	tmp, _ := os.CreateTemp("", "mac-onboarding-system-*")
	tmp.WriteString(content)
	tmp.Close()
	defer os.Remove(tmp.Name())

	if err := archive.AddFile(dst, tmp.Name(), "system/defaults.sh"); err != nil {
		return fmt.Errorf("system defaults: %w", err)
	}

	fmt.Printf("  system: captured %d defaults\n", len(manifest)-3)
	return nil
}

func (m *Module) Install(cfg *config.Config, opts runner.Options, src string) error {
	if opts.DryRun {
		fmt.Println("  dry-run: would restore macOS system defaults")
		return nil
	}

	tmp, err := archive.ExtractFile(src, "system/defaults.sh")
	if err != nil {
		fmt.Println("  system: defaults not in archive — skipping")
		return nil
	}
	defer os.Remove(tmp)

	data, _ := os.ReadFile(tmp)
	lines := strings.Split(string(data), "\n")

	count := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 || parts[0] != "defaults" {
			continue
		}

		// Execute: defaults write <domain> <key> <type> <value>
		// Skip if domain is MDM-controlled.
		if len(parts) >= 3 {
			domain := parts[2]
			if !defaultsAllowlist[domain] {
				fmt.Printf("  system: skip restricted domain %s\n", domain)
				continue
			}
		}

		cmd := exec.Command("/bin/sh", "-c", line)
		if err := cmd.Run(); err != nil {
			fmt.Printf("  system: %s failed: %v\n", line, err)
			continue
		}
		count++
	}

	fmt.Printf("  system: restored %d defaults\n", count)
	fmt.Println("  system: restart Dock and Finder for changes to take effect:")
	fmt.Println("    killall Dock Finder")
	return nil
}
