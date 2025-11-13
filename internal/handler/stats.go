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

/*
formatWorkerStats adds computed fields like uptime duration
TODO: Add composite scoring system:
  - Success rate (jobs_completed / total_jobs)
  - Throughput (jobs per minute)
  - Speed score (average response time)
  - Reliability score (uptime + consistency)
  - Token processing rate (tokens/second)
*/
func formatWorkerStats(stats map[string]internal.WorkerStats) map[string]interface{} {
	formatted := make(map[string]interface{})

	for url, stat := range stats {
		uptime := time.Since(stat.StartTime)
		score := calculateWorkerScore(stat, uptime)

		formatted[url] = map[string]interface{}{
			"jobs_completed": stat.JobsCompleted,
			"jobs_failed":    stat.JobsFailed,
			"uptime_seconds": int(uptime.Seconds()),
			"uptime_pretty":  uptime.Round(time.Second).String(),
			"start_time":     stat.StartTime.Format(time.RFC3339),
			"score":          score,
		}
	}

	return formatted
}

/*
calculateWorkerScore computes a simple weighted performance score (0-100)
Magic constants explained:
- 0.7: Success rate weight (70%) - prioritizes reliability over speed
- 0.3: Throughput weight (30%) - rewards productive workers
- 10.0: Throughput normalizer - assumes 10 jobs/min is "excellent" throughput
- 60.0: Minimum uptime (seconds) - prevents division by zero, gives new workers a chance
*/
func calculateWorkerScore(stat internal.WorkerStats, uptime time.Duration) float64 {
	totalJobs := stat.JobsCompleted + stat.JobsFailed
	if totalJobs == 0 {
		return 50.0 // Neutral score for workers with no jobs yet
	}

	successRate := float64(stat.JobsCompleted) / float64(totalJobs)

	uptimeMinutes := uptime.Minutes()
	// Use minimum 1 minute to avoid extreme throughput scores
	// TODO: however... we probably want to be "cautious" of very short worker uptimes.
	//     for example, it would be very easy to basically do a DOS style attack by signing up lots of temporary
	//     workers and immediately killing them when they get in a request.
	if uptimeMinutes < 1.0 {
		uptimeMinutes = 1.0
	}
	throughput := float64(stat.JobsCompleted) / uptimeMinutes

	throughputScore := throughput / 10.0 // 10 jobs/min = perfect throughput score
	if throughputScore > 1.0 {
		throughputScore = 1.0 // Cap at 100%
	}

	score := (successRate * 0.7) + (throughputScore * 0.3)
	return score * 100.0 // Convert to 0-100 scale
}
