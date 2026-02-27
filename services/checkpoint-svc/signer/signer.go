package signer

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
)

// Signer represents a service capable of signing bytes and returning the signature.
type Signer interface {
	Sign([]byte) ([]byte, error)
	KeyRef() string
}

// LocalSigner uses an exported ed25519 private key (in memory) to sign payloads.
type LocalSigner struct {
	priv   ed25519.PrivateKey
	keyRef string
}

func NewLocalSignerFromBase64(b64 string) (*LocalSigner, error) {
	dec, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	if len(dec) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid private key size")
	}
	return &LocalSigner{priv: ed25519.PrivateKey(dec), keyRef: "local:file-or-env"}, nil
}

func NewLocalSignerFromBase64WithRef(b64, ref string) (*LocalSigner, error) {
	s, err := NewLocalSignerFromBase64(b64)
	if err != nil {
		return nil, err
	}
	if ref != "" {
		s.keyRef = ref
	}
	return s, nil
}

func (s *LocalSigner) Sign(b []byte) ([]byte, error) {
	sig := ed25519.Sign(s.priv, b)
	return sig, nil
}

func (s *LocalSigner) KeyRef() string {
	return s.keyRef
}

// KMSSigner abstracts provider-backed signing implementations.
type KMSSigner struct {
	Provider string
	KeyID    string
	signer   Signer
}

func NewKMSSigner(provider, keyid string) (*KMSSigner, error) {
	var impl Signer
	switch provider {
	case "local-hsm-emulator":
		b64 := os.Getenv("KMS_PRIVATE_KEY_B64")
		if b64 == "" {
			return nil, errors.New("KMS_PRIVATE_KEY_B64 is required for local-hsm-emulator")
		}
		s, err := NewLocalSignerFromBase64WithRef(b64, fmt.Sprintf("%s:%s", provider, keyid))
		if err != nil {
			return nil, err
		}
		impl = s
	default:
		return nil, fmt.Errorf("unsupported KMS_PROVIDER %q", provider)
	}
	return &KMSSigner{Provider: provider, KeyID: keyid, signer: impl}, nil
}

func (k *KMSSigner) Sign(b []byte) ([]byte, error) {
	if k.signer == nil {
		return nil, errors.New("kms signer backend not initialized")
	}
	return k.signer.Sign(b)
}

func (k *KMSSigner) KeyRef() string {
	return fmt.Sprintf("%s:%s", k.Provider, k.KeyID)
}
