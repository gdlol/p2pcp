package node

import (
	"context"
	"fmt"
	"slices"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

func getBootstrapPeers() []peer.AddrInfo {
	return dht.GetDefaultBootstrapPeerAddrInfos()
}

func createDHT(ctx context.Context, host host.Host) (*dual.DHT, error) {
	dualDHT, err := dual.New(ctx, host,
		dual.DHTOption(dht.BootstrapPeers(getBootstrapPeers()...)),
		dual.WanDHTOption(dht.AddressFilter(func(m []multiaddr.Multiaddr) []multiaddr.Multiaddr {
			return slices.DeleteFunc(m, func(addr multiaddr.Multiaddr) bool {
				return !manet.IsPublicAddr(addr)
			})
		})))
	if err != nil {
		return nil, fmt.Errorf("error creating DHT: %w", err)
	}
	err = dualDHT.Bootstrap(ctx)
	if err != nil {
		return nil, fmt.Errorf("error bootstrapping DHT: %w", err)
	}
	return dualDHT, nil
}
