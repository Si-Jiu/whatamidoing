package config

import (
	"os"
	"strings"
	"time"
)

// Config holds all runtime configuration for the server.
type Config struct {
	Port           string
	IdleTimeout    time.Duration
	DataFile       string
	SetupToken     string
	TrustedProxies []string
}

// FromEnv builds Config from environment variables.
//
//	PORT           监听端口，默认 8080
//	IDLE_TIMEOUT   离线判定阈值，默认 30s（如 "45s"）
//	DATA_FILE      持久化数据文件路径（管理员/设备/token），默认 data.json
//	SETUP_TOKEN    必填（首次初始化管理员时）：初始化令牌，见 README
//	TRUSTED_PROXIES 可选：可信反向代理 IP/CIDR（逗号分隔）。设置后限流才信任
//	                  X-Forwarded-For 来源 IP；否则一律用直连对端 IP（防伪造绕过限流）
//
// 管理员密码、网页密码、设备 token 均在管理面板中管理并持久化到 DATA_FILE，
// 不再需要 REPORT_TOKEN / VIEWER_PASSWORD 环境变量。
func FromEnv() Config {
	return Config{
		Port:           getenv("PORT", "8080"),
		IdleTimeout:    getDuration("IDLE_TIMEOUT", 30*time.Second),
		DataFile:       getenv("DATA_FILE", "data.json"),
		SetupToken:     os.Getenv("SETUP_TOKEN"),
		TrustedProxies: splitList(os.Getenv("TRUSTED_PROXIES")),
	}
}

func splitList(v string) []string {
	var out []string
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return def
}
