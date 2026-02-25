package checkpoint

import "time"

type SignedTreeHead struct {
	TreeSize    int64     `json:"tree_size"`
	RootHash    string    `json:"root_hash"`
	PublishedAt time.Time `json:"published_at"`
	KeyID       string    `json:"key_id"`
	Signature   string    `json:"signature"`
}
