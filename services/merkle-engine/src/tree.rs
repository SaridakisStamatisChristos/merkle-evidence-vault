use sha2::{Digest, Sha256};
use std::fmt::Write;

pub fn leaf_hash(data: &[u8]) -> [u8; 32] {
    let mut hasher = Sha256::new();
    hasher.update(&[0x00]);
    hasher.update(data);
    let out = hasher.finalize();
    let mut a = [0u8; 32];
    a.copy_from_slice(&out);
    a
}

pub fn node_hash(left: &[u8; 32], right: &[u8; 32]) -> [u8; 32] {
    let mut hasher = Sha256::new();
    hasher.update(&[0x01]);
    hasher.update(left);
    hasher.update(right);
    let out = hasher.finalize();
    let mut a = [0u8; 32];
    a.copy_from_slice(&out);
    a
}

#[derive(Clone)]
pub struct MerkleTree {
    leaves: Vec<[u8; 32]>,
}

impl MerkleTree {
    pub fn new() -> Self { MerkleTree { leaves: Vec::new() } }

    pub fn append(&mut self, data: &[u8]) -> usize {
        let h = leaf_hash(data);
        self.leaves.push(h);
        self.leaves.len() - 1
    }

    pub fn root(&self) -> String {
        if self.leaves.is_empty() { return "".to_string(); }
        let root = subtree_hash(&self.leaves, 0, self.leaves.len());
        hex::encode(root)
    }
}

fn subtree_hash(leaves: &[[u8;32]], lo: usize, hi: usize) -> [u8;32] {
    if hi - lo == 1 { return leaves[lo]; }
    let k = largest_pow2_less_than(hi - lo);
    let l = subtree_hash(leaves, lo, lo + k);
    let r = subtree_hash(leaves, lo + k, hi);
    node_hash(&l, &r)
}

fn largest_pow2_less_than(n: usize) -> usize {
    let mut k = 1usize;
    while k < n { k <<= 1; }
    k >> 1
}
