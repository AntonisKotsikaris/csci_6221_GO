package pool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gollama/internal"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

/*
Pool holds the last-known pool of workers. Workers are verified, used, or discarded as they're called upon by the
jobProcessor.
*/
type Pool struct {
	jobs        chan internal.WorkerJob          //job queue channel - send jobs messages to this channel
	workerStats map[string]*internal.WorkerStats // worker stats by URL
	workerOrder []string                         // ordered list of worker URLs for round-robin
	mu          sync.RWMutex                     // Protects worker data during concurrent calls
	nextIdx     int
}

/*
New creates a new worker pool
*/
func New(queueSize int) *Pool {
	return &Pool{
		jobs:        make(chan internal.WorkerJob, queueSize),
		workerStats: make(map[string]*internal.WorkerStats),
		workerOrder: make([]string, 0),
	}
}

/*
Start initializes the worker pool
*/
func (p *Pool) Start() {
	for i := 1; i <= 50; i++ { // 50 concurrent job processors
		go p.jobProcessor(i)
	}
	log.Println("Worker pool initialized with 50 job processors")
}

/*
jobProcessor handles jobs and manages worker health
NOTE: Under high load, some requests may get "stuck in limbo" - they appear to hang
while newer requests continue processing. This could be due to worker health check delays,
channel/goroutine leaks, or mutex contention from heavy stats logging.
*/
func (p *Pool) jobProcessor(id int) {
	for job := range p.jobs {
		log.Printf("[Processor %d] Processing job with worker %s", id, job.WorkerURL)

		result, latencyMS := p.callWorker(job.WorkerURL, job.Request)

		if isError(result) {
			log.Printf("[Processor %d] Worker %s failed, removing from pool", id, job.WorkerURL)
			p.updateWorkerStats(job.WorkerURL, false, 0)
			p.RemoveWorker(job.WorkerURL)

			// Retry the job with a different worker if retries are available
			if job.RetryCount < job.MaxRetries {
				job.RetryCount++
				job.WorkerURL = p.GetWorker() // Get a new worker
				if job.WorkerURL != "" {
					log.Printf("[Processor %d] Retrying job (attempt %d/%d) with worker %s",
						id, job.RetryCount, job.MaxRetries, job.WorkerURL)
					p.SubmitJob(job) // Requeue the job
					return           // Don't send response yet, let the retry handle it
				} else {
					log.Printf("[Processor %d] No workers available for retry", id)
					job.ReplyCh <- "Error: No available workers for retry"
				}
			} else {
				log.Printf("[Processor %d] Job exceeded max retries (%d), giving up", id, job.MaxRetries)
				job.ReplyCh <- "Error: Job failed after maximum retries"
			}
		} else {
			p.updateWorkerStats(job.WorkerURL, true, latencyMS)
			job.ReplyCh <- result
		}
	}
}

/*
callWorker sends an inference request to a worker via its standardized /execute endpoint
Returns the response text and the latency in milliseconds
*/
func (p *Pool) callWorker(workerURL string, req internal.LlamaRequest) (string, float64) {
	startTime := time.Now()

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Sprintf("Error marshaling request: %v", err), 0
	}

	/*
		TODO: right now this is hard-coded. We need to update this so that callWorker adds info about
			what endpoint we're trying to hit. So whether it's chat or otherwise.
			Should depend on the gollama endpoint.
	*/
	executeReq := map[string]interface{}{
		"endpoint": "/v1/chat/completions",
		"body":     json.RawMessage(jsonData),
	}

	executePayload, err := json.Marshal(executeReq)
	if err != nil {
		return fmt.Sprintf("Error marshaling execute request: %v", err), 0
	}

	resp, err := http.Post(
		fmt.Sprintf("%s/execute", workerURL),
		"application/json",
		bytes.NewBuffer(executePayload),
	)
	if err != nil {
		return fmt.Sprintf("Error contacting worker: %v", err), 0
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error reading response: %v", err), 0
	}

	var workerResp internal.LlamaResponse
	err = json.Unmarshal(body, &workerResp)
	if err != nil {
		return fmt.Sprintf("Error parsing response: %v", err), 0
	}

	if workerResp.Error != "" {
		return fmt.Sprintf("Worker error: %s", workerResp.Error), 0
	}

	if len(workerResp.Choices) == 0 {
		return "Worker error: no choices in response", 0
	}

	latencyMS := float64(time.Since(startTime).Microseconds()) / 1000.0
	return workerResp.Choices[0].Message.Content, latencyMS
}

/*
updateWorkerStats updates the statistics for a worker after job completion
*/
func (p *Pool) updateWorkerStats(url string, success bool, latencyMS float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	stats, exists := p.workerStats[url]
	if !exists {
		return // Worker not found, probably already removed
	}

	if success {
		stats.JobsCompleted++
		stats.Requests++
		stats.LastActive = time.Now()
		stats.Healthy = true

		// Update running average response time
		if stats.AvgResponseMS == 0 {
			stats.AvgResponseMS = latencyMS
		} else {
			// Calculate new average: (old_avg * old_count + new_value) / new_count
			stats.AvgResponseMS = ((stats.AvgResponseMS * float64(stats.Requests-1)) + latencyMS) / float64(stats.Requests)
		}

		log.Printf("Worker %s completed job in %.2fms (avg: %.2fms, total: %d, failed: %d, uptime: %s)",
			url, latencyMS, stats.AvgResponseMS, stats.JobsCompleted, stats.JobsFailed, time.Since(stats.StartTime).Round(time.Second))
	} else {
		stats.JobsFailed++
		stats.Healthy = false
		log.Printf("Worker %s failed job (total: %d, failed: %d, uptime: %s)",
			url, stats.JobsCompleted, stats.JobsFailed, time.Since(stats.StartTime).Round(time.Second))
	}
}

/*
isError checks if the result is an error message
*/
func isError(result string) bool {
	return len(result) > 5 && (result[:5] == "Error" || result[:5] == "error" || result[:6] == "Worker")
}

/*
SubmitJob adds a job to the worker pool queue
*/
func (p *Pool) SubmitJob(job internal.WorkerJob) {
	p.jobs <- job
}

/*
AddWorker adds a new worker to the pool
*/
func (p *Pool) AddWorker(url string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if worker already exists - don't add them to the pool if they do
	if _, exists := p.workerStats[url]; exists {
		log.Printf("Worker %s already registered", url)
		return
	}

	//initialize stats.
	p.workerStats[url] = &internal.WorkerStats{
		URL:           url,
		JobsCompleted: 0,
		JobsFailed:    0,
		StartTime:     time.Now(),
		AvgResponseMS: 0,
		Requests:      0,
		LastActive:    time.Now(),
	}

	p.workerOrder = append(p.workerOrder, url)
	log.Printf("Added worker: %s (total workers: %d)", url, len(p.workerOrder))
}

/*
RemoveWorker removes a worker from the pool
*/
func (p *Pool) RemoveWorker(url string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove from stats map
	if stats, exists := p.workerStats[url]; exists {
		log.Printf("Removing worker: %s (completed: %d, failed: %d, uptime: %s)",
			url, stats.JobsCompleted, stats.JobsFailed, time.Since(stats.StartTime).Round(time.Second))
		delete(p.workerStats, url)
	}

	// Remove from worker map
	for i, w := range p.workerOrder {
		if w == url {
			p.workerOrder = append(p.workerOrder[:i], p.workerOrder[i+1:]...)
			log.Printf("Worker %s removed (total workers: %d)", url, len(p.workerOrder))
			return
		}
	}
}

/*
GetWorker returns a worker - currently it's specified by URL
returns a URL linked to the worker
*/
func (p *Pool) GetWorker() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.workerOrder) == 0 {
		return "" //error - no workers
	}

	/*
		basically just keep getting the next worker, reset back to first worker when all workers have been handled.
		TODO: Get a list of IDLE workers. Then from there, follow round-robin. But prioritize idle workers.
			Probably lots of clever ways to handle distributing worker load.
			But this implies the worker needs to maintain some kind of state - idle / not idle etc.
	*/
	// Reset nextIdx if it's out of bounds (happens when workers are removed)
	if p.nextIdx >= len(p.workerOrder) {
		p.nextIdx = 0
	}

	worker := p.workerOrder[p.nextIdx]
	p.nextIdx = (p.nextIdx + 1) % len(p.workerOrder)

	return worker
}

/*
GetWorkerCount returns the number of available workers
*/
func (p *Pool) GetWorkerCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.workerOrder)
}

/*
GetWorkerStats returns a copy of all worker statistics
*/
func (p *Pool) GetWorkerStats() map[string]internal.WorkerStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make(map[string]internal.WorkerStats)
	for url, workerStats := range p.workerStats {
		stats[url] = *workerStats
	}
	return stats
}
