package merkle

type InclusionProof struct {
	LeafIndex int64    `json:"leaf_index"`
	TreeSize  int64    `json:"tree_size"`
	Root      string   `json:"root"`
	Path      []string `json:"path"`
}

type ConsistencyProof struct {
	OldSize int64    `json:"old_size"`
	NewSize int64    `json:"new_size"`
	Path    []string `json:"path"`
}

type Engine interface {
	AppendLeaf(leaf []byte) (leafIndex int64, root []byte, err error)
	InclusionProof(leafIndex int64) (*InclusionProof, error)
	ConsistencyProof(oldSize int64, newSize int64) (*ConsistencyProof, error)
	TreeSize() (int64, error)
	Root() ([]byte, error)
}
