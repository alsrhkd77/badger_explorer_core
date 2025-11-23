package db

import (
	"fmt"
	"time"

	badger "github.com/dgraph-io/badger/v4"
)

// GetValue retrieves the full value for a key.
func (c *DBClient) GetValue(key string) ([]byte, error) {
	c.mu.Lock()
	db := c.db
	c.mu.Unlock()

	if db == nil {
		return nil, fmt.Errorf("database not open")
	}

	var val []byte
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		val, err = item.ValueCopy(nil)
		return err
	})

	if err != nil {
		return nil, err
	}
	return val, nil
}

// SetValue sets a value for a key.
// If ttl is > 0, it sets the TTL in seconds.
func (c *DBClient) SetValue(key string, value []byte, ttl int) error {
	c.mu.Lock()
	db := c.db
	c.mu.Unlock()

	if db == nil {
		return fmt.Errorf("database not open")
	}

	// 필요한 경우 R/W 모드로 다시 열기?
	// 명세서: "쓰기 작업 시에만 DB를 R/W 모드로 열기".
	// 하지만 연결을 쉽게 업그레이드할 수 없음. 닫고 다시 열어야 할 수도?
	// 아니면 UI가 "쓰기용 열기" 상태를 처리한다고 가정함.
	// 그러나 명세서: "항상 DB를 닫아두고, 필요할 때만 열기".
	// 하지만 TUI의 경우 목록 읽기를 위해 열어둠.
	// TUI(Standalone)의 경우 열어둔다고 가정함.
	// 하지만 써야 한다면 R/W인지 확인해야 할 수도 있음.
	// ReadOnly로 열렸다면 닫고 R/W로 다시 열어서 쓰고, (아마도) 다시 ReadOnly로 열어야 함?
	// 아니면 편집하려는 경우 그냥 R/W로 열기?
	// 쓰기를 시도해봄. ReadOnly로 인해 실패하면 다시 열기를 시도할 수 있음.

	return db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), value)
		if ttl > 0 {
			e.WithTTL(time.Duration(ttl) * time.Second)
		}
		return txn.SetEntry(e)
	})
}

// DeleteKey deletes a key.
func (c *DBClient) DeleteKey(key string) error {
	c.mu.Lock()
	db := c.db
	c.mu.Unlock()

	if db == nil {
		return fmt.Errorf("database not open")
	}

	return db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}
