package updater

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	formulaName       = "mac-onboarding"
	envAutoUpdate     = "MAC_ONBOARDING_AUTOUPDATE"
	envAutoUpdateDone = "MAC_ONBOARDING_AUTOUPDATE_DONE"
)

func MaybeUpdate(version string, stderr io.Writer) error {
	if shouldSkip(version) {
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return nil
	}
	resolvedExe, err := filepath.EvalSymlinks(exe)
	if err != nil {
		resolvedExe = exe
	}
	stableExe, ok := brewStableExecutable(resolvedExe)
	if !ok {
		return nil
	}

	brewPath, err := exec.LookPath("brew")
	if err != nil {
		return nil
	}

	outdated, err := exec.Command(brewPath, "outdated", "--quiet", formulaName).Output()
	if err != nil || !isOutdated(outdated) {
		return nil
	}

	fmt.Fprintln(stderr, "autoupdate: upgrading mac-onboarding via Homebrew...")

	upgradeCmd := exec.Command(brewPath, "upgrade", formulaName)
	upgradeCmd.Stdout = stderr
	upgradeCmd.Stderr = stderr
	if err := upgradeCmd.Run(); err != nil {
		fmt.Fprintf(stderr, "autoupdate: brew upgrade failed: %v\n", err)
		return nil
	}

	env := append(os.Environ(), envAutoUpdateDone+"=1")
	if err := syscall.Exec(stableExe, os.Args, env); err != nil {
		fmt.Fprintf(stderr, "autoupdate: restart failed: %v\n", err)
	}

	return nil
}

func shouldSkip(version string) bool {
	if os.Getenv(envAutoUpdateDone) == "1" {
		return true
	}
	if strings.EqualFold(os.Getenv(envAutoUpdate), "0") {
		return true
	}
	if version == "" || version == "dev" {
		return true
	}
	return false
}

func brewStableExecutable(path string) (string, bool) {
	clean := filepath.Clean(path)

	for _, marker := range []string{
		string(filepath.Separator) + "Cellar" + string(filepath.Separator) + formulaName + string(filepath.Separator),
		string(filepath.Separator) + "Caskroom" + string(filepath.Separator) + formulaName + string(filepath.Separator),
	} {
		idx := strings.Index(clean, marker)
		if idx == -1 {
			continue
		}
		prefix := clean[:idx]
		if prefix == "" {
			return "", false
		}
		return filepath.Join(prefix, "bin", formulaName), true
	}

	return "", false
}

func isOutdated(out []byte) bool {
	for _, line := range bytes.Split(out, []byte{'\n'}) {
		name := strings.TrimSpace(string(line))
		if name == formulaName {
			return true
		}
	}
	return false
}
