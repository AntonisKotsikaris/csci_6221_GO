package handler

import (
	"encoding/json"
	"net/http"

	"gollama/internal/pool"
)

/*
HandleHealth provides a health check endpoint
*/
func HandleHealth(p *pool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"status":  http.StatusOK,
			"workers": p.GetWorkerCount(),
		}
		_ = json.NewEncoder(w).Encode(response)
	}
}
