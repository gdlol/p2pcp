package node

import (
	"context"
	"log/slog"

	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
)

type dhtRouting struct {
	dht *dual.DHT
}

func (d *dhtRouting) FindPeer(ctx context.Context, id peer.ID) (peer.AddrInfo, error) {
	if d.dht == nil {
		slog.Warn("dhtRouting: DHT not initialized.")
		return peer.AddrInfo{ID: id}, nil
	} else {
		return d.dht.FindPeer(ctx, id)
	}
}

var _ routing.PeerRouting = (*dhtRouting)(nil)
