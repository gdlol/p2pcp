package node

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	mathRand "math/rand"
	"p2pcp/internal/auth"
	"p2pcp/internal/errors"
	"project/pkg/project"
	"slices"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	coreRouting "github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/backoff"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	b58 "github.com/mr-tron/base58/base58"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

type NodeID interface {
	String() string
	Bytes() []byte
}

type Node interface {
	ID() NodeID
	GetHost() host.Host
	StartMdns()
	AdvertiseLAN(ctx context.Context, topic string) error
	AdvertiseWAN(ctx context.Context, topic string) error
	FindPeers(ctx context.Context, topic string) (<-chan peer.AddrInfo, error)
	RegisterErrorHandler(peerID peer.ID, handler func(string))
	SendError(ctx context.Context, peerID peer.ID, errStr string)
	Close()
}

type nodeID struct {
	value []byte
}

func (k *nodeID) String() string {
	return b58.Encode(k.value)
}

func (k *nodeID) Bytes() []byte {
	return k.value
}

type node struct {
	id              NodeID
	host            host.Host
	privateMode     bool
	dht             *dual.DHT
	mdnsService     mdns.Service
	peerSource      chan peer.AddrInfo
	peerSourceLimit chan int
}

func (n *node) ID() NodeID {
	return n.id
}

func (n *node) GetHost() host.Host {
	return n.host
}

func (n *node) StartMdns() {
	err := n.mdnsService.Start()
	errors.Unexpected(err, "start mDNS")
}

func (n *node) AdvertiseLAN(ctx context.Context, topic string) error {
	discovery := routing.NewRoutingDiscovery(n.dht.LAN)
	_, err := discovery.Advertise(ctx, topic)
	return err
}

func (n *node) waitForWAN(ctx context.Context, continuation func() error) error {
	for ctx.Err() == nil {
		if !n.privateMode && !n.dht.WANActive() {
			time.Sleep(time.Second)
			continue
		} else {
			return continuation()
		}
	}
	return ctx.Err()
}

func (n *node) AdvertiseWAN(ctx context.Context, topic string) error {
	return n.waitForWAN(ctx, func() error {
		discovery := routing.NewRoutingDiscovery(n.dht.WAN)
		_, err := discovery.Advertise(ctx, topic)
		return err
	})
}

func (n *node) FindPeers(ctx context.Context, topic string) (<-chan peer.AddrInfo, error) {
	var peers <-chan peer.AddrInfo
	var err error
	err = n.waitForWAN(ctx, func() error {
		discovery := routing.NewRoutingDiscovery(n.dht)
		peers, err = discovery.FindPeers(ctx, topic)
		return err
	})
	return peers, err
}

func findPeersForAutoRelay(ctx context.Context, n node) {
	backoffStrategy := backoff.NewExponentialBackoff(
		time.Second, 6*time.Second, backoff.NoJitter,
		time.Second, 2, 0,
		mathRand.NewSource(0))

	for ctx.Err() == nil {
		var num int
		select {
		case <-ctx.Done():
			return
		case num = <-n.peerSourceLimit:
		}

		// Get random peers from DHT.
		b := backoffStrategy()
		var peers []peer.ID
		for ctx.Err() == nil {
			slog.Debug("Getting peers from DHT for auto relay...")
			var err error
			err = n.waitForWAN(ctx, func() error {
				peers, err = n.dht.WAN.GetClosestPeers(ctx, rand.Text())
				return err
			})
			if err == nil {
				if len(peers) > 0 {
					slog.Debug(fmt.Sprintf("Feeding %d peers from DHT for auto relay.", len(peers)))
					break
				} else {
					time.Sleep(b.Delay())
				}
			} else if ctx.Err() == nil {
				slog.Debug("Error getting peers from DHT for auto relay.", "error", err)
			}
		}

		// Feed peers with public addresses to auto relay.
		for i := 0; ctx.Err() == nil && num > 0 && i < len(peers); i++ {
			peerID := peers[i]
			addrs := n.host.Peerstore().Addrs(peerID)
			addrs = slices.DeleteFunc(addrs, func(addr multiaddr.Multiaddr) bool {
				return !manet.IsPublicAddr(addr)
			})
			if len(addrs) >= 0 {
				addrInfo := peer.AddrInfo{
					ID:    peerID,
					Addrs: addrs,
				}
				n.peerSource <- addrInfo
				num--
			}
		}
	}
}

func (n *node) RegisterErrorHandler(peerID peer.ID, handler func(string)) {
	registerErrorHandler(n.host, peerID, handler)
}

func (n *node) SendError(ctx context.Context, peerID peer.ID, errStr string) {
	sendError(ctx, n.host, peerID, errStr)
}

func (n *node) Close() {
	n.mdnsService.Close()
	n.dht.Close()
	n.host.Close()
}

// Gets a hashed ID, as peerID.String() may or may not have been hashed.
func GetNodeID(peerID peer.ID) NodeID {
	pubKey, err := peerID.ExtractPublicKey()
	errors.Unexpected(err, "GetNodeID")
	keyBytes, err := pubKey.Raw()
	errors.Unexpected(err, "GetNodeID")
	hashValue := auth.ComputeHash(keyBytes)
	return &nodeID{value: hashValue}
}

func NewNode(ctx context.Context, privateMode bool, options ...libp2p.Option) Node {
	peerSource := make(chan peer.AddrInfo)
	peerSourceLimit := make(chan int, 1)
	routing := &dhtRouting{}

	if privateMode {
		options = append([]libp2p.Option{libp2p.ConnectionGater(privateAddressGater{})}, options...)
	} else {
		options = append([]libp2p.Option{
			libp2p.EnableAutoNATv2(),
			libp2p.EnableHolePunching(),
			libp2p.EnableAutoRelayWithPeerSource(
				func(ctx context.Context, num int) <-chan peer.AddrInfo {
					select {
					case peerSourceLimit <- num:
					case <-ctx.Done():
					}
					return peerSource
				},
				autorelay.WithBootDelay(6*time.Second)),
			libp2p.Routing(func(host.Host) (coreRouting.PeerRouting, error) {
				return routing, nil
			}),
			libp2p.ForceReachabilityPrivate(), // Force auto relay to start,
		}, options...)
	}

	host, err := libp2p.New(options...)
	errors.Unexpected(err, "libp2p.New")
	routing.host = host

	id := GetNodeID(host.ID())

	dht := createDHT(ctx, host)
	routing.dht = dht

	mdnsService := createMdnsService(ctx, host, project.Name)

	node := &node{
		id:              id,
		host:            host,
		privateMode:     privateMode,
		dht:             dht,
		mdnsService:     mdnsService,
		peerSource:      peerSource,
		peerSourceLimit: peerSourceLimit,
	}

	go findPeersForAutoRelay(ctx, *node)

	return node
}
