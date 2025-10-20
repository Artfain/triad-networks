package api

import (
	"encoding/json"
	"net/http"

	"github.com/Artfain/triad-networks/core"
)

func SetupREST(state *core.State) {
	http.HandleFunc("/blocks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data, _ := json.Marshal(state.Blockchain.Root)
		w.Write(data)
	})

	http.HandleFunc("/validate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		valid := state.ValidateBlockchain()
		json.NewEncoder(w).Encode(map[string]bool{"valid": valid})
	})

	http.ListenAndServe(":8081", nil) // Run on different port to not conflict with WebSocket
}
