use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize)]
pub enum BieProtocol {
    Token(String),
    FileChunk(Vec<u8>),
    EndOfFile,
}

impl From<BieProtocol> for Vec<u8> {
    fn from(protocol: BieProtocol) -> Vec<u8> {
        minicbor_serde::to_vec(&protocol).unwrap()
    }
}

pub fn generate_secure_random_string(length: usize) -> String {
    let rng = ring::rand::SystemRandom::new();
    let mut random_bytes = vec![0u8; length];

    ring::rand::SecureRandom::fill(&rng, &mut random_bytes)
        .expect("Failed to generate random bytes");

    // Convert the random bytes to a string using only alphanumeric characters
    let random_string: String = random_bytes
        .into_iter()
        .map(|b| (b % 62) as u8)
        .map(|b| match b {
            0..=9 => (b + b'0') as char,
            10..=35 => (b - 10 + b'a') as char,
            _ => (b - 36 + b'A') as char,
        })
        .collect();

    random_string
}
