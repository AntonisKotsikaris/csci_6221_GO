package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// Credential represents a user credential from the database
type Credential struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	User     string `json:"user"`
}

// CredentialStore manages loading and validating credentials
type CredentialStore struct {
	credentials map[string]Credential
}

// NewCredentialStore loads credentials from the auth.json file
func NewCredentialStore(filePath string) (*CredentialStore, error) {
	cs := &CredentialStore{
		credentials: make(map[string]Credential),
	}

	err := cs.loadCredentials(filePath)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

// loadCredentials reads and parses the auth.json file
func (cs *CredentialStore) loadCredentials(filePath string) error {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read credentials file: %w", err)
	}

	var creds []Credential
	err = json.Unmarshal(file, &creds)
	if err != nil {
		return fmt.Errorf("failed to parse credentials file: %w", err)
	}

	// Store credentials by username for quick lookup
	for _, cred := range creds {
		cs.credentials[cred.User] = cred
	}

	log.Printf("Loaded %d credentials from %s", len(creds), filePath)
	return nil
}

// ValidateCredentials checks if the provided username and password are valid
func (cs *CredentialStore) ValidateCredentials(username, password string) bool {
	cred, exists := cs.credentials[username]
	if !exists {
		log.Printf("User not found: %s", username)
		return false
	}

	if cred.Password != password {
		log.Printf("Invalid password for user: %s", username)
		return false
	}

	log.Printf("Credentials validated for user: %s", username)
	return true
}

// GetUser returns the credential for a given username
func (cs *CredentialStore) GetUser(username string) (Credential, bool) {
	cred, exists := cs.credentials[username]
	return cred, exists
}
