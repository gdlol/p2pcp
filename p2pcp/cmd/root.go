package cmd

import (
	"fmt"
	"log/slog"
	"os"

	receive "p2pcp/cmd/receive"
	send "p2pcp/cmd/send"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "p2pcp",
	Short:         "Peer to Peer Copy, a peer-to-peer data transfer tool.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Show debug logs")
	rootCmd.PersistentFlags().BoolP("private", "p", false, "Use private network only")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
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
