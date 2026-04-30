package cmd

import (
	"fmt"
	"os"

	"github.com/oleg-koval/mac-onboarding/internal/config"
	"github.com/oleg-koval/mac-onboarding/internal/runner"
	"github.com/spf13/cobra"
)

var installInput string

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install configs and apps on this Mac from an archive",
	Long: `Reads the archive produced by 'export' and restores apps, configs,
and settings on this (target) Mac. MDM-managed paths are skipped safely.

Example:
  mac-onboarding install --input ~/onboard-20250430.tar.gz`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if installInput == "" {
			return fmt.Errorf("--input is required")
		}
		if _, err := os.Stat(installInput); err != nil {
			return fmt.Errorf("archive not found: %s", installInput)
		}

		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}

		opts := runner.Options{
			DryRun:  dryRun,
			Only:    only,
			Verbose: verbose,
			Input:   installInput,
		}

		if dryRun {
			fmt.Fprintln(os.Stdout, "dry-run: no changes will be made")
		}

		return runner.Install(cfg, opts)
	},
}

func init() {
	installCmd.Flags().StringVarP(&installInput, "input", "i", "", "input archive path (required)")
	rootCmd.AddCommand(installCmd)
}
