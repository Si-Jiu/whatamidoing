/// What the user is doing right now, in one device's foreground.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ForegroundInfo {
    /// Foreground application display name, e.g. "Google Chrome".
    pub app: String,
    /// Active window title, e.g. "GitHub · whatamidoing". May be empty.
    pub window_title: String,
}
