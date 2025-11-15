package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Reply string `json:"reply"`
}

type TestMetrics struct {
	TotalRequests   int
	SuccessRequests int
	FailedRequests  int
	TotalDuration   time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	AvgLatency      time.Duration
	Latencies       []time.Duration
	mu              sync.Mutex
}

func (m *TestMetrics) AddResult(latency time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests++
	m.Latencies = append(m.Latencies, latency)

	if success {
		m.SuccessRequests++
	} else {
		m.FailedRequests++
	}

	if m.MinLatency == 0 || latency < m.MinLatency {
		m.MinLatency = latency
	}
	if latency > m.MaxLatency {
		m.MaxLatency = latency
	}
}

func (m *TestMetrics) PrintResults() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Latencies) == 0 {
		log.Println("No results to display")
		return
	}

	// Calculate average
	var total time.Duration
	for _, lat := range m.Latencies {
		total += lat
	}
	m.AvgLatency = total / time.Duration(len(m.Latencies))

	// Calculate percentiles (simple approach)
	sorted := make([]time.Duration, len(m.Latencies))
	copy(sorted, m.Latencies)
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	p50 := sorted[len(sorted)/2]
	p95 := sorted[int(float64(len(sorted))*0.95)]
	p99 := sorted[int(float64(len(sorted))*0.99)]

	fmt.Printf("\n=== LOAD TEST RESULTS ===\n")
	fmt.Printf("Total Requests: %d\n", m.TotalRequests)
	fmt.Printf("Successful: %d (%.1f%%)\n", m.SuccessRequests, float64(m.SuccessRequests)/float64(m.TotalRequests)*100)
	fmt.Printf("Failed: %d (%.1f%%)\n", m.FailedRequests, float64(m.FailedRequests)/float64(m.TotalRequests)*100)
	fmt.Printf("Test Duration: %v\n", m.TotalDuration)
	fmt.Printf("Requests/sec: %.1f\n", float64(m.TotalRequests)/m.TotalDuration.Seconds())
	fmt.Printf("\nLatency Stats:\n")
	fmt.Printf("  Min: %v\n", m.MinLatency)
	fmt.Printf("  Avg: %v\n", m.AvgLatency)
	fmt.Printf("  Max: %v\n", m.MaxLatency)
	fmt.Printf("  P50: %v\n", p50)
	fmt.Printf("  P95: %v\n", p95)
	fmt.Printf("  P99: %v\n", p99)
}

func sendChatRequest(url string, message string) (time.Duration, bool) {
	start := time.Now()

	req := ChatRequest{Message: message}
	jsonData, err := json.Marshal(req)
	if err != nil {
		return time.Since(start), false
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return time.Since(start), false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return time.Since(start), false
	}

	var chatResp ChatResponse
	err = json.NewDecoder(resp.Body).Decode(&chatResp)
	if err != nil {
		return time.Since(start), false
	}

	return time.Since(start), true
}

func main() {
	numRequests := flag.Int("requests", 100, "Number of total requests to send")
	concurrency := flag.Int("concurrency", 10, "Number of concurrent requests")
	serverURL := flag.String("url", "http://localhost:9000/chat", "GoLlama server chat endpoint")
	message := flag.String("message", "Hello, this is a load test", "Message to send in each request")
	flag.Parse()

	log.Printf("Starting load test with %d requests, %d concurrent", *numRequests, *concurrency)
	log.Printf("Target: %s", *serverURL)

	metrics := &TestMetrics{}
	startTime := time.Now()

	// Channel to control concurrency
	sem := make(chan struct{}, *concurrency)
	var wg sync.WaitGroup

	for i := 0; i < *numRequests; i++ {
		wg.Add(1)
		go func(reqNum int) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			latency, success := sendChatRequest(*serverURL, fmt.Sprintf("%s #%d", *message, reqNum))
			metrics.AddResult(latency, success)

			if reqNum%50 == 0 {
				log.Printf("Completed request %d", reqNum)
			}
		}(i)
	}

	wg.Wait()
	metrics.TotalDuration = time.Since(startTime)
	metrics.PrintResults()
}
