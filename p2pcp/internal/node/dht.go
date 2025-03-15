package node

import (
	"context"
	"fmt"
	"log/slog"
	"p2pcp/internal/errors"
	"p2pcp/pkg/config"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

func getBootstrapPeers(peers []string) []peer.AddrInfo {
	if len(peers) > 0 {
		bootstrapPeers := make([]peer.AddrInfo, 0, len(peers))
		for _, addr := range peers {
			ma, err := multiaddr.NewMultiaddr(addr)
			var addrInfo *peer.AddrInfo
			if err == nil {
				addrInfo, err = peer.AddrInfoFromP2pAddr(ma)
			}
			if err != nil {
				slog.Error(fmt.Sprintf("error parsing bootstrap peer %s: %s", addr, err))
				continue
			}
			bootstrapPeers = append(bootstrapPeers, *addrInfo)
		}
		return bootstrapPeers
	} else {
		return dht.GetDefaultBootstrapPeerAddrInfos()
	}
}

func createDHT(ctx context.Context, host host.Host) *dual.DHT {
	config := config.GetConfig()
	bootstrapPeers := getBootstrapPeers(config.BootstrapPeers)
	dualDHT, err := dual.New(ctx, host, dual.DHTOption(dht.BootstrapPeers(bootstrapPeers...)))
	errors.Unexpected(err, "create DHT")
	err = dualDHT.Bootstrap(ctx)
	errors.Unexpected(err, "bootstrap DHT")
	return dualDHT
}
