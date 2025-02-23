package send

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"moul.io/drunken-bishop/drunkenbishop"
)

func Send(ctx context.Context, path string) error {
	ctx = network.WithAllowLimitedConn(ctx, "hole-punching")

	fmt.Println("Preparing sender...")
	sender, err := NewAdvertisedSender(ctx)
	if err != nil {
		return fmt.Errorf("error creating sender: %w", err)
	}
	node := sender.GetNode()
	defer node.Close()

	id := node.ID()
	topic := sender.GetAdvertiseTopic()
	err = node.StartMdns(func(ai peer.AddrInfo) {
		go func() {
			for ctx.Err() != nil {
				time.Sleep(time.Second)
				slog.Debug("Advertising to LAN DHT...", "topic", topic)
				err := node.AdvertiseLAN(ctx, sender.GetAdvertiseTopic())
				if err != nil {
					slog.Debug("Error advertising to LAN DHT, retrying...", "error", err)
				} else {
					slog.Debug("Advertised to LAN DHT.")
					break
				}
			}
		}()
	})
	if err != nil {
		return fmt.Errorf("error starting mDNS service: %w", err)
	}

	fmt.Println("Node ID:", id.String())
	room := drunkenbishop.FromBytes(id.Bytes())
	fmt.Println(room)

	fmt.Println("Please run the following command on receiver side:")
	fmt.Println("p2pcp", "receive", topic)

	fmt.Println("Sending...")
	err = sender.Send(ctx, path)
	if err == nil {
		fmt.Println("Done.")
	}
	return err
}
