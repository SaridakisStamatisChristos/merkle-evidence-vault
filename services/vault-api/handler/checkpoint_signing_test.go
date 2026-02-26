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
	"github.com/rs/zerolog/log"
)

func TestCheckpointSigningIntegration(t *testing.T) {
	// generate ed25519 keypair
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	// signing server: returns base64 signature of request body
	signer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		// sign raw body
		sig := ed25519.Sign(priv, b)
		enc := base64.StdEncoding.EncodeToString(sig)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"signature": enc})
	}))
	defer signer.Close()

	// set env so handler will call signing service
	d := signer.URL + "/sign"
	os.Setenv("CHECKPOINT_SIGNING_URL", d)
	defer os.Unsetenv("CHECKPOINT_SIGNING_URL")

	// enable test JWT mode so middleware maps token to roles
	os.Setenv("ENABLE_TEST_JWT", "true")
	defer os.Unsetenv("ENABLE_TEST_JWT")

	// seed an entry so tree_size > 0
	mu.Lock()
	storeMap = map[string]*evidenceRecord{"abc": {ID: "abc", LeafIndex: func() *int64 { x := int64(1); return &x }()}}
	mu.Unlock()

	// build router same as server
	h := NewIngestHandler()
	r := chi.NewRouter()
	r.With(middleware.JWT).Get("/api/v1/checkpoints/latest", h.GetCheckpointsLatest)

	// create request with auditor token
	req := httptest.NewRequest("GET", "/api/v1/checkpoints/latest", nil)
	req.Header.Set("Authorization", "Bearer auditor-token")

	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)

	if rw.Code != 200 {
		t.Fatalf("expected 200; got %d; body=%s", rw.Code, rw.Body.String())
	}

	var got struct {
		TreeSize  int64  `json:"tree_size"`
		RootHash  string `json:"root_hash"`
		Signature string `json:"signature"`
	}
	if err := json.NewDecoder(rw.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if got.Signature == "" {
		t.Fatalf("expected signature from signing service")
	}

	// Verify signature matches payload
	payload := map[string]interface{}{"tree_size": got.TreeSize, "root_hash": got.RootHash}
	pb, _ := json.Marshal(payload)
	sig, err := base64.StdEncoding.DecodeString(got.Signature)
	if err != nil {
		t.Fatalf("bad base64 sig: %v", err)
	}
	if !ed25519.Verify(pub, pb, sig) {
		t.Fatalf("signature did not verify")
	}

	log.Info().Msg("checkpoint signing integration test passed")
}
