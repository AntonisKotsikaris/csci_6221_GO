package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"gollama/internal/pool"
)

type chat_req struct {
	Message string `json:"message"`
}

type chat_resp struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func HandleChat(p *pool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req chat_req
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if req.Message == "" {
			http.Error(w, "message is required", http.StatusBadRequest)
			return
		}

		// Build the Llama.cpp chat request body.
		llamaBody := map[string]any{
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": req.Message,
				},
			},
			"max_tokens": 256,
		}

		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()

		job := &pool.Job_W{
			Endpoint:     "/v1/chat/completions",
			Body:         llamaBody,
			ResponseChan: make(chan []byte, 1),
			ErrorChan:    make(chan error, 1),
			Ctx:          ctx,
		}

		if err := p.EnqueueJob(job); err != nil {
			http.Error(w, "failed to enqueue job: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		var (
			respBytes []byte
			err       error
		)

		select {
		case respBytes = <-job.ResponseChan:
			// ok
		case err = <-job.ErrorChan:
			http.Error(w, "worker error: "+err.Error(), http.StatusBadGateway)
			return
		case <-ctx.Done():
			http.Error(w, "timeout waiting for worker", http.StatusGatewayTimeout)
			return
		}

		var response chat_resp
		if err := json.Unmarshal(respBytes, &response); err != nil {
			http.Error(w, "invalid llama response: "+err.Error(), http.StatusBadGateway)
			return
		}

		if len(response.Choices) == 0 {
			http.Error(w, "llama response had no choices", http.StatusBadGateway)
			return
		}

		reply := response.Choices[0].Message.Content

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Reply string `json:"reply"`
		}{
			Reply: reply,
		})
	}
}
