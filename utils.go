package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

var state = NewState()
var tree = NewTriadTree()

var connections = struct {
	sync.Mutex
	m map[string]map[*websocket.Conn]bool
}{
	m: make(map[string]map[*websocket.Conn]bool),
}

var loadLimits = struct {
	sync.Mutex
	m map[string]uint64
}{
	m: make(map[string]uint64),
}

var powerData = struct {
	sync.Mutex
	m map[string]struct {
		CPUPercent float64
		MemoryMB   float64
		Storage    uint64
		Bandwidth  uint64
		Uptime     uint64
		EcoActions uint64
	}
}{
	m: make(map[string]struct {
		CPUPercent float64
		MemoryMB   float64
		Storage    uint64
		Bandwidth  uint64
		Uptime     uint64
		EcoActions uint64
	}),
}

var reputations = struct {
	sync.Mutex
	m map[string]*Reputation
}{
	m: make(map[string]*Reputation),
}

var rateLimiter = rate.NewLimiter(rate.Every(time.Minute/100), 100)
var anomalyLimiter = rate.NewLimiter(rate.Every(time.Minute), 100000)

var mfaTokens = struct {
	sync.Mutex
	m map[string]struct {
		Token     string
		ExpiresAt time.Time
	}
}{
	m: make(map[string]struct {
		Token     string
		ExpiresAt time.Time
	}),
}

var weights = struct {
	CPU        float64
	Storage    float64
	Bandwidth  float64
	Uptime     float64
	EcoActions float64
}{
	CPU:        1.0,
	Storage:    0.5,
	Bandwidth:  0.2,
	Uptime:     0.1,
	EcoActions: 2.0,
}

func init() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(log.Writer(), &slog.HandlerOptions{Level: slog.LevelDebug})))
	go realTimeMinting()
}

func calculateEmissionFactor() float64 {
	participantCount := state.GetParticipantCount()
	if participantCount < 10 {
		return 2.0
	} else if participantCount < 100 {
		return 1.5
	}
	return 1.0 / math.Log(float64(participantCount)+1)
}

func realTimeMinting() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		connections.Lock()
		for address := range connections.m {
			userData, err := GetData(address, "macbook")
			if err != nil {
				continue
			}
			powerData.Lock()
			power := powerData.m[address+":macbook"]
			powerData.Unlock()
			if power.CPUPercent == 0 && power.MemoryMB == 0 && power.Storage == 0 && power.Bandwidth == 0 && power.Uptime == 0 && power.EcoActions == 0 {
				continue
			}
			computations := uint64(power.CPUPercent*10 + power.MemoryMB*5)
			computations = PerformUsefulWork(computations)
			isCheating := DetectCheat(computations)
			reputations.Lock()
			reputation, exists := reputations.m[address]
			if !exists {
				reputation = NewReputation()
				reputations.m[address] = reputation
			}
			reputationScore := UpdateReputation(reputation, power.Uptime, !isCheating).Score
			reputations.Unlock()
			userData.Reputation = reputation
			emissionFactor := calculateEmissionFactor()
			tokens := int64(float64(computations)*weights.CPU*reputationScore*emissionFactor +
				float64(power.Storage)*weights.Storage*reputationScore*emissionFactor +
				float64(power.Bandwidth)*weights.Bandwidth*reputationScore*emissionFactor +
				float64(power.Uptime)*weights.Uptime*reputationScore*emissionFactor +
				float64(power.EcoActions)*weights.EcoActions*reputationScore*emissionFactor)
			userData.Balance += tokens
			userData.PoCContribution.Computations += computations
			userData.PoCContribution.Storage += power.Storage
			userData.PoCContribution.Bandwidth += power.Bandwidth
			userData.PoCContribution.Uptime += power.Uptime
			userData.PoCContribution.EcoActions += power.EcoActions
			if err := UpdateData(address, "macbook", userData); err != nil {
				continue
			}
			if err := UpdateTreesPlanted(computations); err != nil {
				continue
			}
			state.UpdateUser(address, userData)
			broadcastUpdate(address, map[string]interface{}{
				"action":       "powerContributed",
				"user":         userData,
				"power":        map[string]interface{}{"cpuPercent": power.CPUPercent, "memoryMB": power.MemoryMB},
				"storage":      power.Storage,
				"bandwidth":    power.Bandwidth,
				"uptime":       power.Uptime,
				"ecoActions":   power.EcoActions,
				"tokensMinted": tokens,
				"reputation":   reputationScore,
			})
		}
		connections.Unlock()
	}
}

func broadcastUpdate(address string, updateData map[string]interface{}) {
	connections.Lock()
	defer connections.Unlock()
	if conns, exists := connections.m[address]; exists {
		for conn := range conns {
			if err := conn.WriteJSON(updateData); err != nil {
				slog.Error("Error broadcasting to", "address", address, "error", err)
				delete(conns, conn)
				conn.Close()
			}
		}
		if len(conns) == 0 {
			delete(connections.m, address)
		}
	}
}

func updateUserData(address, deviceID string, userData UserData, computations uint64, storage, bandwidth float64, uptime, ecoActions uint64, cpuPercent, memoryMB float64) error {
	loadLimits.Lock()
	limit, exists := loadLimits.m[address+":"+deviceID]
	if !exists {
		limit = 100000 // Увеличен лимит с 10000 до 100000
	}
	loadLimits.Unlock()

	if computations > limit {
		computations = limit
		slog.Warn("Computations limited", "address", address, "deviceID", deviceID, "limit", limit)
	}

	isCheating := DetectCheat(computations)
	reputations.Lock()
	reputation, exists := reputations.m[address]
	if !exists {
		reputation = NewReputation()
		reputations.m[address] = reputation
	}
	reputationScore := UpdateReputation(reputation, uptime, !isCheating).Score
	reputations.Unlock()
	userData.Reputation = reputation

	emissionFactor := calculateEmissionFactor()
	tokens := int64(float64(computations)*weights.CPU*reputationScore*emissionFactor +
		storage*weights.Storage*reputationScore*emissionFactor +
		bandwidth*weights.Bandwidth*reputationScore*emissionFactor +
		float64(uptime)*weights.Uptime*reputationScore*emissionFactor +
		float64(ecoActions)*weights.EcoActions*reputationScore*emissionFactor)

	userData.PoCContribution.Computations += computations
	userData.PoCContribution.Storage += uint64(math.Round(storage))
	userData.PoCContribution.Bandwidth += uint64(math.Round(bandwidth))
	userData.PoCContribution.Uptime += uptime
	userData.PoCContribution.EcoActions += ecoActions
	userData.Balance += tokens
	if userData.Balance > 1<<63-1 {
		return fmt.Errorf("balance overflow")
	}
	if err := UpdateData(address, deviceID, userData); err != nil {
		return fmt.Errorf("update user data: %v", err)
	}
	if err := UpdateTreesPlanted(computations); err != nil {
		return fmt.Errorf("update trees: %v", err)
	}
	state.UpdateUser(address, userData)

	powerData.Lock()
	powerData.m[address+":"+deviceID] = struct {
		CPUPercent float64
		MemoryMB   float64
		Storage    uint64
		Bandwidth  uint64
		Uptime     uint64
		EcoActions uint64
	}{CPUPercent: cpuPercent, MemoryMB: memoryMB, Storage: uint64(math.Round(storage)), Bandwidth: uint64(math.Round(bandwidth)), Uptime: uptime, EcoActions: ecoActions}
	powerData.Unlock()

	return nil
}

func verifyMFAToken(address, token string) bool {
	mfaTokens.Lock()
	defer mfaTokens.Unlock()
	data, exists := mfaTokens.m[address]
	if !exists || data.Token != token || data.ExpiresAt.Before(time.Now()) {
		return false
	}
	return true
}

func generateMFAToken(address string) string {
	mfaTokens.Lock()
	defer mfaTokens.Unlock()
	if data, exists := mfaTokens.m[address]; exists && data.ExpiresAt.After(time.Now()) {
		return data.Token
	}
	token := fmt.Sprintf("%x", sha256.Sum256([]byte(address+fmt.Sprintf("%d", time.Now().UnixNano()))))
	mfaTokens.m[address] = struct {
		Token     string
		ExpiresAt time.Time
	}{Token: token, ExpiresAt: time.Now().Add(5 * time.Minute)}
	return token
}

func validateInput(message Message) error {
	if message.UserData.Address != "" && len(message.UserData.Address) > 64 {
		return fmt.Errorf("address too long")
	}
	if message.DeviceID != "" && len(message.DeviceID) > 64 {
		return fmt.Errorf("deviceID too long")
	}
	if message.Transaction.Amount < 0 {
		return fmt.Errorf("negative amount")
	}
	if message.Power.CPUPercent < 0 || message.Power.MemoryMB < 0 {
		return fmt.Errorf("negative power values")
	}
	if message.Storage < 0 || message.Bandwidth < 0 || message.Uptime < 0 || message.EcoActions < 0 {
		return fmt.Errorf("negative contribution values")
	}
	if message.CPULoad < 0 || message.CPULoad > 100 {
		return fmt.Errorf("invalid CPU load percentage")
	}
	return nil
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !rateLimiter.Allow() {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Error upgrading to WebSocket", "error", err)
		return
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(180 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(180 * time.Second))
		return nil
	})
	conn.SetCloseHandler(func(code int, text string) error {
		slog.Info("WebSocket closed", "code", code, "reason", text)
		return nil
	})

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	var userAddress string
	var mfaToken string

	for {
		select {
		case <-ctx.Done():
			return
		case <-pingTicker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Error("Error sending ping", "error", err)
				return
			}
		default:
			_, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					slog.Info("WebSocket closed by client", "error", err)
				} else {
					slog.Error("Error reading message", "error", err)
				}
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
				return
			}

			var message Message
			if err := json.Unmarshal(msg, &message); err != nil {
				slog.Error("Invalid JSON received", "error", err, "message", string(msg))
				conn.WriteJSON(map[string]string{"status": "error", "message": fmt.Sprintf("Invalid JSON: %v", err)})
				continue
			}

			if err := validateInput(message); err != nil {
				slog.Error("Invalid input", "error", err)
				conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
				continue
			}

			if userAddress == "" && message.UserData.Address != "" {
				userAddress = message.UserData.Address
				connections.Lock()
				if _, exists := connections.m[userAddress]; !exists {
					connections.m[userAddress] = make(map[*websocket.Conn]bool)
				}
				connections.m[userAddress][conn] = true
				connections.Unlock()
				slog.Info("User connected", "address", userAddress)
				mfaToken = generateMFAToken(userAddress)
				conn.WriteJSON(map[string]interface{}{"status": "success", "mfaToken": mfaToken})
			}

			if message.Action == "sendTransaction" || message.Action == "removeDevice" || message.Action == "contributePower" || message.Action == "startContributing" {
				if !verifyMFAToken(userAddress, message.MFAToken) {
					slog.Error("Invalid MFA token", "address", userAddress, "token", message.MFAToken)
					reputations.Lock()
					if rep, exists := reputations.m[userAddress]; exists {
						rep.InvalidMFAAttempts++
						if rep.InvalidMFAAttempts > 5 {
							rep.Score = math.Max(0.1, rep.Score*0.8)
						}
						reputations.m[userAddress] = rep
					}
					reputations.Unlock()
					conn.WriteJSON(map[string]string{"status": "error", "message": "Invalid MFA token"})
					continue
				}
			}

			switch message.Action {
			case "addUser":
				if message.UserData.Address == "" || message.DeviceID == "" {
					slog.Error("Empty address or deviceID")
					conn.WriteJSON(map[string]string{"status": "error", "message": "Empty address or deviceID"})
					continue
				}
				qli, privKey, err := CreateQLI(message.UserData.Address, message.DeviceID)
				if err != nil {
					slog.Error("Error creating QLI", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				// Форматируем публичный ключ с ведущими нулями для фиксированной длины 128 символов
				xBytes := privKey.PublicKey.X.Bytes()
				yBytes := privKey.PublicKey.Y.Bytes()
				xStr := fmt.Sprintf("%064x", xBytes)
				yStr := fmt.Sprintf("%064x", yBytes)
				message.UserData.PublicKey = xStr + yStr
				if len(message.UserData.PublicKey) != 128 {
					slog.Error("Invalid public key length", "length", len(message.UserData.PublicKey))
					conn.WriteJSON(map[string]string{"status": "error", "message": fmt.Sprintf("Invalid public key length: %d", len(message.UserData.PublicKey))})
					continue
				}
				message.UserData.LastNonce = 0
				message.UserData.Reputation = NewReputation()
				if err := state.AddUser(message.UserData.Address, message.DeviceID, message.UserData, tree, qli); err != nil {
					slog.Error("Error adding user", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				reputations.Lock()
				reputations.m[message.UserData.Address] = message.UserData.Reputation
				reputations.Unlock()
				slog.Info("User added", "address", message.UserData.Address)
				conn.WriteJSON(map[string]interface{}{"status": "success", "qli": qli, "mfaToken": mfaToken})

			case "getUser":
				qli, err := tree.GetUser(message.UserData.Address, message.DeviceID)
				if err != nil {
					slog.Error("Error getting user", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				userData, err := GetData(message.UserData.Address, message.DeviceID)
				if err != nil {
					slog.Error("Error getting user data", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				conn.WriteJSON(map[string]interface{}{"status": "success", "user": userData, "qli": qli})

			case "syncUser":
				qli, _, err := CreateQLI(message.UserData.Address, message.DeviceID)
				if err != nil {
					slog.Error("Error creating QLI for sync", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				userData, err := GetData(message.UserData.Address, message.DeviceID)
				if err != nil {
					slog.Error("Error getting user data for sync", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				if err := updateUserData(message.UserData.Address, message.DeviceID, userData, message.UserData.PoCContribution.Computations, float64(message.UserData.PoCContribution.Storage), float64(message.UserData.PoCContribution.Bandwidth), message.UserData.PoCContribution.Uptime, message.UserData.PoCContribution.EcoActions, 0, 0); err != nil {
					slog.Error("Error updating user data for sync", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				if err := tree.AddUser(message.UserData.Address, message.DeviceID, userData, qli); err != nil {
					slog.Error("Error adding user to tree for sync", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				state.UpdateUser(message.UserData.Address, userData)
				slog.Info("User synced", "address", message.UserData.Address)
				conn.WriteJSON(map[string]interface{}{"status": "success", "qli": qli, "mfaToken": mfaToken})
				broadcastUpdate(message.UserData.Address, map[string]interface{}{
					"action": "userUpdated",
					"user":   userData,
					"qli":    qli,
				})

			case "sendTransaction":
				tx := message.Transaction
				tx.Timestamp = time.Now().UnixNano()
				if tx.Amount <= 0 {
					slog.Error("Invalid transaction amount")
					conn.WriteJSON(map[string]string{"status": "error", "message": "Invalid amount"})
					continue
				}
				if err := state.VerifyTransaction(tx); err != nil {
					slog.Error("Error verifying transaction", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				if err := state.ExecuteTransaction(tx.From, tx.To, tx.Amount, tx.Nonce); err != nil {
					slog.Error("Error executing transaction", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				if err := StoreTransaction(tx); err != nil {
					slog.Error("Error storing transaction", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				fromUser, err := GetData(tx.From, message.DeviceID)
				if err != nil {
					slog.Error("Error getting from user", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				toUser, err := GetData(tx.To, message.DeviceID)
				if err != nil {
					slog.Error("Error getting to user", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				if err := StoreData(tx.From, message.DeviceID, fromUser); err != nil {
					slog.Error("Error storing from user", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				if err := StoreData(tx.To, message.DeviceID, toUser); err != nil {
					slog.Error("Error storing to user", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				slog.Info("Transaction completed", "from", tx.From, "to", tx.To, "amount", tx.Amount)
				conn.WriteJSON(map[string]interface{}{"status": "success", "message": "Transaction completed", "mfaToken": mfaToken})
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

			case "contributePower":
				if message.Power.CPUPercent == 0 && message.Power.MemoryMB == 0 && message.Storage == 0 && message.Bandwidth == 0 && message.Uptime == 0 && message.EcoActions == 0 {
					slog.Error("No contribution provided")
					conn.WriteJSON(map[string]string{"status": "error", "message": "No contribution provided"})
					continue
				}
				if len(message.DeviceID) > 64 {
					slog.Error("Invalid deviceID length", "deviceID", message.DeviceID)
					conn.WriteJSON(map[string]string{"status": "error", "message": "DeviceID too long"})
					continue
				}
				if err := state.AddDevice(message.UserData.Address, message.DeviceID); err != nil && err.Error() != "device already added" {
					slog.Error("Error adding device", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				userData, err := GetData(message.UserData.Address, message.DeviceID)
				if err != nil {
					slog.Error("Error getting user data for device", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				computations := uint64(message.Power.CPUPercent*10 + message.Power.MemoryMB*5)
				computations = PerformUsefulWork(computations)
				if err := updateUserData(message.UserData.Address, message.DeviceID, userData, computations, message.Storage, message.Bandwidth, message.Uptime, message.EcoActions, message.Power.CPUPercent, message.Power.MemoryMB); err != nil {
					slog.Error("Error updating user data for power", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				slog.Info("Contribution processed", "address", message.UserData.Address, "deviceID", message.DeviceID, "cpuPercent", message.Power.CPUPercent, "memoryMB", message.Power.MemoryMB, "storage", message.Storage, "bandwidth", message.Bandwidth, "uptime", message.Uptime, "ecoActions", message.EcoActions)
				conn.WriteJSON(map[string]interface{}{"status": "success", "message": "Contribution processed and tokens minted", "mfaToken": mfaToken})
				broadcastUpdate(message.UserData.Address, map[string]interface{}{
					"action":     "powerContributed",
					"user":       userData,
					"power":      map[string]interface{}{"cpuPercent": message.Power.CPUPercent, "memoryMB": message.Power.MemoryMB},
					"storage":    message.Storage,
					"bandwidth":  message.Bandwidth,
					"uptime":     message.Uptime,
					"ecoActions": message.EcoActions,
				})

			case "startContributing":
				if message.CPULoad == 0 {
					slog.Error("No CPU load provided")
					conn.WriteJSON(map[string]string{"status": "error", "message": "No CPU load provided"})
					continue
				}
				if len(message.DeviceID) > 64 {
					slog.Error("Invalid deviceID length", "deviceID", message.DeviceID)
					conn.WriteJSON(map[string]string{"status": "error", "message": "DeviceID too long"})
					continue
				}
				_, err := GetData(message.UserData.Address, message.DeviceID)
				if err != nil {
					slog.Error("Error getting user data for device", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				slog.Info("Started contributing power", "address", message.UserData.Address, "deviceID", message.DeviceID, "cpuLoad", message.CPULoad)
				conn.WriteJSON(map[string]interface{}{"status": "success", "message": "Started contributing power", "mfaToken": mfaToken})

			case "getMFAToken":
				if userAddress == "" {
					slog.Error("No user address provided")
					conn.WriteJSON(map[string]string{"status": "error", "message": "No user address provided"})
					continue
				}
				mfaToken = generateMFAToken(userAddress)
				conn.WriteJSON(map[string]interface{}{"status": "success", "mfaToken": mfaToken})

			case "getTransactions":
				transactions, err := GetTransactions(message.UserData.Address)
				if err != nil {
					slog.Error("Error getting transactions", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				conn.WriteJSON(map[string]interface{}{"status": "success", "transactions": transactions})

			case "addDevice":
				if message.UserData.PoCContribution.Computations == 0 {
					slog.Error("No computational power provided")
					conn.WriteJSON(map[string]string{"status": "error", "message": "No computational power provided"})
					continue
				}
				if len(message.DeviceID) > 64 {
					slog.Error("Invalid deviceID length", "deviceID", message.DeviceID)
					conn.WriteJSON(map[string]string{"status": "error", "message": "DeviceID too long"})
					continue
				}
				if err := state.AddDevice(message.UserData.Address, message.DeviceID); err != nil && err.Error() != "device already added" {
					slog.Error("Error adding device", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				userData, err := GetData(message.UserData.Address, message.DeviceID)
				if err != nil {
					slog.Error("Error getting user data for device", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				if err := updateUserData(message.UserData.Address, message.DeviceID, userData, message.UserData.PoCContribution.Computations, float64(message.UserData.PoCContribution.Storage), float64(message.UserData.PoCContribution.Bandwidth), message.UserData.PoCContribution.Uptime, message.UserData.PoCContribution.EcoActions, 0, 0); err != nil {
					slog.Error("Error updating user data for device", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				slog.Info("Device power clustered", "address", message.UserData.Address, "deviceID", message.DeviceID, "computations", message.UserData.PoCContribution.Computations)
				conn.WriteJSON(map[string]interface{}{"status": "success", "message": "Device power clustered", "mfaToken": mfaToken})
				broadcastUpdate(message.UserData.Address, map[string]interface{}{
					"action": "deviceUpdated",
					"user":   userData,
				})

			case "getDevices":
				devices, err := state.GetDevices(message.UserData.Address)
				if err != nil {
					slog.Error("Error getting devices", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				conn.WriteJSON(map[string]interface{}{"status": "success", "devices": devices})

			case "getEnergyUsage":
				userData, err := GetData(message.UserData.Address, message.DeviceID)
				if err != nil {
					slog.Error("Error getting energy usage", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				loadLimits.Lock()
				limit := loadLimits.m[message.UserData.Address+":"+message.DeviceID]
				loadLimits.Unlock()
				powerData.Lock()
				power := powerData.m[message.UserData.Address+":"+message.DeviceID]
				powerData.Unlock()
				conn.WriteJSON(map[string]interface{}{
					"status":       "success",
					"energyUsage":  userData.PoCContribution.Computations,
					"tokensEarned": userData.Balance,
					"loadLimit":    limit,
					"cpuPercent":   power.CPUPercent,
					"memoryMB":     power.MemoryMB,
					"storage":      power.Storage,
					"bandwidth":    power.Bandwidth,
					"uptime":       power.Uptime,
					"ecoActions":   power.EcoActions,
					"reputation":   userData.Reputation.Score,
				})

			case "getAllEnergyUsage":
				devices, err := state.GetDevices(message.UserData.Address)
				if err != nil {
					slog.Error("Error getting devices for energy usage", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				var totalComputations uint64
				energyData := make(map[string]uint64)
				limitsData := make(map[string]uint64)
				cpuData := make(map[string]float64)
				memoryData := make(map[string]float64)
				storageData := make(map[string]uint64)
				bandwidthData := make(map[string]uint64)
				uptimeData := make(map[string]uint64)
				ecoActionsData := make(map[string]uint64)
				reputationData := make(map[string]float64)
				for _, deviceID := range devices {
					userData, err := GetData(message.UserData.Address, deviceID)
					if err != nil {
						slog.Error("Error getting user data for device", "deviceID", deviceID, "error", err)
						continue
					}
					totalComputations += userData.PoCContribution.Computations
					energyData[deviceID] = userData.PoCContribution.Computations
					loadLimits.Lock()
					limit := loadLimits.m[message.UserData.Address+":"+deviceID]
					if limit == 0 {
						limit = 100000
					}
					limitsData[deviceID] = limit
					loadLimits.Unlock()
					powerData.Lock()
					power := powerData.m[message.UserData.Address+":"+deviceID]
					powerData.Unlock()
					cpuData[deviceID] = power.CPUPercent
					memoryData[deviceID] = power.MemoryMB
					storageData[deviceID] = power.Storage
					bandwidthData[deviceID] = power.Bandwidth
					uptimeData[deviceID] = power.Uptime
					ecoActionsData[deviceID] = power.EcoActions
					reputationData[deviceID] = userData.Reputation.Score
				}
				conn.WriteJSON(map[string]interface{}{
					"status":              "success",
					"totalEnergyUsage":    totalComputations,
					"energyPerDevice":     energyData,
					"loadLimits":          limitsData,
					"cpuPerDevice":        cpuData,
					"memoryPerDevice":     memoryData,
					"storagePerDevice":    storageData,
					"bandwidthPerDevice":  bandwidthData,
					"uptimePerDevice":     uptimeData,
					"ecoActionsPerDevice": ecoActionsData,
					"reputationPerDevice": reputationData,
					"tokensEarned":        state.GetTokens(message.UserData.Address),
				})

			case "setLoadLimit":
				if len(message.DeviceID) > 64 {
					slog.Error("Invalid deviceID length", "deviceID", message.DeviceID)
					conn.WriteJSON(map[string]string{"status": "error", "message": "DeviceID too long"})
					continue
				}
				if message.LoadLimit == 0 {
					slog.Error("Invalid load limit")
					conn.WriteJSON(map[string]string{"status": "error", "message": "Invalid load limit"})
					continue
				}
				loadLimits.Lock()
				loadLimits.m[message.UserData.Address+":"+message.DeviceID] = message.LoadLimit
				loadLimits.Unlock()
				slog.Info("Load limit set", "address", message.UserData.Address, "deviceID", message.DeviceID, "limit", message.LoadLimit)
				conn.WriteJSON(map[string]interface{}{"status": "success", "message": "Load limit set", "mfaToken": mfaToken})

			case "removeDevice":
				if err := state.RemoveDevice(message.UserData.Address, message.DeviceID); err != nil {
					slog.Error("Error removing device", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				slog.Info("Device removed", "address", message.UserData.Address, "deviceID", message.DeviceID)
				conn.WriteJSON(map[string]interface{}{"status": "success", "message": "Device removed", "mfaToken": mfaToken})

			case "getNodes":
				nodes := state.GetNodes()
				conn.WriteJSON(map[string]interface{}{"status": "success", "nodes": nodes})

			case "getTokens":
				tokens := state.GetTokens(message.UserData.Address)
				conn.WriteJSON(map[string]interface{}{"status": "success", "tokens": tokens})

			case "getTrees":
				trees, err := GetTreesPlanted()
				if err != nil {
					slog.Error("Error getting trees", "error", err)
					conn.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
					continue
				}
				conn.WriteJSON(map[string]interface{}{"status": "success", "trees": trees})

			default:
				slog.Error("Unknown action", "action", message.Action)
				conn.WriteJSON(map[string]string{"status": "error", "message": "Unknown action"})
			}
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
	resp := map[string]interface{}{"tokens": tokens}
	json.NewEncoder(w).Encode(resp)
}
