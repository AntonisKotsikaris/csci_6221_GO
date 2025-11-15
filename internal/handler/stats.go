package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"gollama/internal"
	"gollama/internal/pool"
)

/*
HandleStats provides worker statistics endpoint
*/
func HandleStats(p *pool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		stats := p.GetWorkerStats()

		response := map[string]interface{}{
			"total_workers": p.GetWorkerCount(),
			"workers":       formatWorkerStats(stats),
		}

		_ = json.NewEncoder(w).Encode(response)
	}
}

func formatWorkerStats(stats map[string]internal.WorkerStats) map[string]interface{} {
	formatted := make(map[string]interface{})

	for url, stat := range stats {
		uptime := time.Since(stat.StartTime)

		formatted[url] = map[string]interface{}{
			"jobs_completed": stat.JobsCompleted,
			"jobs_failed":    stat.JobsFailed,
			"uptime_seconds": int(uptime.Seconds()),
			"uptime_pretty":  uptime.Round(time.Second).String(),
			"start_time":     stat.StartTime.Format(time.RFC3339),
			"score":          calculateWorkerScore(stat, uptime),
		}
	}

	return formatted
}

/*
calculateWorkerScore computes a raw productivity score
Formula: jobs_completed × uptime_minutes
This rewards both speed AND reliability - workers who complete many jobs
and stay online longer get higher scores. No artificial caps or percentages.

TODO: This would probably better if we were logging tokens and time to complete tasks.
*/
func calculateWorkerScore(stat internal.WorkerStats, uptime time.Duration) float64 {
	if stat.JobsCompleted == 0 {
		return 0.0 // No productivity yet
	}

	// Raw productivity: jobs completed × minutes active
	return float64(stat.JobsCompleted) * uptime.Minutes()
}
