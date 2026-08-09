/// What the user is doing right now, in one device's foreground.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ForegroundInfo {
    /// Raw process / window class identifier, e.g. "kitty", "chrome",
    /// or bundle id on macOS. The server maps this to a display name.
    pub app_id: String,
    /// Active window title, e.g. "GitHub · whatamidoing". May be empty.
    pub window_title: String,
}
