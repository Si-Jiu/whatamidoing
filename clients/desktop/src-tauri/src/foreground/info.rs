/// What the user is doing right now, in one device's foreground.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ForegroundInfo {
    /// Raw process / window class identifier, e.g. "kitty", "chrome".
    /// Android uses the package name instead (handled on that client).
    pub app_id: String,
    /// Foreground application display name, e.g. "终端" (mapped from app_id).
    pub app: String,
    /// Active window title, e.g. "GitHub · whatamidoing". May be empty.
    pub window_title: String,
}
