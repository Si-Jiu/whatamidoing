//! 进程名 → 友好显示名的映射规则。
//!
//! 规则按顺序匹配，命中第一条即生效；无命中保留原始名称。
//! 内置默认映射表（终端/浏览器/编辑器等常见应用），可用
//! `~/.config/whatamidoing/rules.json` 完全覆盖（文件存在时优先）。

use std::path::PathBuf;

use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum MatchType {
    Exact,
    Prefix,
    Contains,
    Regex,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MappingRule {
    pub match_type: MatchType,
    pub pattern: String,
    pub display: String,
}

impl MappingRule {
    pub fn matches(&self, app: &str) -> bool {
        match self.match_type {
            MatchType::Exact => app.eq_ignore_ascii_case(&self.pattern),
            MatchType::Prefix => app.to_lowercase().starts_with(&self.pattern.to_lowercase()),
            MatchType::Contains => app.to_lowercase().contains(&self.pattern.to_lowercase()),
            MatchType::Regex => regex::Regex::new(&self.pattern)
                .map(|r| r.is_match(app))
                .unwrap_or(false),
        }
    }
}

pub fn rules_path() -> PathBuf {
    dirs::config_dir()
        .map(|d| d.join("whatamidoing").join("rules.json"))
        .unwrap_or_else(|| PathBuf::from("rules.json"))
}

/// 读取规则：优先用户自定义 `rules.json`；不存在或解析失败时用内置默认。
pub fn load() -> Vec<MappingRule> {
    let path = rules_path();
    std::fs::read_to_string(&path)
        .ok()
        .and_then(|s| serde_json::from_str(&s).ok())
        .unwrap_or_else(default_rules)
}

/// 按顺序应用规则，返回第一条命中的显示名；无命中返回原名称。
pub fn apply(app: &str, rules: &[MappingRule]) -> String {
    rules
        .iter()
        .find(|r| r.matches(app))
        .map(|r| r.display.clone())
        .unwrap_or_else(|| app.to_string())
}

fn rule(match_type: MatchType, pattern: &str, display: &str) -> MappingRule {
    MappingRule {
        match_type,
        pattern: pattern.to_string(),
        display: display.to_string(),
    }
}

/// 内置默认映射表（常见应用 → 中文友好名）。
pub fn default_rules() -> Vec<MappingRule> {
    use MatchType::*;
    vec![
        // 终端
        rule(Contains, "kitty", "终端"),
        rule(Contains, "alacritty", "终端"),
        rule(Contains, "wezterm", "终端"),
        rule(Contains, "foot", "终端"),
        rule(Contains, "konsole", "终端"),
        rule(Contains, "gnome-terminal", "终端"),
        rule(Contains, "terminal", "终端"),
        rule(Contains, "powershell", "终端"),
        rule(Contains, "windowsterminal", "终端"),
        rule(Contains, "cmd.exe", "终端"),
        // 浏览器
        rule(Contains, "firefox", "浏览器"),
        rule(Contains, "chrome", "浏览器"),
        rule(Contains, "chromium", "浏览器"),
        rule(Contains, "edge", "浏览器"),
        rule(Contains, "brave", "浏览器"),
        rule(Contains, "safari", "浏览器"),
        // 编辑器
        rule(Contains, "code", "编辑器"),
        rule(Contains, "vscodium", "编辑器"),
        rule(Contains, "vim", "编辑器"),
        rule(Contains, "nvim", "编辑器"),
        rule(Contains, "neovim", "编辑器"),
        rule(Contains, "emacs", "编辑器"),
        rule(Contains, "sublime", "编辑器"),
        rule(Contains, "intellij", "编辑器"),
        rule(Contains, "idea", "编辑器"),
        rule(Contains, "pycharm", "编辑器"),
        // 聊天
        rule(Contains, "wechat", "微信"),
        rule(Contains, "weixin", "微信"),
        rule(Contains, "qq", "QQ"),
        rule(Contains, "telegram", "Telegram"),
        rule(Contains, "discord", "Discord"),
        rule(Contains, "slack", "Slack"),
        rule(Contains, "whatsapp", "WhatsApp"),
        // 音乐 / 视频
        rule(Contains, "spotify", "音乐"),
        rule(Contains, "netease", "音乐"),
        rule(Contains, "music", "音乐"),
        rule(Contains, "mpv", "视频"),
        rule(Contains, "vlc", "视频"),
        rule(Contains, "bilibili", "哔哩哔哩"),
        rule(Contains, "youtube", "YouTube"),
        // 文件管理器
        rule(Contains, "thunar", "文件管理器"),
        rule(Contains, "nautilus", "文件管理器"),
        rule(Contains, "dolphin", "文件管理器"),
        rule(Contains, "explorer.exe", "文件管理器"),
        // 桌面 / 锁屏兜底
        rule(Contains, "lock", "锁屏"),
        rule(Exact, "", "桌面"),
    ]
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn maps_kitty_to_terminal() {
        assert_eq!(apply("kitty", &default_rules()), "终端");
    }

    #[test]
    fn matches_case_insensitively() {
        assert_eq!(apply("Google Chrome", &default_rules()), "浏览器");
        assert_eq!(apply("google-chrome", &default_rules()), "浏览器");
        assert_eq!(apply("KITTY", &default_rules()), "终端");
    }

    #[test]
    fn unknown_app_keeps_original() {
        assert_eq!(apply("weird-app-xyz", &default_rules()), "weird-app-xyz");
    }

    #[test]
    fn first_matching_rule_wins() {
        let rules = vec![
            rule(MatchType::Contains, "foo", "第一个"),
            rule(MatchType::Contains, "foo", "第二个"),
        ];
        assert_eq!(apply("foobar", &rules), "第一个");
    }

    #[test]
    fn exact_match_does_not_prefix_match() {
        let rules = vec![rule(MatchType::Exact, "kitty", "精确终端")];
        assert_eq!(apply("kitty-dir", &rules), "kitty-dir");
    }

    #[test]
    fn regex_rule_matches() {
        let rules = vec![MappingRule {
            match_type: MatchType::Regex,
            pattern: r"^chrom.*".to_string(),
            display: "浏览器".to_string(),
        }];
        assert_eq!(apply("chromium", &rules), "浏览器");
        assert_eq!(apply("firefox", &rules), "firefox");
    }

    #[test]
    fn empty_app_falls_back_to_desktop() {
        assert_eq!(apply("", &default_rules()), "桌面");
    }
}
