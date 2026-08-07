use std::sync::{Arc, Mutex};
use std::time::Duration;

use crate::config::ClientConfig;
use crate::foreground;
use crate::foreground::info::ForegroundInfo;
use crate::rules;

/// Background loop: polls the foreground app and reports it to the server.
/// Runs forever; reads the shared config each iteration so toggling sharing
/// or editing settings takes effect immediately.
pub fn run(cfg: Arc<Mutex<ClientConfig>>) {
    let client = match reqwest::blocking::Client::builder()
        .timeout(Duration::from_secs(5))
        .build()
    {
        Ok(c) => c,
        Err(e) => {
            log::error!("无法创建 HTTP 客户端: {e}");
            return;
        }
    };

    let mut last: Option<ForegroundInfo> = None;
    let mut app_started = chrono::Utc::now();

    loop {
        let cfg_guard = match cfg.lock() {
            Ok(g) => g,
            Err(_) => {
                std::thread::sleep(Duration::from_secs(2));
                continue;
            }
        };
        let c = cfg_guard.clone();
        drop(cfg_guard);

        if !c.enabled {
            std::thread::sleep(Duration::from_secs(2));
            continue;
        }
        let interval = if c.interval_secs == 0 { 5 } else { c.interval_secs };

        match foreground::current() {
            Some(info) => {
                // Foreground changed → reset the "since" timestamp.
                if last.as_ref() != Some(&info) {
                    app_started = chrono::Utc::now();
                    last = Some(info.clone());
                }
                // 应用映射表规则：进程名 → 友好显示名。
                let mut reported = info;
                reported.app = rules::apply(&reported.app, &rules::load());
                report(&client, &c, &reported, app_started);
            }
            None => {
                // No foreground available (lock screen / no compositor access):
                // keep the previous state alive as a heartbeat.
                if let Some(info) = &last {
                    report(&client, &c, info, app_started);
                }
            }
        }
        sleep_interruptible(interval);
    }
}

fn report(client: &reqwest::blocking::Client, c: &ClientConfig, info: &ForegroundInfo, started: chrono::DateTime<chrono::Utc>) {
    let url = format!("{}/api/v1/report", c.server_url.trim_end_matches('/'));
    let body = serde_json::json!({
        "device_id": c.device_id,
        "device_name": c.device_name,
        "platform": foreground::platform(),
        "app": info.app,
        "window_title": info.window_title,
        "app_started_at": started.to_rfc3339(),
    });

    match client
        .post(&url)
        .bearer_auth(&c.token)
        .json(&body)
        .send()
    {
        Ok(resp) if resp.status().is_success() => {}
        Ok(resp) => log::warn!("上报被拒绝: HTTP {}", resp.status()),
        Err(e) => log::debug!("上报失败: {e}"),
    }
}

/// Sleep in 1s steps so config changes (e.g. stop sharing) feel responsive.
fn sleep_interruptible(secs: u64) {
    let mut left = secs;
    while left > 0 {
        std::thread::sleep(Duration::from_secs(1));
        left -= 1;
    }
}
