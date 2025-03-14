package node

import (
	"context"
	"log/slog"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

type mdnsNotifee struct {
	host host.Host
	ctx  context.Context
}

func (notifee *mdnsNotifee) HandlePeerFound(addrInfo peer.AddrInfo) {
	if notifee.ctx.Err() == nil {
		slog.Debug("mdns: found new peer.", "peer", addrInfo.ID)
		err := notifee.host.Connect(notifee.ctx, addrInfo)
		if err == nil {
			notifee.host.ConnManager().Protect(addrInfo.ID, "mdns")
		}
	}
}

func createMdnsService(ctx context.Context, host host.Host, serviceName string) mdns.Service {
	notifee := mdnsNotifee{
		host: host,
		ctx:  ctx,
	}
	return mdns.NewMdnsService(host, serviceName, &notifee)
}
