package config

import (
	"os"
	"time"
)

// Config holds all runtime configuration for the server.
type Config struct {
	Port           string
	ReportToken    string
	ViewerPassword string
	IdleTimeout    time.Duration
}

// FromEnv builds Config from environment variables.
//
//	PORT            监听端口，默认 8080
//	REPORT_TOKEN    设备上报 token，必填
//	VIEWER_PASSWORD 可选，设置后查看网页需登录
//	IDLE_TIMEOUT    离线判定阈值，默认 30s（如 "45s"）
func FromEnv() Config {
	return Config{
		Port:           getenv("PORT", "8080"),
		ReportToken:    os.Getenv("REPORT_TOKEN"),
		ViewerPassword: os.Getenv("VIEWER_PASSWORD"),
		IdleTimeout:    getDuration("IDLE_TIMEOUT", 30*time.Second),
	}
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
