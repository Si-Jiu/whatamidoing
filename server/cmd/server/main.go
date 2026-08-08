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
	handler := api.New(cfg, st, h, ds, web.Files)

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
