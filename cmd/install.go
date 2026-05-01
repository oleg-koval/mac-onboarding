package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/oleg-koval/mac-onboarding/internal/config"
	"github.com/oleg-koval/mac-onboarding/internal/runner"
	"github.com/spf13/cobra"
)

var installInput string
var installFromStdin bool

var installCmd = &cobra.Command{
	Use:   "install [ARCHIVE_PATH]",
	Short: "Install configs and apps on this Mac from an archive",
	Long: `Reads the archive produced by 'export' and restores apps, configs,
and settings on this (target) Mac. MDM-managed paths are skipped safely.

The archive path can be specified as a positional argument, via --input flag,
or via --from-stdin to read from a pipe.

Example:
  mac-onboarding install ~/onboard.tar.gz
  mac-onboarding install --dry-run ~/onboard.tar.gz
  ssh source-mac "mac-onboarding export --to-stdout" | mac-onboarding install --from-stdin`,
	RunE: func(cmd *cobra.Command, args []string) error {
		archivePath := installInput
		if archivePath == "" && len(args) > 0 {
			archivePath = args[0]
		}

		// If reading from stdin, copy to temp file
		if installFromStdin {
			f, err := os.CreateTemp("", "mac-onboarding-install-*")
			if err != nil {
				return err
			}
			archivePath = f.Name()
			defer os.Remove(archivePath)

			if _, err := io.Copy(f, os.Stdin); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}

		if archivePath == "" {
			return fmt.Errorf("--input or --from-stdin is required")
		}
		if _, err := os.Stat(archivePath); err != nil {
			return fmt.Errorf("archive not found: %s", archivePath)
		}

		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}

		opts := runner.Options{
			DryRun:  dryRun,
			Only:    only,
			Verbose: verbose,
			Input:   archivePath,
		}

		if dryRun {
			fmt.Fprintln(os.Stderr, "dry-run: no changes will be made")
		}

		return runner.Install(cfg, opts)
	},
}

func init() {
	installCmd.Flags().StringVarP(&installInput, "input", "i", "", "input archive path (required unless using --from-stdin)")
	installCmd.Flags().BoolVar(&installFromStdin, "from-stdin", false, "read archive from stdin instead of file")
	rootCmd.AddCommand(installCmd)
}
