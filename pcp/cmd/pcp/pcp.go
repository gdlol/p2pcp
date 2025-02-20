package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"

	"p2pcp/pkg/receive"
	"p2pcp/pkg/send"
)

var (
	// RawVersion and build tag of the
	// PCP command line tool. This is
	// replaced on build via e.g.:
	// -ldflags "-X main.RawVersion=${VERSION}"
	RawVersion  = "dev"
	ShortCommit = "5f3759df" // quake
)

func main() {
	// ShortCommit version tag
	verTag := fmt.Sprintf("v%s+%s", RawVersion, ShortCommit)

	app := &cli.App{
		Name: "pcp",
		Authors: []*cli.Author{
			{
				Name:  "Dennis Trautwein",
				Email: "pcp@dtrautwein.eu",
			},
		},
		Usage:                "Peer Copy, a peer-to-peer data transfer tool.",
		Version:              verTag,
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			receive.Command,
			send.Command,
		},
		Before: func(c *cli.Context) error {
			if c.Bool("debug") {
				slog.SetLogLoggerLevel(slog.LevelDebug)
			}
			return nil
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "enables debug log output",
			},
			&cli.BoolFlag{
				Name:  "dht",
				Usage: "Only advertise via the DHT",
			},
			&cli.BoolFlag{
				Name:  "mdns",
				Usage: "Only advertise via multicast DNS",
			},
			&cli.BoolFlag{
				Name:   "homebrew",
				Usage:  "if set transfers a hard coded file with a hard coded word sequence",
				Hidden: true,
			},
		},
	}

	sigs := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	go func() {
		<-sigs
		slog.Info("Stopping...")
		signal.Stop(sigs)
		cancel()
	}()

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		slog.Error(fmt.Sprintf("error: %v\n", err))
		os.Exit(1)
	}
}
