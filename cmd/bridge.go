package cmd

import (
	"github.com/spf13/cobra"
)

var bridgeCmd = &cobra.Command{
	Use:   "bridge",
	Short: "Live pull configs from a source Mac over Tailscale",
	Long: `Bridge mode lets the target Mac pull any module's config live from
the source Mac over SSH via Tailscale — no archive needed.

Source Mac:  mac-onboarding bridge serve
Target Mac:  mac-onboarding bridge pull --from <tailscale-hostname> --module kitty`,
}

var bridgeServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start bridge server on this (source) Mac",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Task 14
		cmd.Println("bridge serve: not yet implemented")
		return nil
	},
}

var (
	bridgeFrom   string
	bridgeModule string
)

var bridgePullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull a module's config from the source Mac",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Task 14
		cmd.Println("bridge pull: not yet implemented")
		return nil
	},
}

func init() {
	bridgePullCmd.Flags().StringVar(&bridgeFrom, "from", "", "source Mac Tailscale hostname (required)")
	bridgePullCmd.Flags().StringVar(&bridgeModule, "module", "", "module to pull (required)")
	bridgeCmd.AddCommand(bridgeServeCmd, bridgePullCmd)
	rootCmd.AddCommand(bridgeCmd)
}
