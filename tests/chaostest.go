package main

//chaostest
import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Reply string `json:"reply"`
}

type TestResults struct {
	TotalRequests  int
	SuccessfulReqs int
	FailedReqs     int
	WorkersSpawned int
	WorkersKilled  int
	TestDuration   time.Duration
	mu             sync.Mutex
}

func (r *TestResults) AddResult(success bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.TotalRequests++
	if success {
		r.SuccessfulReqs++
	} else {
		r.FailedReqs++
	}
}

func (r *TestResults) AddWorkerSpawned() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.WorkersSpawned++
}

func (r *TestResults) AddWorkerKilled() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.WorkersKilled++
}

func (r *TestResults) PrintResults() {
	r.mu.Lock()
	defer r.mu.Unlock()

	fmt.Printf("\n=== CHAOS TEST RESULTS ===\n")
	fmt.Printf("Test Duration: %v\n", r.TestDuration)
	fmt.Printf("Total Requests: %d\n", r.TotalRequests)
	fmt.Printf("Successful: %d (%.1f%%)\n", r.SuccessfulReqs, float64(r.SuccessfulReqs)/float64(r.TotalRequests)*100)
	fmt.Printf("Failed: %d (%.1f%%)\n", r.FailedReqs, float64(r.FailedReqs)/float64(r.TotalRequests)*100)
	fmt.Printf("Workers Spawned: %d\n", r.WorkersSpawned)
	fmt.Printf("Workers Killed: %d\n", r.WorkersKilled)

	if r.TotalRequests > 0 {
		fmt.Printf("Requests/sec: %.1f\n", float64(r.TotalRequests)/r.TestDuration.Seconds())
	}
}

type Worker struct {
	Port      int
	LlamaPort int
	Cmd       *exec.Cmd
	Running   bool
	mu        sync.Mutex
}

func NewWorker(port, llamaPort int) *Worker {
	return &Worker{
		Port:      port,
		LlamaPort: llamaPort,
		Running:   false,
	}
}

func (w *Worker) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.Running {
		return fmt.Errorf("worker already running on port %d", w.Port)
	}

	// Need to run from project root directory
	w.Cmd = exec.Command("go", "run", "../cmd/client/main.go",
		"-port", fmt.Sprintf("%d", w.Port),
		"-llama-port", fmt.Sprintf("%d", w.LlamaPort))
	w.Cmd.Dir = ".." // Set working directory to project root

	// Redirect output to avoid cluttering test logs
	w.Cmd.Stdout = nil
	w.Cmd.Stderr = nil

	err := w.Cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start worker on port %d: %v", w.Port, err)
	}

	w.Running = true
	log.Printf("Started worker on port %d (llama.cpp: %d)", w.Port, w.LlamaPort)

	// Give worker time to start up and connect
	time.Sleep(3 * time.Second)
	return nil
}

func (w *Worker) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.Running || w.Cmd == nil {
		return nil
	}

	err := w.Cmd.Process.Kill()
	if err != nil {
		return fmt.Errorf("failed to kill worker: %v", err)
	}

	w.Cmd.Wait() // Clean up
	w.Running = false
	w.Cmd = nil
	log.Printf("Killed worker on port %d", w.Port)
	return nil
}

func (w *Worker) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.Running
}

type WorkerManager struct {
	Workers []*Worker
	mu      sync.Mutex
}

func NewWorkerManager() *WorkerManager {
	llamaPorts := []int{8080, 8081, 8082}
	workers := make([]*Worker, 6) // Support up to 6 workers

	for i := 0; i < 6; i++ {
		port := 9001 + i
		llamaPort := llamaPorts[i%len(llamaPorts)] // Distribute across llama instances
		workers[i] = NewWorker(port, llamaPort)
	}

	return &WorkerManager{Workers: workers}
}

func (wm *WorkerManager) SpawnRandomWorker() error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Find a stopped worker to start
	for _, worker := range wm.Workers {
		if !worker.IsRunning() {
			return worker.Start()
		}
	}
	return fmt.Errorf("all workers already running")
}

func (wm *WorkerManager) KillRandomWorker() error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Find running workers
	var runningWorkers []*Worker
	for _, worker := range wm.Workers {
		if worker.IsRunning() {
			runningWorkers = append(runningWorkers, worker)
		}
	}

	if len(runningWorkers) == 0 {
		return fmt.Errorf("no workers to kill")
	}

	// Kill a random running worker
	victim := runningWorkers[rand.Intn(len(runningWorkers))]
	return victim.Stop()
}

func (wm *WorkerManager) GetRunningCount() int {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	count := 0
	for _, worker := range wm.Workers {
		if worker.IsRunning() {
			count++
		}
	}
	return count
}

func (wm *WorkerManager) StopAll() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	for _, worker := range wm.Workers {
		worker.Stop()
	}
}

func sendChatRequest(url, message string) bool {
	req := ChatRequest{Message: message}
	jsonData, err := json.Marshal(req)
	if err != nil {
		return false
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func chaosWorkerManager(wm *WorkerManager, results *TestResults, duration time.Duration) {
	ticker := time.NewTicker(5 * time.Second) // Chaos every 5 seconds
	defer ticker.Stop()

	timeout := time.After(duration)

	for {
		select {
		case <-timeout:
			return
		case <-ticker.C:
			running := wm.GetRunningCount()

			// Chaos decision logic
			if running == 0 {
				// Always spawn if no workers
				if err := wm.SpawnRandomWorker(); err == nil {
					results.AddWorkerSpawned()
				}
			} else if running >= 4 {
				// Kill if too many workers
				if err := wm.KillRandomWorker(); err == nil {
					results.AddWorkerKilled()
				}
			} else {
				// Random chaos
				if rand.Float64() < 0.6 { // 60% chance to spawn
					if err := wm.SpawnRandomWorker(); err == nil {
						results.AddWorkerSpawned()
					}
				} else { // 40% chance to kill
					if err := wm.KillRandomWorker(); err == nil {
						results.AddWorkerKilled()
					}
				}
			}
		}
	}
}

func requestSender(serverURL string, results *TestResults, duration time.Duration, reqInterval time.Duration) {
	ticker := time.NewTicker(reqInterval)
	defer ticker.Stop()

	timeout := time.After(duration)
	counter := 0

	for {
		select {
		case <-timeout:
			return
		case <-ticker.C:
			counter++
			message := fmt.Sprintf("Chaos test message #%d", counter)
			success := sendChatRequest(serverURL, message)
			results.AddResult(success)

			if counter%10 == 0 {
				log.Printf("Sent %d requests", counter)
			}
		}
	}
}

func main() {
	duration := flag.Duration("duration", 60*time.Second, "Duration of chaos test")
	serverURL := flag.String("url", "http://localhost:9000/chat", "GoLlama server chat endpoint")
	reqInterval := flag.Duration("interval", 2*time.Second, "Interval between requests")
	initialWorkers := flag.Int("initial", 2, "Initial number of workers to start")
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	log.Printf("Starting chaos test for %v", *duration)
	log.Printf("Target: %s", *serverURL)
	log.Printf("Request interval: %v", *reqInterval)

	results := &TestResults{}
	wm := NewWorkerManager()

	// Ensure cleanup
	defer wm.StopAll()

	// Start initial workers
	log.Printf("Starting %d initial workers...", *initialWorkers)
	for i := 0; i < *initialWorkers && i < len(wm.Workers); i++ {
		if err := wm.Workers[i].Start(); err != nil {
			log.Printf("Failed to start initial worker %d: %v", i, err)
		} else {
			results.AddWorkerSpawned()
		}
	}

	startTime := time.Now()

	// Start chaos goroutines
	var wg sync.WaitGroup

	// Worker chaos manager
	wg.Add(1)
	go func() {
		defer wg.Done()
		chaosWorkerManager(wm, results, *duration)
	}()

	// Request sender
	wg.Add(1)
	go func() {
		defer wg.Done()
		requestSender(*serverURL, results, *duration, *reqInterval)
	}()

	// Wait for test to complete
	wg.Wait()

	results.TestDuration = time.Since(startTime)
	results.PrintResults()

	log.Println("Chaos test completed!")
}
