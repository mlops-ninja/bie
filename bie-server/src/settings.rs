/// Settings for the server
#[derive(Debug, serde::Deserialize)]
pub struct Settings {
    /// The port to bind the server to
    /// Default: 8080
    #[serde(default = "default_port")]
    pub port: u16,

    /// The maximum file size to accept
    /// Default: 100MB
    #[serde(default = "default_max_file_size")]
    pub max_file_size: u64,
}

fn default_port() -> u16 {
    3000
}

fn default_max_file_size() -> u64 {
    10 * 1024 * 1024
}

impl Settings {
    /// Load settings from environment variables
    /// They should be prefixed with `BIE_`
    /// Example: `BIE_PORT=8080`
    pub fn load() -> Result<Self, anyhow::Error> {
        let settings = config::Config::builder()
            .add_source(config::Environment::with_prefix("BIE"))
            .build()
            .unwrap();

        let settings: Settings = settings.try_deserialize()?;
        Ok(settings)
    }
}
