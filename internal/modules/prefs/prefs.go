// Package prefs handles macOS app preferences via plist export/import.
// Each app is a sub-module that registers itself; this file provides shared helpers.
package prefs

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
	runner.Register(&swiftbarModule{})
	runner.Register(&alfredModule{})
	runner.Register(&simplePlistModule{"klack", "com.trphotography.Klack", "Klack"})
	runner.Register(&simplePlistModule{"flux", "org.herf.Flux", "f.lux"})
	runner.Register(&simplePlistModule{"betterdisplay", "pro.betterdisplay.BetterDisplay", "BetterDisplay"})
	runner.Register(&simplePlistModule{"tailscale", "io.tailscale.ipn.macos", "Tailscale"})
	runner.Register(&simplePlistModule{"shottr", "cc.ffitch.shottr", "Shottr"})
	runner.Register(&simplePlistModule{"orbstack", "dev.orbstack.OrbStack", "OrbStack"})
	runner.Register(&synologyModule{})
	runner.Register(&onepasswordModule{})
}

// ---- generic plist helpers -------------------------------------------------

func exportPlist(domain, archiveName, dst string, dry bool) error {
	tmp, err := os.CreateTemp("", "mac-onboarding-plist-*")
	if err != nil {
		return err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	if dry {
		fmt.Printf("  dry-run: would export plist %s\n", domain)
		return nil
	}

	out, err := exec.Command("defaults", "export", domain, tmp.Name()).CombinedOutput()
	if err != nil {
		// Domain may not exist yet — skip silently.
		fmt.Printf("  prefs: %s not found (%s) — skipping\n", domain, strings.TrimSpace(string(out)))
		return nil
	}

	return archive.AddFile(dst, tmp.Name(), archiveName)
}

func importPlist(archivePath, archiveName, domain string, dry bool) error {
	if dry {
		fmt.Printf("  dry-run: would import plist %s → %s\n", archiveName, domain)
		return nil
	}

	tmp, err := archive.ExtractFile(archivePath, archiveName)
	if err != nil {
		fmt.Printf("  prefs: %s not in archive — skipping\n", archiveName)
		return nil
	}
	defer os.Remove(tmp)

	out, err := exec.Command("defaults", "import", domain, tmp).CombinedOutput()
	if err != nil {
		fmt.Printf("  prefs: import %s failed: %s\n", domain, strings.TrimSpace(string(out)))
	}
	return nil
}

// ---- SwiftBar --------------------------------------------------------------

type swiftbarModule struct{}

func (m *swiftbarModule) Name() string { return "swiftbar" }

func (m *swiftbarModule) Export(cfg *config.Config, opts runner.Options, dst string) error {
	home, _ := os.UserHomeDir()
	pluginsDir := filepath.Join(home, "Library", "Application Support", "SwiftBar")

	if _, err := os.Stat(pluginsDir); err != nil {
		fmt.Println("  swiftbar: plugins dir not found — skipping")
		return nil
	}
	if opts.DryRun {
		fmt.Printf("  dry-run: would archive SwiftBar plugins from %s\n", pluginsDir)
		return nil
	}
	if err := archive.AddDir(dst, pluginsDir, "swiftbar/plugins"); err != nil {
		return fmt.Errorf("swiftbar plugins: %w", err)
	}
	return exportPlist("com.ameba.SwiftBar", "swiftbar/prefs.plist", dst, opts.DryRun)
}

func (m *swiftbarModule) Install(cfg *config.Config, opts runner.Options, src string) error {
	home, _ := os.UserHomeDir()
	pluginsDir := filepath.Join(home, "Library", "Application Support", "SwiftBar")

	if !opts.DryRun {
		os.MkdirAll(pluginsDir, 0755)
		if err := archive.ExtractDir(src, "swiftbar/plugins", pluginsDir); err != nil {
			fmt.Printf("  swiftbar: plugins restore failed: %v\n", err)
		}
		// Make plugin scripts executable.
		filepath.Walk(pluginsDir, func(path string, fi os.FileInfo, err error) error {
			if err == nil && !fi.IsDir() {
				os.Chmod(path, 0755)
			}
			return nil
		})
	}
	return importPlist(src, "swiftbar/prefs.plist", "com.ameba.SwiftBar", opts.DryRun)
}

// ---- Alfred ----------------------------------------------------------------

type alfredModule struct{}

func (m *alfredModule) Name() string { return "alfred" }

func (m *alfredModule) Export(cfg *config.Config, opts runner.Options, dst string) error {
	syncDir := alfredSyncDir(cfg)
	if syncDir == "" {
		fmt.Println("  alfred: no sync_dir configured — skipping")
		fmt.Println("         set modules.alfred.options.sync_dir in onboard.yaml")
		return nil
	}
	if _, err := os.Stat(syncDir); err != nil {
		fmt.Printf("  alfred: sync dir not found at %s — skipping\n", syncDir)
		return nil
	}
	if opts.DryRun {
		fmt.Printf("  dry-run: would archive Alfred sync dir %s\n", syncDir)
		fmt.Println("  ⚠  alfred: workflows may contain credentials — review archive before sharing")
		return nil
	}
	fmt.Println("  ⚠  alfred: workflows may contain credentials — review archive before sharing")
	return archive.AddDir(dst, syncDir, "alfred/sync")
}

func (m *alfredModule) Install(cfg *config.Config, opts runner.Options, src string) error {
	syncDir := alfredSyncDir(cfg)
	if syncDir == "" {
		fmt.Println("  alfred: no sync_dir configured in onboard.yaml — skipping")
		return nil
	}
	if opts.DryRun {
		fmt.Printf("  dry-run: would restore Alfred sync → %s\n", syncDir)
		return nil
	}
	os.MkdirAll(syncDir, 0755)
	if err := archive.ExtractDir(src, "alfred/sync", syncDir); err != nil {
		return fmt.Errorf("alfred: %w", err)
	}
	fmt.Printf("  alfred: sync dir restored to %s\n", syncDir)
	fmt.Println("  alfred: open Alfred → Preferences → Advanced → set sync folder to above path")
	return nil
}

func alfredSyncDir(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	if m, ok := cfg.Modules["alfred"]; ok {
		if p, ok := m.Options["sync_dir"]; ok && p != "" {
			return expandHome(p)
		}
	}
	return ""
}

// ---- Simple plist-only apps ------------------------------------------------

type simplePlistModule struct {
	name_   string
	domain  string
	appName string
}

func (m *simplePlistModule) Name() string { return m.name_ }
func (m *simplePlistModule) Export(cfg *config.Config, opts runner.Options, dst string) error {
	return exportPlist(m.domain, m.name_+"/prefs.plist", dst, opts.DryRun)
}
func (m *simplePlistModule) Install(cfg *config.Config, opts runner.Options, src string) error {
	if err := importPlist(src, m.name_+"/prefs.plist", m.domain, opts.DryRun); err != nil {
		return err
	}
	if !opts.DryRun {
		fmt.Printf("  %s: preferences restored (restart the app to apply)\n", m.appName)
	}
	return nil
}


// ---- Synology --------------------------------------------------------------

type synologyModule struct{}

func (m *synologyModule) Name() string { return "synology" }

func (m *synologyModule) Export(cfg *config.Config, opts runner.Options, dst string) error {
	nasHost := synologyHost(cfg)
	if nasHost == "" {
		fmt.Println("  synology: no nas_host configured — skipping")
		return nil
	}
	if opts.DryRun {
		fmt.Printf("  dry-run: would record NAS host %s\n", nasHost)
		return nil
	}
	// Store hostname only — credentials are never saved.
	tmp, _ := os.CreateTemp("", "mac-onboarding-synology-*")
	tmp.WriteString("nas_host=" + nasHost + "\n")
	tmp.Close()
	defer os.Remove(tmp.Name())
	fmt.Printf("  synology: recording NAS host %s (credentials NOT stored)\n", nasHost)
	return archive.AddFile(dst, tmp.Name(), "synology/config.env")
}

func (m *synologyModule) Install(cfg *config.Config, opts runner.Options, src string) error {
	if opts.DryRun {
		fmt.Println("  dry-run: would print Synology setup instructions")
		return nil
	}
	tmp, err := archive.ExtractFile(src, "synology/config.env")
	if err != nil {
		fmt.Println("  synology: not in archive — skipping")
		return nil
	}
	data, _ := os.ReadFile(tmp)
	os.Remove(tmp)

	fmt.Println("  synology: setup instructions")
	fmt.Println("  ─────────────────────────────────────────────")
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "nas_host=") {
			host := strings.TrimPrefix(line, "nas_host=")
			fmt.Printf("  1. Install Synology Drive Client from https://www.synology.com/en-us/support/download\n")
			fmt.Printf("  2. Open Synology Drive Client → Set Up Now\n")
			fmt.Printf("  3. Server address: %s\n", host)
			fmt.Println("  4. Enter your Synology credentials when prompted")
		}
	}
	fmt.Println("  ─────────────────────────────────────────────")
	return nil
}

func synologyHost(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	if m, ok := cfg.Modules["synology"]; ok {
		if h, ok := m.Options["nas_host"]; ok {
			return h
		}
	}
	return ""
}

// ---- 1Password (guide only) ------------------------------------------------

type onepasswordModule struct{}

func (m *onepasswordModule) Name() string { return "onepassword" }

func (m *onepasswordModule) Export(cfg *config.Config, opts runner.Options, dst string) error {
	fmt.Println("  1password: vault cannot be exported automatically (by design)")
	return nil
}

func (m *onepasswordModule) Install(cfg *config.Config, opts runner.Options, src string) error {
	fmt.Println("  1password: setup guide")
	fmt.Println("  ─────────────────────────────────────────────")
	fmt.Println("  1. Install 1Password from https://1password.com/downloads/mac/")
	fmt.Println("  2. Open 1Password → Sign In with existing account")
	fmt.Println("  3. Use your Secret Key + Master Password (not stored here)")
	fmt.Println("  4. Vaults sync automatically after sign-in")
	fmt.Println("  ─────────────────────────────────────────────")
	return nil
}

// ---- shared helpers --------------------------------------------------------

func expandHome(p string) string {
	if len(p) >= 2 && p[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}
