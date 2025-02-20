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

func (notifee mdnsNotifee) HandlePeerFound(addrInfo peer.AddrInfo) {
	slog.Debug("mdns: HandlePeerFound", "peer", addrInfo)
	err := notifee.host.Connect(notifee.ctx, addrInfo)
	if err != nil {
		slog.Warn("mdns: failed to connect to peer", "peer", addrInfo, "error", err)
		return
	}
	notifee.host.ConnManager().Protect(addrInfo.ID, "mdns")
}

func createMdnsService(ctx context.Context, host host.Host, serviceName string) (mdns.Service, error) {
	notifee := mdnsNotifee{
		host: host,
		ctx:  ctx,
	}
	service := mdns.NewMdnsService(host, serviceName, notifee)
	err := service.Start()
	if err != nil {
		service.Close()
		return nil, err
	}
	return service, nil
}
