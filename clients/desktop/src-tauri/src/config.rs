use std::path::PathBuf;

use anyhow::Context;
use serde::{Deserialize, Serialize};

/// Persistent client configuration, stored as JSON in the user config dir.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(default)]
pub struct ClientConfig {
    pub server_url: String,
    pub device_id: String,
    pub device_name: String,
    pub token: String,
    pub interval_secs: u64,
    pub enabled: bool,
}

impl Default for ClientConfig {
    fn default() -> Self {
        Self {
            server_url: String::new(),
            device_id: "desktop".to_string(),
            device_name: hostname().unwrap_or_else(|| "我的设备".to_string()),
            token: String::new(),
            interval_secs: 5,
            enabled: false,
        }
    }
}

pub fn config_path() -> PathBuf {
    dirs::config_dir()
        .map(|d| d.join("whatamidoing").join("config.json"))
        .unwrap_or_else(|| PathBuf::from("config.json"))
}

pub fn load() -> ClientConfig {
    let path = config_path();
    std::fs::read_to_string(&path)
        .ok()
        .and_then(|s| serde_json::from_str(&s).ok())
        .unwrap_or_default()
}

pub fn save(cfg: &ClientConfig) -> anyhow::Result<()> {
    let path = config_path();
    if let Some(dir) = path.parent() {
        std::fs::create_dir_all(dir).context("创建配置目录")?;
    }
    let json = serde_json::to_string_pretty(cfg)?;
    std::fs::write(&path, json).context("写入配置")
}

fn hostname() -> Option<String> {
    std::env::var("HOSTNAME")
        .ok()
        .filter(|s| !s.is_empty())
        .or_else(|| {
            std::fs::read_to_string("/etc/hostname")
                .ok()
                .map(|s| s.trim().to_string())
        })
}
