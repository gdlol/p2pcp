package node

import (
	"context"
	"log/slog"
	"sync"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"

	p2p "p2pcp/pkg/pb"
)

// pattern: /protocol-name/request-or-response-message/version
const ProtocolPushRequest = "/pcp/push/0.0.1"

// PushProtocol .
type PushProtocol struct {
	node *Node
	lk   sync.RWMutex
	prh  PushRequestHandler
}

type PushRequestHandler interface {
	HandlePushRequest(*p2p.PushRequest) (bool, error)
}

func NewPushProtocol(node *Node) *PushProtocol {
	return &PushProtocol{node: node, lk: sync.RWMutex{}}
}

func (p *PushProtocol) RegisterPushRequestHandler(prh PushRequestHandler) {
	slog.Debug("Registering push request handler")
	p.lk.Lock()
	defer p.lk.Unlock()
	p.prh = prh
	p.node.SetStreamHandler(ProtocolPushRequest, p.onPushRequest)
}

func (p *PushProtocol) UnregisterPushRequestHandler() {
	slog.Debug("Unregistering push request handler")
	p.lk.Lock()
	defer p.lk.Unlock()
	p.node.RemoveStreamHandler(ProtocolPushRequest)
	p.prh = nil
}

func (p *PushProtocol) onPushRequest(s network.Stream) {
	defer s.Close()
	defer p.node.ResetOnShutdown(s)()

	if !p.node.IsAuthenticated(s.Conn().RemotePeer()) {
		slog.Info("Received push request from unauthenticated peer")
		s.Reset() // Tell peer to go away
		return
	}

	req := &p2p.PushRequest{}
	if err := p.node.Read(s, req); err != nil {
		slog.Info("Error reading push request", "err", err)
		return
	}
	slog.Debug("Received push request", "name", req.Name, "size", req.Size)

	p.lk.RLock()
	defer p.lk.RUnlock()
	accept, err := p.prh.HandlePushRequest(req)
	if err != nil {
		slog.Info("Error handling push request", "err", err)
		accept = false
		// Fall through and tell peer we won't handle the request
	}

	if err := p.node.Send(s, p2p.NewPushResponse(accept)); err != nil {
		slog.Info("Error sending push response", "err", err)
		return
	}

	if err = p.node.WaitForEOF(s); err != nil {
		slog.Info("Error waiting for EOF", "err", err)
		return
	}
}

func (p *PushProtocol) SendPushRequest(ctx context.Context, peerID peer.ID, filename string, size int64, isDir bool) (bool, error) {
	s, err := p.node.NewStream(ctx, peerID, ProtocolPushRequest)
	if err != nil {
		return false, err
	}
	defer s.Close()

	slog.Debug("Sending push request", "filename", filename, "size", size)
	if err = p.node.Send(s, p2p.NewPushRequest(filename, size, isDir)); err != nil {
		return false, err
	}

	resp := &p2p.PushResponse{}
	if err = p.node.Read(s, resp); err != nil {
		return false, err
	}

	return resp.Accept, nil
}
