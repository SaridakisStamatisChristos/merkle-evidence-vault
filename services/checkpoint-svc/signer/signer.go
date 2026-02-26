package signer

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
)

// Signer represents a service capable of signing bytes and returning the signature.
type Signer interface {
	Sign([]byte) ([]byte, error)
}

// LocalSigner uses an exported ed25519 private key (in memory) to sign payloads.
type LocalSigner struct {
	priv ed25519.PrivateKey
}

func NewLocalSignerFromBase64(b64 string) (*LocalSigner, error) {
	dec, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	if len(dec) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid private key size")
	}
	return &LocalSigner{priv: ed25519.PrivateKey(dec)}, nil
}

func (s *LocalSigner) Sign(b []byte) ([]byte, error) {
	sig := ed25519.Sign(s.priv, b)
	return sig, nil
}

// KMSSigner is a placeholder for KMS-backed signing implementations.
// Production implementations should implement this type to call provider-specific
// Sign APIs (AWS KMS, Azure Key Vault, Google Cloud KMS) which typically do
// not expose the private key material but provide signing RPCs.
// For now KMSSigner is a stub that returns an informative error.
type KMSSigner struct {
	Provider string
	KeyID    string
}

func NewKMSSigner(provider, keyid string) (*KMSSigner, error) {
	// TODO: implement provider-specific clients and return a KMSSigner that
	// performs remote Sign operations. Keep the interface minimal (Sign).
	return &KMSSigner{Provider: provider, KeyID: keyid}, nil
}

func (k *KMSSigner) Sign(b []byte) ([]byte, error) {
	return nil, errors.New("KMSSigner not implemented: configure provider-specific implementation (AWS KMS / Azure Key Vault / GCP KMS)")
}
