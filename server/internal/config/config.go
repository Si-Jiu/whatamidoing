package config

import (
	"os"
	"time"
)

// Config holds all runtime configuration for the server.
type Config struct {
	Port        string
	IdleTimeout time.Duration
	DataFile    string
	SetupToken  string
}

// FromEnv builds Config from environment variables.
//
//	PORT         监听端口，默认 8080
//	IDLE_TIMEOUT 离线判定阈值，默认 30s（如 "45s"）
//	DATA_FILE    持久化数据文件路径（管理员/设备/token），默认 data.json
//	SETUP_TOKEN  可选：首次初始化管理员的令牌；未设置时服务端随机生成并打印到启动日志
//
// 管理员密码、网页密码、设备 token 均在管理面板中管理并持久化到 DATA_FILE，
// 不再需要 REPORT_TOKEN / VIEWER_PASSWORD 环境变量。
func FromEnv() Config {
	return Config{
		Port:        getenv("PORT", "8080"),
		IdleTimeout: getDuration("IDLE_TIMEOUT", 30*time.Second),
		DataFile:    getenv("DATA_FILE", "data.json"),
		SetupToken:  os.Getenv("SETUP_TOKEN"),
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
