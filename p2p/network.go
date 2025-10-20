package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/Artfain/triad-networks/core"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// P2P manages the peer-to-peer network.
type P2P struct {
	host  host.Host
	peers map[peer.ID]struct{}
	mutex sync.Mutex
	state *core.State
}

func NewP2P(state *core.State) (*P2P, error) {
	h, err := libp2p.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %v", err)
	}
	p := &P2P{
		host:  h,
		peers: make(map[peer.ID]struct{}),
		state: state,
	}
	h.SetStreamHandler(protocol.ID("/triad/1.0.0"), p.handleStream)
	return p, nil
}

// AddPeer adds a peer to the network.
func (p *P2P) AddPeer(multiaddr string) error {
	addrInfo, err := peer.AddrInfoFromString(multiaddr)
	if err != nil {
		return fmt.Errorf("failed to parse peer address: %v", err)
	}
	if err := p.host.Connect(context.Background(), *addrInfo); err != nil {
		return fmt.Errorf("failed to connect to peer: %v", err)
	}
	p.mutex.Lock()
	p.peers[addrInfo.ID] = struct{}{}
	p.mutex.Unlock()
	slog.Info("Connected to peer", "peer", addrInfo.ID.String())
	return nil
}

// BroadcastBlock broadcasts a new block to all peers.
func (p *P2P) BroadcastBlock(block *core.Block) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	data, _ := json.Marshal(block)
	for peerID := range p.peers {
		stream, err := p.host.NewStream(context.Background(), peerID, protocol.ID("/triad/1.0.0"))
		if err != nil {
			slog.Error("Failed to open stream to peer", "peer", peerID, "error", err)
			continue
		}
		_, err = stream.Write(data)
		if err != nil {
			slog.Error("Failed to send block to peer", "peer", peerID, "error", err)
		}
		stream.Close()
	}
}

// handleStream handles incoming streams from peers.
func (p *P2P) handleStream(stream network.Stream) {
	defer stream.Close()
	buf := make([]byte, 1024*1024) // 1MB buffer
	n, err := stream.Read(buf)
	if err != nil {
		slog.Error("Failed to read from stream", "error", err)
		return
	}
	var block core.Block
	if err := json.Unmarshal(buf[:n], &block); err != nil {
		slog.Error("Failed to unmarshal block", "error", err)
		return
	}
	slog.Info("Received block from peer", "peer", stream.Conn().RemotePeer().String(), "block", block.Hash)
	// Add received block to the triad tree
	p.state.AddBlock(block.Data, block.Validator, block.ParentHash)
}

// Host returns the libp2p host.
func (p *P2P) Host() host.Host {
	return p.host
}
