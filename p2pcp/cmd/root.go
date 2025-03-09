package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"project/pkg/project"

	"p2pcp/cmd/receive"
	"p2pcp/cmd/send"
	"p2pcp/pkg/config"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           project.Name,
	Short:         "Peer to Peer Copy, a peer-to-peer data transfer tool.",
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       project.Version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.PersistentFlags().BoolP("debug", "d", false, "show debug logs")
	rootCmd.PersistentFlags().BoolP("private", "p", false, "only connect to private networks")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		config.LoadConfig()

		debug, _ := cmd.Flags().GetBool("debug")
		if debug {
			slog.SetLogLoggerLevel(slog.LevelDebug)
		} else {
			slog.SetLogLoggerLevel(slog.LevelWarn)
		}
	}

	rootCmd.AddCommand(send.SendCmd)
	rootCmd.AddCommand(receive.ReceiveCmd)
}
