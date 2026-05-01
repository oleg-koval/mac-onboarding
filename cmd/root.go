package cmd

import (
	"fmt"
	"os"

	"github.com/oleg-koval/mac-onboarding/internal/updater"
	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	cfgFile string
	dryRun  bool
	only    []string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:     "mac-onboarding",
	Short:   "Restore your Mac environment from a source machine",
	Version: Version,
	Long: `mac-onboarding exports app configs and settings from a source Mac,
then installs them on a new MDM-managed Mac — without Time Machine.

Source Mac:  mac-onboarding export --output ~/onboard-archive.tar.gz
Target Mac:  mac-onboarding install --input ~/onboard-archive.tar.gz

Bridge mode: mac-onboarding bridge serve   (source)
             mac-onboarding bridge pull --from <hostname> --module kitty   (target)`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		help, _ := cmd.Flags().GetBool("help")
		if help || dryRun {
			return nil
		}
		return updater.MaybeUpdate(Version, os.Stderr)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./onboard.yaml or ~/.config/mac-onboarding/onboard.yaml)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print what would happen without making changes")
	rootCmd.PersistentFlags().StringSliceVar(&only, "only", nil, "run only these modules (comma-separated)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}
