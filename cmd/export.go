package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/oleg-koval/mac-onboarding/internal/config"
	"github.com/oleg-koval/mac-onboarding/internal/runner"
	"github.com/spf13/cobra"
)

var exportOutput string

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export configs and settings from this Mac into an archive",
	Long: `Packages your app configs, dotfiles, and settings into a portable archive.
Secrets are redacted. Run on your source (current) Mac.

Example:
  mac-onboarding export --output ~/onboard-$(date +%Y%m%d).tar.gz`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if exportOutput == "" {
			exportOutput = fmt.Sprintf("onboard-%s.tar.gz", time.Now().Format("20060102"))
		}

		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}

		opts := runner.Options{
			DryRun:  dryRun,
			Only:    only,
			Verbose: verbose,
			Output:  exportOutput,
		}

		if dryRun {
			fmt.Fprintln(os.Stdout, "dry-run: no changes will be made")
		}

		return runner.Export(cfg, opts)
	},
}

func init() {
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output archive path (default: onboard-YYYYMMDD.tar.gz)")
	rootCmd.AddCommand(exportCmd)
}
