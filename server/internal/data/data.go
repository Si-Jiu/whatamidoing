package data

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// Device is a registered device managed in the admin panel.
// The per-device Token is what the client uses as the report credential.
type Device struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform,omitempty"`
	Distro   string `json:"distro,omitempty"`
	Token    string `json:"token"`
}

// Data is the persisted server state.
type Data struct {
	AdminInitialized   bool     `json:"admin_initialized"`
	AdminPasswordHash  string   `json:"admin_password_hash,omitempty"`
	ViewerPasswordHash string   `json:"viewer_password_hash,omitempty"`
	Devices            []Device `json:"devices"`
}

// Store persists Data to a JSON file with a mutex and atomic writes.
type Store struct {
	mu   sync.Mutex
	path string
	d    Data
}

// New loads the store from path, creating an empty file if absent.
func New(path string) (*Store, error) {
	s := &Store{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		s.d = Data{Devices: []Device{}}
		return s.save()
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &s.d); err != nil {
		return err
	}
	if s.d.Devices == nil {
		s.d.Devices = []Device{}
	}
	return nil
}

func (s *Store) save() error {
	if dir := filepath.Dir(s.path); dir != "" {
		// 数据含密码 hash 与设备 token，目录仅属主可读写。
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	b, err := json.MarshalIndent(&s.d, "", "  ")
	if err != nil {
		return err
	}
	// 原子写：先写临时文件再改名，避免进程中断损坏数据。
	// 0o600 仅属主可读写，防止同机其他用户读取凭据。
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// --- admin ---

func (s *Store) IsAdminInitialized() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.d.AdminInitialized
}

// SetAdminPassword records the admin password hash and marks setup complete.
func (s *Store) SetAdminPassword(hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.d.AdminPasswordHash = hash
	s.d.AdminInitialized = true
	return s.save()
}

func (s *Store) AdminPasswordHash() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.d.AdminPasswordHash
}

// --- viewer password ---

func (s *Store) ViewerPasswordHash() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.d.ViewerPasswordHash
}

// SetViewerPasswordHash stores the viewer password hash; empty clears it.
func (s *Store) SetViewerPasswordHash(hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.d.ViewerPasswordHash = hash
	return s.save()
}

// --- devices ---

func (s *Store) Devices() []Device {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Device, len(s.d.Devices))
	copy(out, s.d.Devices)
	return out
}

// AddDevice registers a new device with a freshly generated token.
func (s *Store) AddDevice(name, platform, distro string) (Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dev := Device{
		ID:       "dev_" + rand.Text(),
		Name:     name,
		Platform: platform,
		Distro:   distro,
		Token:    "tok_" + rand.Text(),
	}
	s.d.Devices = append(s.d.Devices, dev)
	if err := s.save(); err != nil {
		return Device{}, err
	}
	return dev, nil
}

func (s *Store) RemoveDevice(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.d.Devices[:0]
	removed := false
	for _, d := range s.d.Devices {
		if d.ID == id {
			removed = true
			continue
		}
		out = append(out, d)
	}
	if !removed {
		return errors.New("设备不存在")
	}
	s.d.Devices = out
	return s.save()
}

// DeviceByToken resolves a report credential to a registered device.
func (s *Store) DeviceByToken(token string) (Device, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.d.Devices {
		if d.Token == token {
			return d, true
		}
	}
	return Device{}, false
}
