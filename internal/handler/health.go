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

		stats := p.GetStats()

		response := map[string]interface{}{
			"status":          "healthy",
			"workers":         stats.TotalWorkers,
			"healthy workers": stats.HealthyWorkers,
			"busy workers":    stats.BusyWorkers,
			"pending jobs":    stats.PendingJobs,
		}
		_ = json.NewEncoder(w).Encode(response)
	}
}
