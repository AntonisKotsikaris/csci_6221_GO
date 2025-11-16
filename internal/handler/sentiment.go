package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"gollama/internal"
	"gollama/internal/pool"
)

// HandleSentiment processes sentiment analysis requests from clients
func HandleSentiment(p *pool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var sentReq internal.SentimentRequest
		err := json.NewDecoder(r.Body).Decode(&sentReq)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		if sentReq.Text == "" {
			http.Error(w, "Text field is required", http.StatusBadRequest)
			return
		}

		if p.GetWorkerCount() == 0 {
			http.Error(w, "No workers available", http.StatusServiceUnavailable)
			return
		}

		log.Printf("Received sentiment analysis request for text: %s...", truncateString(sentReq.Text, 50))

		// Construct the sentiment analysis prompt
		prompt := fmt.Sprintf("Analyze the sentiment of the following text and respond with only one word: positive, negative, or neutral.\n\nText: %s\n\nSentiment:", sentReq.Text)

		llamaReq := internal.LlamaRequest{
			Messages: []internal.Message{
				{Role: "user", Content: prompt},
			},
			MaxTokens: 50,
		}

		replyCh := make(chan string)

		job := internal.WorkerJob{
			Request:    llamaReq,
			ReplyCh:    replyCh,
			WorkerURL:  p.GetWorker(),
			RetryCount: 0,
			MaxRetries: 3,
		}
		log.Printf("Sentiment job assigned to worker: %s", job.WorkerURL)

		p.SubmitJob(job)
		reply := <-replyCh

		sentResp := internal.SentimentResponse{Sentiment: reply}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sentResp)
	}
}

// truncateString is a helper function to truncate long strings for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
