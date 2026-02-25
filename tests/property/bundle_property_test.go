package property

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/SaridakisStamatisChristos/vault-api/domain/bundle"
)

func TestBundleManifest_RoundTrip(t *testing.T) {
	manifest := &bundle.Manifest{
		Version:       "1",
		CreatedAt:     time.Now().UTC().Truncate(time.Second),
		CreatedBy:     "test-actor",
		TreeSize:      1000,
		RootHash:      "aabb" + strings.Repeat("00", 30),
		LeafRange:     bundle.LeafRange{First: 0, Last: 999},
		EvidenceCount: 1000,
		Entries:       nil,
		Checkpoint: bundle.CheckpointRef{
			Filename:  bundle.PathSTH,
			KeyID:     "cc" + strings.Repeat("00", 31),
			Signature: "dd" + strings.Repeat("00", 63),
		},
	}

	data, err := bundle.MarshalManifest(manifest)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := bundle.UnmarshalManifest(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.TreeSize != manifest.TreeSize {
		t.Errorf("TreeSize mismatch: got %d want %d", got.TreeSize, manifest.TreeSize)
	}
	if got.RootHash != manifest.RootHash {
		t.Errorf("RootHash mismatch")
	}
	if got.Version != bundle.BundleVersion {
		t.Errorf("Version mismatch: got %q want %q", got.Version, bundle.BundleVersion)
	}
}

func TestBundleManifest_WrongVersionRejected(t *testing.T) {
	raw := map[string]interface{}{
		"version":        "99",
		"tree_size":      1,
		"root_hash":      "aa",
		"evidence_count": 0,
	}
	data, _ := json.Marshal(raw)
	_, err := bundle.UnmarshalManifest(data)
	if err == nil {
		t.Error("expected error for unsupported version")
	}
}

func TestBundleManifest_InvalidJSONRejected(t *testing.T) {
	_, err := bundle.UnmarshalManifest([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
