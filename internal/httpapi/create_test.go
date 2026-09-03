package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pastebin/internal/store"
)

func doCreate(t *testing.T, s *store.Store, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/pastes", strings.NewReader(body))
	rec := httptest.NewRecorder()
	CreateHandler(s)(rec, req)
	return rec
}

func TestCreateValidReturns201MetadataOnly(t *testing.T) {
	s := store.New()
	rec := doCreate(t, s, `{"content":"hello world","language":"text","expires_in_seconds":60}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var resp struct {
		ID        string     `json:"id"`
		Language  string     `json:"language"`
		CreatedAt time.Time  `json:"created_at"`
		ExpiresAt *time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID == "" {
		t.Fatal("id is empty")
	}
	if resp.Language != "text" {
		t.Fatalf("language = %q, want text", resp.Language)
	}
	if resp.CreatedAt.IsZero() {
		t.Fatal("created_at is zero")
	}
	if resp.ExpiresAt == nil {
		t.Fatal("expires_at is nil, want set")
	}

	if strings.Contains(rec.Body.String(), `"content"`) {
		t.Fatalf("response must not contain content: %s", rec.Body.String())
	}
}

func TestCreateNoExpiryReturnsNullExpiresAt(t *testing.T) {
	s := store.New()
	rec := doCreate(t, s, `{"content":"hello"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		ExpiresAt *time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ExpiresAt != nil {
		t.Fatalf("expires_at = %v, want nil", resp.ExpiresAt)
	}
}

func TestCreateInvalidJSONReturns400(t *testing.T) {
	s := store.New()
	rec := doCreate(t, s, `{"content": "not closed"`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"error"`) {
		t.Fatalf("body missing error field: %s", rec.Body.String())
	}
}

func TestCreateEmptyContentReturns400(t *testing.T) {
	s := store.New()
	rec := doCreate(t, s, `{"content":""}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"error"`) {
		t.Fatalf("body missing error field: %s", rec.Body.String())
	}
}

func TestCreateMissingContentReturns400(t *testing.T) {
	s := store.New()
	rec := doCreate(t, s, `{"language":"text"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateNegativeExpiresReturns400(t *testing.T) {
	s := store.New()
	rec := doCreate(t, s, `{"content":"hello","expires_in_seconds":-1}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateZeroExpiresReturns400(t *testing.T) {
	s := store.New()
	rec := doCreate(t, s, `{"content":"hello","expires_in_seconds":0}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateNonNumericExpiresReturns400(t *testing.T) {
	s := store.New()
	rec := doCreate(t, s, `{"content":"hello","expires_in_seconds":"abc"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCreateBodyOver1MiBReturns413(t *testing.T) {
	s := store.New()
	body := `{"content":"` + strings.Repeat("a", 1<<20) + `"}`
	rec := doCreate(t, s, body)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"error"`) {
		t.Fatalf("body missing error field: %s", rec.Body.String())
	}
}

func TestCreateStoreFullReturns503(t *testing.T) {
	s := store.New()
	for i := 0; i < store.MaxPastes; i++ {
		if _, err := s.Create("x", "", 0); err != nil {
			t.Fatalf("failed to fill store: %v", err)
		}
	}

	rec := doCreate(t, s, `{"content":"hello"}`)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"error"`) {
		t.Fatalf("body missing error field: %s", rec.Body.String())
	}
}

func TestCreateErrorsDoNotContainContent(t *testing.T) {
	s := store.New()
	for i := 0; i < store.MaxPastes; i++ {
		if _, err := s.Create("x", "", 0); err != nil {
			t.Fatalf("failed to fill store: %v", err)
		}
	}

	cases := []struct {
		name string
		body string
	}{
		{"invalid json", `{"content": "secret`},
		{"empty content", `{"content":""}`},
		{"negative expires", `{"content":"secret-content","expires_in_seconds":-1}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := doCreate(t, s, c.body)
			if strings.Contains(rec.Body.String(), "secret") {
				t.Fatalf("error response leaked content: %s", rec.Body.String())
			}
		})
	}
}
