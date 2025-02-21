package node

import (
	"context"
	"crypto"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	b58 "github.com/mr-tron/base58/base58"
)

type NodeID interface {
	String() string
	Bytes() []byte
}

type Node interface {
	ID() NodeID
	GetHost() host.Host
	StartMdns(handleMdnsPeerFound HandleMdnsPeerFound) error
	AdvertiseLAN(ctx context.Context, topic string) error
	AdvertiseWAN(ctx context.Context, topic string) error
	FindPeers(ctx context.Context, topic string) (<-chan peer.AddrInfo, error)
	GetProtocol() protocol.ID
	Close()
}

type nodeID struct {
	value []byte
}

type node struct {
	id          NodeID
	host        host.Host
	dht         *dual.DHT
	mdnsService MdnsService
	protocol    protocol.ID
}

func (k *nodeID) String() string {
	return b58.Encode(k.value)
}

func (k *nodeID) Bytes() []byte {
	return k.value
}

func (n *node) ID() NodeID {
	return n.id
}

func (n *node) GetHost() host.Host {
	return n.host
}

func (n *node) StartMdns(handlePeerFound HandleMdnsPeerFound) error {
	n.mdnsService.SetHandler(handlePeerFound)
	return n.mdnsService.Start()
}

func (n *node) AdvertiseLAN(ctx context.Context, topic string) error {
	discovery := routing.NewRoutingDiscovery(n.dht.LAN)
	_, err := discovery.Advertise(ctx, topic)
	return err
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
				peers, err := node.dht.WAN.GetClosestPeers(ctx, node.id.String())
				if err != nil {
					slog.Warn("Error getting closest peers.", "error", err)
					continue
				}
				slog.Debug("getPeerSource: Got closest peers.", "len(peers)", len(peers))
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

func sha256(input []byte) ([]byte, error) {
	hash := crypto.SHA256.New()
	_, err := hash.Write(input)
	if err != nil {
		return nil, err
	}
	return hash.Sum(nil), nil
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
	hashValue, err := sha256(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("error hashing public key: %w", err)
	}
	return &nodeID{value: hashValue}, nil
}

func NewNode(ctx context.Context, options ...libp2p.Option) (Node, error) {
	success := false
	closeIfError := func(closer io.Closer) {
		if !success {
			closer.Close()
		}
	}

	resultChan := make(chan node)
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

	id, err := GetNodeID(host.ID())
	if err != nil {
		return nil, fmt.Errorf("error getting node ID: %w", err)
	}

	dht, err := createDHT(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("error creating DHT: %w", err)
	}
	defer closeIfError(dht)

	mdnsService := createMdnsService(ctx, host, "p2pcp")
	defer closeIfError(mdnsService)

	node := &node{
		id:          id,
		host:        host,
		dht:         dht,
		mdnsService: mdnsService,
		protocol:    "/p2pcp/transfer/0.1.0",
	}

	go func() {
		select {
		case resultChan <- *node:
		case <-ctx.Done():
		}
	}()
	success = true
	return node, nil
}
