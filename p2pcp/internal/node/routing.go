package node

import (
	"context"
	"log/slog"

	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/routing"
)

const PeerRoutingTag = "p2pcp/peer-routing"

type dhtRouting struct {
	host host.Host
	dht  *dual.DHT
}

func (d *dhtRouting) FindPeer(ctx context.Context, id peer.ID) (peer.AddrInfo, error) {
	if d.dht == nil {
		slog.Warn("dhtRouting: DHT not initialized.")
		return peer.AddrInfo{ID: id}, nil
	} else {
		_, err := d.host.Peerstore().Get(id, PeerRoutingTag)
		if err == nil {
			slog.Debug("dhtRouting: Finding peer address with DHT.", "peer", id)
			return d.dht.FindPeer(ctx, id)
		}
		if err != peerstore.ErrNotFound {
			slog.Warn("dhtRouting: Error getting PeerRoutingTag for peer.", "peer", id, "error", err)
		}
		return peer.AddrInfo{ID: id}, nil
	}
}

var _ routing.PeerRouting = (*dhtRouting)(nil)
