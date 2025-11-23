package db

import (
	"fmt"
	"os"
	"testing"
)

func TestDBClient(t *testing.T) {
	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "badger-test")
	if err != nil {
		t.Fatal(err)
	}
	//defer os.RemoveAll(tmpDir)

	client := NewDBClient()

	// Test Open
	err = client.Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer client.Close()

	// Test SetValue
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		val := []byte(fmt.Sprintf("test-value-%d", i))
		err = client.SetValue(key, val, 0)
		if err != nil {
			t.Errorf("Failed to set value: %v", err)
		}
	}
	key := "test-key"
	val := []byte("test-value")
	err = client.SetValue(key, val, 0)
	if err != nil {
		t.Errorf("Failed to set value: %v", err)
	}

	// Test GetValue
	got, err := client.GetValue(key)
	if err != nil {
		t.Errorf("Failed to get value: %v", err)
	}
	if string(got) != string(val) {
		t.Errorf("Expected %s, got %s", val, got)
	}

	// Test ListKeys
	// opts := ListKeysOptions{
	// 	Prefix: "",
	// 	Mode:   "prefix",
	// 	Limit:  10,
	// }
	// keys, _, err := client.ListKeys(opts)
	// if err != nil {
	// 	t.Errorf("Failed to list keys: %v", err)
	// }
	// if len(keys) != 1 {
	// 	t.Errorf("Expected 1 key, got %d", len(keys))
	// }
	// if keys[0].Key != key {
	// 	t.Errorf("Expected key %s, got %s", key, keys[0].Key)
	// }

	// // Test DeleteKey
	// err = client.DeleteKey(key)
	// if err != nil {
	// 	t.Errorf("Failed to delete key: %v", err)
	// }

	// // Verify deletion
	// _, err = client.GetValue(key)
	// if err == nil {
	// 	t.Error("Expected error after deletion, got nil")
	// }
}
