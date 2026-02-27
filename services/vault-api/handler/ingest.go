package handler

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SaridakisStamatisChristos/vault-api/middleware"
	"github.com/SaridakisStamatisChristos/vault-api/store"
	"github.com/google/uuid"
)

type IngestHandler struct{}

func NewIngestHandler() *IngestHandler { return &IngestHandler{} }

type evidenceRecord struct {
	ID        string `json:"id"`
	LeafIndex *int64 `json:"leaf_index,omitempty"`
}

type auditEntry struct {
	ID        string    `json:"id"`
	Actor     string    `json:"actor"`
	Timestamp time.Time `json:"timestamp"`
}

type checkpointPayload struct {
	TreeSize int64  `json:"tree_size"`
	RootHash string `json:"root_hash"`
}

type checkpointResponse struct {
	TreeSize  int64  `json:"tree_size"`
	RootHash  string `json:"root_hash"`
	Signature string `json:"signature"`
	KeyRef    string `json:"key_ref,omitempty"`
}

var (
	mu sync.Mutex
	// keep old globals for memory fallback compatibility; store package
	// will be used when initialized.
	storeMap                = map[string]*evidenceRecord{}
	next              int64 = 0
	audits                  = []auditEntry{}
	checkpointHistory       = map[int64]checkpointResponse{}
	checkpointOrder         = []int64{}
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
	storeMap[id] = &evidenceRecord{ID: id, LeafIndex: nil}
	actor := middleware.SubjectFromContext(r.Context())
	audits = append(audits, auditEntry{ID: id, Actor: actor, Timestamp: time.Now()})
	mu.Unlock()

	if s := store.Current(); s != nil {
		_ = s.SaveEvidence(r.Context(), id)
		_ = s.SaveAudit(r.Context(), store.AuditEntry{ID: "", ResourceID: id, Actor: actor, Timestamp: time.Now()})
	}

	resp := map[string]interface{}{"id": id, "content_hash": "", "status": "pending"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(202)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *IngestHandler) GetAudit(w http.ResponseWriter, r *http.Request) {
	roles := middleware.RolesFromContext(r.Context())
	allowed := false
	for _, rr := range roles {
		if rr == "auditor" {
			allowed = true
			break
		}
	}
	if !allowed {
		w.WriteHeader(403)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	var entries []map[string]interface{}
	for _, a := range audits {
		entries = append(entries, map[string]interface{}{"action": "ingest", "resource_id": a.ID})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"entries": entries})
}

func (h *IngestHandler) GetCheckpointsLatest(w http.ResponseWriter, r *http.Request) {
	cp, status := buildLatestCheckpoint(r.Context())
	if status != http.StatusOK {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cp)
}

// GetCheckpointsHistory returns known checkpoints (latest first).
func (h *IngestHandler) GetCheckpointsHistory(w http.ResponseWriter, r *http.Request) {
	if !hasCheckpointAccess(r.Context()) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	_, _ = buildLatestCheckpoint(r.Context())

	mu.Lock()
	defer mu.Unlock()
	entries := make([]checkpointResponse, 0, len(checkpointOrder))
	for i := len(checkpointOrder) - 1; i >= 0; i-- {
		ts := checkpointOrder[i]
		entries = append(entries, checkpointHistory[ts])
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"entries": entries})
}

// VerifyLatestCheckpoint verifies the latest checkpoint signature against
// CHECKPOINT_VERIFY_PUBLIC_KEY_B64 and returns a verification verdict.
func (h *IngestHandler) VerifyLatestCheckpoint(w http.ResponseWriter, r *http.Request) {
	cp, status := buildLatestCheckpoint(r.Context())
	if status != http.StatusOK {
		w.WriteHeader(status)
		return
	}
	verifyCheckpointResponse(w, *cp)
}

// VerifyCheckpointByTreeSize verifies a specific checkpoint by tree size.
func (h *IngestHandler) VerifyCheckpointByTreeSize(w http.ResponseWriter, r *http.Request) {
	if !hasCheckpointAccess(r.Context()) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	prefix := "/api/v1/checkpoints/"
	path := r.URL.Path
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, "/verify") {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	part := strings.TrimSuffix(strings.TrimPrefix(path, prefix), "/verify")
	part = strings.Trim(part, "/")
	treeSize, err := strconv.ParseInt(part, 10, 64)
	if err != nil || treeSize <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	mu.Lock()
	cp, ok := checkpointHistory[treeSize]
	mu.Unlock()
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	verifyCheckpointResponse(w, cp)
}

func buildLatestCheckpoint(ctx context.Context) (*checkpointResponse, int) {
	if !hasCheckpointAccess(ctx) {
		return nil, http.StatusForbidden
	}

	var maxLeaf int64 = -1
	mu.Lock()
	for _, rec := range storeMap {
		if rec.LeafIndex != nil && *rec.LeafIndex > maxLeaf {
			maxLeaf = *rec.LeafIndex
		}
	}
	mu.Unlock()

	treeSize := int64(0)
	if maxLeaf >= 0 {
		treeSize = maxLeaf + 1
	}
	if treeSize == 0 {
		return nil, http.StatusNotFound
	}

	mu.Lock()
	if cp, exists := checkpointHistory[treeSize]; exists {
		mu.Unlock()
		return &cp, http.StatusOK
	}
	mu.Unlock()

	root := strings.Repeat("0", 64)
	payload := checkpointPayload{TreeSize: treeSize, RootHash: root}
	payloadBytes, _ := json.Marshal(payload)
	signature := strings.Repeat("a", 64)
	keyRef := "local:dev-default"

	if svc := os.Getenv("CHECKPOINT_SIGNING_URL"); svc != "" {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Post(svc, "application/json", strings.NewReader(string(payloadBytes)))
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var got map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&got); err == nil {
					if s, ok := got["signature"]; ok && s != "" {
						signature = s
					}
					if kr, ok := got["key_ref"]; ok && kr != "" {
						keyRef = kr
					}
				}
			}
		}
	}

	cp := checkpointResponse{TreeSize: treeSize, RootHash: root, Signature: signature, KeyRef: keyRef}
	mu.Lock()
	if _, exists := checkpointHistory[treeSize]; !exists {
		checkpointOrder = append(checkpointOrder, treeSize)
	}
	checkpointHistory[treeSize] = cp
	mu.Unlock()
	return &cp, http.StatusOK
}

func hasCheckpointAccess(ctx context.Context) bool {
	roles := middleware.RolesFromContext(ctx)
	for _, rr := range roles {
		if rr == "auditor" || rr == "ingester" {
			return true
		}
	}
	return false
}

func verifyCheckpointResponse(w http.ResponseWriter, cp checkpointResponse) {
	pubB64 := strings.TrimSpace(os.Getenv("CHECKPOINT_VERIFY_PUBLIC_KEY_B64"))
	if pubB64 == "" {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"verified": false, "reason": "missing CHECKPOINT_VERIFY_PUBLIC_KEY_B64"})
		return
	}
	pubRaw, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil || len(pubRaw) != ed25519.PublicKeySize {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"verified": false, "reason": "invalid CHECKPOINT_VERIFY_PUBLIC_KEY_B64"})
		return
	}
	payload, err := json.Marshal(checkpointPayload{TreeSize: cp.TreeSize, RootHash: cp.RootHash})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sig, err := base64.StdEncoding.DecodeString(cp.Signature)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"verified": false, "reason": "checkpoint signature is not base64", "tree_size": cp.TreeSize, "root_hash": cp.RootHash, "key_ref": cp.KeyRef})
		return
	}
	verified := ed25519.Verify(ed25519.PublicKey(pubRaw), payload, sig)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"verified":  verified,
		"tree_size": cp.TreeSize,
		"root_hash": cp.RootHash,
		"signature": cp.Signature,
		"key_ref":   cp.KeyRef,
	})
}

func (h *IngestHandler) GetEvidence(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/v1/evidence/"):]
	if s := store.Current(); s != nil {
		ev, err := s.GetEvidence(r.Context(), id)
		if err != nil {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"leaf_index": ev.LeafIndex})
		return
	}
	mu.Lock()
	rec, ok := storeMap[id]
	mu.Unlock()
	if !ok {
		w.WriteHeader(404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"leaf_index": rec.LeafIndex})
}

func (h *IngestHandler) GetProof(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/v1/evidence/"):]
	if len(id) > 6 && id[len(id)-6:] == "/proof" {
		id = id[:len(id)-6]
	}
	mu.Lock()
	rec, ok := storeMap[id]
	mu.Unlock()
	if !ok || rec.LeafIndex == nil {
		w.WriteHeader(404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	root := strings.Repeat("0", 64)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"leaf_index": *rec.LeafIndex, "tree_size": *rec.LeafIndex + 1, "root": root, "path": []string{}})
}

func StartCommitter(period time.Duration) {
	go func() {
		for {
			time.Sleep(period)
			if s := store.Current(); s != nil {
				if ev, err := s.AssignNextPendingLeaf(context.Background()); err == nil && ev != nil {
					mu.Lock()
					if m, ok := storeMap[ev.ID]; ok {
						m.LeafIndex = ev.LeafIndex
					}
					mu.Unlock()
					continue
				}
			}
			mu.Lock()
			for _, rec := range storeMap {
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
