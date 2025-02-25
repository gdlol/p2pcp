package send

import (
	"context"
	"crypto/rand"
	"fmt"
	"p2pcp/internal/auth"
	"time"

	"github.com/briandowns/spinner"
	"github.com/libp2p/go-libp2p/core/network"
	"moul.io/drunken-bishop/drunkenbishop"
)

func Send(ctx context.Context, path string, strict bool, private bool) error {
	ctx = network.WithAllowLimitedConn(ctx, "hole-punching")

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Preparing sender..."
	s.Start()
	sender, err := NewAdvertisedSender(ctx, strict, private)
	s.Stop()
	if err != nil {
		return fmt.Errorf("error creating sender: %w", err)
	}
	defer sender.Close()
	node := sender.GetNode()

	err = node.StartMdns()
	if err != nil {
		return fmt.Errorf("error starting mDNS service: %w", err)
	}

	var secret string
	if !strict {
		secret, err = auth.GetPin()
		if err != nil {
			return fmt.Errorf("error generating pin: %w", err)
		}
	} else {
		secret = rand.Text()
	}

	if !strict {
		fmt.Println("Node ID:", node.ID())
		room := drunkenbishop.FromBytes(node.ID().Bytes())
		fmt.Println(room)
	}

	var id string
	if strict {
		id = node.ID().String()
	} else {
		id = sender.GetAdvertiseTopic()
	}
	fmt.Println("Please run the following command on receiver side:")
	fmt.Println()
	if private {
		fmt.Println("p2pcp", "receive", id, secret, "--private")
	} else {
		fmt.Println("p2pcp", "receive", id, secret)
	}
	fmt.Println()

	fmt.Println("Sending...")
	secretHash := auth.ComputeHash([]byte(secret))
	err = sender.Send(ctx, secretHash, path)
	if err == nil {
		fmt.Println("Done.")
	}
	return err
}
