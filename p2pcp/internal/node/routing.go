package node

import (
	"context"
	"log/slog"
	"math"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/backoff"
)

const DhtRoutingTag = "p2pcp/peer-routing"

type dhtRouting struct {
	host host.Host
	dht  routing.PeerRouting
}

func (d *dhtRouting) FindPeer(ctx context.Context, id peer.ID) (peer.AddrInfo, error) {
	if d.dht == nil {
		slog.Warn("dhtRouting: DHT not initialized.")
		return peer.AddrInfo{ID: id}, nil
	} else {
		_, err := d.host.Peerstore().Get(id, DhtRoutingTag)
		if err != nil {
			if err != peerstore.ErrNotFound {
				slog.Warn("dhtRouting: Error getting DhtRoutingTag for peer.", "peer", id, "error", err)
			}
			return peer.AddrInfo{ID: id}, err
		}
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		addrInfo := peer.AddrInfo{ID: id}
		b := backoff.NewExponentialBackoff(
			0, 3*time.Second, backoff.NoJitter,
			100*time.Millisecond, math.Sqrt2, 0,
			rand.NewSource(0))()
		for ctx.Err() == nil {
			addrInfo, err = d.dht.FindPeer(ctx, id)
			if len(addrInfo.Addrs) == 0 || err != nil {
				slog.Debug("dhtRouting: Failed to find peer with DHT.", "peer", id, "error", err)
				time.Sleep(b.Delay())
				continue
			} else {
				return addrInfo, nil
			}
		}
		return addrInfo, ctx.Err()
	}
}

var _ routing.PeerRouting = (*dhtRouting)(nil)
