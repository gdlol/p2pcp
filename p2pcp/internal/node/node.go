package node

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
)

type Node interface {
	GetHost() host.Host
	GetDiscovery() *routing.RoutingDiscovery
	GetProtocol() protocol.ID
	Close()
}

type node struct {
	host        host.Host
	dht         *dual.DHT
	mdnsService mdns.Service
	discovery   *routing.RoutingDiscovery
	protocol    protocol.ID
}

func (n *node) GetHost() host.Host {
	return n.host
}

func (n *node) GetDiscovery() *routing.RoutingDiscovery {
	return n.discovery
}

func (n *node) GetProtocol() protocol.ID {
	return n.protocol
}

func (n *node) Close() {
	n.mdnsService.Close()
	n.dht.Close()
	n.host.Close()
}

// Get peers from DHT for auto relay.
func getPeerSource(getNode <-chan node) autorelay.PeerSource {
	return func(ctx context.Context, num int) <-chan peer.AddrInfo {
		peerSource := make(chan peer.AddrInfo, num)
		go func() {
			defer close(peerSource)

			node := <-getNode
			for {
				slog.Debug("getPeerSource: Getting closest peers...")
				peers, err := node.dht.WAN.GetClosestPeers(ctx, node.host.ID().String())
				if err != nil {
					slog.Warn("Error getting closest peers", "error", err)
					continue
				}
				slog.Debug("getPeerSource: Got closest peers", "peers", peers)
				for _, peerID := range peers {
					addrs := node.host.Peerstore().Addrs(peerID)
					if len(addrs) == 0 {
						continue
					}
					addrInfo := peer.AddrInfo{
						ID:    peerID,
						Addrs: addrs,
					}
					if num > 0 {
						select {
						case peerSource <- addrInfo:
							num--
						case <-ctx.Done():
							return
						}
					} else {
						return
					}
				}
				time.Sleep(time.Second)
			}
		}()
		return peerSource
	}
}

func NewNode(ctx context.Context, options ...libp2p.Option) (Node, error) {
	success := false
	closeIfError := func(closer io.Closer) {
		if !success {
			closer.Close()
		}
	}

	resultChan := make(chan node)
	defer close(resultChan)
	peerSource := getPeerSource(resultChan)
	options = append([]libp2p.Option{
		libp2p.EnableAutoNATv2(),
		libp2p.EnableHolePunching(),
		libp2p.EnableAutoRelayWithPeerSource(peerSource),
	}, options...)

	host, err := libp2p.New(options...)
	if err != nil {
		return nil, fmt.Errorf("error creating host: %w", err)
	}
	defer closeIfError(host)

	dht, err := createDHT(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("error creating DHT: %w", err)
	}
	defer closeIfError(dht)

	mdnsService, err := createMdnsService(ctx, host, "p2pcp")
	if err != nil {
		return nil, fmt.Errorf("error creating mDNS service: %w", err)
	}
	defer closeIfError(mdnsService)

	discovery := routing.NewRoutingDiscovery(dht)

	node := &node{
		host:        host,
		dht:         dht,
		mdnsService: mdnsService,
		discovery:   discovery,
		protocol:    "/p2pcp/transfer/0.1.0",
	}

	resultChan <- *node
	success = true
	return node, nil
}
