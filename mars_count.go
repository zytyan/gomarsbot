package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/dgraph-io/badger/v4"
	"github.com/fxamacker/cbor/v2"
	"github.com/puzpuzpuz/xsync/v4"
	"reflect"
	"sync"
)

type MarsInfoKey struct {
	ChatId int64 `cbor:"0,keyasint"`
	Hash   PHash `cbor:"1,keyasint"`
}

type MarsInfo struct {
	LastMsgId int64
	Count     int64
	// 控制是否要忽略该图片，如果忽略该图片，即使火星也不会返回消息
	IsIgnored bool
}
type TgImgIdKey struct {
	Id string `cbor:"0,keyasint"`
}

func encodeKeyWithType(obj any) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, 64))
	t := reflect.TypeOf(obj)
	buf.WriteString(t.Name())
	buf.WriteByte(',')
	err := cbor.NewEncoder(buf).Encode(obj)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func GetDHashByUniqueId(uid string) (PHash, error) {
	var dHash PHash
	e := db.View(func(tx *badger.Txn) error {
		key := encodeKeyWithType(TgImgIdKey{Id: uid})
		item, err := tx.Get(key)
		if err != nil {
			return err
		}
		err = item.Value(func(val []byte) error {
			if len(val) != 8 {
				return fmt.Errorf("get PHash from badger database len = %d", len(val))
			}
			copy(dHash.hash[:], val)
			return nil
		})
		return err
	})
	return dHash, e
}

func SetDHashByUniqueId(uid string, dHash PHash) error {
	e := db.Update(func(tx *badger.Txn) error {
		key := encodeKeyWithType(TgImgIdKey{Id: uid})
		return tx.Set(key, dHash.hash[:])
	})
	return e
}

func (m MarsInfoKey) toBadgerKey() []byte {
	return encodeKeyWithType(m)
}

func (m MarsInfoKey) GetInfo(tx *badger.Txn) (res MarsInfo, e error) {
	key := m.toBadgerKey()
	item, err := tx.Get(key)
	if errors.Is(err, badger.ErrKeyNotFound) {
		res = MarsInfo{
			LastMsgId: 0,
			Count:     0,
			IsIgnored: false,
		}
		return
	}
	if err != nil {
		return
	}
	err = item.Value(func(val []byte) error {
		return gob.NewDecoder(bytes.NewReader(val)).Decode(&res)
	})
	return

}

func (m MarsInfoKey) SetInfo(tx *badger.Txn, info MarsInfo) error {
	key := m.toBadgerKey()
	buf := bytes.NewBuffer(make([]byte, 0, 64))
	err := gob.NewEncoder(buf).Encode(info)
	if err != nil {
		return err
	}
	return tx.Set(key, buf.Bytes())
}

func (m MarsInfoKey) String() string {
	return fmt.Sprintf("MarsInfoKey{ChatId:%d,Hash:%x}", m.ChatId, m.Hash)
}

var intKeyLocker = xsync.NewMap[int64, *sync.Mutex]()

func GetLockByKey(key int64) *sync.Mutex {
	mu, _ := intKeyLocker.LoadOrCompute(key, func() (newValue *sync.Mutex, cancel bool) {
		return new(sync.Mutex), false
	})
	return mu
}
