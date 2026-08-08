package main

import (
	"crypto/rand"
	"log"
	"net/http"

	"whatamidoing/server/internal/api"
	"whatamidoing/server/internal/config"
	"whatamidoing/server/internal/data"
	"whatamidoing/server/internal/hub"
	"whatamidoing/server/internal/store"
	"whatamidoing/server/web"
)

func main() {
	cfg := config.FromEnv()

	st := store.New(cfg.IdleTimeout)
	ds, err := data.New(cfg.DataFile)
	if err != nil {
		log.Fatalf("初始化数据存储失败: %v", err)
	}
	h := hub.New()

	// 首次初始化管理员所需的令牌：未配置 SETUP_TOKEN 时随机生成并打印到日志。
	setupToken := cfg.SetupToken
	if setupToken == "" {
		setupToken = rand.Text()
	}
	if !ds.IsAdminInitialized() {
		log.Printf("⚠️  首次设置管理员需要初始化令牌：%s（设 SETUP_TOKEN 可自定；初始化后失效）", setupToken)
	}

	handler := api.New(cfg, st, h, ds, setupToken, web.Files)

	addr := ":" + cfg.Port
	adminState := "未初始化"
	if ds.IsAdminInitialized() {
		adminState = "已初始化"
	}
	log.Printf("whatamidoing server 监听 %s (管理员 %s, 数据文件 %s)", addr, adminState, cfg.DataFile)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
