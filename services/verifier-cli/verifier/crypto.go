package verifier

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
)

func VerifySTH(pubHex string, msg []byte, sigHex string) bool {
	pk, err := hex.DecodeString(pubHex)
	if err != nil {
		return false
	}
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}
	return ed25519.Verify(pk, msg, sig)
}

func SHA256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
