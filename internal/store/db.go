package store

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/dgraph-io/badger/v4"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/zalando/go-keyring"
)

var db *badger.DB

// Initialize 初始化存储
func Initialize() error {
	dbKey, err := keyring.Get(constant.AppName, constant.DbEncryptKey)
	if err != nil {
		dbKey = generateRandom32Bytes()
		err = keyring.Set(constant.AppName, constant.DbEncryptKey, dbKey)
		if err != nil {
			return err
		}
	}

	decodedKey, err := base64.StdEncoding.DecodeString(dbKey)

	if len(decodedKey) != 32 {
		return fmt.Errorf("db key length must be 32 bytes")
	}

	db, err = badger.Open(badger.DefaultOptions("./data").WithEncryptionKey(decodedKey).
		WithIndexCacheSize(1 << 20). // 1M缓存
		WithMemTableSize(64 << 20).  // 64M内存表
		WithMaxLevels(4),            // 4层
	)
	if err != nil {
		return err
	}

	slog.Info("Store opened", "path", "./data")

	return nil
}

// Shutdown 关闭存储
func Shutdown() error {
	if db != nil {
		err := db.Close()
		if err != nil {
			return err
		}
		slog.Info("Store closed")
	}
	return nil
}

func generateRandom32Bytes() string {
	// 生成随机32字节
	key := make([]byte, 32)
	rand.Read(key)
	return base64.StdEncoding.EncodeToString(key)
}
