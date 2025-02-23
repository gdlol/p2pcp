package send

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"p2pcp/internal/auth"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"moul.io/drunken-bishop/drunkenbishop"
)

func Send(ctx context.Context, path string, strict bool) error {
	ctx = network.WithAllowLimitedConn(ctx, "hole-punching")

	fmt.Println("Preparing sender...")
	sender, err := NewAdvertisedSender(ctx)
	if err != nil {
		return fmt.Errorf("error creating sender: %w", err)
	}
	node := sender.GetNode()
	defer node.Close()

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
		id = topic
	}
	fmt.Println("Please run the following command on receiver side:")
	fmt.Println("p2pcp", "receive", id, secret)

	fmt.Println("Sending...")
	secretHash := auth.ComputeHash([]byte(secret))
	err = sender.Send(ctx, secretHash, path, strict)
	if err == nil {
		fmt.Println("Done.")
	}
	return err
}
