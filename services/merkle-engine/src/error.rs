use thiserror::Error;

#[derive(Error, Debug)]
pub enum EngineError {
    #[error("persistence error: {0}")]
    Persistence(String),
    #[error("invalid argument: {0}")]
    InvalidArgument(String),
}
