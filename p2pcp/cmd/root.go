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

var RootCmd = &cobra.Command{
	Use:           project.Name,
	Short:         "Peer to Peer Copy, a peer-to-peer data transfer tool",
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       project.Version,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.CompletionOptions.DisableDefaultCmd = true

	RootCmd.PersistentFlags().BoolP("debug", "d", false, "show debug logs")
	RootCmd.PersistentFlags().BoolP("private", "p", false, "only connect to private networks")
	RootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		config.LoadConfig()

		debug, _ := cmd.Flags().GetBool("debug")
		if debug {
			slog.SetLogLoggerLevel(slog.LevelDebug)
		} else {
			slog.SetLogLoggerLevel(slog.LevelWarn)
		}
	}

	RootCmd.AddCommand(send.SendCmd)
	RootCmd.AddCommand(receive.ReceiveCmd)
	os.Setenv("QUIC_GO_DISABLE_RECEIVE_BUFFER_WARNING", "true")
}
