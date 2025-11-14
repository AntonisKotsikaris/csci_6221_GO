package pool

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"
)

type Status_W string

const (
	StatusUnknown   Status_W = "unknown"
	StatusHealthy   Status_W = "healthy"
	StatusUnhealthy Status_W = "unhealthy"
)

// structure of worker
type Worker struct {
	URL           string    `json:"url"`
	Model         string    `json:"model"`
	Status        Status_W  `json:"status"`
	Busy          bool      `json:"busy"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`

	FailedPings   int    `json:"-"`
	TotalJobs     int    `json:"totalJobs"`
	TotalFailures int    `json:"totalFailures"`
	LastError     string `json:"lastError"`
}

// units of work to be executed
type Job_W struct {
	Endpoint     string
	Body         any
	ResponseChan chan []byte
	ErrorChan    chan error
	Ctx          context.Context
}

// statistcs
type Stats struct {
	TotalWorkers   int       `json:"totalWorkers"`
	HealthyWorkers int       `json:"healthyWorkers"`
	BusyWorkers    int       `json:"busyWorkers"`
	PendingJobs    int       `json:"pendingJobs"`
	Workers        []*Worker `json:"workers"`
}

//pool struct

type Pool struct {
	mu                sync.Mutex
	workers           []*Worker
	jobChan           chan *Job_W
	httpClient        *http.Client
	heartbeatInterval time.Duration
	maxFailedPings    int

	stopCh    chan struct{}
	onceStart sync.Once
	onceStop  sync.Once
}

// New creates a new pool with a bounded job queue.
func New(queueSize int) *Pool {
	return &Pool{
		workers:           make([]*Worker, 0),
		jobChan:           make(chan *Job_W, queueSize),
		httpClient:        &http.Client{Timeout: 15 * time.Second},
		heartbeatInterval: 5 * time.Second,
		maxFailedPings:    3,
		stopCh:            make(chan struct{}),
	}
}

// Start launches background goroutines (job loop + heartbeat loop).
func (p *Pool) Start() {
	p.onceStart.Do(func() {
		go p.jobLoop()
		go p.heartbeatLoop()
	})
}

// Stop gracefully stops the pool.
func (p *Pool) Stop() {
	p.onceStop.Do(func() {
		close(p.stopCh)
	})
}

// AddWorker is called by /connectWorker to register or refresh a worker.
func (p *Pool) AddWorker(url, model string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// If worker already exists, refresh it.
	for _, w := range p.workers {
		if w.URL == url {
			w.Model = model
			w.Status = StatusHealthy
			w.Busy = false
			w.LastHeartbeat = time.Now()
			w.FailedPings = 0
			return
		}
	}

	w := &Worker{
		URL:           url,
		Model:         model,
		Status:        StatusHealthy,
		Busy:          false,
		LastHeartbeat: time.Now(),
	}
	p.workers = append(p.workers, w)
}

// EnqueueJob puts a job into the queue.
func (p *Pool) EnqueueJob(job *Job_W) error {
	select {
	case p.jobChan <- job:
		return nil
	case <-job.Ctx.Done():
		return job.Ctx.Err()
	}
}

// GetStats returns a snapshot of pool state.
func (p *Pool) GetStats() Stats {
	p.mu.Lock()
	defer p.mu.Unlock()

	stats := Stats{
		TotalWorkers: len(p.workers),
		PendingJobs:  len(p.jobChan),
		Workers:      make([]*Worker, 0, len(p.workers)),
	}

	for _, w := range p.workers {
		if w.Status == StatusHealthy {
			stats.HealthyWorkers++
		}
		if w.Busy {
			stats.BusyWorkers++
		}
		stats.Workers = append(stats.Workers, w)
	}

	return stats
}

func (p *Pool) jobLoop() {
	for {
		select {
		case <-p.stopCh:
			return
		case job := <-p.jobChan:
			p.handleJob(job)
		}
	}
}

func (p *Pool) handleJob(job *Job_W) {
	worker := p.pickAvailableWorker()
	if worker == nil {
		job.ErrorChan <- errors.New("no healthy workers available")
		return
	}

	// Mark worker busy.
	p.mu.Lock()
	worker.Busy = true
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		worker.Busy = false
		p.mu.Unlock()
	}()

	// Prepare request to worker: POST <workerURL>/execute
	payload := map[string]any{
		"endpoint": job.Endpoint,
		"body":     job.Body,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		job.ErrorChan <- err
		return
	}

	req, err := http.NewRequestWithContext(job.Ctx, http.MethodPost, worker.URL+"/execute", bytes.NewReader(data))
	if err != nil {
		job.ErrorChan <- err
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.markFailure(worker, err.Error())
		job.ErrorChan <- err
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		p.markFailure(worker, resp.Status)
		job.ErrorChan <- errors.New("worker returned status " + resp.Status)
		return
	}

	buf := new(bytes.Buffer)
	if _, err = buf.ReadFrom(resp.Body); err != nil {
		p.markFailure(worker, err.Error())
		job.ErrorChan <- err
		return
	}

	p.markSuccess(worker)
	job.ResponseChan <- buf.Bytes()
}

func (p *Pool) heartbeatLoop() {
	ticker := time.NewTicker(p.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.checkWorkers()
		}
	}
}

func (p *Pool) checkWorkers() {
	p.mu.Lock()
	defer p.mu.Unlock()

	alive := p.workers[:0] // reuse underlying slice

	for _, w := range p.workers {
		ok := p.pingWorker(w)
		if ok {
			alive = append(alive, w)
		}
		// if !ok, we drop it by not appending
	}

	p.workers = alive
}

func (p *Pool) pingWorker(w *Worker) bool {
	resp, err := p.httpClient.Get(w.URL + "/health")
	if err != nil {
		w.FailedPings++
		if w.FailedPings >= p.maxFailedPings {
			w.Status = StatusUnhealthy
		}
		return w.Status == StatusHealthy
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		w.Status = StatusHealthy
		w.FailedPings = 0
		w.LastHeartbeat = time.Now()
		return true
	}

	w.FailedPings++
	if w.FailedPings >= p.maxFailedPings {
		w.Status = StatusUnhealthy
	}
	return w.Status == StatusHealthy
}

// pickAvailableWorker returns a healthy, not-busy worker (simple scan + wait).
func (p *Pool) pickAvailableWorker() *Worker {
	for i := 0; i < 1000; i++ { // crude cap to avoid infinite loop
		p.mu.Lock()
		var chosen *Worker
		for _, w := range p.workers {
			if w.Status == StatusHealthy && !w.Busy {
				chosen = w
				break
			}
		}
		p.mu.Unlock()

		if chosen != nil {
			return chosen
		}

		// Everyone is busy at the moment, wait a bit.
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

func (p *Pool) markSuccess(w *Worker) {
	p.mu.Lock()
	defer p.mu.Unlock()
	w.TotalJobs++
	w.LastError = ""
	w.Status = StatusHealthy
}

func (p *Pool) markFailure(w *Worker, msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	w.TotalJobs++
	w.TotalFailures++
	w.LastError = msg
}
