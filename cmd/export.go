package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/oleg-koval/mac-onboarding/internal/config"
	"github.com/oleg-koval/mac-onboarding/internal/runner"
	"github.com/spf13/cobra"
)

var exportOutput string
var exportToStdout bool

var exportCmd = &cobra.Command{
	Use:   "export [ARCHIVE_PATH]",
	Short: "Export configs and settings from this Mac into an archive",
	Long: `Packages your app configs, dotfiles, and settings into a portable archive.
Secrets are redacted. Run on your source (current) Mac.

The archive path can be specified as a positional argument or via --output flag.
If neither is provided, defaults to onboard-YYYYMMDD.tar.gz in current directory.

Example:
  mac-onboarding export ~/onboard.tar.gz
  mac-onboarding export --to-stdout | ssh target-mac "mac-onboarding install"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		archivePath := exportOutput
		if archivePath == "" && len(args) > 0 {
			archivePath = args[0]
		}
		if archivePath == "" {
			archivePath = fmt.Sprintf("onboard-%s.tar.gz", time.Now().Format("20060102"))
		}

		// If outputting to stdout, use a temp file internally
		if exportToStdout {
			f, err := os.CreateTemp("", "mac-onboarding-export-*")
			if err != nil {
				return err
			}
			archivePath = f.Name()
			f.Close()
			defer os.Remove(archivePath)
		}

		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}

		opts := runner.Options{
			DryRun:  dryRun,
			Only:    only,
			Verbose: verbose,
			Output:  archivePath,
		}

		if dryRun {
			fmt.Fprintln(os.Stderr, "dry-run: no changes will be made")
		}

		if err := runner.Export(cfg, opts); err != nil {
			return err
		}

		// If --to-stdout, pipe archive to stdout after successful export
		if exportToStdout && !dryRun {
			f, err := os.Open(archivePath)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(os.Stdout, f)
			return err
		}

		return nil
	},
}

func init() {
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output archive path (default: onboard-YYYYMMDD.tar.gz)")
	exportCmd.Flags().BoolVar(&exportToStdout, "to-stdout", false, "write archive to stdout instead of file (for piping)")
	rootCmd.AddCommand(exportCmd)
}
