package bundle

import (
	"encoding/json"
	"time"
)

const (
	BundleVersion = "1"
	PathSTH       = "checkpoint/sth.json"
)

type LeafRange struct {
	First int `json:"first"`
	Last  int `json:"last"`
}

type CheckpointRef struct {
	Filename  string `json:"filename"`
	KeyID     string `json:"key_id"`
	Signature string `json:"signature"`
}

type Manifest struct {
	Version       string        `json:"version"`
	CreatedAt     time.Time     `json:"created_at"`
	CreatedBy     string        `json:"created_by"`
	TreeSize      int64         `json:"tree_size"`
	RootHash      string        `json:"root_hash"`
	LeafRange     LeafRange     `json:"leaf_range"`
	EvidenceCount int           `json:"evidence_count"`
	Entries       []interface{} `json:"entries"`
	Checkpoint    CheckpointRef `json:"checkpoint"`
}

func MarshalManifest(m *Manifest) ([]byte, error) {
	m.Version = BundleVersion
	return json.Marshal(m)
}

func UnmarshalManifest(b []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m.Version != BundleVersion {
		return nil, ErrUnsupportedVersion
	}
	return &m, nil
}

var ErrUnsupportedVersion = &json.UnmarshalTypeError{Value: "version", Type: nil}
