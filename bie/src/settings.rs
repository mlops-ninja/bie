use anyhow::Result;
use config::{Config, File as ConfigFile};
use toml;
use xdg;

use serde::{Deserialize, Serialize};

#[derive(Debug, Deserialize, Serialize)]
pub struct Settings {
    pub bastion_server_url: String,
}

impl Default for Settings {
    fn default() -> Self {
        Settings {
            bastion_server_url: "http://localhost:8080".to_string(),
        }
    }
}

impl Settings {
    pub fn load() -> Result<Self> {
        let xdg_dirs = xdg::BaseDirectories::with_prefix("bie")?;
        let config_path = xdg_dirs.place_config_file("config.toml")?;

        if !config_path.exists() {
            let default_settings = Settings::default();
            let toml = toml::to_string(&default_settings)?;
            std::fs::write(&config_path, toml)?;
        }

        let config = Config::builder()
            .add_source(ConfigFile::from(config_path))
            .build()?;

        Ok(config.try_deserialize()?)
    }
}
