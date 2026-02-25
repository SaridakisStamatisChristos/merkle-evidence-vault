// Minimal RPC shim placeholder.
use crate::tree::MerkleTree;

pub struct MerkleEngineService {
    pub tree: MerkleTree,
}

impl MerkleEngineService {
    pub fn new() -> Self { MerkleEngineService { tree: MerkleTree::new() } }
}
