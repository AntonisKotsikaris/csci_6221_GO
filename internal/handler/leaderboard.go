package handler

import (
	"encoding/json"
	"net/http"
	"sort"

	"gollama/internal"
	"gollama/internal/pool"
)

// HandleLeaderboard returns a sorted list of workers by performance
func HandleLeaderboard(p *pool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get worker stats
		statsMap := p.GetWorkerStats()

		// Convert map to slice for sorting
		workers := make([]internal.WorkerStats, 0, len(statsMap))
		for _, stats := range statsMap {
			workers = append(workers, stats)
		}

		// Sort by average response time (ascending - lower is better)
		// Workers with no requests go to the end
		sort.Slice(workers, func(i, j int) bool {
			// If one worker has no requests, put it after workers with requests
			if workers[i].Requests == 0 && workers[j].Requests > 0 {
				return false
			}
			if workers[i].Requests > 0 && workers[j].Requests == 0 {
				return true
			}
			// If both have requests, sort by average response time
			if workers[i].Requests > 0 && workers[j].Requests > 0 {
				return workers[i].AvgResponseMS < workers[j].AvgResponseMS
			}
			// If both have no requests, sort by URL for consistency
			return workers[i].URL < workers[j].URL
		})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(workers)
	}
}
