package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"whatamidoing/server/internal/config"
	"whatamidoing/server/internal/hub"
	"whatamidoing/server/internal/store"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var validPlatforms = map[string]bool{
	"windows": true,
	"macos":   true,
	"linux":   true,
	"android": true,
}

// Server bundles the HTTP handlers and their dependencies.
type Server struct {
	cfg      config.Config
	store    *store.Store
	hub      *hub.Hub
	sessions *sessionStore
}

// New builds the HTTP handler for the whole server.
func New(cfg config.Config, st *store.Store, h *hub.Hub, assets fs.FS) http.Handler {
	s := &Server{
		cfg:      cfg,
		store:    st,
		hub:      h,
		sessions: newSessionStore(),
	}
	go s.runOfflineDetector()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/report", s.handleReport)
	mux.HandleFunc("GET /api/v1/state", s.requireViewer(s.handleState))
	mux.HandleFunc("GET /ws", s.requireViewer(s.handleWS))
	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.Handle("/", http.FileServer(http.FS(assets)))
	return mux
}

// --- report ---

type reportRequest struct {
	DeviceID     string     `json:"device_id"`
	DeviceName   string     `json:"device_name"`
	Platform     string     `json:"platform"`
	App          string     `json:"app"`
	WindowTitle  string     `json:"window_title"`
	AppStartedAt *time.Time `json:"app_started_at"`
}

func (r reportRequest) validate() error {
	if strings.TrimSpace(r.DeviceID) == "" {
		return errors.New("device_id 不能为空")
	}
	if strings.TrimSpace(r.DeviceName) == "" {
		return errors.New("device_name 不能为空")
	}
	if !validPlatforms[r.Platform] {
		return errors.New("platform 必须为 windows/macos/linux/android 之一")
	}
	if strings.TrimSpace(r.App) == "" {
		return errors.New("app 不能为空")
	}
	return nil
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	if !s.authorizedDevice(r) {
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
	updated := s.store.Upsert(store.DeviceState{
		DeviceID:     in.DeviceID,
		DeviceName:   in.DeviceName,
		Platform:     in.Platform,
		App:          in.App,
		WindowTitle:  in.WindowTitle,
		AppStartedAt: started,
	})
	s.hub.Broadcast(updateMessage(updated))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) authorizedDevice(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	return strings.HasPrefix(auth, "Bearer ") &&
		strings.TrimPrefix(auth, "Bearer ") == s.cfg.ReportToken
}

// --- state / login / ws ---

func (s *Server) handleState(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"devices": s.store.List()})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.cfg.ViewerPassword == "" {
		writeError(w, http.StatusNotFound, "未启用查看密码")
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Password != s.cfg.ViewerPassword {
		writeError(w, http.StatusUnauthorized, "密码错误")
		return
	}
	tok := s.sessions.issue()
	http.SetCookie(w, &http.Cookie{
		Name:     "viewer_session",
		Value:    tok,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((24 * time.Hour).Seconds()),
	})
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
	case c.Send <- stateMessage(s.store.List()):
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
		if s.cfg.ViewerPassword == "" {
			next(w, r)
			return
		}
		c, err := r.Cookie("viewer_session")
		if err != nil || !s.sessions.valid(c.Value) {
			writeError(w, http.StatusUnauthorized, "需要登录")
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

// --- viewer sessions ---

type sessionStore struct {
	mu     sync.Mutex
	tokens map[string]time.Time
}

func newSessionStore() *sessionStore {
	return &sessionStore{tokens: make(map[string]time.Time)}
}

func (s *sessionStore) issue() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		log.Printf("session: crypto/rand failed: %v", err)
	}
	tok := hex.EncodeToString(buf)
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
