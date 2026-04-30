package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"

	"github.com/oleg-koval/mac-onboarding/internal/config"
	"github.com/oleg-koval/mac-onboarding/internal/runner"
	"github.com/spf13/cobra"
)

var bridgeCmd = &cobra.Command{
	Use:   "bridge",
	Short: "Pull configs live from source Mac via Tailscale SSH (no archive)",
	Long: `Bridge mode connects to your source Mac via Tailscale SSH and pulls
modules live without creating an intermediate archive.

Requires:
  - source.host set in onboard.yaml (your source Mac's Tailscale hostname)
  - Tailscale running on both Macs
  - Same username on both Macs

Example:
  mac-onboarding bridge pull --dry-run
  mac-onboarding bridge pull --only brew,shell`,
}

var bridgePullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull modules from source Mac via SSH",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}

		if cfg == nil || cfg.Source.Host == "" {
			return fmt.Errorf("bridge: source.host not set in config")
		}

		sourceHost := cfg.Source.Host
		currentUser, err := user.Current()
		if err != nil {
			return err
		}
		username := currentUser.Username

		if dryRun {
			fmt.Fprintf(os.Stderr, "dry-run: would pull from %s@%s\n", username, sourceHost)
			return nil
		}

		// Build remote export command
		remoteCmd := "mac-onboarding export --to-stdout"
		if len(only) > 0 {
			remoteCmd += " --only " + only[0]
			for _, m := range only[1:] {
				remoteCmd += "," + m
			}
		}

		sshTarget := fmt.Sprintf("%s@%s", username, sourceHost)
		sshCmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", sshTarget, remoteCmd)
		sshCmd.Stderr = os.Stderr

		fmt.Fprintf(os.Stderr, "bridge: pulling from %s...\n", sshTarget)

		// Get SSH stdout and pipe to temp file
		stdout, err := sshCmd.StdoutPipe()
		if err != nil {
			return err
		}

		if err := sshCmd.Start(); err != nil {
			return fmt.Errorf("bridge: ssh failed: %w", err)
		}

		// Write to temp file
		f, err := os.CreateTemp("", "mac-onboarding-bridge-*")
		if err != nil {
			sshCmd.Wait()
			return err
		}
		archivePath := f.Name()
		defer os.Remove(archivePath)

		if _, err := io.Copy(f, stdout); err != nil {
			f.Close()
			sshCmd.Wait()
			return err
		}
		f.Close()

		if err := sshCmd.Wait(); err != nil {
			return fmt.Errorf("bridge: ssh command failed: %w", err)
		}

		fmt.Fprintf(os.Stderr, "bridge: received, installing...\n")

		// Install from temp archive
		opts := runner.Options{
			DryRun:  false,
			Only:    only,
			Verbose: verbose,
			Input:   archivePath,
		}

		return runner.Install(cfg, opts)
	},
}

func init() {
	bridgePullCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would happen")
	bridgePullCmd.Flags().StringSliceVar(&only, "only", nil, "run only these modules")
	bridgePullCmd.Flags().BoolVar(&verbose, "verbose", false, "verbose output")
	bridgeCmd.AddCommand(bridgePullCmd)
	rootCmd.AddCommand(bridgeCmd)
}
