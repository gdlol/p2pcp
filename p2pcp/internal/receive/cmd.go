package receive

import (
	"context"
	"fmt"
	"p2pcp/internal/node"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"moul.io/drunken-bishop/drunkenbishop"
)

func Receive(ctx context.Context, topic string, basePath string) error {
	ctx = network.WithAllowLimitedConn(ctx, "hole-punching")

	n, err := node.NewNode(ctx)
	if err != nil {
		return fmt.Errorf("error creating new node: %w", err)
	}
	defer n.Close()

	err = n.StartMdns(func(ai peer.AddrInfo) {})
	if err != nil {
		return fmt.Errorf("error starting mDNS service: %w", err)
	}
	receiver := NewReceiver(n)

	fmt.Println("Finding sender...")
	peer, err := receiver.FindPeer(ctx, topic)
	if err != nil {
		return fmt.Errorf("error finding peer: %w", err)
	}

	nodeID, err := node.GetNodeID(peer.ID)
	if err != nil {
		return fmt.Errorf("error getting node ID for %v: %w", peer.ID, err)
	}
	fmt.Println("Sender ID:", nodeID.String())
	room := drunkenbishop.FromBytes(nodeID.Bytes())
	fmt.Println(room)
	fmt.Println("Are you sure you want to connect to this peer? [y/N]")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "y" {
		return nil
	}

	fmt.Println("Receiving...")
	err = receiver.Receive(ctx, *peer, basePath)
	if err == nil {
		fmt.Println("Done.")
	}
	return err
}
