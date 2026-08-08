package main

import (
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

	// 初始化令牌必须显式设置，绝不自动生成（避免凭证出现在日志里）。
	if !ds.IsAdminInitialized() && cfg.SetupToken == "" {
		log.Fatal("管理员尚未初始化：请先设置 SETUP_TOKEN 环境变量（首次初始化管理员需要），再启动")
	}

	handler := api.New(cfg, st, h, ds, cfg.SetupToken, web.Files)

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
