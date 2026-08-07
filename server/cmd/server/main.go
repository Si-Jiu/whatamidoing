package main

import (
	"log"
	"net/http"

	"whatamidoing/server/internal/api"
	"whatamidoing/server/internal/config"
	"whatamidoing/server/internal/hub"
	"whatamidoing/server/internal/store"
	"whatamidoing/server/web"
)

func main() {
	cfg := config.FromEnv()
	if cfg.ReportToken == "" {
		log.Fatal("REPORT_TOKEN 环境变量未设置")
	}

	st := store.New(cfg.IdleTimeout)
	h := hub.New()
	handler := api.New(cfg, st, h, web.Files)

	addr := ":" + cfg.Port
	passwordState := "未启用"
	if cfg.ViewerPassword != "" {
		passwordState = "已启用"
	}
	log.Printf("whatamidoing server 监听 %s (查看密码 %s)", addr, passwordState)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
