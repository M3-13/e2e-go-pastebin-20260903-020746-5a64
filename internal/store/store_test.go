package store

import (
	"testing"
	"time"
)

func TestCreateAndGet(t *testing.T) {
	s := New()
	p, err := s.Create("hello", "text", 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if p.Content != "hello" {
		t.Fatalf("content = %q, want hello", p.Content)
	}
	if p.Language != "text" {
		t.Fatalf("language = %q, want text", p.Language)
	}
	if p.ExpiresAt != nil {
		t.Fatal("expected nil ExpiresAt for expiresIn 0")
	}
	if !p.CreatedAt.Before(time.Now().Add(time.Second)) {
		t.Fatal("CreatedAt should be in the past")
	}

	got, ok := s.Get(p.ID)
	if !ok {
		t.Fatal("Get returned false for existing paste")
	}
	if got.ID != p.ID || got.Content != "hello" {
		t.Fatalf("Get mismatch: %+v", got)
	}
}

func TestGetUnknown(t *testing.T) {
	s := New()
	if _, ok := s.Get("missing"); ok {
		t.Fatal("Get should return false for unknown id")
	}
}

func TestList(t *testing.T) {
	s := New()
	if _, err := s.Create("a", "text", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create("b", "text", 0); err != nil {
		t.Fatal(err)
	}
	got := s.List()
	if len(got) != 2 {
		t.Fatalf("List length = %d, want 2", len(got))
	}
	for _, p := range got {
		if p.Content == "" {
			t.Fatal("List returned paste with empty content")
		}
	}
}

func TestDelete(t *testing.T) {
	s := New()
	p, err := s.Create("x", "text", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Delete(p.ID) {
		t.Fatal("Delete returned false for existing paste")
	}
	if s.Delete(p.ID) {
		t.Fatal("Delete returned true for already-deleted paste")
	}
	if _, ok := s.Get(p.ID); ok {
		t.Fatal("Get returned true after delete")
	}
	if s.Len() != 0 {
		t.Fatalf("Len = %d, want 0", s.Len())
	}
}

func TestDeleteUnknown(t *testing.T) {
	s := New()
	if s.Delete("missing") {
		t.Fatal("Delete should return false for unknown id")
	}
}

func TestExpiry(t *testing.T) {
	s := New()
	p, err := s.Create("x", "text", 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if p.ExpiresAt == nil {
		t.Fatal("expected non-nil ExpiresAt")
	}
	if _, ok := s.Get(p.ID); !ok {
		t.Fatal("paste should exist before expiry")
	}

	time.Sleep(20 * time.Millisecond)

	if _, ok := s.Get(p.ID); ok {
		t.Fatal("paste should be gone after expiry")
	}
	// expired paste is removed from the store
	if s.Len() != 0 {
		t.Fatalf("Len = %d after expiry, want 0", s.Len())
	}
	if s.Delete(p.ID) {
		t.Fatal("Delete should return false for expired paste")
	}
}

func TestListExcludesExpired(t *testing.T) {
	s := New()
	if _, err := s.Create("expired", "text", 10*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create("live", "text", time.Hour); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)

	got := s.List()
	if len(got) != 1 {
		t.Fatalf("List length = %d, want 1", len(got))
	}
	if got[0].Content != "live" {
		t.Fatalf("expected only live paste, got %+v", got)
	}
}

func TestMaxPastes(t *testing.T) {
	s := New()
	for i := 0; i < MaxPastes; i++ {
		if _, err := s.Create("x", "text", 0); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}
	if s.Len() != MaxPastes {
		t.Fatalf("Len = %d, want %d", s.Len(), MaxPastes)
	}
	if _, err := s.Create("x", "text", 0); err != ErrStoreFull {
		t.Fatalf("expected ErrStoreFull, got %v", err)
	}
}

func TestIDEntropy(t *testing.T) {
	s := New()
	seen := make(map[string]bool)
	const n = 100
	for i := 0; i < n; i++ {
		p, err := s.Create("x", "text", 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(p.ID) != 22 {
			t.Fatalf("ID length = %d, want 22 (base64.RawURLEncoding of 16 bytes)", len(p.ID))
		}
		if seen[p.ID] {
			t.Fatal("duplicate ID generated")
		}
		seen[p.ID] = true
	}
}
