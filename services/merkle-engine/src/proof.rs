use crate::tree::{leaf_hash, node_hash};
use sha2::{Digest, Sha256};

pub struct InclusionProof {
    pub leaf_index: usize,
    pub tree_size: usize,
    pub path: Vec<[u8;32]>,
}

impl InclusionProof {
    pub fn verify(&self, leaf_data: &[u8], root_hex: &str) -> bool {
        let mut cur = leaf_hash(leaf_data);
        let mut index = self.leaf_index;
        for sibling in &self.path {
            if index % 2 == 0 {
                cur = node_hash(&cur, sibling);
            } else {
                cur = node_hash(sibling, &cur);
            }
            index /= 2;
        }
        hex::encode(cur) == root_hex
    }
}
