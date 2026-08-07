//! macOS foreground detection via AppKit: `NSWorkspace.frontmostApplication`.
//!
//! Reports the frontmost app's display name (no special permission needed).
//! Window title is intentionally left empty: reading another app's window title
//! requires the **Screen Recording** permission plus the CGWindowList API,
//! which is a documented follow-up (see README).

use objc2::rc::autoreleasepool;
use objc2_app_kit::NSWorkspace;

use super::info::ForegroundInfo;

pub fn current() -> Option<ForegroundInfo> {
    autoreleasepool(|_| unsafe {
        let ws = NSWorkspace::sharedWorkspace();
        let app = ws.frontmostApplication()?;
        let name = app.localizedName().map(|s| s.to_string()).unwrap_or_default();
        if name.is_empty() {
            return None;
        }
        Some(ForegroundInfo {
            app: name,
            window_title: String::new(),
        })
    })
}
