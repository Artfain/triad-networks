package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Artfain/triad-networks/api"
	"github.com/Artfain/triad-networks/core"
	"github.com/Artfain/triad-networks/p2p"
)

func main() {
	// Initialize state
	state := core.NewState()

	// Initialize P2P network
	p2p, err := p2p.NewP2P(state)
	if err != nil {
		slog.Error("Failed to create P2P", "error", err)
		return
	}
	// Print multiaddr for this node
	fmt.Println("P2P multiaddr:", p2p.Host().Addrs()[0].String()+"/p2p/"+p2p.Host().ID().String())

	// Start WebSocket server
	http.HandleFunc("/ws", api.HandleWebSocket)
	http.HandleFunc("/nodes", api.GetNodes)
	http.HandleFunc("/tokens", api.GetTokens)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "client/index.html")
	})

	// Start REST API in a separate goroutine
	go api.SetupREST(state)

	// Start server
	slog.Info("Starting server", "websocket", "ws://localhost:8080/ws", "http", "http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		slog.Error("Failed to start server", "error", err)
	}
}
