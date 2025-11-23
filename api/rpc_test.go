package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"

	"badger_explorer_core/db"
)

func TestAPIHandler(t *testing.T) {
	// Setup Temp DB
	tmpDir, err := os.MkdirTemp("", "badger-api-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Setup Client & Handler
	client := db.NewDBClient()
	var outBuf bytes.Buffer
	handler := NewHandler(client, &outBuf)

	// Helper to send request and get response
	sendRequest := func(req Request) Response {
		reqBytes, _ := json.Marshal(req)
		handler.handleLine(reqBytes)

		// Read response from outBuf
		line, err := outBuf.ReadBytes('\n')
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		var resp Response
		if err := json.Unmarshal(line, &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		return resp
	}

	// 1. Open DB
	openParams, _ := json.Marshal(OpenDBParams{Path: tmpDir})
	resp := sendRequest(Request{ID: "1", Type: TypeOpenDB, Params: openParams})
	if resp.Error != nil {
		t.Fatalf("OpenDB failed: %v", resp.Error)
	}

	// 2. Put Value (Small - using chunk logic for now as per implementation)
	// Note: Our implementation expects put_value -> put_chunk -> put_commit sequence for everything currently.
	key := "test-key"
	val := "Hello World"
	valBytes := []byte(val)

	// 2-1. Put Value Init
	putParams, _ := json.Marshal(PutValueParams{Key: key, ValueLength: len(valBytes), TTL: 0})
	resp = sendRequest(Request{ID: "2", Type: TypePutValue, Params: putParams})
	if resp.Error != nil {
		t.Fatalf("PutValue init failed: %v", resp.Error)
	}

	// 2-2. Put Chunk (Base64)
	// In test, we can just use raw string if we didn't enforce base64 in test helper,
	// but the handler expects base64 in Data field.
	encodedVal := base64.StdEncoding.EncodeToString(valBytes)

	chunkParams, _ := json.Marshal(PutChunkParams{ID: "2", ChunkIndex: 0, Data: encodedVal})
	resp = sendRequest(Request{ID: "3", Type: TypePutChunk, Params: chunkParams})
	if resp.Error != nil {
		t.Fatalf("PutChunk failed: %v", resp.Error)
	}

	// 2-3. Put Commit
	commitParams, _ := json.Marshal(PutCommitParams{ID: "2", Key: key, TTL: 0})
	resp = sendRequest(Request{ID: "4", Type: TypePutCommit, Params: commitParams})
	if resp.Error != nil {
		t.Fatalf("PutCommit failed: %v", resp.Error)
	}

	// 3. Get Value
	getParams, _ := json.Marshal(GetValueParams{Key: key})
	resp = sendRequest(Request{ID: "5", Type: TypeGetValue, Params: getParams})
	if resp.Error != nil {
		t.Fatalf("GetValue failed: %v", resp.Error)
	}

	// Result is map[string]interface{} when unmarshaled to interface{}, need to cast or re-marshal
	// But we know the structure.
	resultBytes, _ := json.Marshal(resp.Result)
	var getResult GetValueResult
	json.Unmarshal(resultBytes, &getResult)

	if getResult.Value != encodedVal {
		t.Errorf("Expected value %s, got %s", encodedVal, getResult.Value)
	}

	// 4. List Keys
	listParams, _ := json.Marshal(ListKeysParams{Prefix: "", Limit: 10})
	resp = sendRequest(Request{ID: "6", Type: TypeListKeys, Params: listParams})
	if resp.Error != nil {
		t.Fatalf("ListKeys failed: %v", resp.Error)
	}

	// 5. Delete Key
	delParams, _ := json.Marshal(DeleteKeyParams{Key: key})
	resp = sendRequest(Request{ID: "7", Type: TypeDeleteKey, Params: delParams})
	if resp.Error != nil {
		t.Fatalf("DeleteKey failed: %v", resp.Error)
	}

	// 6. Close DB
	resp = sendRequest(Request{ID: "8", Type: TypeCloseDB, Params: nil})
	if resp.Error != nil {
		t.Fatalf("CloseDB failed: %v", resp.Error)
	}
}
