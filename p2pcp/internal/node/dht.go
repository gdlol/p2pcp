package node

import (
	"context"
	"fmt"
	"p2pcp/internal/config"
	"slices"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

func getBootstrapPeers() ([]peer.AddrInfo, error) {
	config := config.GetConfig()
	if len(config.BootstrapPeers) > 0 {
		bootstrapPeers := make([]peer.AddrInfo, 0, len(config.BootstrapPeers))
		for _, addr := range config.BootstrapPeers {
			ma, err := multiaddr.NewMultiaddr(addr)
			var addrInfo *peer.AddrInfo
			if err == nil {
				addrInfo, err = peer.AddrInfoFromP2pAddr(ma)
			}
			if err != nil {
				return nil, fmt.Errorf("error getting bootstrap peers: %w", err)
			}
			bootstrapPeers = append(bootstrapPeers, *addrInfo)
		}
		return bootstrapPeers, nil
	} else {
		return dht.GetDefaultBootstrapPeerAddrInfos(), nil
	}
}

func createDHT(ctx context.Context, host host.Host) (*dual.DHT, error) {
	bootstrapPeers, err := getBootstrapPeers()
	if err != nil {
		return nil, err
	}
	dualDHT, err := dual.New(ctx, host,
		dual.DHTOption(dht.BootstrapPeers(bootstrapPeers...)),
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
