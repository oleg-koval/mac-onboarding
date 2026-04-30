package bootstrap

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/oleg-koval/mac-onboarding/internal/config"
	"github.com/oleg-koval/mac-onboarding/internal/mdm"
	"github.com/oleg-koval/mac-onboarding/internal/runner"
)

func init() {
	runner.Register(&Module{})
}

type Module struct{}

func (m *Module) Name() string { return "bootstrap" }

func (m *Module) Export(cfg *config.Config, opts runner.Options, dst string) error {
	// Nothing to export — bootstrap is install-only.
	return nil
}

func (m *Module) Install(cfg *config.Config, opts runner.Options, src string) error {
	if s := mdm.Probe(); s.Enrolled {
		fmt.Println("  ⚠  MDM enrolled — some installs may be restricted or require IT approval")
	}

	if err := ensureXcodeCLT(opts.DryRun); err != nil {
		return err
	}
	if err := ensureHomebrew(opts.DryRun); err != nil {
		return err
	}
	return nil
}

func ensureXcodeCLT(dry bool) error {
	_, err := exec.LookPath("xcode-select")
	if err != nil {
		fmt.Println("  xcode-select: not found — skipping (unusual)")
		return nil
	}

	out, err := exec.Command("xcode-select", "-p").Output()
	if err == nil && len(out) > 0 {
		fmt.Println("  xcode-select: already installed")
		return nil
	}

	fmt.Println("  xcode-select: installing Command Line Tools...")
	if dry {
		fmt.Println("  dry-run: would run: xcode-select --install")
		return nil
	}

	// xcode-select --install opens a GUI dialog; inform user and wait.
	cmd := exec.Command("xcode-select", "--install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Exit code 1 means "already installed" on some macOS versions.
		fmt.Println("  xcode-select: install dialog triggered — re-run after it completes")
	}
	return nil
}

func ensureHomebrew(dry bool) error {
	if _, err := exec.LookPath("brew"); err == nil {
		fmt.Println("  homebrew: already installed")
		return nil
	}

	fmt.Println("  homebrew: not found — installing...")
	if dry {
		fmt.Println(`  dry-run: would run: /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`)
		return nil
	}

	cmd := exec.Command("/bin/bash", "-c",
		`/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("homebrew install: %w", err)
	}

	// Apple Silicon: brew lands in /opt/homebrew — add to PATH for this process.
	for _, candidate := range []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"} {
		if _, err := os.Stat(candidate); err == nil {
			_ = os.Setenv("PATH", candidate[:len(candidate)-4]+":"+os.Getenv("PATH"))
			break
		}
	}

	fmt.Println("  homebrew: installed")
	return nil
}
