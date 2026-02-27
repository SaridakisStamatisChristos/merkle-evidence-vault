package handler

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/SaridakisStamatisChristos/vault-api/middleware"
	"github.com/go-chi/chi/v5"
)

func TestCheckpointVerifyLatest(t *testing.T) {
	pub, _, signer := setupCheckpointSigner(t)
	defer signer.Close()

	os.Setenv("CHECKPOINT_SIGNING_URL", signer.URL)
	os.Setenv("CHECKPOINT_VERIFY_PUBLIC_KEY_B64", base64.StdEncoding.EncodeToString(pub))
	os.Setenv("ENABLE_TEST_JWT", "true")
	defer os.Unsetenv("CHECKPOINT_SIGNING_URL")
	defer os.Unsetenv("CHECKPOINT_VERIFY_PUBLIC_KEY_B64")
	defer os.Unsetenv("ENABLE_TEST_JWT")

	resetCheckpointState()
	mu.Lock()
	storeMap = map[string]*evidenceRecord{"abc": {ID: "abc", LeafIndex: func() *int64 { x := int64(1); return &x }()}}
	mu.Unlock()

	h := NewIngestHandler()
	r := chi.NewRouter()
	r.With(middleware.JWT).Get("/api/v1/checkpoints/latest/verify", h.VerifyLatestCheckpoint)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/checkpoints/latest/verify", nil)
	req.Header.Set("Authorization", "Bearer auditor-token")
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d; body=%s", rw.Code, rw.Body.String())
	}

	var got struct {
		Verified bool   `json:"verified"`
		KeyRef   string `json:"key_ref"`
	}
	if err := json.NewDecoder(rw.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.Verified {
		t.Fatalf("expected verified=true")
	}
	if got.KeyRef != "local-hsm-emulator:test-kid" {
		t.Fatalf("unexpected key_ref %q", got.KeyRef)
	}
}

func TestCheckpointHistoryAndVerifyByTreeSize(t *testing.T) {
	pub, _, signer := setupCheckpointSigner(t)
	defer signer.Close()

	os.Setenv("CHECKPOINT_SIGNING_URL", signer.URL)
	os.Setenv("CHECKPOINT_VERIFY_PUBLIC_KEY_B64", base64.StdEncoding.EncodeToString(pub))
	os.Setenv("ENABLE_TEST_JWT", "true")
	defer os.Unsetenv("CHECKPOINT_SIGNING_URL")
	defer os.Unsetenv("CHECKPOINT_VERIFY_PUBLIC_KEY_B64")
	defer os.Unsetenv("ENABLE_TEST_JWT")

	resetCheckpointState()
	mu.Lock()
	storeMap = map[string]*evidenceRecord{
		"one": {ID: "one", LeafIndex: func() *int64 { x := int64(0); return &x }()},
	}
	mu.Unlock()

	h := NewIngestHandler()
	r := chi.NewRouter()
	r.With(middleware.JWT).Get("/api/v1/checkpoints", h.GetCheckpointsHistory)
	r.With(middleware.JWT).Get("/api/v1/checkpoints/latest", h.GetCheckpointsLatest)
	r.With(middleware.JWT).Get("/api/v1/checkpoints/{treeSize}/verify", h.VerifyCheckpointByTreeSize)

	// materialize first checkpoint (tree_size=1)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/checkpoints/latest", nil)
	req.Header.Set("Authorization", "Bearer auditor-token")
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)
	if rw.Code != http.StatusOK {
		t.Fatalf("latest #1 expected 200 got %d", rw.Code)
	}

	// grow tree and materialize second checkpoint (tree_size=2)
	mu.Lock()
	storeMap["two"] = &evidenceRecord{ID: "two", LeafIndex: func() *int64 { x := int64(1); return &x }()}
	mu.Unlock()
	rw = httptest.NewRecorder()
	r.ServeHTTP(rw, req)
	if rw.Code != http.StatusOK {
		t.Fatalf("latest #2 expected 200 got %d", rw.Code)
	}

	// list history should include both checkpoints
	hReq := httptest.NewRequest(http.MethodGet, "/api/v1/checkpoints", nil)
	hReq.Header.Set("Authorization", "Bearer auditor-token")
	hRW := httptest.NewRecorder()
	r.ServeHTTP(hRW, hReq)
	if hRW.Code != http.StatusOK {
		t.Fatalf("history expected 200 got %d", hRW.Code)
	}
	var list struct {
		Entries []checkpointResponse `json:"entries"`
	}
	if err := json.NewDecoder(hRW.Body).Decode(&list); err != nil {
		t.Fatalf("decode history: %v", err)
	}
	if len(list.Entries) < 2 {
		t.Fatalf("expected at least 2 checkpoints got %d", len(list.Entries))
	}

	// verify checkpoint at tree size 1
	vReq := httptest.NewRequest(http.MethodGet, "/api/v1/checkpoints/1/verify", nil)
	vReq.Header.Set("Authorization", "Bearer auditor-token")
	vRW := httptest.NewRecorder()
	r.ServeHTTP(vRW, vReq)
	if vRW.Code != http.StatusOK {
		t.Fatalf("verify by tree size expected 200 got %d body=%s", vRW.Code, vRW.Body.String())
	}
	var ver struct {
		Verified bool `json:"verified"`
	}
	if err := json.NewDecoder(vRW.Body).Decode(&ver); err != nil {
		t.Fatalf("decode verify: %v", err)
	}
	if !ver.Verified {
		t.Fatalf("expected verified=true")
	}
}

func setupCheckpointSigner(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, *httptest.Server) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	signer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload checkpointPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		pb, _ := json.Marshal(payload)
		sig := ed25519.Sign(priv, pb)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"signature": base64.StdEncoding.EncodeToString(sig),
			"key_ref":   "local-hsm-emulator:test-kid",
		})
	}))
	return pub, priv, signer
}

func resetCheckpointState() {
	mu.Lock()
	defer mu.Unlock()
	checkpointHistory = map[int64]checkpointResponse{}
	checkpointOrder = []int64{}
}
