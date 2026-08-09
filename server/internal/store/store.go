package store

import (
	"sort"
	"sync"
	"time"
)

// DeviceState is the live state of one reporting device.
type DeviceState struct {
	DeviceID     string    `json:"device_id"`
	DeviceName   string    `json:"device_name"`
	Platform     string    `json:"platform"`
	Distro       string    `json:"distro,omitempty"`
	App          string    `json:"app"`
	WindowTitle  string    `json:"window_title,omitempty"`
	AppStartedAt time.Time `json:"app_started_at"`
	LastSeen     time.Time `json:"last_seen"`
	Online       bool      `json:"online"`
}

// Store keeps the latest reported state per device in memory.
type Store struct {
	mu         sync.Mutex
	states     map[string]DeviceState
	lastOnline map[string]bool
	idle       time.Duration
	now        func() time.Time
}

// New returns a Store that marks devices offline after idle.
func New(idle time.Duration) *Store {
	return &Store{
		states:     make(map[string]DeviceState),
		lastOnline: make(map[string]bool),
		idle:       idle,
		now:        time.Now,
	}
}

// SetIdle updates the offline threshold at runtime (admin panel).
func (s *Store) SetIdle(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.idle = d
}

// Upsert records a report and returns the resulting state.
func (s *Store) Upsert(d DeviceState) DeviceState {
	s.mu.Lock()
	defer s.mu.Unlock()
	d.LastSeen = s.now()
	d.Online = true
	s.states[d.DeviceID] = d
	s.lastOnline[d.DeviceID] = true
	return d
}

// List returns all device states with online/offline computed against now,
// ordered by device ID.
func (s *Store) List() []DeviceState {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	out := make([]DeviceState, 0, len(s.states))
	for _, d := range s.states {
		d.Online = now.Sub(d.LastSeen) <= s.idle
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DeviceID < out[j].DeviceID })
	return out
}

// ListChanges returns device states whose online flag flipped since the last
// call, updating internal bookkeeping. Used by the offline-transition detector
// so viewers get pushed when a device goes offline without a new report.
func (s *Store) ListChanges() []DeviceState {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	var out []DeviceState
	for id, d := range s.states {
		online := now.Sub(d.LastSeen) <= s.idle
		if s.lastOnline[id] != online {
			d.Online = online
			out = append(out, d)
		}
		s.lastOnline[id] = online
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DeviceID < out[j].DeviceID })
	return out
}
