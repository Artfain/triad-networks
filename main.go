package main

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return r.Host == "localhost:8080"
	},
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/nodes", getNodes)
	http.HandleFunc("/tokens", getTokens)
	fs := http.FileServer(http.Dir("client"))
	http.Handle("/", fs)
	slog.Info("Starting server", "websocket", "ws://localhost:8080/ws", "http", "http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		slog.Error("Error starting server", "error", err)
	}
}
