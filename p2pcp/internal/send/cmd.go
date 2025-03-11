package send

import (
	"context"
	"fmt"
	"p2pcp/internal/auth"
	"project/pkg/project"
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
	n := sender.GetNode()

	err = n.StartMdns()
	if err != nil {
		return fmt.Errorf("error starting mDNS service: %w", err)
	}

	var secret string
	if !strict {
		secret = auth.GetOneTimeSecret()
	} else {
		secret = auth.GetStrongSecret()
	}

	if !strict {
		fmt.Println("Node ID:", n.ID())
		room := drunkenbishop.FromBytes(n.ID().Bytes())
		fmt.Println(room)
	}

	var id string
	if strict {
		id = n.ID().String()
	} else {
		id = sender.GetAdvertiseTopic()
	}
	fmt.Println("Please run the following command on the receiver's side:")
	fmt.Println()
	if private {
		fmt.Println(project.Name, "receive", id, secret, "--private")
	} else {
		fmt.Println(project.Name, "receive", id, secret)
	}
	fmt.Println()

	secretHash := auth.ComputeHash([]byte(secret))
	receiver, err := sender.WaitForReceiver(ctx, secretHash, path)
	if err != nil {
		return fmt.Errorf("error waiting for receiver: %w", err)
	}

	fmt.Println("Sending...")
	err = sender.Send(ctx, receiver, path)
	if err == nil {
		fmt.Println("Done.")
	}
	return err
}
