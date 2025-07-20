package main

import (
	"errors"
	"fmt"
	"github.com/dgraph-io/badger/v4"
	"log"
	"testing"
)

func TestHashAndPhotoId(t *testing.T) {
	dHash := PHash{hash: [8]uint8{1, 2, 3, 4, 5, 6, 7, 8}}
	err := SetDHashByUniqueId("12345678", dHash)
	if err != nil {
		log.Fatal(err)
	}
	newHash, err := GetDHashByUniqueId("12345678")
	if err != nil {
		log.Fatal(err)
	}
	if newHash != dHash {
		log.Fatal("hash error")
	}
	_, err = GetDHashByUniqueId("87654321")
	if !errors.Is(err, badger.ErrKeyNotFound) {
		log.Fatal(err)
	}

}

func TestMarsInfoGetSet(t *testing.T) {
	key := MarsInfoKey{
		ChatId: 12345678,
		Hash:   PHash{hash: [8]uint8{1, 2, 3, 4, 5, 6, 7, 8}},
	}
	value := MarsInfo{
		LastMsgId: 100,
		Count:     200,
		IsIgnored: true,
	}
	tx := db.NewTransaction(true)
	defer tx.Commit()
	err := key.SetInfo(tx, value)
	if err != nil {
		log.Fatal(err)
	}

	newValue, err := key.GetInfo(tx)
	if err != nil {
		log.Fatal(err)
	}
	if newValue != value {
		log.Fatalf("value error, newValue = %v, value = %v", newValue, value)
	}
	notKey := MarsInfoKey{
		ChatId: 0,
		Hash:   PHash{},
	}
	_, err = notKey.GetInfo(tx)
	if !errors.Is(err, badger.ErrKeyNotFound) {
		log.Fatal(err)
	}
}

func TestEncodeKey(t *testing.T) {
	key := MarsInfoKey{ChatId: 12345678, Hash: PHash{hash: [8]uint8{1, 2, 3, 4, 5, 6, 7, 8}}}
	b := encodeKeyWithType(key)
	fmt.Printf("%s\n", b)
}
