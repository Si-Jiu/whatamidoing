package main

import (
	"crypto/rand"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"whatamidoing/server/internal/api"
	"whatamidoing/server/internal/config"
	"whatamidoing/server/internal/data"
	"whatamidoing/server/internal/hub"
	"whatamidoing/server/internal/store"
	"whatamidoing/server/web"
)

func main() {
	cfg := config.FromEnv()

	ds, err := data.New(cfg.DataFile)
	if err != nil {
		log.Fatalf("初始化数据存储失败: %v", err)
	}
	// 离线阈值：管理面板持久化的值优先，否则用环境变量默认。
	idle := cfg.IdleTimeout
	if secs := ds.IdleTimeout(); secs > 0 {
		idle = time.Duration(secs) * time.Second
	}
	st := store.New(idle)
	h := hub.New()

	// 初始化令牌：管理员未初始化时，可用 SETUP_TOKEN 环境变量固定，否则程序自动生成并打印到日志。
	setupToken := cfg.SetupToken
	if !ds.IsAdminInitialized() {
		if setupToken == "" {
			setupToken = "setup_" + rand.Text()
			log.Printf("首次初始化管理员需要初始化令牌：%s（在初始化页面填写；也可用 SETUP_TOKEN 环境变量固定）", setupToken)
		}
	}

	// 开发模式：从磁盘目录提供前端静态文件，并注入自动刷新脚本——文件一改页面立即重载。
	var assets fs.FS = web.Files
	if devDir := os.Getenv("WAID_DEV_WEB_DIR"); devDir != "" {
		assets = os.DirFS(devDir)
		log.Printf("开发模式：前端静态资源来自 %s（修改即时生效，页面自动刷新）", devDir)
	}

	handler := api.New(cfg, st, h, ds, setupToken, assets)
	if devDir := os.Getenv("WAID_DEV_WEB_DIR"); devDir != "" {
		handler = &devLiveReload{dir: devDir, next: handler}
	}

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
