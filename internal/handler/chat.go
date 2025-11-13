package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"gollama/internal"
	"gollama/internal/pool"
)

// HandleChat processes chat requests from clients
func HandleChat(p *pool.Pool) http.HandlerFunc {
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
			MaxTokens: 100,
		}

		replyCh := make(chan string)

		job := internal.WorkerJob{
			Request:   llamaReq,
			ReplyCh:   replyCh,
			WorkerURL: p.GetWorkerURL(),
		}
		log.Printf("Worker job processed at URL: %s", job.WorkerURL)

		p.SubmitJob(job)
		reply := <-replyCh

		chatResp := internal.ChatResponse{Reply: reply}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(chatResp)
	}
}
