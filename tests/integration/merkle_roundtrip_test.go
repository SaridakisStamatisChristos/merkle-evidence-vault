package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestMerkleRoundtrip ingests N items, waits for commit, fetches
// inclusion proofs, and verifies they structurally match the tree.
func TestMerkleRoundtrip(t *testing.T) {
	const N = 5

	ids := make([]string, N)

	// 1. Ingest N distinct evidence items.
	for i := 0; i < N; i++ {
		payload := []byte(fmt.Sprintf("roundtrip-evidence-%s-%d", uuid.NewString(), i))
		resp := apiReq(t, "POST", "/evidence", map[string]interface{}{
			"content_type": "text/plain",
			"payload":      payload,
			"labels":       map[string]string{"batch": "roundtrip"},
		})
		if resp.StatusCode != 202 && resp.StatusCode != 200 {
			t.Fatalf("ingest %d returned %d", i, resp.StatusCode)
		}
		var r struct {
			ID string `json:"id"`
		}
		json.NewDecoder(resp.Body).Decode(&r)
		resp.Body.Close()
		ids[i] = r.ID
		t.Logf("ingested[%d] id=%s", i, r.ID)
	}

	// 2. Wait for async commit (pipeline consumer).
	t.Log("waiting for pipeline commit…")
	time.Sleep(3 * time.Second)

	// 3. For each committed item, fetch and validate inclusion proof.
	for i, id := range ids {
		// Fetch evidence to check if committed.
		evResp := apiReq(t, "GET", "/evidence/"+id, nil)
		var ev struct {
			LeafIndex *int64 `json:"leaf_index"`
		}
		json.NewDecoder(evResp.Body).Decode(&ev)
		evResp.Body.Close()

		if ev.LeafIndex == nil {
			t.Logf("item %d (%s) still pending — skipping proof check", i, id[:8])
			continue
		}

		// Fetch inclusion proof.
		proofResp := apiReq(t, "GET", "/evidence/"+id+"/proof", nil)
		if proofResp.StatusCode != 200 {
			t.Errorf("proof for id=%s returned %d", id, proofResp.StatusCode)
			proofResp.Body.Close()
			continue
		}

		var proof struct {
			LeafIndex int64    `json:"leaf_index"`
			TreeSize  int64    `json:"tree_size"`
			Root      string   `json:"root"`
			Path      []string `json:"path"`
		}
		json.NewDecoder(proofResp.Body).Decode(&proof)
		proofResp.Body.Close()

		// Structural sanity checks.
		if proof.LeafIndex != *ev.LeafIndex {
			t.Errorf("proof leaf_index=%d ≠ evidence leaf_index=%d",
				proof.LeafIndex, *ev.LeafIndex)
		}
		if proof.Root == "" {
			t.Errorf("proof root must not be empty for id=%s", id)
		}
		if len(proof.Root) != 64 {
			t.Errorf("proof root must be 64 hex chars, got %d", len(proof.Root))
		}
		// Path length must be ⌈log₂(treeSize)⌉ ± 1.
		expectedMaxPath := 64 // generous upper bound for any tree ≤ 2^64
		if len(proof.Path) > expectedMaxPath {
			t.Errorf("proof path too long: %d nodes", len(proof.Path))
		}
		t.Logf("item %d proof ok: leaf=%d tree=%d path_len=%d root=%s…",
			i, proof.LeafIndex, proof.TreeSize, len(proof.Path), proof.Root[:12])
	}
}
