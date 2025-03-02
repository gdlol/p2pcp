package cmd

// spell-checker: ignore adrg

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"project/pkg/project"

	"p2pcp/cmd/receive"
	"p2pcp/cmd/send"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:           project.Name,
	Short:         "Peer to Peer Copy, a peer-to-peer data transfer tool.",
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       version,
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
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		debug, _ := cmd.Flags().GetBool("debug")
		if debug {
			slog.SetLogLoggerLevel(slog.LevelDebug)
		} else {
			slog.SetLogLoggerLevel(slog.LevelWarn)
		}

		viper.SetConfigName("config")
		viper.SetConfigType("json")
		viper.AddConfigPath(filepath.Join(xdg.ConfigHome, project.Name))
		if err := viper.ReadInConfig(); err != nil {
			return err
		}

		return nil
	}

	rootCmd.AddCommand(send.SendCmd)
	rootCmd.AddCommand(receive.ReceiveCmd)
}
