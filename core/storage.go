package core

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/syndtr/goleveldb/leveldb"
)

// StoreData stores user data in LevelDB.
func StoreData(address, deviceID string, data UserData) error {
	db, err := leveldb.OpenFile("data.db", nil)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %v", err)
	}
	key := fmt.Sprintf("%s:%s", address, deviceID)
	if err := db.Put([]byte(key), dataBytes, nil); err != nil {
		return fmt.Errorf("failed to store data: %v", err)
	}
	return nil
}

// GetData retrieves user data from LevelDB.
func GetData(address, deviceID string) (UserData, error) {
	db, err := leveldb.OpenFile("data.db", nil)
	if err != nil {
		return UserData{}, fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	key := fmt.Sprintf("%s:%s", address, deviceID)
	dataBytes, err := db.Get([]byte(key), nil)
	if err != nil {
		return UserData{}, fmt.Errorf("failed to get data: %v", err)
	}

	var data UserData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return UserData{}, fmt.Errorf("failed to unmarshal data: %v", err)
	}
	return data, nil
}

// StoreTransaction stores a transaction in LevelDB.
func StoreTransaction(tx Transaction) error {
	db, err := leveldb.OpenFile("data.db", nil)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	dataBytes, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %v", err)
	}
	key := fmt.Sprintf("tx:%s:%d", tx.From, tx.Nonce)
	if err := db.Put([]byte(key), dataBytes, nil); err != nil {
		return fmt.Errorf("failed to store transaction: %v", err)
	}
	return nil
}

// GetTransactions retrieves all transactions for a user.
func GetTransactions(address string) ([]Transaction, error) {
	db, err := leveldb.OpenFile("data.db", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
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
				continue
			}
			transactions = append(transactions, tx)
		}
	}
	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %v", err)
	}
	return transactions, nil
}
