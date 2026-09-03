package httpapi

import (
	"net/http"

	"pastebin/internal/store"
)

func GetHandler(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		p, ok := s.Get(id)
		if !ok {
			writeError(w, http.StatusNotFound, "paste not found")
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}
