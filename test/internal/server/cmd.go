package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"p2pcp/pkg/config"

	"github.com/libp2p/go-libp2p/core/peer"
)

func Run(ctx context.Context) error {
	// Create server node.
	host, err := NewServerNode(ctx)
	if err != nil {
		return err
	}
	defer host.Close()

	// Get listen addresses.
	addrInfo := host.Peerstore().PeerInfo(host.ID())
	p2pAddrs, err := peer.AddrInfoToP2pAddrs(&addrInfo)
	if err != nil {
		return err
	}

	// Generate config file for clients.
	cfg := config.NewConfig()
	bootStrapPeers := make([]string, 0, len(p2pAddrs))
	for _, addr := range p2pAddrs {
		bootStrapPeers = append(bootStrapPeers, addr.String())
	}
	cfg.BootstrapPeers = bootStrapPeers
	jsonConfig, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("config:\n%s\n", jsonConfig)
	fs, err := os.OpenFile("/config/config.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	func() {
		defer fs.Close()
		_, err = fs.Write(jsonConfig)
	}()
	if err != nil {
		return err
	}

	// Mark server ready.
	ready, err := os.OpenFile("/tmp/ready", os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	ready.Close()

	fmt.Println("Server is ready.")
	<-ctx.Done()
	return nil
}

func Ready() error {
	_, err := os.Stat("/tmp/ready")
	return err
}
