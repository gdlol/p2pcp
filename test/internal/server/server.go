package server

import (
	"context"
	"io"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/host"
)

// Creates a server node for testing, helps bootstrapping DHT and relay traffic.
func NewServerNode(ctx context.Context) (host.Host, error) {
	success := false
	closeIfError := func(closer io.Closer) {
		if !success {
			closer.Close()
		}
	}

	options := []libp2p.Option{
		libp2p.ForceReachabilityPublic(),
		libp2p.EnableNATService(),
		libp2p.AutoNATServiceRateLimit(0, 0, 0),
		libp2p.EnableRelayService(),
	}
	host, err := libp2p.New(options...)
	if err != nil {
		return nil, err
	}
	defer closeIfError(host)

	dualDHT, err := dual.New(ctx, host, dual.DHTOption(dht.Mode(dht.ModeServer)))
	if err != nil {
		return nil, err
	}
	defer closeIfError(dualDHT)

	err = dualDHT.Bootstrap(ctx)
	if err != nil {
		return nil, err
	}

	success = true
	return host, nil
}
