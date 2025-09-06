package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Разрешить все подключения (для примера)
	},
}

var state = NewState()
var tree = NewTriadTree()

// connections хранит WebSocket-соединения для каждого пользователя
var connections = struct {
	sync.Mutex
	m map[string]map[*websocket.Conn]bool // address -> conn -> true
}{
	m: make(map[string]map[*websocket.Conn]bool),
}

type Message struct {
	Action      string      `json:"action"`
	UserData    UserData    `json:"userData"`
	DeviceID    string      `json:"deviceID"`
	Transaction Transaction `json:"transaction"`
}

func broadcastUpdate(address string, updateData map[string]interface{}) {
	connections.Lock()
	defer connections.Unlock()

	if conns, exists := connections.m[address]; exists {
		for conn := range conns {
			err := conn.WriteJSON(updateData)
			if err != nil {
				log.Println("Error broadcasting update to", address, ":", err)
				delete(conns, conn)
				conn.Close()
			}
		}
		if len(conns) == 0 {
			delete(connections.m, address)
		}
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer conn.Close()

	var userAddress string

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			// Удаляем соединение при отключении
			if userAddress != "" {
				connections.Lock()
				if conns, exists := connections.m[userAddress]; exists {
					delete(conns, conn)
					if len(conns) == 0 {
						delete(connections.m, userAddress)
					}
				}
				connections.Unlock()
			}
			break
		}

		var message Message
		if err := json.Unmarshal(msg, &message); err != nil {
			conn.WriteJSON(map[string]string{"status": "error", "message": "Invalid JSON"})
			continue
		}

		// Сохраняем адрес пользователя для отслеживания соединений
		if userAddress == "" && message.UserData.Address != "" {
			userAddress = message.UserData.Address
			connections.Lock()
			if _, exists := connections.m[userAddress]; !exists {
				connections.m[userAddress] = make(map[*websocket.Conn]bool)
			}
			connections.m[userAddress][conn] = true
			connections.Unlock()
		}

		switch message.Action {
		case "addUser":
			qli, err := CreateQLI(message.UserData)
			if err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}
			if err := tree.AddUser(message.UserData.Address, message.UserData, qli); err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}
			if err := StoreData(message.UserData.Address, message.UserData); err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}
			state.AddUser(message.UserData.Address, message.UserData)
			conn.WriteJSON(map[string]string{"status": "success", "qli": qli})

		case "getUser":
			qli, err := tree.GetUser(message.UserData.Address)
			if err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}
			userData, err := GetData(message.UserData.Address)
			if err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}
			conn.WriteJSON(map[string]interface{}{"status": "success", "user": userData, "qli": qli})

		case "syncUser":
			qli, err := CreateQLI(message.UserData)
			if err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}
			if err := UpdateData(message.UserData.Address, qli, message.UserData); err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}
			if err := tree.AddUser(message.UserData.Address, message.UserData, qli); err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}
			if err := UpdateTreesPlanted(message.UserData.PoCContribution.Computations); err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}
			state.UpdateUser(message.UserData.Address, message.UserData)
			conn.WriteJSON(map[string]string{"status": "success", "qli": qli})

			// Отправляем обновление всем устройствам пользователя
			updateData := map[string]interface{}{
				"action": "userUpdated",
				"user":   message.UserData,
				"qli":    qli,
			}
			broadcastUpdate(message.UserData.Address, updateData)

		case "sendTransaction":
			tx := message.Transaction
			tx.Timestamp = time.Now().UnixNano()

			// Выполняем транзакцию
			if err := state.ExecuteTransaction(tx.From, tx.To, tx.Amount); err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}

			// Сохраняем транзакцию в LevelDB
			if err := StoreTransaction(tx); err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}

			// Обновляем данные пользователей в хранилище
			fromUser, err := GetData(tx.From)
			if err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}
			toUser, err := GetData(tx.To)
			if err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}
			if err := StoreData(tx.From, fromUser); err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}
			if err := StoreData(tx.To, toUser); err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}

			conn.WriteJSON(map[string]string{"status": "success", "message": "Transaction completed"})

			// Уведомляем отправителя и получателя
			broadcastUpdate(tx.From, map[string]interface{}{
				"action":      "transaction",
				"transaction": tx,
				"user":        fromUser,
			})
			broadcastUpdate(tx.To, map[string]interface{}{
				"action":      "transaction",
				"transaction": tx,
				"user":        toUser,
			})

		case "getTransactions":
			transactions, err := GetTransactions(message.UserData.Address)
			if err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}
			conn.WriteJSON(map[string]interface{}{"status": "success", "transactions": transactions})

		case "getTrees":
			trees, err := GetTreesPlanted()
			if err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}
			conn.WriteJSON(map[string]interface{}{"status": "success", "trees": trees})

		case "addDevice":
			if err := state.AddDevice(message.UserData.Address, message.DeviceID); err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}
			conn.WriteJSON(map[string]string{"status": "success", "message": "Device added"})

		case "getDevices":
			devices, err := state.GetDevices(message.UserData.Address)
			if err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}
			conn.WriteJSON(map[string]interface{}{"status": "success", "devices": devices})

		case "removeDevice":
			if err := state.RemoveDevice(message.UserData.Address, message.DeviceID); err != nil {
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}
			conn.WriteJSON(map[string]string{"status": "success", "message": "Device removed"})

		case "getNodes":
			nodes := state.GetNodes()
			conn.WriteJSON(map[string]interface{}{"status": "success", "nodes": nodes})

		case "getTokens":
			tokens := state.GetTokens(message.UserData.Address)
			conn.WriteJSON(map[string]interface{}{"status": "success", "tokens": tokens})

		default:
			conn.WriteJSON(map[string]string{"status": "error", "message": "Unknown action"})
		}
	}
}

func getNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{"nodes": state.GetNodes()}
	json.NewEncoder(w).Encode(resp)
}

func getTokens(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	address := r.URL.Query().Get("address")
	tokens := state.GetTokens(address)
	json.NewEncoder(w).Encode(map[string]interface{}{"tokens": tokens})
}

func main() {
	// Настраиваем WebSocket и API-эндпоинты
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/nodes", getNodes)
	http.HandleFunc("/tokens", getTokens)

	// Настраиваем обработчик для статических файлов из директории client/
	fs := http.FileServer(http.Dir("client"))
	http.Handle("/", fs)

	log.Println("WebSocket server running on ws://localhost:8080/ws")
	log.Println("HTTP server running on http://localhost:8080/nodes and http://localhost:8080/tokens")
	log.Println("Static files serving on http://localhost:8080/ (from client/ directory)")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Error starting server:", err)
	}
}
