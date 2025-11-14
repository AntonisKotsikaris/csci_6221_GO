package handler

import (
	"encoding/json"
	"net/http"

	"gollama/internal/pool"
)

func HandleStats(p *pool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		stats := p.GetStats()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(stats); err != nil {
			http.Error(w, "failed to encode stats", http.StatusInternalServerError)
			return
		}
	}
}
