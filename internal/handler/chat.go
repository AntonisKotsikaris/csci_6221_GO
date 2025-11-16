package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"gollama/internal"
	"gollama/internal/pool"
)

// HandleChat processes chat requests from clients
func HandleChat(p *pool.Pool, defaultMaxTokens int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var chatReq internal.ChatRequest
		err := json.NewDecoder(r.Body).Decode(&chatReq)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		if p.GetWorkerCount() == 0 {
			http.Error(w, "No workers available", http.StatusServiceUnavailable)
			return
		}

		log.Printf("Received message: %s", chatReq.Message)

		llamaReq := internal.LlamaRequest{
			Messages: []internal.Message{
				{Role: "user", Content: chatReq.Message},
			},
			MaxTokens: defaultMaxTokens,
		}

		replyCh := make(chan string)

		job := internal.WorkerJob{
			Request:    llamaReq,
			ReplyCh:    replyCh,
			WorkerURL:  p.GetWorker(),
			RetryCount: 0,
			MaxRetries: p.GetMaxRetries(),
		}
		log.Printf("Worker job processed at URL: %s", job.WorkerURL)

		p.SubmitJob(job)
		reply := <-replyCh //must wait for reply from the job reply channel

		chatResp := internal.ChatResponse{Reply: reply}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(chatResp)
	}
}
