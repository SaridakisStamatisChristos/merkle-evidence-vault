package handler

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type IngestHandler struct{}

func NewIngestHandler() *IngestHandler { return &IngestHandler{} }

type evidenceRecord struct {
	ID        string `json:"id"`
	LeafIndex *int64 `json:"leaf_index,omitempty"`
}

var (
	mu    sync.Mutex
	store       = map[string]*evidenceRecord{}
	next  int64 = 0
)

func (h *IngestHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContentType string `json:"content_type"`
		Payload     []byte `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(400)
		return
	}
	if len(req.Payload) == 0 {
		w.WriteHeader(400)
		return
	}

	id := uuid.NewString()
	mu.Lock()
	store[id] = &evidenceRecord{ID: id, LeafIndex: nil}
	mu.Unlock()

	resp := map[string]interface{}{"id": id, "content_hash": "", "status": "pending"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(202)
	json.NewEncoder(w).Encode(resp)
}

func (h *IngestHandler) GetEvidence(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/v1/evidence/"):]
	mu.Lock()
	rec, ok := store[id]
	mu.Unlock()
	if !ok {
		w.WriteHeader(404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"leaf_index": rec.LeafIndex})
}

func (h *IngestHandler) GetProof(w http.ResponseWriter, r *http.Request) {
	// Minimal proof stub — return 200 with empty path if committed
	id := r.URL.Path[len("/api/v1/evidence/"):]
	// strip trailing /proof
	if len(id) > 6 && id[len(id)-6:] == "/proof" {
		id = id[:len(id)-6]
	}
	mu.Lock()
	rec, ok := store[id]
	mu.Unlock()
	if !ok || rec.LeafIndex == nil {
		w.WriteHeader(404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"leaf_index": *rec.LeafIndex, "tree_size": *rec.LeafIndex + 1, "root": "", "path": []string{}})
}

// StartCommitter launches a background goroutine that marks pending records as
// committed by assigning a monotonically increasing leaf index every `period`.
func StartCommitter(period time.Duration) {
	go func() {
		for {
			time.Sleep(period)
			mu.Lock()
			for _, rec := range store {
				if rec.LeafIndex == nil {
					idx := next
					next++
					rec.LeafIndex = &idx
					break
				}
			}
			mu.Unlock()
		}
	}()
}
