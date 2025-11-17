package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"gollama/internal"
	"gollama/internal/pool"
)

// HandleTranslate processes translation requests from clients
func HandleTranslate(p *pool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var transReq internal.TranslateRequest
		err := json.NewDecoder(r.Body).Decode(&transReq)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		if transReq.Text == "" {
			http.Error(w, "Text field is required", http.StatusBadRequest)
			return
		}

		if transReq.Language == "" {
			http.Error(w, "Language field is required", http.StatusBadRequest)
			return
		}

		if p.GetWorkerCount() == 0 {
			http.Error(w, "No workers available", http.StatusServiceUnavailable)
			return
		}

		log.Printf("Received translate request to %s for text: %s...", transReq.Language, truncateText(transReq.Text, 50))

		// Construct the translation prompt
		prompt := fmt.Sprintf("Translate the following text to %s:\n\n%s", transReq.Language, transReq.Text)

		llamaReq := internal.LlamaRequest{
			Messages: []internal.Message{
				{Role: "user", Content: prompt},
			},
			MaxTokens: 200,
		}

		replyCh := make(chan string)

		job := internal.WorkerJob{
			Request:    llamaReq,
			ReplyCh:    replyCh,
			WorkerURL:  p.GetWorker(),
			RetryCount: 0,
			MaxRetries: 3,
		}
		log.Printf("Translate job assigned to worker: %s", job.WorkerURL)

		p.SubmitJob(job)
		reply := <-replyCh

		transResp := internal.TranslateResponse{Translation: reply}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(transResp)
	}
}

// truncateText is a helper function to truncate long strings for logging
func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
