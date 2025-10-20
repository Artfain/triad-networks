package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Artfain/triad-networks/core"
	"github.com/gorilla/websocket"
)

// Message represents a WebSocket message.
type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity
	},
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade to WebSocket", "error", err)
		return
	}
	defer conn.Close()

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			slog.Error("Failed to read WebSocket message", "error", err)
			return
		}

		switch msg.Type {
		case "register":
			var data struct {
				Address  string `json:"address"`
				DeviceID string `json:"deviceID"`
			}
			if err := json.Unmarshal(msg.Data, &data); err != nil {
				conn.WriteJSON(map[string]string{"error": "invalid data"})
				continue
			}
			userData := core.UserData{
				Balance:      1000,
				Reputation:   core.NewReputation(),
				Devices:      []string{data.DeviceID},
				TreesPlanted: 0,
			}
			if err := state.AddUser(data.Address, data.DeviceID, userData); err != nil {
				conn.WriteJSON(map[string]string{"error": err.Error()})
				continue
			}
			conn.WriteJSON(map[string]string{"status": "registered"})

		case "contribute":
			var data struct {
				Address      string               `json:"address"`
				DeviceID     string               `json:"deviceID"`
				Contribution core.PoCContribution `json:"contribution"`
				Trees        int64                `json:"trees"`
			}
			if err := json.Unmarshal(msg.Data, &data); err != nil {
				conn.WriteJSON(map[string]string{"error": "invalid data"})
				continue
			}
			userData, exists := state.GetData(data.Address)
			if !exists {
				conn.WriteJSON(map[string]string{"error": "user not found"})
				continue
			}
			isHonest := !core.DetectCheat(data.Contribution.Computations)
			userData.Reputation = core.UpdateReputation(userData.Reputation, data.Contribution.Uptime, isHonest)
			state.UpdateData(data.Address, data.DeviceID, data.Contribution, data.Trees)
			conn.WriteJSON(map[string]string{"status": "contribution recorded"})

		case "get_data":
			var data struct {
				Address string `json:"address"`
			}
			if err := json.Unmarshal(msg.Data, &data); err != nil {
				conn.WriteJSON(map[string]string{"error": "invalid data"})
				continue
			}
			userData, exists := state.GetData(data.Address)
			if !exists {
				conn.WriteJSON(map[string]string{"error": "user not found"})
				continue
			}
			conn.WriteJSON(userData)

		case "get_transactions":
			var data struct {
				Address string `json:"address"`
			}
			if err := json.Unmarshal(msg.Data, &data); err != nil {
				conn.WriteJSON(map[string]string{"error": "invalid data"})
				continue
			}
			transactions, err := core.GetTransactions(data.Address)
			if err != nil {
				conn.WriteJSON(map[string]string{"error": err.Error()})
				continue
			}
			conn.WriteJSON(transactions)

		case "get_trees":
			var data struct {
				Address string `json:"address"`
			}
			if err := json.Unmarshal(msg.Data, &data); err != nil {
				conn.WriteJSON(map[string]string{"error": "invalid data"})
				continue
			}
			trees := state.GetTreesPlanted(data.Address)
			conn.WriteJSON(map[string]int64{"treesPlanted": trees})
		}
	}
}

var state = core.NewState()

func GetNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal(state.Blockchain.Nodes)
	w.Write(data)
}

func GetTokens(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal(state.Users)
	w.Write(data)
}
