//! Linux foreground detection.
//!
//! Strategy, in priority order:
//! 1. **Hyprland IPC** (`hyprctl -j activewindow`) — exact class + title on
//!    wlroots-style Hyprland, no X needed.
//! 2. **X11** `_NET_ACTIVE_WINDOW` + `_NET_WM_NAME` — works on X11 sessions
//!    and (for X clients) under XWayland.
//! 3. Otherwise nothing — GNOME/KDE Wayland have no global foreground API;
//!    this is a documented limitation.

use std::process::Command;

use x11rb::connection::Connection;
use x11rb::protocol::xproto::ConnectionExt;

use super::info::ForegroundInfo;

pub fn current() -> Option<ForegroundInfo> {
    if let Some(info) = hyprland() {
        return Some(info);
    }
    if let Some(info) = x11() {
        return Some(info);
    }
    None
}

/// Hyprland's IPC gives exact foreground info without any X dependency.
fn hyprland() -> Option<ForegroundInfo> {
    if std::env::var("HYPRLAND_INSTANCE_SIGNATURE").is_err() {
        return None;
    }
    let out = Command::new("hyprctl").args(["-j", "activewindow"]).output().ok()?;
    if !out.status.success() {
        return None;
    }
    let v: serde_json::Value = serde_json::from_slice(&out.stdout).ok()?;
    let class = v.get("class").and_then(|c| c.as_str()).unwrap_or("").trim().to_string();
    let title = v.get("title").and_then(|c| c.as_str()).unwrap_or("").trim().to_string();
    if class.is_empty() && title.is_empty() {
        return None;
    }
    Some(ForegroundInfo { app_id: class, window_title: title })
}

/// X11 EWMH: `_NET_ACTIVE_WINDOW` on the root → window title + class.
fn x11() -> Option<ForegroundInfo> {
    if std::env::var("DISPLAY").is_err() {
        return None;
    }
    let (conn, screen_num) = x11rb::connect(None).ok()?;
    let screen = &conn.setup().roots[screen_num];
    let root = screen.root;

    let net_active = conn.intern_atom(false, b"_NET_ACTIVE_WINDOW").ok()?.reply().ok()?.atom;
    let reply = conn
        .get_property(false, root, net_active, x11rb::protocol::xproto::AtomEnum::WINDOW, 0, 1)
        .ok()?
        .reply()
        .ok()?;
    let window = reply.value32()?.next()?;
    if window == 0 {
        return None;
    }

    let title = string_prop(&conn, window, b"_NET_WM_NAME")
        .or_else(|| string_prop(&conn, window, b"WM_NAME"))
        .unwrap_or_default();
    let app_id = class_prop(&conn, window).unwrap_or_default();
    if app_id.is_empty() && title.is_empty() {
        return None;
    }
    Some(ForegroundInfo { app_id, window_title: title })
}

fn string_prop(conn: &impl x11rb::connection::Connection, window: u32, name: &[u8]) -> Option<String> {
    let atom = conn.intern_atom(false, name).ok()?.reply().ok()?.atom;
    let reply = conn
        .get_property(false, window, atom, x11rb::protocol::xproto::AtomEnum::ANY, 0, 10_000)
        .ok()?
        .reply()
        .ok()?;
    String::from_utf8(reply.value).ok().filter(|s| !s.is_empty())
}

/// WM_CLASS is a null-separated pair (instance, class); prefer the class.
fn class_prop(conn: &impl x11rb::connection::Connection, window: u32) -> Option<String> {
    let reply = conn
        .get_property(false, window, x11rb::protocol::xproto::AtomEnum::STRING, x11rb::protocol::xproto::AtomEnum::STRING, 0, 1_000)
        .ok()?
        .reply()
        .ok()?;
    let parts: Vec<String> = reply
        .value
        .split(|b| *b == 0)
        .filter(|s| !s.is_empty())
        .filter_map(|s| String::from_utf8(s.to_vec()).ok())
        .collect();
    parts.last().filter(|s| !s.is_empty()).cloned()
}
