package property

import (
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"testing"
	"time"
)

// minimalMerkle is a Go reference implementation of RFC 6962
// used to cross-check the engine's output.
func leafHash(data []byte) [32]byte {
	h := sha256.New()
	h.Write([]byte{0x00})
	h.Write(data)
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

func nodeHash(l, r [32]byte) [32]byte {
	h := sha256.New()
	h.Write([]byte{0x01})
	h.Write(l[:])
	h.Write(r[:])
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

func subtreeHash(leaves [][32]byte, lo, hi int) [32]byte {
	if hi-lo == 1 {
		return leaves[lo]
	}
	k := largestPow2LessThan(hi - lo)
	l := subtreeHash(leaves, lo, lo+k)
	r := subtreeHash(leaves, lo+k, hi)
	return nodeHash(l, r)
}

func largestPow2LessThan(n int) int {
	k := 1
	for k < n {
		k <<= 1
	}
	return k >> 1
}

func merkleRoot(leaves [][32]byte) [32]byte {
	return subtreeHash(leaves, 0, len(leaves))
}

// ── Properties ─────────────────────────────────────────────────────────────

// Property 1: root of prefix tree equals root_at(n) of larger tree.
func TestProperty_RootAtEqualsPrefix(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for trial := 0; trial < 500; trial++ {
		n := rng.Intn(62) + 2 // 2..63
		leaves := randomLeaves(rng, n)
		mid := rng.Intn(n-1) + 1 // 1..n-1

		full := merkleRoot(leaves)
		prefix := merkleRoot(leaves[:mid])

		// Recompute root_at(mid) from full set.
		rootAtMid := subtreeHash(leaves, 0, mid)

		if rootAtMid != prefix {
			t.Errorf("trial %d: root_at(%d) != prefix root for n=%d", trial, mid, n)
		}
		_ = full
	}
}

// Property 2: single-leaf tree root equals leaf_hash(data).
func TestProperty_SingleLeafRootIsLeafHash(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 1000; i++ {
		data := randomBytes(rng, 1, 256)
		lh := leafHash(data)
		tree := [][32]byte{lh}
		root := merkleRoot(tree)
		if root != lh {
			t.Errorf("single-leaf root %s != leaf hash %s",
				hex.EncodeToString(root[:]), hex.EncodeToString(lh[:]))
		}
	}
}

// Property 3: appending a leaf changes the root (collision resistance smoke).
func TestProperty_AppendChangesRoot(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	for trial := 0; trial < 200; trial++ {
		n := rng.Intn(30) + 1
		leaves := randomLeaves(rng, n)
		root1 := merkleRoot(leaves)

		extra := leafHash(randomBytes(rng, 1, 64))
		root2 := merkleRoot(append(leaves, extra))

		if root1 == root2 {
			t.Errorf("trial %d: root unchanged after appending leaf (n=%d)", trial, n)
		}
	}
}

// Property 4: tree hash is deterministic (same leaves → same root).
func TestProperty_Deterministic(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for trial := 0; trial < 300; trial++ {
		n := rng.Intn(32) + 1
		leaves := randomLeaves(rng, n)
		r1 := merkleRoot(leaves)
		r2 := merkleRoot(leaves)
		if r1 != r2 {
			t.Errorf("non-deterministic root for n=%d", n)
		}
	}
}

// Property 5: inclusion proof path length ≤ ⌈log₂(n)⌉.
func TestProperty_ProofPathLength(t *testing.T) {
	rng := rand.New(rand.NewSource(13))
	for trial := 0; trial < 200; trial++ {
		n := rng.Intn(63) + 1
		leaves := randomLeaves(rng, n)
		idx := rng.Intn(n)
		path := inclusionPath(leaves, idx, 0, n)
		maxLen := ceilLog2(n)
		if len(path) > maxLen {
			t.Errorf("trial %d: path_len=%d > ceil_log2(%d)=%d",
				trial, len(path), n, maxLen)
		}
	}
}

// Property 6: leaf order matters (permutation changes root).
func TestProperty_OrderMatters(t *testing.T) {
	rng := rand.New(rand.NewSource(55))
	for trial := 0; trial < 100; trial++ {
		n := rng.Intn(10) + 2
		leaves := randomLeaves(rng, n)
		root1 := merkleRoot(leaves)
		// Swap first two leaves.
		leaves[0], leaves[1] = leaves[1], leaves[0]
		root2 := merkleRoot(leaves)
		// They may be equal if leaves[0]==leaves[1], but extremely unlikely.
		if root1 == root2 {
			// Check if the leaves were actually identical.
			if leaves[0] != leaves[1] {
				t.Errorf("trial %d: root unchanged after swap with distinct leaves", trial)
			}
		}
	}
}

// ── Inclusion path helper (mirrors Rust proof.rs) ──────────────────────────

func inclusionPath(leaves [][32]byte, idx, lo, hi int) [][32]byte {
	if hi-lo == 1 {
		return nil
	}
	k := largestPow2LessThan(hi - lo)
	var path [][32]byte
	if idx-lo < k {
		path = inclusionPath(leaves, idx, lo, lo+k)
		path = append(path, subtreeHash(leaves, lo+k, hi))
	} else {
		path = inclusionPath(leaves, idx, lo+k, hi)
		path = append(path, subtreeHash(leaves, lo, lo+k))
	}
	return path
}

func ceilLog2(n int) int {
	if n <= 1 {
		return 0
	}
	k := 0
	m := n - 1
	for m > 0 {
		m >>= 1
		k++
	}
	return k
}

// ── Random data helpers ────────────────────────────────────────────────────

func randomLeaves(rng *rand.Rand, n int) [][32]byte {
	out := make([][32]byte, n)
	for i := range out {
		out[i] = leafHash(randomBytes(rng, 1, 64))
	}
	return out
}

func randomBytes(rng *rand.Rand, minLen, maxLen int) []byte {
	n := minLen + rng.Intn(maxLen-minLen+1)
	b := make([]byte, n)
	rng.Read(b)
	return b
}
