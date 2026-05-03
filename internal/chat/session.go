package chat

import (
	"sync"
	"time"

	"vekta/internal/model"
)

const (
	defaultMaxMessages = 20
	defaultTTL         = 45 * time.Minute
)

// Message adalah satu unit percakapan dalam session.
type Message = model.ChatMessage

// Session menyimpan history percakapan satu user/koneksi.
// Thread-safe: setiap session punya mutex sendiri.
type Session struct {
	mu        sync.Mutex
	ID        string
	Messages  []Message
	CreatedAt time.Time
	UpdatedAt time.Time
	MaxMsg    int
}

func newSession(id string, maxMsg int) *Session {
	if maxMsg <= 0 {
		maxMsg = defaultMaxMessages
	}
	return &Session{
		ID:        id,
		Messages:  make([]Message, 0, maxMsg),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		MaxMsg:    maxMsg,
	}
}

// Add menambah pesan ke session dan trim jika melebihi MaxMsg.
// Thread-safe.
func (s *Session) Add(role, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Messages = append(s.Messages, Message{Role: role, Content: content})
	s.UpdatedAt = time.Now()

	if len(s.Messages) > s.MaxMsg {
		if len(s.Messages) > 0 && s.Messages[0].Role == "system" {
			s.Messages = append(s.Messages[:1], s.Messages[2:]...)
		} else {
			s.Messages = s.Messages[1:]
		}
	}
}

// History mengembalikan copy messages — aman untuk dikirim ke model.
func (s *Session) History() []Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Message, len(s.Messages))
	copy(out, s.Messages)
	return out
}

// Manager adalah registry semua session aktif.
// Thread-safe via RWMutex.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	ttl      time.Duration
	maxMsg   int
}

func NewManager(ttl time.Duration, maxMsg int) *Manager {
	if ttl <= 0 {
		ttl = defaultTTL
	}
	m := &Manager{
		sessions: make(map[string]*Session),
		ttl:      ttl,
		maxMsg:   maxMsg,
	}
	go m.runEviction()
	return m
}

// Get mengambil session by ID, buat baru jika belum ada.
func (m *Manager) Get(id string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.sessions[id]; ok {
		return s
	}
	s := newSession(id, m.maxMsg)
	m.sessions[id] = s
	return s
}

// Delete menghapus session secara eksplisit.
func (m *Manager) Delete(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
}

func (m *Manager) runEviction() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		m.evict()
	}
}

func (m *Manager) evict() {
	m.mu.Lock()
	defer m.mu.Unlock()
	cutoff := time.Now().Add(-m.ttl)
	for id, s := range m.sessions {
		s.mu.Lock()
		updated := s.UpdatedAt
		s.mu.Unlock()
		if updated.Before(cutoff) {
			delete(m.sessions, id)
		}
	}
}

// Count mengembalikan jumlah session aktif.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}
