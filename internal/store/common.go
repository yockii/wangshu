package store

import (
	"encoding/json"

	"github.com/dgraph-io/badger/v4"
	"github.com/rs/xid"
	"github.com/yockii/wangshu/internal/types"
)

func List[T types.BaseConfig](prefixKey string) (list []T, err error) {
	err = db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(prefixKey)
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			var t T
			if err := json.Unmarshal(val, &t); err != nil {
				return err
			}

			list = append(list, t)
		}
		return nil
	})
	return
}

func Get[T types.BaseConfig](prefixKey, id string) (t T, err error) {
	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(prefixKey + id))
		if err != nil {
			return err
		}
		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(val, &t); err != nil {
			return err
		}
		return nil
	})
	return
}

func Save[T types.BaseConfig](prefixKey string, t T) (err error) {
	if t.GetID() == "" {
		t.SetID(xid.New().String())
	}
	err = db.Update(func(txn *badger.Txn) error {
		key := []byte(prefixKey + t.GetID())
		val, err := json.Marshal(t)
		if err != nil {
			return err
		}
		return txn.Set(key, val)
	})
	return err
}

func Delete[T types.BaseConfig](prefixKey, id string) (err error) {
	err = db.Update(func(txn *badger.Txn) error {
		key := []byte(prefixKey + id)
		return txn.Delete(key)
	})
	return err
}
