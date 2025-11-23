package api

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"badger_explorer_core/db"
)

// Request types
const (
	TypeOpenDB    = "open_db"
	TypeListKeys  = "list_keys"
	TypeGetValue  = "get_value"
	TypePutValue  = "put_value"
	TypePutChunk  = "put_chunk"
	TypePutCommit = "put_commit"
	TypeDeleteKey = "delete_key"
	TypeCloseDB   = "close_db"
)

// Request represents a JSON-RPC request.
type Request struct {
	ID     string          `json:"id"`
	Type   string          `json:"type"`
	Params json.RawMessage `json:"params"`
}

// Response represents a JSON-RPC response.
type Response struct {
	ID     string      `json:"id"`
	Type   string      `json:"type"`
	Result interface{} `json:"result,omitempty"`
	Error  *Error      `json:"error,omitempty"`
}

// Error represents an error in the response.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Handler handles API requests.
type Handler struct {
	dbClient *db.DBClient
	out      io.Writer
	mu       sync.Mutex

	// Chunking state
	chunkBuffer map[string][]byte // requestID -> data buffer
}

// NewHandler creates a new API handler.
func NewHandler(dbClient *db.DBClient, out io.Writer) *Handler {
	return &Handler{
		dbClient:    dbClient,
		out:         out,
		chunkBuffer: make(map[string][]byte),
	}
}

// Run starts reading from stdin and handling requests.
func (h *Handler) Run(in io.Reader) {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		go h.handleLine(line)
	}
}

func (h *Handler) handleLine(line []byte) {
	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		h.sendError(req.ID, 1003, "Invalid request format")
		return
	}

	var err error
	var result interface{}

	switch req.Type {
	case TypeOpenDB:
		result, err = h.handleOpenDB(req.Params)
	case TypeListKeys:
		result, err = h.handleListKeys(req.Params)
	case TypeGetValue:
		result, err = h.handleGetValue(req.Params)
	case TypePutValue:
		result, err = h.handlePutValue(req.ID, req.Params)
	case TypePutChunk:
		result, err = h.handlePutChunk(req.Params)
	case TypePutCommit:
		result, err = h.handlePutCommit(req.Params)
	case TypeDeleteKey:
		result, err = h.handleDeleteKey(req.Params)
	case TypeCloseDB:
		result, err = h.handleCloseDB()
	default:
		h.sendError(req.ID, 1000, "Unknown request type")
		return
	}

	if err != nil {
		h.sendError(req.ID, 1000, err.Error())
	} else {
		h.sendResponse(req.ID, req.Type+"_resp", result)
	}
}

func (h *Handler) sendResponse(id, typeStr string, result interface{}) {
	resp := Response{
		ID:     id,
		Type:   typeStr,
		Result: result,
	}
	h.writeJSON(resp)
}

func (h *Handler) sendError(id string, code int, msg string) {
	resp := Response{
		ID:   id,
		Type: "error",
		Error: &Error{
			Code:    code,
			Message: msg,
		},
	}
	h.writeJSON(resp)
}

func (h *Handler) writeJSON(v interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()

	bytes, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.out.Write(bytes)
	h.out.Write([]byte("\n"))
}

// --- Handlers ---

type OpenDBParams struct {
	Path     string `json:"path"`
	ReadOnly bool   `json:"readonly"`
}

func (h *Handler) handleOpenDB(params json.RawMessage) (interface{}, error) {
	var p OpenDBParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	err := h.dbClient.Open(p.Path)
	return nil, err
}

type ListKeysParams struct {
	Prefix string `json:"prefix"`
	Mode   string `json:"mode"`
	Sort   string `json:"sort"` // "asc", "desc"
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type ListKeysResult struct {
	Keys    []db.KeyItem `json:"keys"`
	HasMore bool         `json:"has_more"`
}

func (h *Handler) handleListKeys(params json.RawMessage) (interface{}, error) {
	var p ListKeysParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	opts := db.ListKeysOptions{
		Prefix:   p.Prefix,
		Mode:     p.Mode,
		SortDesc: p.Sort == "desc",
		Limit:    p.Limit,
		Offset:   p.Offset,
	}

	keys, hasMore, err := h.dbClient.ListKeys(opts)
	if err != nil {
		return nil, err
	}

	return ListKeysResult{Keys: keys, HasMore: hasMore}, nil
}

type GetValueParams struct {
	Key string `json:"key"`
}

type GetValueResult struct {
	Value string `json:"value"` // Base64 encoded
}

func (h *Handler) handleGetValue(params json.RawMessage) (interface{}, error) {
	var p GetValueParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	val, err := h.dbClient.GetValue(p.Key)
	if err != nil {
		return nil, err
	}

	return GetValueResult{Value: base64.StdEncoding.EncodeToString(val)}, nil
}

type PutValueParams struct {
	Key         string `json:"key"`
	ValueLength int    `json:"value_length"`
	TTL         int    `json:"ttl"`
}

func (h *Handler) handlePutValue(reqID string, params json.RawMessage) (interface{}, error) {
	var p PutValueParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	// If small enough, maybe value is included?
	// Spec says: "put_value" (large value chunking).
	// But what if it's small?
	// Let's assume standard put_value is just init for chunking OR
	// we can support a "Value" field in params for small puts.
	// The spec example shows "value_length" and then "put_chunk".
	// Let's support both: if "value" is present, do it. If not, init chunking.

	// For now, let's implement the chunking init as per spec example.
	h.mu.Lock()
	h.chunkBuffer[reqID] = make([]byte, 0, p.ValueLength)
	h.mu.Unlock()

	return nil, nil // Acknowledge init
}

type PutChunkParams struct {
	ID         string `json:"id"` // Original Request ID
	ChunkIndex int    `json:"chunk_index"`
	Data       string `json:"data"` // Base64
}

func (h *Handler) handlePutChunk(params json.RawMessage) (interface{}, error) {
	var p PutChunkParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	data, err := base64.StdEncoding.DecodeString(p.Data)
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	buf, ok := h.chunkBuffer[p.ID]
	if !ok {
		return nil, fmt.Errorf("unknown upload session: %s", p.ID)
	}

	// Append
	h.chunkBuffer[p.ID] = append(buf, data...)
	return nil, nil
}

type PutCommitParams struct {
	ID  string `json:"id"`
	Key string `json:"key"`
	TTL int    `json:"ttl"`
}

func (h *Handler) handlePutCommit(params json.RawMessage) (interface{}, error) {
	var p PutCommitParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	h.mu.Lock()
	buf, ok := h.chunkBuffer[p.ID]
	delete(h.chunkBuffer, p.ID)
	h.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("unknown upload session: %s", p.ID)
	}

	err := h.dbClient.SetValue(p.Key, buf, p.TTL)
	return nil, err
}

type DeleteKeyParams struct {
	Key string `json:"key"`
}

func (h *Handler) handleDeleteKey(params json.RawMessage) (interface{}, error) {
	var p DeleteKeyParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	err := h.dbClient.DeleteKey(p.Key)
	return nil, err
}

func (h *Handler) handleCloseDB() (interface{}, error) {
	err := h.dbClient.Close()
	return nil, err
}
