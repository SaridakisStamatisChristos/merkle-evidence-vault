#![no_main]
extern crate libfuzzer_sys;

use libfuzzer_sys::fuzz_target;
use merkle_engine::tree::MerkleTree;

fuzz_target!(|data: &[u8]| {
    if data.is_empty() { return; }
    let n = (data[0] as usize % 8) + 1;
    let mut t = MerkleTree::new();
    let rest = &data[1..];
    if rest.is_empty() {
        t.append(&[]);
        let _ = t.root();
        return;
    }
    let chunk_len = (rest.len() + n - 1) / n;
    for chunk in rest.chunks(chunk_len) {
        t.append(chunk);
    }
    let _ = t.root();
});
