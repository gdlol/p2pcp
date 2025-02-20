package dht

import (
	"context"
	"log/slog"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"

	"p2pcp/internal/wrap"
)

// Discoverer is responsible for reading the DHT for an
// entry with the channel ID given below.
type Discoverer struct {
	*protocol
}

// NewDiscoverer creates a new Discoverer.
func NewDiscoverer(h host.Host, dht wrap.IpfsDHT) *Discoverer {
	return &Discoverer{newProtocol(h, dht)}
}

// Discover establishes a connection to a set of bootstrap peers
// that we're using to connect to the DHT. It tries to find
func (d *Discoverer) Discover(chanID int, handler func(info peer.AddrInfo)) error {
	if err := d.ServiceStarted(); err != nil {
		return err
	}
	defer d.ServiceStopped()

	if err := d.Bootstrap(); err != nil {
		return err
	}

	for {
		did := d.DiscoveryID(chanID)
		slog.Debug("DHT - Discovering", "did", did)
		cID, err := strToCid(did)
		if err != nil {
			return err
		}

		// Find new provider with a timeout, so the discovery ID is renewed if necessary.
		ctx, cancel := context.WithTimeout(d.ServiceContext(), provideTimeout)
		for pi := range d.dht.FindProvidersAsync(ctx, cID, 100) {
			slog.Debug("DHT - Found peer ", "id", pi.ID)
			pi.Addrs = onlyPublic(pi.Addrs)
			if isRoutable(pi) {
				go handler(pi)
			}
		}
		slog.Debug("DHT - Discovering done.")

		// cannot defer cancel in this for loop
		cancel()

		select {
		case <-d.SigShutdown():
			return nil
		default:
		}
	}
}

func (d *Discoverer) SetOffset(offset time.Duration) *Discoverer {
	d.offset = offset
	return d
}

func (d *Discoverer) Shutdown() {
	d.Service.Shutdown()
}

// Filter out addresses that are local - only allow public ones.
func onlyPublic(addrs []ma.Multiaddr) []ma.Multiaddr {
	routable := []ma.Multiaddr{}
	for _, addr := range addrs {
		if manet.IsPublicAddr(addr) {
			routable = append(routable, addr)
			slog.Debug("DHT - Found public address", "addr", addr.String())
		} else {
			slog.Debug("DHT - Found private address", "addr", addr.String())
		}
	}
	return routable
}

func isRoutable(pi peer.AddrInfo) bool {
	return len(pi.Addrs) > 0
}
