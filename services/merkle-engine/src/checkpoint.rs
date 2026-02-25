use ed25519_dalek::{Keypair, PublicKey, Signature, Signer, Verifier};
use sha2::{Digest, Sha256};

pub struct SignedTreeHead {
    pub tree_size: u64,
    pub root_hash: String,
    pub signature: String,
    pub key_id: String,
}

pub struct SignerWrapper {
    kp: Keypair,
}

impl SignerWrapper {
    pub fn from_seed_hex(seed_hex: &str) -> anyhow::Result<Self> {
        let seed = hex::decode(seed_hex)?;
        let kp = Keypair::from_bytes(&[&seed[..], &[0u8;0]].concat()).map_err(|e| anyhow::anyhow!(e.to_string()))?;
        Ok(SignerWrapper { kp })
    }

    pub fn sign_sth(&self, tree_size: u64, root_hash: &str) -> SignedTreeHead {
        let mut ctx = Vec::new();
        ctx.extend_from_slice(&tree_size.to_be_bytes());
        ctx.extend_from_slice(root_hash.as_bytes());
        let sig = self.kp.sign(&ctx);
        let key_id = {
            let pk = self.kp.public;
            let mut h = Sha256::new(); h.update(pk.as_bytes()); hex::encode(h.finalize())
        };
        SignedTreeHead {
            tree_size,
            root_hash: root_hash.to_string(),
            signature: hex::encode(sig.to_bytes()),
            key_id,
        }
    }
}
