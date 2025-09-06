package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"

	"github.com/ugorji/go/codec"
	"go.etcd.io/bbolt"
)

var cborHandle = &codec.CborHandle{}
var db *bbolt.DB

const (
	usersBucket        = "users"
	transactionsBucket = "transactions"
)

// init открывает базу данных при старте приложения
func init() {
	var err error
	db, err = bbolt.Open("triad.db", 0600, nil)
	if err != nil {
		log.Fatal("Error opening database:", err)
	}

	// Создаём бакеты, если их нет
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(usersBucket))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte(transactionsBucket))
		return err
	})
	if err != nil {
		log.Fatal("Error creating buckets:", err)
	}
}

func StoreData(address string, data UserData) error {
	var buf bytes.Buffer
	enc := codec.NewEncoder(&buf, cborHandle)
	if err := enc.Encode(data); err != nil {
		return err
	}

	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(usersBucket))
		return bucket.Put([]byte(address), buf.Bytes())
	})
}

func GetData(address string) (UserData, error) {
	var userData UserData

	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(usersBucket))
		data := bucket.Get([]byte(address))
		if data == nil {
			return errors.New("user not found")
		}

		dec := codec.NewDecoder(bytes.NewReader(data), cborHandle)
		return dec.Decode(&userData)
	})

	if err != nil {
		return UserData{}, err
	}
	return userData, nil
}

func UpdateData(address string, qli string, diff UserData) error {
	currentData, err := GetData(address)
	if err != nil {
		return err
	}

	updatedData := UserData{
		Address:         currentData.Address,
		Balance:         diff.Balance,
		PoCContribution: diff.PoCContribution,
	}
	return StoreData(address, updatedData)
}

func StoreTransaction(tx Transaction) error {
	var buf bytes.Buffer
	enc := codec.NewEncoder(&buf, cborHandle)
	if err := enc.Encode(tx); err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%s:%d", tx.From, tx.To, tx.Timestamp)
	return db.Update(func(txn *bbolt.Tx) error {
		bucket := txn.Bucket([]byte(transactionsBucket))
		return bucket.Put([]byte(key), buf.Bytes())
	})
}

func GetTransactions(address string) ([]Transaction, error) {
	var transactions []Transaction

	err := db.View(func(txn *bbolt.Tx) error {
		bucket := txn.Bucket([]byte(transactionsBucket))
		return bucket.ForEach(func(k, v []byte) error {
			var tx Transaction
			dec := codec.NewDecoder(bytes.NewReader(v), cborHandle)
			if err := dec.Decode(&tx); err != nil {
				return err
			}
			if tx.From == address || tx.To == address {
				transactions = append(transactions, tx)
			}
			return nil
		})
	})

	if err != nil {
		return nil, err
	}
	return transactions, nil
}
