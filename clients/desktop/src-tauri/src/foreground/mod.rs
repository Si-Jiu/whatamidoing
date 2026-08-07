//! Per-platform foreground-window detection.

pub mod info;

#[cfg(target_os = "windows")]
mod windows;
#[cfg(target_os = "macos")]
mod macos;
#[cfg(target_os = "linux")]
mod linux;

pub use info::ForegroundInfo;

/// Return the current foreground app + window title, if detectable.
pub fn current() -> Option<ForegroundInfo> {
    #[cfg(target_os = "windows")]
    {
        windows::current()
    }
    #[cfg(target_os = "macos")]
    {
        macos::current()
    }
    #[cfg(target_os = "linux")]
    {
        linux::current()
    }
    #[cfg(not(any(target_os = "windows", target_os = "macos", target_os = "linux")))]
    {
        None
    }
}

/// Wire-format platform string, per docs/protocol.md.
pub fn platform() -> &'static str {
    #[cfg(target_os = "windows")]
    {
        "windows"
    }
    #[cfg(target_os = "macos")]
    {
        "macos"
    }
    #[cfg(target_os = "linux")]
    {
        "linux"
    }
    #[cfg(not(any(target_os = "windows", target_os = "macos", target_os = "linux")))]
    {
        "unknown"
    }
}
