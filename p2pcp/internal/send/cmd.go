package send

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"moul.io/drunken-bishop/drunkenbishop"
)

func Send(ctx context.Context, path string) error {
	fmt.Println("Preparing sender...")
	sender, err := NewAdvertisedSender(ctx)
	if err != nil {
		return fmt.Errorf("error creating sender: %w", err)
	}
	node := sender.GetNode()
	err = node.StartMdns(func(ai peer.AddrInfo) {
		go func() {
			for {
				time.Sleep(time.Second)
				slog.Debug("Advertising to LAN DHT...", "topic", sender.GetAdvertiseTopic())
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
	id := node.ID()
	topic := sender.GetAdvertiseTopic()

	fmt.Println("Node ID:", id.String())
	room := drunkenbishop.FromBytes(id.Bytes())
	fmt.Println(room)

	fmt.Println("Please run the following command on receiver side:")
	fmt.Println("p2pcp", "receive", topic)

	return sender.Send(path)
}
