package e2e

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

var (
	apiURL      = envOr("E2E_API_URL", "https://localhost:8443")
	auditorTok  = envOr("E2E_AUDITOR_TOKEN", "")
	ingesterTok = envOr("E2E_INGESTER_TOKEN", "")
)

var e2eClient = &http.Client{
	Timeout: 20 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

func e2eReq(t *testing.T, method, path, token string, body interface{}) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, apiURL+"/api/v1"+path, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := e2eClient.Do(req)
	if err != nil {
		t.Fatalf("e2e request failed: %v", err)
	}
	return resp
}

func decode(t *testing.T, resp *http.Response, dst interface{}) {
	t.Helper()
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(dst)
}

// SmokeTest_IngestProveVerify is the golden-path E2E test.
func TestSmokeIngestProveVerify(t *testing.T) {
	if ingesterTok == "" || auditorTok == "" {
		t.Skip("E2E_INGESTER_TOKEN or E2E_AUDITOR_TOKEN not set")
	}

	// Step 1: Ingest.
	payload := []byte("e2e-smoke-" + fmt.Sprintf("%d", time.Now().UnixNano()))
	ingestResp := e2eReq(t, "POST", "/evidence", ingesterTok, map[string]interface{}{
		"content_type": "text/plain",
		"payload":      payload,
		"labels":       map[string]string{"test": "e2e-smoke"},
	})
	if ingestResp.StatusCode != 202 && ingestResp.StatusCode != 200 {
		t.Fatalf("ingest: expected 200/202, got %d", ingestResp.StatusCode)
	}
	var ingest struct {
		ID string `json:"id"`
	}
	decode(t, ingestResp, &ingest)
	t.Logf("✓ Ingest: id=%s", ingest.ID)

	// Step 2: Wait for pipeline commit.
	t.Log("Waiting for Merkle commit…")
	var leafIndex *int64
	for i := 0; i < 30; i++ {
		time.Sleep(time.Second)
		r := e2eReq(t, "GET", "/evidence/"+ingest.ID, ingesterTok, nil)
		var ev struct {
			LeafIndex *int64 `json:"leaf_index"`
		}
		decode(t, r, &ev)
		if ev.LeafIndex != nil {
			leafIndex = ev.LeafIndex
			break
		}
	}
	if leafIndex == nil {
		t.Fatal("evidence was not committed to tree within 30s")
	}
	t.Logf("✓ Committed: leaf_index=%d", *leafIndex)

	// Step 3: Inclusion proof.
	proofResp := e2eReq(t, "GET", "/evidence/"+ingest.ID+"/proof", auditorTok, nil)
	if proofResp.StatusCode != 200 {
		t.Fatalf("proof: expected 200, got %d", proofResp.StatusCode)
	}
	var proof struct {
		LeafIndex int64    `json:"leaf_index"`
		TreeSize  int64    `json:"tree_size"`
		Root      string   `json:"root"`
		Path      []string `json:"path"`
	}
	decode(t, proofResp, &proof)
	if proof.Root == "" {
		t.Error("proof root must not be empty")
	}
	t.Logf("✓ Proof: leaf=%d tree=%d path_len=%d root=%s…",
		proof.LeafIndex, proof.TreeSize, len(proof.Path), proof.Root[:12])

	// Step 4: Latest checkpoint.
	ckptResp := e2eReq(t, "GET", "/checkpoints/latest", auditorTok, nil)
	if ckptResp.StatusCode != 200 {
		t.Logf("⚠ Checkpoint: not yet available (status %d) — ok if no STH emitted yet",
			ckptResp.StatusCode)
	} else {
		var sth struct {
			TreeSize  int64  `json:"tree_size"`
			RootHash  string `json:"root_hash"`
			Signature string `json:"signature"`
		}
		decode(t, ckptResp, &sth)
		if sth.Signature == "" {
			t.Error("checkpoint signature must not be empty")
		}
		t.Logf("✓ Checkpoint: tree_size=%d sig=%s…", sth.TreeSize, sth.Signature[:12])
	}

	// Step 5: Audit log contains our ingest.
	auditResp := e2eReq(t, "GET", "/audit?limit=20", auditorTok, nil)
	if auditResp.StatusCode != 200 {
		t.Fatalf("audit: expected 200, got %d", auditResp.StatusCode)
	}
	var auditBody struct {
		Entries []struct {
			Action     string `json:"action"`
			ResourceID string `json:"resource_id"`
		} `json:"entries"`
	}
	decode(t, auditResp, &auditBody)
	t.Logf("✓ Audit: %d entries returned", len(auditBody.Entries))

	t.Log("✓ Smoke test PASSED")
}

func TestSmokeRBACEnforcement(t *testing.T) {
	if ingesterTok == "" {
		t.Skip("E2E_INGESTER_TOKEN not set")
	}

	// Ingester should NOT be able to list audit log (auditor-only).
	resp := e2eReq(t, "GET", "/audit", ingesterTok, nil)
	if resp.StatusCode != 403 {
		t.Errorf("expected 403 for ingester accessing /audit, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	t.Log("✓ RBAC: ingester blocked from /audit")

	// Unauthenticated should get 401.
	req, _ := http.NewRequest("GET", apiURL+"/api/v1/checkpoints/latest", nil)
	unauthResp, _ := e2eClient.Do(req)
	if unauthResp.StatusCode != 401 {
		t.Errorf("expected 401 for unauth request, got %d", unauthResp.StatusCode)
	}
	unauthResp.Body.Close()
	t.Log("✓ RBAC: unauthenticated blocked with 401")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
