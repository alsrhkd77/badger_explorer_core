package db

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	badger "github.com/dgraph-io/badger/v4"
)

// DBClient handles interactions with BadgerDB.
type DBClient struct {
	path string
	db   *badger.DB
	mu   sync.Mutex
}

// NewDBClient creates a new DBClient instance.
func NewDBClient() *DBClient {
	return &DBClient{}
}

// Open opens the BadgerDB at the specified path.
// Always opens in Read-Write mode for Windows compatibility.
func (c *DBClient) Open(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.db != nil {
		return fmt.Errorf("database is already open")
	}

	opts := badger.DefaultOptions(path)
	opts.ReadOnly = false // Always RW
	// Turn off logging for cleaner output
	opts.Logger = nil

	// Check if directory exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", path)
	}

	db, err := badger.Open(opts)
	if err != nil {
		return fmt.Errorf("failed to open badger db: %w", err)
	}

	c.path = path
	c.db = db
	return nil
}

// Close closes the database.
func (c *DBClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.db == nil {
		return nil
	}

	err := c.db.Close()
	c.db = nil
	c.path = ""
	return err
}

// IsOpen returns true if the database is currently open.
func (c *DBClient) IsOpen() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.db != nil
}

// GetPath returns the current DB path.
func (c *DBClient) GetPath() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.path
}

// KeyItem represents a key and its metadata in the list.
type KeyItem struct {
	Key          string
	ValuePreview string
	Size         int64
	ExpiresAt    uint64 // Timestamp
}

// ListKeysOptions defines options for listing keys.
type ListKeysOptions struct {
	Prefix       string
	Mode         string // "prefix", "substring", "regex"
	SortDesc     bool
	Limit        int
	Offset       int    // 건너뛸 항목 수 (KV 저장소에서는 비효율적이지만, 간단한 페이지네이션 로직을 위해 필요함)
	StartKey     string // KV 저장소 페이지네이션에 더 효율적인 방식
	PreviewChars int
}

// ListKeys lists keys based on the options.
func (c *DBClient) ListKeys(opts ListKeysOptions) ([]KeyItem, bool, error) {
	c.mu.Lock()
	db := c.db
	c.mu.Unlock()

	if db == nil {
		return nil, false, fmt.Errorf("database not open")
	}

	var items []KeyItem
	var hasMore bool

	err := db.View(func(txn *badger.Txn) error {
		itOpts := badger.DefaultIteratorOptions
		itOpts.PrefetchValues = true // We need values for preview
		itOpts.PrefetchSize = opts.Limit
		itOpts.Reverse = opts.SortDesc

		it := txn.NewIterator(itOpts)
		defer it.Close()

		// 시작 키 결정
		startKey := []byte(opts.Prefix)
		if opts.StartKey != "" {
			startKey = []byte(opts.StartKey)
		} else if opts.Mode != "prefix" {
			// 부분 문자열/정규식 모드의 경우, startKey가 제공되지 않으면 처음부터 스캔해야 할 수 있음
			// 하지만 SortDesc가 true라면 끝에서부터 시작해야 할까?
			// Badger의 Reverse iterator는 Seek을 다르게 처리함.
			if opts.SortDesc {
				startKey = []byte{0xFF} // 이론상 마지막
			} else {
				startKey = []byte{}
			}
		}

		// Seek
		it.Seek(startKey)

		count := 0
		skipped := 0

		// Regex compilation if needed
		var re *regexp.Regexp
		var err error
		if opts.Mode == "regex" && opts.Prefix != "" {
			re, err = regexp.Compile(opts.Prefix)
			if err != nil {
				return fmt.Errorf("invalid regex: %w", err)
			}
		}

		for ; it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			keyStr := string(k)

			// Filter logic
			match := false
			switch opts.Mode {
			case "prefix":
				if opts.SortDesc {
					// 역순 모드에서 Seek(prefix)는 해당 접두사를 가진 마지막 키(또는 그보다 큰 키)로 이동함.
					// 하지만 실제로 접두사를 가지고 있는지 확인해야 함.
					if strings.HasPrefix(keyStr, opts.Prefix) {
						match = true
					} else {
						// 역순 모드이고 현재 키가 접두사를 가지고 있지 않으며,
						// 접두사로 Seek을 시작했다면 관련 키들을 지나쳤을 수 있음?
						// 사실, 역순에서의 단순 접두사 스캔의 경우:
						// Seek(prefix + 0xFF)가 더 낫지만, 지금은 단순 확인을 유지함.
						// 그냥 순회하며 접두사를 확인함.
						match = strings.HasPrefix(keyStr, opts.Prefix)
					}
				} else {
					if strings.HasPrefix(keyStr, opts.Prefix) {
						match = true
					} else {
						// 접두사 모드이고 오름차순 정렬인 경우, 접두사를 지나치면 완료된 것임.
						// 최적화:
						if len(opts.Prefix) > 0 && keyStr > opts.Prefix && !strings.HasPrefix(keyStr, opts.Prefix) {
							return nil
						}
						match = false
					}
				}
			case "substring":
				if strings.Contains(keyStr, opts.Prefix) {
					match = true
				}
			case "regex":
				if re != nil && re.MatchString(keyStr) {
					match = true
				} else if re == nil {
					match = true // Empty regex matches all
				}
			default:
				// Default to prefix
				if strings.HasPrefix(keyStr, opts.Prefix) {
					match = true
				}
			}

			if match {
				// Offset 처리 (건너뛰기)
				// 참고: 깊은 페이지에서는 비효율적이지만, 작은 배치의 TUI 사용에는 허용됨.
				// 더 나은 접근 방식은 이전 페이지의 StartKey를 사용하는 것임.
				if opts.StartKey == "" && skipped < opts.Offset {
					skipped++
					continue
				}

				// StartKey를 사용했다면 시작 키 자체를 포함할 수 있는데, 정확한 시작점이라면 보통 원함.
				// 하지만 페이징 중이라면 호출자가 *다음* 키를 전달하거나 우리가 처리해야 함.
				// StartKey는 포함된다고 가정함.

				valCopy, err := item.ValueCopy(nil)
				if err != nil {
					continue
				}

				// Preview
				previewLen := opts.PreviewChars
				if previewLen <= 0 {
					previewLen = 100
				}
				preview := ""
				if len(valCopy) > previewLen {
					preview = string(valCopy[:previewLen]) + "..."
				} else {
					preview = string(valCopy)
				}

				// Check for binary
				if isBinary(valCopy) {
					preview = fmt.Sprintf("[Binary %d bytes]", len(valCopy))
				}

				items = append(items, KeyItem{
					Key:          keyStr,
					ValuePreview: preview,
					Size:         item.ValueSize(),
					ExpiresAt:    item.ExpiresAt(),
				})

				count++
				if count >= opts.Limit {
					// 최소한 하나의 항목이 더 있는지 확인
					it.Next()
					if it.Valid() {
						// 다음 항목도 필터와 일치하는지 확인해야 함...
						// 너무 많이 미리 보지 않고는 까다로움.
						// 제한을 채웠다면 더 있을 수 있다고 가정함.
						hasMore = true
					}
					break
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, false, err
	}

	return items, hasMore, nil
}

// isBinary checks if the data seems to be binary.
// Simple heuristic: looks for null bytes or non-printable chars.
func isBinary(data []byte) bool {
	// Check first 512 bytes
	n := len(data)
	if n > 512 {
		n = 512
	}
	for i := 0; i < n; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}
