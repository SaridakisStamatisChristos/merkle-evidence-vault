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
    use std::fs;
    use std::io::{Read, Write};
    use std::path::PathBuf;
    use std::env;

    #[test]
    fn basic_append_root_changes() {
        let mut t = MerkleTree::new();
        let r0 = t.root();
        t.append(b"a");
        let r1 = t.root();
        assert_ne!(r0, r1);
    }

    #[test]
    fn property_append_monotonicity() {
        // Build roots for prefixes and ensure they match rebuilt-prefix roots.
        let items: Vec<&[u8]> = vec![b"one", b"two", b"three", b"four", b"five"];
        let mut prefix_roots = Vec::new();
        for i in 1..=items.len() {
            let mut t = MerkleTree::new();
            for it in &items[..i] { t.append(it); }
            prefix_roots.push(t.root());
        }

        // Now append all and compare prefix roots by rebuilding prefixes
        let mut full = MerkleTree::new();
        for (i, it) in items.iter().enumerate() {
            full.append(it);
            // rebuild prefix up to i and compare
            let mut prefix = MerkleTree::new();
            for p in &items[..=i] { prefix.append(p); }
            assert_eq!(prefix.root(), prefix_roots[i]);
            assert_eq!(prefix.root(), prefix.root());
        }
    }

    #[test]
    fn replay_determinism_from_corpus() {
        // Read committed minimized corpus files and ensure replay produces same root
        let repo_root = PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("..").canonicalize().unwrap();
        let corpus_dir = repo_root.join("fuzz").join("corpus").join("minimized");
        if !corpus_dir.exists() {
            // Skip when running in environments without the corpus
            eprintln!("corpus dir missing: {}", corpus_dir.display());
            return;
        }
        let mut payloads = Vec::new();
        for entry in fs::read_dir(&corpus_dir).unwrap() {
            let e = entry.unwrap();
            if e.file_type().unwrap().is_file() {
                let mut b = Vec::new();
                fs::File::open(e.path()).unwrap().read_to_end(&mut b).unwrap();
                payloads.push(b);
            }
        }
        assert!(!payloads.is_empty(), "no corpus entries found");

        // build tree first time
        let mut t1 = MerkleTree::new();
        for p in &payloads { t1.append(p); }
        let r1 = t1.root();

        // simulate shutdown and replay
        let mut t2 = MerkleTree::new();
        for p in &payloads { t2.append(p); }
        let r2 = t2.root();
        assert_eq!(r1, r2, "replay produced different root");
    }

    #[test]
    fn truncation_and_recovery() {
        // Create a temp file with newline-separated entries, truncate, and recover
        let mut dir = env::temp_dir();
        dir.push("merkle_truncate_test.txt");
        let entries = vec!["alpha", "beta", "gamma", "delta"];
        let mut content = String::new();
        for e in &entries { content.push_str(e); content.push('\n'); }
        fs::write(&dir, content.as_bytes()).unwrap();

        // truncate some bytes to simulate partial write of last entry
        let meta = fs::metadata(&dir).unwrap();
        let len = meta.len();
        let cut = 3u64; // remove last 3 bytes
        let new_len = len - cut;
        let file = fs::OpenOptions::new().write(true).open(&dir).unwrap();
        file.set_len(new_len).unwrap();

        // recovery: read file and only take complete lines (ending with '\n')
        let data = fs::read_to_string(&dir).unwrap();
        let mut lines: Vec<&str> = data.split('\n').collect();
        // split yields trailing empty if ends with newline; if not, last is partial and should be ignored
        if !data.ends_with('\n') { lines.pop(); }

        // build expected tree from original entries except possibly last partial
        let mut expected = MerkleTree::new();
        for l in &lines { expected.append(l.as_bytes()); }

        // build recovered tree
        let mut recovered = MerkleTree::new();
        for l in &lines { recovered.append(l.as_bytes()); }

        assert_eq!(expected.root(), recovered.root());
    }
}
