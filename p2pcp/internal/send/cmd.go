package send

import (
	"context"
	"fmt"
	"p2pcp/internal/auth"
	"project/pkg/project"
	"time"

	"github.com/briandowns/spinner"
	"github.com/libp2p/go-libp2p/core/network"
)

func Send(ctx context.Context, basePath string, strict bool, private bool) error {
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

	n.StartMdns()

	if !strict {
		fmt.Println("Node ID:", n.ID())
		fmt.Println(auth.RandomArt(n.ID().Bytes()))
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
		fmt.Println(project.Name, "receive", id, "--private")
	} else {
		fmt.Println(project.Name, "receive", id)
	}

	var secret string
	if !strict {
		secret = auth.GetOneTimeSecret()
		fmt.Printf("PIN: %s\n", secret)
	} else {
		secret = auth.GetStrongSecret()
		fmt.Printf("token: %s\n", secret)
	}
	fmt.Println()

	secretHash := auth.ComputeHash([]byte(secret))
	receiver, err := sender.WaitForReceiver(ctx, secretHash)
	if err != nil {
		return fmt.Errorf("error waiting for receiver: %w", err)
	}

	fmt.Println("Sending...")
	err = sender.Send(ctx, receiver, basePath)
	if err == nil {
		fmt.Println("Done.")
	}
	return err
}
