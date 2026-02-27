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
	defer signer.Close()

	os.Setenv("CHECKPOINT_SIGNING_URL", signer.URL)
	os.Setenv("CHECKPOINT_VERIFY_PUBLIC_KEY_B64", base64.StdEncoding.EncodeToString(pub))
	os.Setenv("ENABLE_TEST_JWT", "true")
	defer os.Unsetenv("CHECKPOINT_SIGNING_URL")
	defer os.Unsetenv("CHECKPOINT_VERIFY_PUBLIC_KEY_B64")
	defer os.Unsetenv("ENABLE_TEST_JWT")

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
