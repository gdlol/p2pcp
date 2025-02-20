package mdns

import (
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/pkg/errors"
	"github.com/whyrusleeping/mdns"
)

type Discoverer struct {
	*protocol
}

func NewDiscoverer(h host.Host) *Discoverer {
	return &Discoverer{newProtocol(h)}
}

func (d *Discoverer) Discover(chanID int, handler func(info peer.AddrInfo)) error {
	if err := d.ServiceStarted(); err != nil {
		return err
	}
	defer d.ServiceStopped()

	for {
		entriesCh := make(chan *mdns.ServiceEntry, 16)
		go d.drainEntriesChan(entriesCh, handler)

		did := d.DiscoveryID(chanID)
		slog.Debug("mDNS - Discovering", "did", did)
		qp := &mdns.QueryParam{
			Domain:  "local",
			Entries: entriesCh,
			Service: did,
			Timeout: time.Second * 5,
		}

		err := mdns.Query(qp)
		slog.Debug(fmt.Sprintf("mDNS - Discovering %s done.", did))
		if err != nil {
			slog.Warn("mDNS - query error", "err", err)
		}
		close(entriesCh)

		select {
		case <-d.SigShutdown():
			return nil
		default:
		}
	}
}

func (d *Discoverer) Shutdown() {
	d.Service.Shutdown()
}

func (d *Discoverer) drainEntriesChan(entries chan *mdns.ServiceEntry, handler func(info peer.AddrInfo)) {
	for entry := range entries {

		pi, err := parseServiceEntry(entry)
		if err != nil {
			continue
		}

		slog.Debug("mDNS - Found peer", "id", pi.ID)

		if pi.ID == d.ID() {
			continue
		}

		pi.Addrs = onlyPrivate(pi.Addrs)
		if !isRoutable(pi) {
			continue
		}

		go handler(pi)
	}
}

func parseServiceEntry(entry *mdns.ServiceEntry) (peer.AddrInfo, error) {
	p, err := peer.Decode(entry.Info)
	if err != nil {
		return peer.AddrInfo{}, errors.Wrap(err, "error parsing peer ID from mdns entry")
	}

	var addr net.IP
	if entry.AddrV4 != nil {
		addr = entry.AddrV4
	} else if entry.AddrV6 != nil {
		addr = entry.AddrV6
	} else {
		return peer.AddrInfo{}, errors.Wrap(err, "error parsing multiaddr from mdns entry: no IP address found")
	}

	maddr, err := manet.FromNetAddr(&net.TCPAddr{IP: addr, Port: entry.Port})
	if err != nil {
		return peer.AddrInfo{}, errors.Wrap(err, "error parsing multiaddr from mdns entry")
	}

	return peer.AddrInfo{
		ID:    p,
		Addrs: []ma.Multiaddr{maddr},
	}, nil
}

func isRoutable(pi peer.AddrInfo) bool {
	return len(pi.Addrs) > 0
}

// Filter out addresses that are public - only allow private ones.
func onlyPrivate(addrs []ma.Multiaddr) []ma.Multiaddr {
	routable := []ma.Multiaddr{}
	for _, addr := range addrs {
		if manet.IsPrivateAddr(addr) {
			routable = append(routable, addr)
			slog.Debug("mDNS - Found private address", "addr", addr.String())
		} else {
			slog.Debug("mDNS - Found public address", "addr", addr.String())
		}
	}
	return routable
}
