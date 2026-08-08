package main

// 开发模式自动刷新：轮询 web 目录文件最新修改时间，变化即重载页面。
// 仅当设置了 WAID_DEV_WEB_DIR 时启用（main.go 中包装），生产构建不含此逻辑。

import (
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type devLiveReload struct {
	dir  string
	next http.Handler
}

func (d *devLiveReload) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/__livereload__" {
		t, err := d.latestMtime()
		if err != nil {
			log.Printf("livereload: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = io.WriteString(w, `{"mtime":`+itoa(t)+`}`)
		return
	}
	if r.URL.Path == "/" || r.URL.Path == "/index.html" {
		if b, err := os.ReadFile(filepath.Join(d.dir, "index.html")); err == nil {
			html := string(b)
			if !strings.Contains(html, "__livereload__") {
				html = strings.Replace(html, "</body>", livereloadScript+"</body>", 1)
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			_, _ = io.WriteString(w, html)
			return
		}
	}
	d.next.ServeHTTP(w, r)
}

// latestMtime 返回目录下全部文件的最新修改时间（Unix 毫秒）。
func (d *devLiveReload) latestMtime() (int64, error) {
	var latest int64
	err := filepath.WalkDir(d.dir, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() {
			return nil
		}
		fi, err := e.Info()
		if err != nil {
			return err
		}
		if t := fi.ModTime().UnixMilli(); t > latest {
			latest = t
		}
		return nil
	})
	return latest, err
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}

const livereloadScript = `<script>
(async function () {
  var last = null;
  while (true) {
    try {
      var r = await fetch("/__livereload__", { cache: "no-store" });
      var t = (await r.json()).mtime;
      if (last !== null && t !== last) location.reload();
      last = t;
    } catch (e) {}
    await new Promise(function (res) { setTimeout(res, 1000); });
  }
})();
</script>`
