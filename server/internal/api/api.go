package api

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/netip"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"

	"whatamidoing/server/internal/config"
	"whatamidoing/server/internal/data"
	"whatamidoing/server/internal/hub"
	"whatamidoing/server/internal/store"
)

var upgrader = websocket.Upgrader{
	// 仅允许同源连接（或非浏览器客户端，无 Origin 头）。
	// 防止跨站 WebSocket 劫持：攻击站点无法借用户的会话建立 WS。
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // 非浏览器客户端
		}
		return origin == "http://"+r.Host || origin == "https://"+r.Host
	},
}

var validPlatforms = map[string]bool{
	"windows": true,
	"macos":   true,
	"linux":   true,
	"android": true,
}

// Server bundles the HTTP handlers and their dependencies.
type Server struct {
	cfg            config.Config
	store          *store.Store // 实时状态（内存）
	data           *data.Store  // 持久化配置（管理员/设备/token）
	hub            *hub.Hub
	viewerSessions *sessionStore
	adminSessions  *sessionStore
	setupToken     string
	authThrottle   *ipThrottle    // 认证端点防暴力破解
	trustedProxies []netip.Prefix // 可信反向代理（限流时才信任 X-Forwarded-For）
}

// New builds the HTTP handler for the whole server.
func New(cfg config.Config, st *store.Store, h *hub.Hub, ds *data.Store, setupToken string, assets fs.FS) http.Handler {
	s := &Server{
		cfg:            cfg,
		store:          st,
		data:           ds,
		hub:            h,
		viewerSessions: newSessionStore(),
		adminSessions:  newSessionStore(),
		setupToken:     setupToken,
		authThrottle:   newIPThrottle(),
		trustedProxies: parseTrustedProxies(cfg.TrustedProxies),
	}
	go s.runOfflineDetector()

	mux := http.NewServeMux()
	// 管理员
	mux.HandleFunc("GET /api/admin/status", s.handleAdminStatus)
	mux.HandleFunc("POST /api/admin/setup", s.handleAdminSetup)
	mux.HandleFunc("POST /api/admin/login", s.handleAdminLogin)
	mux.HandleFunc("GET /api/admin/devices", s.requireAdmin(s.handleAdminDevices))
	mux.HandleFunc("POST /api/admin/devices", s.requireAdmin(s.handleAdminAddDevice))
	mux.HandleFunc("DELETE /api/admin/devices/{id}", s.requireAdmin(s.handleAdminDeleteDevice))
	mux.HandleFunc("POST /api/admin/viewer-password", s.requireAdmin(s.handleAdminViewerPassword))
	// 设备上报 / 查看
	mux.HandleFunc("POST /api/v1/report", s.handleReport)
	mux.HandleFunc("GET /api/v1/state", s.requireViewer(s.handleState))
	mux.HandleFunc("GET /ws", s.requireViewer(s.handleWS))
	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("GET /healthz", s.handleHealth)
	// 静态资源不缓存：资源小且随版本变化，避免浏览器/CDN 缓存旧版
	mux.Handle("/", noCache(http.FileServer(http.FS(assets))))
	return mux
}

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		next.ServeHTTP(w, r)
	})
}

// --- 管理员 ---

func (s *Server) handleAdminStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"initialized": s.data.IsAdminInitialized()})
}

func (s *Server) handleAdminSetup(w http.ResponseWriter, r *http.Request) {
	if s.data.IsAdminInitialized() {
		writeError(w, http.StatusConflict, "管理员已初始化")
		return
	}
	if err := s.authThrottle.check(s.clientIP(r)); err != nil {
		writeError(w, http.StatusTooManyRequests, err.Error())
		return
	}
	var in struct {
		SetupToken string `json:"setup_token"`
		Password   string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if s.setupToken == "" || in.SetupToken != s.setupToken {
		s.authThrottle.fail(s.clientIP(r))
		writeError(w, http.StatusUnauthorized, "初始化令牌错误（见服务端启动日志）")
		return
	}
	if len(in.Password) < 6 {
		writeError(w, http.StatusBadRequest, "管理员密码至少 6 位")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "密码处理失败")
		return
	}
	if err := s.data.SetAdminPassword(string(hash)); err != nil {
		writeError(w, http.StatusInternalServerError, "保存失败")
		return
	}
	s.setAdminCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if !s.data.IsAdminInitialized() {
		writeError(w, http.StatusNotFound, "尚未初始化")
		return
	}
	if err := s.authThrottle.check(s.clientIP(r)); err != nil {
		writeError(w, http.StatusTooManyRequests, err.Error())
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	hash := s.data.AdminPasswordHash()
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		s.authThrottle.fail(s.clientIP(r))
		writeError(w, http.StatusUnauthorized, "密码错误")
		return
	}
	s.authThrottle.success(s.clientIP(r))
	s.setAdminCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAdminDevices(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"devices": s.data.Devices()})
}

func (s *Server) handleAdminAddDevice(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		Platform string `json:"platform"`
		Distro   string `json:"distro"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "设备名称不能为空")
		return
	}
	platform := strings.TrimSpace(in.Platform)
	if platform != "" && !validPlatforms[platform] && len(platform) > 24 {
		writeError(w, http.StatusBadRequest, "自定义设备类型最多 24 字符")
		return
	}
	distro := strings.TrimSpace(in.Distro)
	if len(distro) > 24 {
		writeError(w, http.StatusBadRequest, "发行版最多 24 字符")
		return
	}
	dev, err := s.data.AddDevice(name, platform, distro)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "保存失败")
		return
	}
	s.hub.Broadcast(stateMessage(s.allDevices()))
	writeJSON(w, http.StatusOK, dev)
}

func (s *Server) handleAdminDeleteDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.data.RemoveDevice(id); err != nil {
		writeError(w, http.StatusNotFound, "设备不存在")
		return
	}
	s.hub.Broadcast(stateMessage(s.allDevices()))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminViewerPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Password) == "" {
		if err := s.data.SetViewerPasswordHash(""); err != nil {
			writeError(w, http.StatusInternalServerError, "保存失败")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "密码处理失败")
		return
	}
	if err := s.data.SetViewerPasswordHash(string(hash)); err != nil {
		writeError(w, http.StatusInternalServerError, "保存失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// --- 设备上报 ---

type reportRequest struct {
	DeviceID     string     `json:"device_id"`
	DeviceName   string     `json:"device_name"`
	Platform     string     `json:"platform"`
	App          string     `json:"app"`
	WindowTitle  string     `json:"window_title"`
	AppStartedAt *time.Time `json:"app_started_at"`
}

func (r reportRequest) validate() error {
	if !validPlatforms[r.Platform] {
		return errors.New("platform 必须为 windows/macos/linux/android 之一")
	}
	if strings.TrimSpace(r.App) == "" {
		return errors.New("app 不能为空")
	}
	return nil
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	dev, ok := s.data.DeviceByToken(token)
	if !ok {
		writeError(w, http.StatusUnauthorized, "token 无效")
		return
	}
	var in reportRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := in.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	started := time.Now()
	if in.AppStartedAt != nil {
		started = *in.AppStartedAt
	}
	// 设备身份以管理面板注册的为准（token → 设备），忽略上报里的 device_id/name
	updated := s.store.Upsert(store.DeviceState{
		DeviceID:     dev.ID,
		DeviceName:   dev.Name,
		Platform:     in.Platform,
		App:          in.App,
		WindowTitle:  in.WindowTitle,
		AppStartedAt: started,
	})
	s.hub.Broadcast(updateMessage(updated))
	w.WriteHeader(http.StatusNoContent)
}

func bearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// --- 查看 ---

// allDevices 返回管理面板注册的所有设备及其实时状态（未上报的显示离线）。
// 平台/发行版以管理面板注册的为准（优先于客户端上报，离线时也能显示）。
func (s *Server) allDevices() []store.DeviceState {
	liveMap := make(map[string]store.DeviceState, 8)
	for _, d := range s.store.List() {
		liveMap[d.DeviceID] = d
	}
	devices := s.data.Devices()
	out := make([]store.DeviceState, 0, len(devices))
	for _, dev := range devices {
		if l, ok := liveMap[dev.ID]; ok {
			if dev.Platform != "" {
				l.Platform = dev.Platform
			}
			if dev.Distro != "" {
				l.Distro = dev.Distro
			}
			out = append(out, l)
		} else {
			out = append(out, store.DeviceState{
				DeviceID:   dev.ID,
				DeviceName: dev.Name,
				Platform:   dev.Platform,
				Distro:     dev.Distro,
				Online:     false,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DeviceID < out[j].DeviceID })
	return out
}

func (s *Server) handleState(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"devices": s.allDevices()})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	hash := s.data.ViewerPasswordHash()
	if hash == "" {
		writeError(w, http.StatusNotFound, "未启用查看密码")
		return
	}
	if err := s.authThrottle.check(s.clientIP(r)); err != nil {
		writeError(w, http.StatusTooManyRequests, err.Error())
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		s.authThrottle.fail(s.clientIP(r))
		writeError(w, http.StatusUnauthorized, "密码错误")
		return
	}
	s.authThrottle.success(s.clientIP(r))
	s.setViewerCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &hub.Client{Conn: conn, Send: make(chan []byte, 16)}
	s.hub.Register(c)

	select {
	case c.Send <- stateMessage(s.allDevices()):
	default:
	}

	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for msg := range c.Send {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	conn.Close()
	s.hub.Unregister(c) // closes c.Send → writer goroutine exits
	<-writerDone
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// requireViewer guards an endpoint with viewer auth (no-op when no password set).
func (s *Server) requireViewer(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.data.ViewerPasswordHash() == "" {
			next(w, r)
			return
		}
		c, err := r.Cookie("viewer_session")
		if err != nil || !s.viewerSessions.valid(c.Value) {
			writeError(w, http.StatusUnauthorized, "需要登录")
			return
		}
		next(w, r)
	}
}

// requireAdmin guards an endpoint with admin session auth.
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("admin_session")
		if err != nil || !s.adminSessions.valid(c.Value) {
			writeError(w, http.StatusUnauthorized, "需要管理员登录")
			return
		}
		next(w, r)
	}
}

// runOfflineDetector periodically pushes online→offline transitions to viewers.
func (s *Server) runOfflineDetector() {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for range t.C {
		for _, d := range s.store.ListChanges() {
			s.hub.Broadcast(updateMessage(d))
		}
	}
}

// --- 会话 ---

type sessionStore struct {
	mu     sync.Mutex
	tokens map[string]time.Time
}

func newSessionStore() *sessionStore {
	return &sessionStore{tokens: make(map[string]time.Time)}
}

func (s *sessionStore) issue() string {
	// rand.Text 在随机源不可用时直接 panic，绝不静默降级为零值 token。
	tok := rand.Text()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[tok] = time.Now().Add(24 * time.Hour)
	return tok
}

func (s *sessionStore) valid(tok string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.tokens[tok]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(s.tokens, tok)
		return false
	}
	return true
}

func (s *Server) setViewerCookie(w http.ResponseWriter, r *http.Request) {
	setSessionCookie(w, r, "viewer_session", s.viewerSessions.issue())
}

func (s *Server) setAdminCookie(w http.ResponseWriter, r *http.Request) {
	setSessionCookie(w, r, "admin_session", s.adminSessions.issue())
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, name, value string) {
	// HTTPS 连接（含反代透传的 X-Forwarded-Proto）时带 Secure 标记；
	// 纯 HTTP/内网部署不设，否则浏览器不会发送该 cookie，登录会失效。
	secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((24 * time.Hour).Seconds()),
	})
}

// --- 认证限流（防暴力破解）---

const (
	maxAuthFails = 5
	authLockTime = 5 * time.Minute
)

// ipThrottle 按来源 IP 记录认证失败次数，超过阈值后临时锁定。
type ipThrottle struct {
	mu       sync.Mutex
	fails    map[string]int
	lockedAt map[string]time.Time
}

func newIPThrottle() *ipThrottle {
	return &ipThrottle{
		fails:    make(map[string]int),
		lockedAt: make(map[string]time.Time),
	}
}

// check 返回是否允许本次尝试；锁定中返回错误。
func (t *ipThrottle) check(ip string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if at, ok := t.lockedAt[ip]; ok {
		if time.Since(at) < authLockTime {
			return errors.New("尝试次数过多，请 5 分钟后再试")
		}
		delete(t.lockedAt, ip)
		delete(t.fails, ip)
	}
	return nil
}

func (t *ipThrottle) fail(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.fails[ip]++
	if t.fails[ip] >= maxAuthFails {
		t.lockedAt[ip] = time.Now()
		delete(t.fails, ip)
	}
}

func (t *ipThrottle) success(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.fails, ip)
	delete(t.lockedAt, ip)
}

func parseTrustedProxies(list []string) []netip.Prefix {
	var out []netip.Prefix
	for _, s := range list {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if p, err := netip.ParsePrefix(s); err == nil {
			out = append(out, p)
		} else if a, err := netip.ParseAddr(s); err == nil {
			out = append(out, netip.PrefixFrom(a, a.BitLen()))
		}
	}
	return out
}

func (s *Server) isTrustedProxy(ip string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	for _, p := range s.trustedProxies {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

// clientIP 取请求来源 IP 用于限流。
// 仅当直连对端是可信代理（TRUSTED_PROXIES）时才信任 X-Forwarded-For，
// 否则直接用对端 IP，防止攻击者伪造 XFF 绕过按 IP 的限流。
func (s *Server) clientIP(r *http.Request) string {
	remote := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		remote = host
	}
	if !s.isTrustedProxy(remote) {
		return remote
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	return remote
}

// --- message + JSON helpers ---

func stateMessage(devices []store.DeviceState) []byte {
	return mustJSON(map[string]any{"type": "state", "devices": devices})
}

func updateMessage(d store.DeviceState) []byte {
	return mustJSON(map[string]any{"type": "update", "device": d})
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("marshal: %v", err)
		return []byte("{}")
	}
	return b
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "请求体非法: "+err.Error())
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(mustJSON(v))
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
