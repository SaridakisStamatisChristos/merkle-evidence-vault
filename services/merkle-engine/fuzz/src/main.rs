use merkle_engine::tree::MerkleTree;
use rand::{rngs::StdRng, RngCore, SeedableRng};
use std::panic;
use std::time::{Duration, Instant};
use std::env;

// Simple duration-based fuzz harness: generate random byte slices and call append.
// Usage: run with `--duration-secs N` to run for N seconds (default 60).
fn main() {
    let mut args = env::args();
    let mut duration_secs: u64 = 60;
    while let Some(a) = args.next() {
        if a == "--duration-secs" {
            if let Some(val) = args.next() {
                if let Ok(n) = val.parse::<u64>() {
                    duration_secs = n;
                }
            }
        }
    }

    println!("merkle-fuzz: running for {}s", duration_secs);
    let dur = Duration::from_secs(duration_secs);
    let start = Instant::now();

    // deterministic seed so runs are reproducible unless the harness uses system randomness
    let mut rng = StdRng::seed_from_u64(0xF00DBABE);

    let mut iters: u64 = 0;
    loop {
        if start.elapsed() >= dur {
            break;
        }
        // sample length up to 1024
        let len = (rng.next_u32() % 1024) as usize;
        let mut buf = vec![0u8; len];
        for b in buf.iter_mut() {
            *b = (rng.next_u32() & 0xFF) as u8;
        }

        // run append inside catch_unwind to capture panics
        let res = panic::catch_unwind(|| {
            let mut t = MerkleTree::new();
            t.append(&buf);
            let _ = t.root();
        });
        if res.is_err() {
            eprintln!("Panic detected on input len={}", len);
            std::process::exit(2);
        }

        iters += 1;
    }

    println!("merkle-fuzz: completed {} iterations", iters);
}
