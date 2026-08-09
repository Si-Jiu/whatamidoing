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
        // app_id = bundle identifier（如 com.google.Chrome）；拿不到时回退应用名。
        let app_id = app.bundleIdentifier().map(|s| s.to_string()).unwrap_or_else(|| name);
        Some(ForegroundInfo {
            app_id,
            window_title: String::new(),
        })
    })
}
