package pool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"gollama/internal"
)

/*
Pool holds the last-known pool of workers. Workers are verified, used, or discarded as they're called upon by the
jobProcessor.
*/
type Pool struct {
	jobs    chan internal.WorkerJob //job queue
	workers []string                // available worker URLs
	mu      sync.RWMutex            // Protects workers slice during concurrent calls
	nextIdx int
}

/*
New creates a new worker pool
*/
func New(queueSize int) *Pool {
	return &Pool{
		jobs:    make(chan internal.WorkerJob, queueSize),
		workers: make([]string, 0),
	}
}

/*
Start initializes the worker pool
*/
func (p *Pool) Start() {
	for i := 1; i <= 10; i++ { // 10 concurrent job processors
		go p.jobProcessor(i)
	}
	log.Println("Worker pool initialized with 10 job processors")
}

/*
jobProcessor handles jobs and manages worker health
*/
func (p *Pool) jobProcessor(id int) {
	for job := range p.jobs {
		log.Printf("[Processor %d] Processing job with worker %s", id, job.WorkerURL)

		result := p.callWorker(job.WorkerURL, job.Request)

		if isError(result) {
			log.Printf("[Processor %d] Worker %s failed health check, removing from pool", id, job.WorkerURL)
			p.RemoveWorker(job.WorkerURL)
			//TODO: Now that the worker failed, re-run the job!!
		}

		job.ReplyCh <- result
	}
}

/*
callWorker sends an inference request to a worker via its standardized /execute endpoint
*/
func (p *Pool) callWorker(workerURL string, req internal.LlamaRequest) string {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Sprintf("Error marshaling request: %v", err)
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
		return fmt.Sprintf("Error marshaling execute request: %v", err)
	}

	resp, err := http.Post(
		fmt.Sprintf("%s/execute", workerURL),
		"application/json",
		bytes.NewBuffer(executePayload),
	)
	if err != nil {
		return fmt.Sprintf("Error contacting worker: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error reading response: %v", err)
	}

	var workerResp internal.LlamaResponse
	err = json.Unmarshal(body, &workerResp)
	if err != nil {
		return fmt.Sprintf("Error parsing response: %v", err)
	}

	if workerResp.Error != "" {
		return fmt.Sprintf("Worker error: %s", workerResp.Error)
	}

	if len(workerResp.Choices) == 0 {
		return "Worker error: no choices in response"
	}

	return workerResp.Choices[0].Message.Content
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

	// look for identical URL - for now that's our marker for a unique worker
	for _, w := range p.workers {
		if w == url {
			log.Printf("Worker %s already registered", url)
			return
		}
	}

	p.workers = append(p.workers, url)
	log.Printf("Added worker: %s (total workers: %d)", url, len(p.workers))
}

/*
RemoveWorker removes a worker from the pool
*/
func (p *Pool) RemoveWorker(url string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, w := range p.workers {
		if w == url {
			p.workers = append(p.workers[:i], p.workers[i+1:]...)
			log.Printf("Removed worker: %s (total workers: %d)", url, len(p.workers))
			return
		}
	}
}

/*
GetWorkerURL returns a worker URL
*/
func (p *Pool) GetWorkerURL() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.workers) == 0 {
		return "" //error - no workers
	}

	/*
		basically just keep getting the next worker, reset back to first worker when all workers have been handled.
		TODO: Get a list of IDLE workers. Then from there, follow round-robin. But prioritize idle workers.
			Probably lots of clever ways to handle distributing worker load.
			But this implies the client needs to maintain some kind of state - idle / not idle etc.
	*/
	worker := p.workers[p.nextIdx]
	p.nextIdx = (p.nextIdx + 1) % len(p.workers)

	return worker
}

/*
GetWorkerCount returns the number of available workers
*/
func (p *Pool) GetWorkerCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.workers)
}
