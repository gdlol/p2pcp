package receive

import (
	"context"
	"fmt"
	"p2pcp/internal/auth"
	"p2pcp/internal/node"
	"time"

	"github.com/briandowns/spinner"
	"github.com/libp2p/go-libp2p/core/network"
)

func Receive(ctx context.Context, id string, secret string, basePath string, private bool) error {
	ctx = network.WithAllowLimitedConn(ctx, "hole-punching")

	n := node.NewNode(ctx, private)
	defer n.Close()

	n.StartMdns()
	receiver := NewReceiver(n)

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Suffix = " Finding sender..."
	s.Start()
	peer, err := receiver.FindPeer(ctx, id)
	s.Stop()
	if err != nil {
		return fmt.Errorf("error finding sender: %w", err)
	}

	nodeID := node.GetNodeID(peer)
	if id != nodeID.String() { // non-strict mode
		fmt.Println("Sender ID:", nodeID.String())
		fmt.Println("Please verify that the following random art matches the one displayed on the sender's side.")
		fmt.Println(auth.RandomArt(nodeID.Bytes()))
		fmt.Println("Are you sure you want to connect to this sender? [y/N]")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" {
			return nil
		}
	}

	fmt.Println("Receiving...")
	secretHash := auth.ComputeHash([]byte(secret))
	err = receiver.Receive(ctx, peer, secretHash, basePath)
	if err == nil {
		fmt.Println("Done.")
	}
	return err
}
