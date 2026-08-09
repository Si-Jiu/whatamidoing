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
    let mut warned_interval: Option<u64> = None;

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
                check_interval(&mut warned_interval, interval, report(&client, &c, &reported, app_started));
            }
            None => {
                // No foreground available (lock screen / no compositor access):
                // keep the previous state alive as a heartbeat.
                if let Some(info) = &last {
                    check_interval(&mut warned_interval, interval, report(&client, &c, info, app_started));
                }
            }
        }
        sleep_interruptible(interval);
    }
}

/// 上报前台状态；成功时返回服务端响应头 `X-Idle-Timeout-Secs` 的离线判定阈值（秒）。
fn report(client: &reqwest::blocking::Client, c: &ClientConfig, info: &ForegroundInfo, started: chrono::DateTime<chrono::Utc>) -> Option<u64> {
    let url = format!("{}/api/v1/report", c.server_url.trim_end_matches('/'));
    // 设备身份由服务端按 token 确定，无需上报 device_id/device_name
    let body = serde_json::json!({
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
        Ok(resp) if resp.status().is_success() => resp
            .headers()
            .get("x-idle-timeout-secs")
            .and_then(|v| v.to_str().ok())
            .and_then(|v| v.parse().ok()),
        Ok(resp) => {
            log::warn!("上报被拒绝: HTTP {}", resp.status());
            None
        }
        Err(e) => {
            log::debug!("上报失败: {e}");
            None
        }
    }
}

/// 若上报间隔 ≥ 服务端离线判定阈值，设备会被误判离线——给出一次性警告（间隔变化时重新警告）。
fn check_interval(warned: &mut Option<u64>, interval: u64, server_idle: Option<u64>) {
    if let Some(idle) = server_idle {
        if interval >= idle && *warned != Some(interval) {
            log::warn!(
                "上报间隔 {interval} 秒不小于服务端离线判定阈值 {idle} 秒，设备会被误判为离线；请把上报间隔调到小于 {idle} 秒"
            );
            *warned = Some(interval);
        }
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
