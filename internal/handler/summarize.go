package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"gollama/internal"
	"gollama/internal/pool"
)

// HandleSummarize processes summarization requests from clients
func HandleSummarize(p *pool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var sumReq internal.SummarizeRequest
		err := json.NewDecoder(r.Body).Decode(&sumReq)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		if sumReq.Text == "" {
			http.Error(w, "Text field is required", http.StatusBadRequest)
			return
		}

		if p.GetWorkerCount() == 0 {
			http.Error(w, "No workers available", http.StatusServiceUnavailable)
			return
		}

		log.Printf("Received summarize request for text: %s...", truncate(sumReq.Text, 50))

		// Construct the summarization prompt
		prompt := fmt.Sprintf("Summarize the following text in a concise manner:\n\n%s", sumReq.Text)

		llamaReq := internal.LlamaRequest{
			Messages: []internal.Message{
				{Role: "user", Content: prompt},
			},
			MaxTokens: 150,
		}

		replyCh := make(chan string)

		job := internal.WorkerJob{
			Request:    llamaReq,
			ReplyCh:    replyCh,
			WorkerURL:  p.GetWorker(),
			RetryCount: 0,
			MaxRetries: 3,
		}
		log.Printf("Summarize job assigned to worker: %s", job.WorkerURL)

		p.SubmitJob(job)
		reply := <-replyCh

		sumResp := internal.SummarizeResponse{Summary: reply}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sumResp)
	}
}

// truncate is a helper function to truncate long strings for logging
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
