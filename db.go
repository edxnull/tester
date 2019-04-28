package main

import (
	"fmt"
	bolt "go.etcd.io/bbolt"
	"os"
)

var FILE_MODE_RW os.FileMode = 0600

func DBOpen() *bolt.DB {
	wdir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	db, err := bolt.Open(wdir+"/my.db", FILE_MODE_RW, nil)
	if err != nil {
		panic(err)
	}
	return db
}

func DBInit(db *bolt.DB, mk map[string]struct{}) error {
	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("TestWords"))
		if err != nil {
			return fmt.Errorf("Failed to create bucket: %v", err)
		}
		for k, _ := range mk {
			if bucket.Get([]byte(k)) == nil { // don't override key/val if exists
				err = bucket.Put([]byte(k), []byte(""))
				if err != nil {
					return fmt.Errorf("Failed to insert '%s': '%v'", k, err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("bbolt db.Update in DBInit failed '%v'", err)
	}
	return nil
}

func DBInsert(db *bolt.DB, k string) error {
	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("TestWords"))
		if err != nil {
			return fmt.Errorf("Failed to create bucket: %v", err)
		}
		if bucket.Get([]byte(k)) == nil { // don't override key/val if exists
			err = bucket.Put([]byte(k), []byte(""))
			if err != nil {
				return fmt.Errorf("Failed to insert '%s': '%v'", k, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("bbolt db.Update in DBInsert failed '%v'", err)
	}
	return nil
}

func DBView(db *bolt.DB, find string) (bool, error) {
	var result bool
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("TestWords"))
		if bucket == nil {
			return fmt.Errorf("Failed to find bucket")
		}
		if bucket.Get([]byte(find)) == nil {
			return fmt.Errorf("Failed to view '%s'", find)
		}
		result = true
		return nil
	})
	if err != nil {
		return result, fmt.Errorf("bbolt db.View in DBView failed '%v'", err)
	}
	return result, nil
}
