package p2p

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/Artfain/triad-networks/core"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// SyncBlockchain syncs the blockchain with all peers.
func (p *P2P) SyncBlockchain() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	for peerID := range p.peers {
		stream, err := p.host.NewStream(context.Background(), peerID, protocol.ID("/triad/sync/1.0.0"))
		if err != nil {
			slog.Error("Failed to open sync stream to peer", "peer", peerID, "error", err)
			continue
		}
		// Send current tree root hash to peer
		data, _ := json.Marshal(struct{ RootHash string }{RootHash: p.state.Blockchain.Root.Block.Hash})
		_, err = stream.Write(data)
		if err != nil {
			slog.Error("Failed to send root hash to peer", "peer", peerID, "error", err)
			continue
		}
		buf := make([]byte, 1024*1024)
		n, err := stream.Read(buf)
		if err != nil {
			slog.Error("Failed to read from sync stream", "error", err)
			continue
		}
		var subtree core.TriadNode
		if err := json.Unmarshal(buf[:n], &subtree); err != nil {
			slog.Error("Failed to unmarshal subtree", "error", err)
			continue
		}
		slog.Info("Received subtree from peer", "peer", peerID)
		// Integrate received subtree (simplified, merge with local tree)
		// For example:
		// p.state.Blockchain.mergeSubtree(subtree)
	}
}
