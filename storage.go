package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
)

var db *leveldb.DB

func initDB() (*leveldb.DB, error) {
	var err error
	db, err = leveldb.OpenFile("./data.db", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %v", err)
	}
	return db, nil
}

func StoreData(address, deviceID string, data UserData) error {
	db, err := initDB()
	if err != nil {
		return err
	}
	defer db.Close()
	key := []byte(fmt.Sprintf("user:%s:%s", address, deviceID))
	userData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal user data: %v", err)
	}
	return db.Put(key, userData, nil)
}

func GetData(address, deviceID string) (UserData, error) {
	db, err := initDB()
	if err != nil {
		return UserData{}, err
	}
	defer db.Close()
	key := []byte(fmt.Sprintf("user:%s:%s", address, deviceID))
	data, err := db.Get(key, nil)
	if err != nil {
		return UserData{}, fmt.Errorf("get user data: %v", err)
	}
	var userData UserData
	if err := json.Unmarshal(data, &userData); err != nil {
		return UserData{}, fmt.Errorf("unmarshal user data: %v", err)
	}
	return userData, nil
}

func UpdateData(address, deviceID string, data UserData) error {
	return StoreData(address, deviceID, data)
}

func StoreTransaction(tx Transaction) error {
	db, err := initDB()
	if err != nil {
		return err
	}
	defer db.Close()
	lastTxHash, err := getLastTransactionHash(tx.From)
	if err != nil && err != leveldb.ErrNotFound {
		return fmt.Errorf("get last tx hash: %v", err)
	}
	tx.PrevHash = lastTxHash
	key := []byte(fmt.Sprintf("tx:%s:%d", tx.From, tx.Timestamp))
	data, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("marshal tx: %v", err)
	}
	if err := db.Put(key, data, nil); err != nil {
		return fmt.Errorf("store tx: %v", err)
	}
	hash := sha256.Sum256(data)
	lastHashKey := []byte(fmt.Sprintf("last_tx:%s", tx.From))
	return db.Put(lastHashKey, hash[:], nil)
}

func getLastTransactionHash(address string) (string, error) {
	db, err := initDB()
	if err != nil {
		return "", err
	}
	defer db.Close()
	key := []byte(fmt.Sprintf("last_tx:%s", address))
	data, err := db.Get(key, nil)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func GetTransactions(address string) ([]Transaction, error) {
	db, err := initDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	var transactions []Transaction
	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		key := string(iter.Key())
		if strings.HasPrefix(key, "tx:"+address+":") {
			var tx Transaction
			if err := json.Unmarshal(iter.Value(), &tx); err != nil {
				return nil, fmt.Errorf("unmarshal tx: %v", err)
			}
			transactions = append(transactions, tx)
		}
	}
	return transactions, iter.Error()
}

func UpdateTreesPlanted(computations uint64) error {
	db, err := initDB()
	if err != nil {
		return err
	}
	defer db.Close()
	key := []byte("trees")
	data, err := db.Get(key, nil)
	var trees uint64
	if err == nil {
		trees = binary.BigEndian.Uint64(data)
	}
	trees += computations
	data = make([]byte, 8)
	binary.BigEndian.PutUint64(data, trees)
	return db.Put(key, data, nil)
}

func GetTreesPlanted() (uint64, error) {
	db, err := initDB()
	if err != nil {
		return 0, err
	}
	defer db.Close()
	data, err := db.Get([]byte("trees"), nil)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(data), nil
}

func UpdateUptime(address, deviceID string, uptime uint64) error {
	db, err := initDB()
	if err != nil {
		return err
	}
	defer db.Close()
	userData, err := GetData(address, deviceID)
	if err != nil {
		return err
	}
	userData.PoCContribution.Uptime += uptime
	return UpdateData(address, deviceID, userData)
}

func UpdateStorage(address, deviceID string, storage uint64) error {
	db, err := initDB()
	if err != nil {
		return err
	}
	defer db.Close()
	userData, err := GetData(address, deviceID)
	if err != nil {
		return err
	}
	userData.PoCContribution.Storage += storage
	return UpdateData(address, deviceID, userData)
}

func UpdateBandwidth(address, deviceID string, bandwidth uint64) error {
	db, err := initDB()
	if err != nil {
		return err
	}
	defer db.Close()
	userData, err := GetData(address, deviceID)
	if err != nil {
		return err
	}
	userData.PoCContribution.Bandwidth += bandwidth
	return UpdateData(address, deviceID, userData)
}

func UpdateEcoActions(address, deviceID string, ecoActions uint64) error {
	db, err := initDB()
	if err != nil {
		return err
	}
	defer db.Close()
	userData, err := GetData(address, deviceID)
	if err != nil {
		return err
	}
	userData.PoCContribution.EcoActions += ecoActions
	return UpdateData(address, deviceID, userData)
}
