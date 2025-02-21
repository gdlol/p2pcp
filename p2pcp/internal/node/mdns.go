package node

import (
	"context"
	"log/slog"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

type HandleMdnsPeerFound func(peer.AddrInfo)

type MdnsService interface {
	mdns.Service
	SetHandler(handlePeerFound HandleMdnsPeerFound)
}

type mdnsNotifee struct {
	host            host.Host
	ctx             context.Context
	handlePeerFound HandleMdnsPeerFound
}

type mdnsService struct {
	service mdns.Service
	notifee *mdnsNotifee
}

func (m *mdnsService) Close() error {
	return m.service.Close()
}

func (m *mdnsService) SetHandler(handlePeerFound HandleMdnsPeerFound) {
	m.notifee.handlePeerFound = handlePeerFound
}

func (m *mdnsService) Start() error {
	return m.service.Start()
}

func (notifee *mdnsNotifee) HandlePeerFound(addrInfo peer.AddrInfo) {
	slog.Debug("mdns: found new peer.", "peer", addrInfo.ID)
	err := notifee.host.Connect(notifee.ctx, addrInfo)
	if err != nil {
		slog.Warn("mdns: failed to connect to peer.", "peer", addrInfo, "error", err)
		return
	}
	notifee.host.ConnManager().Protect(addrInfo.ID, "mdns")
	notifee.handlePeerFound(addrInfo)
}

func createMdnsService(ctx context.Context, host host.Host, serviceName string) MdnsService {
	notifee := mdnsNotifee{
		host:            host,
		ctx:             ctx,
		handlePeerFound: func(ai peer.AddrInfo) {},
	}
	service := mdns.NewMdnsService(host, serviceName, &notifee)
	return &mdnsService{
		service: service,
		notifee: &notifee,
	}
}
