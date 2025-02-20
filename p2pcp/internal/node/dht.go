package node

import (
	"context"
	"fmt"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/host"
)

func createDHT(ctx context.Context, host host.Host) (*dual.DHT, error) {
	dualDHT, err := dual.New(ctx, host, dual.DHTOption(dht.BootstrapPeers(dht.GetDefaultBootstrapPeerAddrInfos()...)))
	if err != nil {
		return nil, fmt.Errorf("error creating DHT: %w", err)
	}
	err = dualDHT.Bootstrap(ctx)
	if err != nil {
		return nil, fmt.Errorf("error bootstrapping DHT: %w", err)
	}
	return dualDHT, nil
}
