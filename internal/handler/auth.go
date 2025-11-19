package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"gollama/internal/auth"
)

// Global credential store (initialized in main)
var credStore *auth.CredentialStore

// InitAuth initializes the credential store
func InitAuth(credentialFilePath string) error {
	var err error
	credStore, err = auth.NewCredentialStore(credentialFilePath)
	return err
}

// HandleGetToken issues a JWT token for a worker with valid credentials
func HandleGetToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			WorkerID string `json:"worker_id"`
			URL      string `json:"url"`
			Username string `json:"username"`
			Password string `json:"password"`
		}

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			log.Printf("Invalid token request: %v", err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		if req.WorkerID == "" || req.URL == "" || req.Username == "" || req.Password == "" {
			http.Error(w, "Missing worker_id, url, username, or password", http.StatusBadRequest)
			return
		}

		// Validate credentials
		if !credStore.ValidateCredentials(req.Username, req.Password) {
			log.Printf("Authentication failed for worker: %s", req.WorkerID)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Get user info
		user, _ := credStore.GetUser(req.Username)

		// Generate token
		token, err := auth.GenerateToken(req.WorkerID, req.URL, req.Username, user.Email, 24)
		if err != nil {
			log.Printf("Token generation failed: %v", err)
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"token": token,
		})

		log.Printf("Token issued for worker: %s (user: %s)", req.WorkerID, req.Username)
	}
}
