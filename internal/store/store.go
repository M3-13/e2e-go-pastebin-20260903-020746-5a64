package store

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

type Paste struct {
	ID        string     `json:"id"`
	Content   string     `json:"content"`
	Language  string     `json:"language"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
}

const MaxPastes = 1000

var ErrStoreFull = errors.New("store full")

type Store struct {
	mu     sync.RWMutex
	pastes map[string]Paste
}

func New() *Store {
	return &Store{
		pastes: make(map[string]Paste),
	}
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Store) Create(content, language string, expiresIn time.Duration) (Paste, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.pastes) >= MaxPastes {
		return Paste{}, ErrStoreFull
	}

	id, err := generateID()
	if err != nil {
		return Paste{}, err
	}

	now := time.Now()
	p := Paste{
		ID:        id,
		Content:   content,
		Language:  language,
		CreatedAt: now,
	}

	if expiresIn > 0 {
		exp := now.Add(expiresIn)
		p.ExpiresAt = &exp
	}

	s.pastes[id] = p
	return p, nil
}

func (s *Store) Get(id string) (Paste, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.pastes[id]
	if !ok {
		return Paste{}, false
	}

	if p.ExpiresAt != nil && time.Now().After(*p.ExpiresAt) {
		delete(s.pastes, id)
		return Paste{}, false
	}

	return p, true
}

func (s *Store) List() []Paste {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	out := make([]Paste, 0, len(s.pastes))
	for id, p := range s.pastes {
		if p.ExpiresAt != nil && now.After(*p.ExpiresAt) {
			delete(s.pastes, id)
			continue
		}
		out = append(out, p)
	}
	return out
}

func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.pastes[id]
	if !ok {
		return false
	}

	if p.ExpiresAt != nil && time.Now().After(*p.ExpiresAt) {
		delete(s.pastes, id)
		return false
	}

	delete(s.pastes, id)
	return true
}

func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pastes)
}
