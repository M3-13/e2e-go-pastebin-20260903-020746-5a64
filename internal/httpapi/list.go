package httpapi

import (
	"net/http"
	"time"

	"pastebin/internal/store"
)

type pasteMeta struct {
	ID        string     `json:"id"`
	Language  string     `json:"language"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func ListHandler(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pastes := s.List()

		out := make([]pasteMeta, 0, len(pastes))
		for _, p := range pastes {
			out = append(out, pasteMeta{
				ID:        p.ID,
				Language:  p.Language,
				CreatedAt: p.CreatedAt,
				ExpiresAt: p.ExpiresAt,
			})
		}

		writeJSON(w, http.StatusOK, out)
	}
}
