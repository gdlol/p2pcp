package node

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	mathRand "math/rand"
	"p2pcp/internal/auth"
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
	IsPrivateMode() bool
	StartMdns() error
	AdvertiseLAN(ctx context.Context, topic string) error
	WaitForWAN(ctx context.Context) error
	AdvertiseWAN(ctx context.Context, topic string) error
	FindPeers(ctx context.Context, topic string) (<-chan peer.AddrInfo, error)
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

func (n *node) IsPrivateMode() bool {
	return n.privateMode
}

func (n *node) StartMdns() error {
	return n.mdnsService.Start()
}

func (n *node) AdvertiseLAN(ctx context.Context, topic string) error {
	discovery := routing.NewRoutingDiscovery(n.dht.LAN)
	_, err := discovery.Advertise(ctx, topic)
	return err
}

func (n *node) WaitForWAN(ctx context.Context) error {
	for ctx.Err() == nil {
		if !n.dht.WANActive() {
			time.Sleep(time.Second)
			continue
		} else {
			break
		}
	}
	return ctx.Err()
}

func (n *node) AdvertiseWAN(ctx context.Context, topic string) error {
	discovery := routing.NewRoutingDiscovery(n.dht.WAN)
	_, err := discovery.Advertise(ctx, topic)
	return err
}

func (n *node) FindPeers(ctx context.Context, topic string) (<-chan peer.AddrInfo, error) {
	discovery := routing.NewRoutingDiscovery(n.dht)
	return discovery.FindPeers(ctx, topic)
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
			err := n.WaitForWAN(ctx)
			if err != nil {
				return
			}
			slog.Debug("Getting peers from DHT for auto relay...")
			peers, err = n.dht.WAN.GetClosestPeers(ctx, rand.Text())
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				slog.Debug("Error getting peers from DHT for auto relay.", "error", err)
			}
			if len(peers) > 0 {
				slog.Debug(fmt.Sprintf("Feeding %d peers from DHT for auto relay.", len(peers)))
				break
			} else {
				time.Sleep(b.Delay())
			}
		}

		for _, peerID := range peers {
			addrs := n.host.Peerstore().Addrs(peerID)
			addrs = slices.DeleteFunc(addrs, func(addr multiaddr.Multiaddr) bool {
				return !manet.IsPublicAddr(addr)
			})
			if len(addrs) == 0 {
				continue
			}
			addrInfo := peer.AddrInfo{
				ID:    peerID,
				Addrs: addrs,
			}
			if num > 0 {
				select {
				case n.peerSource <- addrInfo:
					num--
				case <-ctx.Done():
					return
				}
			} else {
				break
			}
		}
	}
}

func (n *node) Close() {
	n.mdnsService.Close()
	n.dht.Close()
	n.host.Close()
}

// Gets a SHA-256 hashed ID, as peerID.String() may or may not have been hashed.
func GetNodeID(peerID peer.ID) (NodeID, error) {
	pubKey, err := peerID.ExtractPublicKey()
	if err != nil {
		return nil, fmt.Errorf("error extracting public key: %w", err)
	}
	keyBytes, err := pubKey.Raw()
	if err != nil {
		return nil, fmt.Errorf("error getting raw public key: %w", err)
	}
	hashValue := auth.ComputeHash(keyBytes)
	return &nodeID{value: hashValue}, nil
}

func NewNode(ctx context.Context, privateMode bool, options ...libp2p.Option) (Node, error) {
	success := false
	closeIfError := func(closer io.Closer) {
		if !success {
			closer.Close()
		}
	}

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
						close(peerSource)
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
	if err != nil {
		return nil, fmt.Errorf("error creating host: %w", err)
	}
	defer closeIfError(host)

	id, err := GetNodeID(host.ID())
	if err != nil {
		return nil, fmt.Errorf("error getting node ID: %w", err)
	}

	dht, err := createDHT(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("error creating DHT: %w", err)
	}
	defer closeIfError(dht)
	routing.dht = dht

	mdnsService := createMdnsService(ctx, host, project.Name)
	defer closeIfError(mdnsService)

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

	success = true
	return node, nil
}
