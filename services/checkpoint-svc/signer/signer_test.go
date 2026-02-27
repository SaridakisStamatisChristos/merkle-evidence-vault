package signer

import (
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"testing"
)

func TestNewKMSSigner_LocalHSMEmulator(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	os.Setenv("KMS_PRIVATE_KEY_B64", base64.StdEncoding.EncodeToString(priv))
	defer os.Unsetenv("KMS_PRIVATE_KEY_B64")

	s, err := NewKMSSigner("local-hsm-emulator", "kid-1")
	if err != nil {
		t.Fatalf("NewKMSSigner: %v", err)
	}
	sig, err := s.Sign([]byte("hello"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if len(sig) == 0 {
		t.Fatalf("expected non-empty signature")
	}
	if s.KeyRef() != "local-hsm-emulator:kid-1" {
		t.Fatalf("unexpected keyref: %s", s.KeyRef())
	}
}
