package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

type Evidence struct {
	ID          string            `json:"id"`
	ContentType string            `json:"content_type"`
	ContentHash string            `json:"content_hash"`
	PayloadRef  string            `json:"payload_ref"`
	Labels      map[string]string `json:"labels"`
	IngestedAt  time.Time         `json:"ingested_at"`
	IngestedBy  string            `json:"ingested_by"`
	LeafIndex   *int64            `json:"leaf_index"`
}

func NewEvidence(id string, contentType string, payload []byte, ingestedBy string) *Evidence {
	h := sha256.Sum256(payload)
	return &Evidence{
		ID:          id,
		ContentType: contentType,
		ContentHash: hex.EncodeToString(h[:]),
		PayloadRef:  "", // set by storage backend
		Labels:      make(map[string]string),
		IngestedAt:  time.Now().UTC(),
		IngestedBy:  ingestedBy,
	}
}

// LeafData returns the canonical leaf bytes used by the Merkle tree.
func (e *Evidence) LeafData() []byte {
	// simple binding: content_hash + content_type
	return []byte(e.ContentHash + ":" + e.ContentType)
}
