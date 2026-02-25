pub mod error;
pub mod tree;
pub mod proof;
pub mod checkpoint;
pub mod rpc;

pub const LEAF_PREFIX: u8 = 0x00;
pub const NODE_PREFIX: u8 = 0x01;

#[cfg(test)]
mod tests {
    use super::tree::MerkleTree;

    #[test]
    fn basic_append_root_changes() {
        let mut t = MerkleTree::new();
        let r0 = t.root();
        t.append(b"a");
        let r1 = t.root();
        assert_ne!(r0, r1);
    }
}
