package handler

import (
	"encoding/json"
	"net/http"

	"gollama/internal/pool"
)

/*
WorkerInfo is the payload for registering a new worker
*/
type WorkerInfo struct {
	URL   string `json:"url"`
	Model string `json:"model"`
}

/*
HandleConnectWorker allows new llama.cpp instances to register with GoLlama
*/
func HandleConnectWorker(p *pool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var workerInfo WorkerInfo
		err := json.NewDecoder(r.Body).Decode(&workerInfo)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		//worker already did health check - should be OK for now
		p.AddWorker(workerInfo.URL)

		w.Header().Set("Content-Type", "application/json")
		response := map[string]string{
			"status": "registered",
			"url":    workerInfo.URL,
		}
		_ = json.NewEncoder(w).Encode(response)
	}
}
